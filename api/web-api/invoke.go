package web_api

import (
	"context"
	"errors"

	endpoint_client "github.com/lexatic/web-backend/pkg/clients/endpoint"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webInvokeGRPCApi struct {
	WebApi
	cfg                 *config.AppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	redis               connectors.RedisConnector
	deployServiceClient endpoint_client.DeploymentServiceClient
}

func NewInvokeGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.DeploymentServer {
	return &webInvokeGRPCApi{
		WebApi:              NewWebApi(config, logger, postgres, redis),
		cfg:                 config,
		logger:              logger,
		postgres:            postgres,
		redis:               redis,
		deployServiceClient: endpoint_client.NewDeploymentServiceClientGRPC(config, logger, redis),
	}

}

// Probe implements lexatic_backend.DeploymentServer.
func (*webInvokeGRPCApi) Probe(context.Context, *web_api.ProbeRequest) (*web_api.ProbeResponse, error) {
	panic("unimplemented")
}

// Update implements lexatic_backend.DeploymentServer.
func (*webInvokeGRPCApi) Update(context.Context, *web_api.UpdateRequest) (*web_api.UpdateResponse, error) {
	panic("unimplemented")
}

func (endpointGRPCApi *webInvokeGRPCApi) Invoke(ctx context.Context, iRequest *web_api.InvokeRequest) (*web_api.InvokeResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to fork endpoint")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.deployServiceClient.Invoke(ctx, iAuth, iRequest)
}
