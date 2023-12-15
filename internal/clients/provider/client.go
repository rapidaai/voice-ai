package provider

import (
	"context"

	"github.com/lexatic/web-backend/config"
	internal_clients "github.com/lexatic/web-backend/internal/clients"
	"github.com/lexatic/web-backend/pkg/commons"
	provider_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type providerServiceClient struct {
	cfg            *config.AppConfig
	logger         commons.Logger
	providerClient provider_api.ProviderServiceClient
}

func NewProviderServiceClientGRPC(config *config.AppConfig, logger commons.Logger) internal_clients.ProviderServiceClient {
	logger.Debugf("conntecting to provider client with %s", config.ProviderHost)
	conn, err := grpc.Dial(config.ProviderHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	providerClient := provider_api.NewProviderServiceClient(conn)
	return &providerServiceClient{
		cfg:            config,
		logger:         logger,
		providerClient: providerClient,
	}
}

func (client *providerServiceClient) GetAllProviders(c context.Context) (*provider_api.GetAllProviderResponse, error) {
	res, err := client.providerClient.GetAllProvider(c, &provider_api.GetAllProviderRequest{})
	if err != nil {
		client.logger.Debugf("Printing error here %v", err)
		return nil, err
	}
	return res, nil
}
