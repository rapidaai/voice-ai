package provider_client

import (
	"context"
	"errors"
	"time"

	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	provider_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProviderServiceClient interface {
	GetModel(c context.Context, auth types.SimplePrinciple, modelId uint64) (*provider_api.Model, error)
	GetAllProviderModel(c context.Context, modelType string) ([]*provider_api.Model, error)
	GetAllProviders(c context.Context) ([]*provider_api.Provider, error)

	CalculateRequestPricingForText(c context.Context, providerModelId uint64, inputText, outputText string) (*provider_api.Cost, error)
	CalculateRequestPricing(c context.Context, providerModelId, inputToken, outputToken uint64) (*provider_api.Cost, error)
}

type providerServiceClient struct {
	clients.InternalClient
	cfg            *config.AppConfig
	logger         commons.Logger
	providerClient provider_api.ProviderServiceClient
	pricingClient  provider_api.PricingServiceClient
}

func NewProviderServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) ProviderServiceClient {
	logger.Debugf("conntecting to provider client with %s", config.ProviderHost)
	conn, err := grpc.Dial(config.ProviderHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v to the provider service", err)
	}
	// providerClient :=
	return &providerServiceClient{
		clients.NewInternalClient(config, logger, redis),
		config,
		logger,
		provider_api.NewProviderServiceClient(conn),
		provider_api.NewPricingServiceClient(conn),
	}
}

// Used by experiment and test. modelType is type of experiment text, chat etc
func (client *providerServiceClient) GetAllProviderModel(c context.Context, modelType string) ([]*provider_api.Model, error) {
	key, value := "endpoint", "complete"
	if modelType == "image" {
		value = "text-to-image"
	}

	if modelType == "chat" {
		value = "chat-complete"
	}

	res, err := client.providerClient.GetAllModel(c, &provider_api.GetAllModelRequest{Criterias: []*provider_api.Criteria{{Key: key, Value: value}}})
	if err != nil {
		client.logger.Errorf("got an error while calling provider service client error here %v", err)
		return nil, err
	}
	if !res.Success {
		return nil, errors.New("illegal request with provider clients")
	}
	return res.GetData(), nil
}

// GetModel implements internal_clients.ProviderServiceClient.
func (client *providerServiceClient) GetModel(c context.Context, auth types.SimplePrinciple, modelId uint64) (*provider_api.Model, error) {
	start := time.Now()
	// Generate cache key
	cacheKey := client.CacheKey(c, "GetModel", *auth.GetCurrentOrganizationId(), modelId)

	// Retrieve data from cache
	cachedValue := client.Retrieve(c, cacheKey)
	if cachedValue.HasError() {
		client.logger.Errorf("Cache missed for the request: %v", cachedValue.Err)
	}

	// Initialize data variable
	data := &provider_api.Model{}

	// Parse cached value into data
	err := cachedValue.ResultStruct(data)
	if err != nil {
		client.logger.Errorf("Failed to parse cached data: %v", err)

		res, err := client.providerClient.GetModel(client.WithAuth(c, auth), &provider_api.GetModelRequest{ModelId: modelId})
		if err != nil {
			client.logger.Errorf("Failed to get credentials from vault service: %v", err)
			return nil, err
		}

		if res.GetSuccess() {
			if res.GetModel() != nil {
				client.Cache(c, cacheKey, res.GetModel())
			}
			client.logger.Debugf("Benchmarking: vaultServiceClient.GetProviderCredential time taken %v", time.Since(start))
			return res.GetModel(), nil
		}

	}

	// Log benchmarking information
	client.logger.Debugf("Benchmarking: vaultServiceClient.GetProviderCredential time taken %v", time.Since(start))
	return data, nil

}

func (client *providerServiceClient) GetAllProviders(c context.Context) ([]*provider_api.Provider, error) {
	res, err := client.providerClient.GetAllProvider(c, &provider_api.GetAllProviderRequest{})

	if err != nil {
		client.logger.Errorf("Failed to get credentials from vault service: %v", err)
		return nil, err
	}
	if res.GetSuccess() {
		return res.GetData(), nil
	}
	return nil, errors.New("unknown error occured while fetching providers")
}

func (client *providerServiceClient) CalculateRequestPricingForText(c context.Context, providerModelId uint64, in, out string) (*provider_api.Cost, error) {
	start := time.Now()
	res, err := client.pricingClient.CalculateRequestPricingForText(c, &provider_api.PricingRequestForText{
		ProviderModelId: providerModelId,
		InputText:       in,
		OutputText:      out,
	})
	if err != nil {
		client.logger.Errorf("got an error while calling provider service client error here %v", err)
		return nil, err
	}
	if !res.Success {
		return nil, errors.New("illegal request for provider clients")
	}
	client.logger.Debugf("Benchmarking: providerServiceClient.CalculateRequestPricingForText time taken %v", time.Since(start))
	return res.GetData(), nil
}

func (client *providerServiceClient) CalculateRequestPricing(c context.Context, providerModelId uint64, in, out uint64) (*provider_api.Cost, error) {
	start := time.Now()
	res, err := client.pricingClient.CalculateRequestPricing(c, &provider_api.PricingRequest{
		ProviderModelId:  providerModelId,
		InputTokenCount:  in,
		OutputTokenCount: out,
	})
	if err != nil {
		client.logger.Errorf("got an error while calling provider service client error here %v", err)
		return nil, err
	}
	if !res.Success {
		return nil, errors.New("illegal request for provider clients")
	}
	client.logger.Debugf("Benchmarking: providerServiceClient.CalculateRequestPricing time taken %v", time.Since(start))
	return res.GetData(), nil
}
