// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package web_client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	web_api "github.com/rapidaai/protos"
)

const (
	scopeAuthorizationCacheVersion = "v2"
	scopeAuthorizationCacheTTL     = 5 * time.Minute
)

var (
	authActorCacheMeter, _ = otel.Meter("github.com/rapidaai/pkg/clients/web/auth").Int64Counter("rapida.auth.actor_cache")
)

type authServiceClient struct {
	clients.InternalClient
	cfg        *config.AppConfig
	logger     commons.Logger
	authClient web_api.AuthenticationServiceClient
}

type AuthClient interface {
	Authorize(ctx context.Context, authToken string, userId uint64) (*web_api.Authentication, error)
	ScopeAuthorize(c context.Context, scopeToken string, scopeType string) (*web_api.ScopedAuthentication, error)
}

func NewAuthenticator(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) AuthClient {
	conn, err := grpc.NewClient(config.Web.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	authClient := web_api.NewAuthenticationServiceClient(conn)
	return &authServiceClient{
		clients.NewInternalClient(config, logger, redis),
		config,
		logger,
		authClient,
	}
}

// Authorize implements types.Authenticator.
func (client *authServiceClient) Authorize(c context.Context, authToken string, userId uint64) (*web_api.Authentication, error) {
	start := time.Now()
	// Generate cache key
	cacheKey := client.CacheKey(c, "Authorize", authToken, fmt.Sprintf("%d", userId))

	// Retrieve data from cache
	cachedValue := client.Retrieve(c, cacheKey)

	// Initialize data variable
	data := &web_api.Authentication{}
	// Parse cached value into data
	err := cachedValue.ResultStruct(data)
	if err != nil {
		client.logger.Errorf("Failed to parse cached data: %v", err)

		// Call the vault service to fetch data
		res, err := client.authClient.Authorize(client.WithToken(c, authToken, userId), &web_api.AuthorizeRequest{})
		if err != nil {
			client.logger.Errorf("Failed to get credentials from auth service: %v", err)
			return nil, err
		}

		// Check if the request was successful
		if res.GetSuccess() && res.GetData() != nil {
			// Cache the fetched data
			_c := client.Cache(c, cacheKey, res.GetData())
			if _c.HasError() {
				client.logger.Errorf("Failed to cache the data %+v: %v", res.GetData(), _c.Err)
			}
			client.logger.Benchmark("Benchmarking: AuthClient.ScopeAuthorize", time.Since(start))
			return res.GetData(), nil
		}

		// Handle error response from vault service
		if res.GetError() != nil {
			errMsg := fmt.Sprintf("Failed to get credentials from vault service: %s", res.GetError().HumanMessage)
			client.logger.Errorf(errMsg)
			return nil, errors.New(errMsg)
		}
	}

	// Log benchmarking information
	client.logger.Benchmark("Benchmarking: AuthClient.ScopeAuthorize", time.Since(start))
	return data, nil
}

func (client *authServiceClient) ScopeAuthorize(c context.Context, scopeToken string, scopeType string) (*web_api.ScopedAuthentication, error) {
	start := time.Now()
	cacheKey := client.scopeAuthorizationCacheKey(c, scopeToken, scopeType)

	// Retrieve data from cache
	cachedValue := client.Retrieve(c, cacheKey)

	// Initialize data variable
	data := &web_api.ScopedAuthentication{}
	// Parse cached value into data
	err := cachedValue.ResultStruct(data)
	if err == nil {
		err = validateScopedAuthentication(data, scopeType)
	}
	if err != nil {
		recordActorCacheResult(c, client.cfg, scopeType, "miss")
		client.logger.Errorf("Failed to parse cached data: %v", err)

		// Call the vault service to fetch data
		res, err := client.authClient.ScopeAuthorize(client.WithScopeToken(c, scopeToken, scopeType), &web_api.ScopeAuthorizeRequest{
			Scope: scopeType,
		})
		if err != nil {
			client.logger.Errorf("Failed to get credentials from vault service: %v", err)
			return nil, err
		}

		// Check if the request was successful
		if res.GetSuccess() && res.GetData() != nil {
			if err := validateScopedAuthentication(res.GetData(), scopeType); err != nil {
				return nil, err
			}
			// Cache the fetched data
			_c := client.CacheWithTTL(c, cacheKey, res.GetData(), scopeAuthorizationCacheTTL)
			if _c.HasError() {
				client.logger.Errorf("Failed to cache the data %+v: %v", res.GetData(), _c.Err)
			}
			client.logger.Benchmark("Benchmarking: AuthClient.ScopeAuthorize", time.Since(start))
			return res.GetData(), nil
		}

		// Handle error response from vault service
		if res.GetError() != nil {
			errMsg := fmt.Sprintf("Failed to get credentials from vault service: %s", res.GetError().HumanMessage)
			client.logger.Errorf(errMsg)
			return nil, errors.New(errMsg)
		}
	}
	if err == nil {
		recordActorCacheResult(c, client.cfg, scopeType, "hit")
	}

	// Log benchmarking information
	client.logger.Benchmark("Benchmarking: AuthClient.ScopeAuthorize", time.Since(start))
	return data, nil
}

func recordActorCacheResult(ctx context.Context, cfg *config.AppConfig, scopeType, result string) {
	if authActorCacheMeter == nil {
		return
	}
	serviceName := "unknown"
	if cfg != nil && strings.TrimSpace(cfg.Name) != "" {
		serviceName = strings.TrimSpace(cfg.Name)
	}
	authActorCacheMeter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("auth.scope_type", scopeType),
		attribute.String("cache.result", result),
	))
}

func validateScopedAuthentication(authentication *web_api.ScopedAuthentication, scopeType string) error {
	if scopeType == "project" && (authentication.GetActorType() != string(types.ActorTypeProject) || authentication.GetActorId() == "") {
		return errors.New("project authentication is missing durable actor identity")
	}
	return nil
}

func (client *authServiceClient) scopeAuthorizationCacheKey(c context.Context, scopeToken string, scopeType string) string {
	mac := hmac.New(sha256.New, []byte(client.cfg.Secret))
	_, _ = mac.Write([]byte(scopeToken))
	fingerprint := hex.EncodeToString(mac.Sum(nil))
	return client.CacheKey(c, "ScopeAuthorize", scopeAuthorizationCacheVersion, scopeType, fingerprint)
}
