package web_api

import (
	"context"

	executor_client "github.com/lexatic/web-backend/pkg/clients/workflow"
	"github.com/lexatic/web-backend/pkg/types"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
)

type webExecutorApi struct {
	WebApi
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	redis          connectors.RedisConnector
	executorClient executor_client.ExecutorServiceClient
}

type webExecutorGRPCApi struct {
	webExecutorApi
}

func NewExecutorGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.ExecutorServiceServer {
	return &webExecutorGRPCApi{
		webExecutorApi{
			WebApi:         NewWebApi(config, logger, postgres, redis),
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			executorClient: executor_client.NewExecutorServiceClientGRPC(config, logger, redis),
		},
	}
}

// GetWorkflowRunOutput implements lexatic_backend.ExecutorServiceServer.
func (executor *webExecutorGRPCApi) GetWorkflowRunOutput(ctx context.Context, iRequest *web_api.GetWorkflowRunOutputRequest) (*web_api.GetWorkflowRunOutputResponse, error) {
	executor.logger.Debugf("Get workflow output request with args %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		executor.logger.Errorf("unauthenticated request to create endpoint tag")
		return utils.AuthenticateError[web_api.GetWorkflowRunOutputResponse]()
	}
	return executor.executorClient.GetWorkflowRunOutput(ctx, iAuth, iRequest)
}

// RunWorkflow implements lexatic_backend.ExecutorServiceServer.
func (executor *webExecutorGRPCApi) RunWorkflow(ctx context.Context, iRequest *web_api.RunWorkflowRequest) (*web_api.RunWorkflowResponse, error) {
	executor.logger.Debugf("Run workflow request with args %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		executor.logger.Errorf("unauthenticated request to create endpoint tag")
		return utils.AuthenticateError[web_api.RunWorkflowResponse]()
	}
	return executor.executorClient.RunWorkflow(ctx, iAuth, iRequest)
}
