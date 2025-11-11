package web_proxy_api

import (
	"context"
	"errors"

	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	protos "github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type webInvokeGRPCApi struct {
	web_api.WebApi
	cfg                 *config.WebAppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	redis               connectors.RedisConnector
	deployServiceClient endpoint_client.DeploymentServiceClient
}

func NewInvokeGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.DeploymentServer {
	return &webInvokeGRPCApi{
		WebApi:              web_api.NewWebApi(config, logger, postgres, redis),
		cfg:                 config,
		logger:              logger,
		postgres:            postgres,
		redis:               redis,
		deployServiceClient: endpoint_client.NewDeploymentServiceClientGRPC(&config.AppConfig, logger, redis),
	}

}

// Probe implements lexatic_backend.DeploymentServer.
func (*webInvokeGRPCApi) Probe(context.Context, *protos.ProbeRequest) (*protos.ProbeResponse, error) {
	panic("unimplemented")
}

// Update implements lexatic_backend.DeploymentServer.
func (*webInvokeGRPCApi) Update(context.Context, *protos.UpdateRequest) (*protos.UpdateResponse, error) {
	panic("unimplemented")
}

func (endpointGRPCApi *webInvokeGRPCApi) Invoke(ctx context.Context, iRequest *protos.InvokeRequest) (*protos.InvokeResponse, error) {
	endpointGRPCApi.logger.Debugf("invoking endpoint with context %v", ctx)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to fork endpoint")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.deployServiceClient.Invoke(ctx, iAuth, iRequest)
}
