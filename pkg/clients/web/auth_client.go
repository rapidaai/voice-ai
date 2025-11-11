package web_client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	web_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	logger.Debugf("conntecting to authentication client with %s", config.WebHost)
	conn, err := grpc.NewClient(config.WebHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	// Generate cache key
	cacheKey := client.CacheKey(c, "ScopeAuthorize", scopeToken, scopeType)

	// Retrieve data from cache
	cachedValue := client.Retrieve(c, cacheKey)

	// Initialize data variable
	data := &web_api.ScopedAuthentication{}
	// Parse cached value into data
	err := cachedValue.ResultStruct(data)
	if err != nil {
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
