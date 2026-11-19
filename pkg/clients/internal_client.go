// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package clients

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

type InternalClient interface {
	WithPlatform(ctx context.Context, principal authentication.Principal) (context.Context, error)
	WithAuth(ctx context.Context, principal authentication.Principal) (context.Context, error)
	WithHttpAuth(c context.Context, principal authentication.Principal, req *http.Request) (*http.Request, error)

	WithToken(ctx context.Context, token string, userId uint64) context.Context
	WithScopeToken(c context.Context, token string, scope string) context.Context

	Cache(c context.Context, key string, value interface{}) *connectors.RedisResponse
	CacheWithTTL(c context.Context, key string, value interface{}, ttl time.Duration) *connectors.RedisResponse
	Retrieve(c context.Context, key string) *connectors.RedisResponse
	CacheKey(c context.Context, funcName string, key ...string) string
}

// ServiceInternalClient can authenticate the local Service without delegation.
type ServiceInternalClient interface {
	InternalClient
	WithService(ctx context.Context) (context.Context, error)
}

type internalClient struct {
	issuer    *authentication_jwt.Issuer
	issuerErr error
	logger    commons.Logger
	redis     connectors.RedisConnector
}

// NewInternalClient constructs an outbound client for exactly one destination.
func NewInternalClient(cfg *config.AppConfig, destination string, logger commons.Logger, redis connectors.RedisConnector) ServiceInternalClient {
	issuer, err := authentication_jwt.NewIssuer(destination, cfg.ServiceAuthenticationKey())
	return &internalClient{issuer: issuer, issuerErr: err, logger: logger, redis: redis}
}

// NewInternalClientWithIssuer constructs an outbound client from a command-owned issuer.
func NewInternalClientWithIssuer(issuer *authentication_jwt.Issuer, logger commons.Logger, redis connectors.RedisConnector) ServiceInternalClient {
	return &internalClient{issuer: issuer, logger: logger, redis: redis}
}

// NewWebInternalClient constructs an outbound client bound to Web API.
func NewWebInternalClient(cfg *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) ServiceInternalClient {
	return NewInternalClient(cfg, "web-api", logger, redis)
}

func (ic *internalClient) WithToken(c context.Context, token string, userId uint64) context.Context {
	md := metadata.New(map[string]string{
		types.AUTHORIZATION_KEY: token,
		types.AUTH_KEY:          strconv.FormatUint(userId, 10),
	})
	return metadata.NewOutgoingContext(c, md)
}

func (ic *internalClient) WithScopeToken(c context.Context, token string, scope string) context.Context {
	if scope == "project" {
		md := metadata.New(map[string]string{
			types.PROJECT_SCOPE_KEY: token,
		})
		return metadata.NewOutgoingContext(c, md)
	}
	md := metadata.New(map[string]string{
		types.ORG_SCOPE_KEY: token,
	})
	return metadata.NewOutgoingContext(c, md)
}

func (ic *internalClient) WithAuth(c context.Context, principal authentication.Principal) (context.Context, error) {
	token, err := ic.createServiceAssertion(principal)
	if err != nil {
		return nil, err
	}
	md := metadata.New(map[string]string{types.SERVICE_SCOPE_KEY: token})
	return metadata.NewOutgoingContext(c, md), nil
}

// WithService authenticates the local direct Service identity without delegated tenant facts.
func (ic *internalClient) WithService(c context.Context) (context.Context, error) {
	if ic.issuerErr != nil {
		return nil, ic.issuerErr
	}
	token, err := ic.issuer.IssueService(authentication.Tenant{})
	if err != nil {
		return nil, err
	}
	return metadata.NewOutgoingContext(c, metadata.New(map[string]string{types.SERVICE_SCOPE_KEY: token})), nil
}

func (ic *internalClient) WithPlatform(c context.Context, principal authentication.Principal) (context.Context, error) {
	token, err := ic.createServiceAssertion(principal)
	if err != nil {
		return nil, err
	}
	_platform := map[string]string{
		types.SERVICE_SCOPE_KEY: token,
	}
	source, ok := utils.GetClientSource(c)
	if ok {
		_platform[utils.HEADER_SOURCE_KEY] = source.Get()
	}

	env, ok := utils.GetClientEnvironment(c)
	if ok {
		_platform[utils.HEADER_ENVIRONMENT_KEY] = env.Get()
	}

	// HEADER_REGION_KEY
	region, ok := utils.GetClientRegion(c)
	if ok {
		_platform[utils.HEADER_REGION_KEY] = region.Get()
	}

	return metadata.NewOutgoingContext(c, metadata.New(_platform)), nil
}

func (ic *internalClient) WithHttpAuth(c context.Context, principal authentication.Principal, req *http.Request) (*http.Request, error) {
	token, err := ic.createServiceAssertion(principal)
	if err != nil {
		return nil, err
	}
	req.Header.Set(types.SERVICE_SCOPE_KEY, token)
	return req.WithContext(c), nil
}

func (ic *internalClient) createServiceAssertion(principal authentication.Principal) (string, error) {
	if ic.issuerErr != nil {
		return "", ic.issuerErr
	}
	return ic.issuer.Issue(principal.Subject(), principal.Tenant(), principal.Authorities())
}

func (client *internalClient) Cache(c context.Context, key string, value interface{}) *connectors.RedisResponse {
	data, err := json.Marshal(value)
	if err != nil {
		client.logger.Errorf("Unable to cache the record as value is not marshalable %v for key %s", err, key)
		return nil
	}
	put := client.redis.Cmd(c, "SET", []string{key, string(data)})
	if put != nil && put.Err != nil {
		client.logger.Errorf("unable to set cache value with err %v for key %s", put, key)
	}
	return put
}

func (client *internalClient) CacheWithTTL(c context.Context, key string, value interface{}, ttl time.Duration) *connectors.RedisResponse {
	if ttl <= 0 {
		return &connectors.RedisResponse{Err: errors.New("cache TTL must be positive")}
	}
	data, err := json.Marshal(value)
	if err != nil {
		client.logger.Errorf("Unable to cache the record as value is not marshalable %v for key %s", err, key)
		return &connectors.RedisResponse{Err: err}
	}
	put := client.redis.Cmd(c, "SET", []string{key, string(data), "EX", strconv.FormatInt(int64(ttl/time.Second), 10)})
	if put != nil && put.Err != nil {
		client.logger.Errorf("unable to set cache value with err %v for key %s", put, key)
	}
	return put
}

func (client *internalClient) Retrieve(c context.Context, key string) *connectors.RedisResponse {
	return client.redis.Cmd(c, "GET", []string{key})
}

func (client *internalClient) CacheKey(c context.Context, funcName string, key ...string) string {
	var builder strings.Builder
	builder.WriteString("INTERNAL::")
	builder.WriteString(funcName)
	builder.WriteString("_")
	builder.WriteString(strings.Join(key, "__"))
	return builder.String()
}
