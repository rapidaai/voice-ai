package web_proxy_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	protos "github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
)

type webInvokeGRPCApi struct {
	web_api.WebApi
	cfg                 *config.WebAppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	redis               connectors.RedisConnector
	deployServiceClient endpoint_client.DeploymentServiceClient
}

func NewInvokeGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector, deploymentClient endpoint_client.DeploymentServiceClient) protos.DeploymentServer {
	return &webInvokeGRPCApi{
		WebApi:              web_api.NewWebApi(config, logger, postgres, redis),
		cfg:                 config,
		logger:              logger,
		postgres:            postgres,
		redis:               redis,
		deployServiceClient: deploymentClient,
	}
}

// Probe implements protos.DeploymentServer.
func (*webInvokeGRPCApi) Probe(context.Context, *protos.ProbeRequest) (*protos.ProbeResponse, error) {
	panic("unimplemented")
}

// Update implements protos.DeploymentServer.
func (*webInvokeGRPCApi) Update(context.Context, *protos.UpdateRequest) (*protos.UpdateResponse, error) {
	panic("unimplemented")
}

func (endpointGRPCApi *webInvokeGRPCApi) Invoke(ctx context.Context, iRequest *protos.InvokeRequest) (*protos.InvokeResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.deployServiceClient.Invoke(ctx, principal, iRequest)
}
