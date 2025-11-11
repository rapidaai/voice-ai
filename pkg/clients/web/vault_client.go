package web_client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	vault_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type VaultClient interface {
	GetProviderCredential(ctx context.Context, auth types.SimplePrinciple, providerId uint64) (*vault_api.VaultCredential, error)
	GetCredential(ctx context.Context, auth types.SimplePrinciple, vaultId uint64) (*vault_api.VaultCredential, error)
	GetOauth2Credential(ctx context.Context, auth types.SimplePrinciple, vaultId uint64) (*vault_api.VaultCredential, error)
}

type vaultServiceClient struct {
	clients.InternalClient
	cfg         *config.AppConfig
	logger      commons.Logger
	vaultClient vault_api.VaultServiceClient
}

func NewVaultClientGRPC(cfg *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) VaultClient {
	logger.Debugf("conntecting to vault client with %s", cfg.WebHost)
	conn, err := grpc.NewClient(cfg.WebHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Errorf("Unable to create connection for vault api %v", err)
	}
	vaultClient := vault_api.NewVaultServiceClient(conn)
	return &vaultServiceClient{
		clients.NewInternalClient(cfg, logger, redis),
		cfg,
		logger,
		vaultClient,
	}
}
func (client *vaultServiceClient) GetProviderCredential(c context.Context, auth types.SimplePrinciple, providerId uint64) (*vault_api.VaultCredential, error) {
	start := time.Now()

	// Generate cache key
	cacheKey := client.CacheKey(c, "GetProviderCredential", fmt.Sprintf("%d", *auth.GetCurrentOrganizationId()), fmt.Sprintf("%d", providerId))

	// Retrieve data from cache
	cachedValue := client.Retrieve(c, cacheKey)
	if cachedValue.HasError() {
		client.logger.Errorf("Cache missed for the request: %v", cachedValue.Err)
	}

	// Initialize data variable
	data := &vault_api.VaultCredential{}
	// Parse cached value into data
	err := cachedValue.ResultStruct(data)
	if err != nil {
		client.logger.Errorf("Failed to parse cached data: %v", err)

		// Call the vault service to fetch data
		res, err := client.vaultClient.GetProviderCredential(client.WithAuth(c, auth), &vault_api.GetProviderCredentialRequest{
			ProviderId: providerId,
		})
		if err != nil {
			client.logger.Errorf("Failed to get credentials from vault service: %v", err)
			return nil, err
		}

		// Check if the request was successful
		if res.GetSuccess() {
			if res.GetData() != nil {
				// Cache the fetched data
				client.Cache(c, cacheKey, res.GetData())
			}
			client.logger.Benchmark("vaultServiceClient.GetProviderCredential", time.Since(start))
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
	client.logger.Benchmark("vaultServiceClient.GetProviderCredential", time.Since(start))
	return data, nil
}

// the request witll be cache with timeout given in the response
// eg
// removing caching for now as token will be expire for time being
// "expiresIn":1800
// {"accessToken":"CNHBzOfGMhIaAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGAYrsSpFyDZ1LAkKMPdoQMyFMO0Oc4C5e9QT-dl5GlbjfCUj-NzOlEAAABBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADZCFMVXu0LRQlBBF2WcFTSZqNS7_QJnSgNuYTFSAFoAYAA","code":"na1-48bb-77e5-443a-b51f-bf04461b4717","connect":"crm/hubspot","expiresIn":1800,"refreshToken":"na1-8779-d688-494a-899e-406752f8c8ac","scope":"","state":"47066eef-367c-4135-9e74-53b367666147"}
func (client *vaultServiceClient) GetOauth2Credential(c context.Context,
	auth types.SimplePrinciple, vaultId uint64) (*vault_api.VaultCredential, error) {
	start := time.Now()

	data := &vault_api.VaultCredential{}
	// Call the vault service to fetch data
	res, err := client.vaultClient.GetOauth2Credential(client.WithAuth(c, auth), &vault_api.GetCredentialRequest{
		VaultId: vaultId,
	})
	if err != nil {
		client.logger.Errorf("Failed to get credentials from vault service: %v", err)
		return nil, err
	}

	// Check if the request was successful
	if res.GetSuccess() {
		if res.GetData() != nil {
			// Cache the fetched data
			// client.Cache(c, cacheKey, res.GetData())
		}
		client.logger.Benchmark("vaultServiceClient.GetOauth2Credential", time.Since(start))
		return res.GetData(), nil
	}

	// Handle error response from vault service
	if res.GetError() != nil {
		errMsg := fmt.Sprintf("Failed to get credentials from vault service: %s", res.GetError().HumanMessage)
		client.logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}
	// Log benchmarking information
	client.logger.Benchmark("vaultServiceClient.GetOauth2Credential", time.Since(start))
	return data, nil
}

func (client *vaultServiceClient) GetCredential(c context.Context, auth types.SimplePrinciple, vaultId uint64) (*vault_api.VaultCredential, error) {
	start := time.Now()

	cacheKey := client.CacheKey(c, "GetCredential", fmt.Sprintf("%d", *auth.GetCurrentOrganizationId()), fmt.Sprintf("vlt__%d", vaultId))

	cachedValue := client.Retrieve(c, cacheKey)
	if cachedValue.HasError() {
		client.logger.Errorf("Cache missed for the request: %v", cachedValue.Err)
	}

	data := &vault_api.VaultCredential{}
	err := cachedValue.ResultStruct(data)

	// Start a goroutine to fetch from API and update cache
	var apiData chan *vault_api.VaultCredential = make(chan *vault_api.VaultCredential, 1)
	bgCtx := context.Background()
	utils.Go(bgCtx, func() {
		res, err := client.vaultClient.GetCredential(client.WithAuth(bgCtx, auth), &vault_api.GetCredentialRequest{
			VaultId: vaultId,
		})
		if err != nil {
			client.logger.Errorf("Failed to get credentials from vault service: %v", err)
			apiData <- nil
			return
		}

		if res.GetSuccess() && res.GetData() != nil {
			client.Cache(bgCtx, cacheKey, res.GetData())
			apiData <- res.GetData()
		} else if res.GetError() != nil {
			client.logger.Errorf("Failed to get credentials from vault service: %s", res.GetError().HumanMessage)
			apiData <- nil
		} else {
			apiData <- nil
		}
	})

	// If cache hit, return cached data immediately
	if err == nil {
		client.logger.Benchmark("vaultServiceClient.GetCredential (cache hit)", time.Since(start))
		return data, nil
	}

	// If cache miss, wait for API response
	apiResult := <-apiData
	if apiResult != nil {
		client.logger.Benchmark("vaultServiceClient.GetCredential (API call)", time.Since(start))
		return apiResult, nil
	}

	client.logger.Benchmark("vaultServiceClient.GetCredential", time.Since(start))
	return nil, errors.New("failed to get credentials from vault service")
}
