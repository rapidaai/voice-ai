package web_api

import (
	"context"

	workflow_client "github.com/lexatic/web-backend/pkg/clients/workflow"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webWorkflowApi struct {
	WebApi
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	redis          connectors.RedisConnector
	workflowClient workflow_client.WorkflowServiceClient
}

type webWorkflowGRPCApi struct {
	webWorkflowApi
}

func NewWorkflowGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.WorkflowServiceServer {
	return &webWorkflowGRPCApi{
		webWorkflowApi{
			WebApi:         NewWebApi(config, logger, postgres, redis),
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			workflowClient: workflow_client.NewWorkflowServiceClientGRPC(config, logger, redis),
		},
	}
}

func (workflow *webWorkflowGRPCApi) GetWorkflow(c context.Context, iRequest *web_api.GetWorkflowRequest) (*web_api.GetWorkflowResponse, error) {
	workflow.logger.Debugf("GetWorkflow from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		workflow.logger.Errorf("unauthenticated request for get actvities")
		return utils.AuthenticateError[web_api.GetWorkflowResponse]()
	}
	_workflow, err := workflow.workflowClient.GetWorkflow(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[web_api.GetWorkflowResponse](
			err,
			"Unable to get your workflow, please try again in sometime.")
	}

	// if _workflow.EndpointProviderModel != nil {
	// 	_workflow.EndpointProviderModel.CreatedUser = endpoint.GetUser(c, iAuth, _workflow.EndpointProviderModel.GetCreatedBy())
	// 	_workflow.EndpointProviderModel.ProviderModel = endpoint.GetProviderModel(c, iAuth, _workflow.EndpointProviderModel.GetProviderModelId())
	// }

	return utils.Success[web_api.GetWorkflowResponse, *web_api.Workflow](_workflow)

}

/*
 */

func (workflow *webWorkflowGRPCApi) GetAllWorkflow(c context.Context, iRequest *web_api.GetAllWorkflowRequest) (*web_api.GetAllWorkflowResponse, error) {
	workflow.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		workflow.logger.Errorf("unauthenticated request for get actvities")
		return utils.AuthenticateError[web_api.GetAllWorkflowResponse]()
	}

	_page, _endpoint, err := workflow.workflowClient.GetAllWorkflow(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllWorkflowResponse](
			err,
			"Unable to get your workflows, please try again in sometime.")
	}

	return utils.PaginatedSuccess[web_api.GetAllWorkflowResponse, []*web_api.Workflow](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoint)
}

// CreateWorkflow implements lexatic_backend.WorkflowServiceServer.
func (workflow *webWorkflowGRPCApi) CreateWorkflow(ctx context.Context, iRequest *web_api.CreateWorkflowRequest) (*web_api.GetWorkflowResponse, error) {
	workflow.logger.Debugf("Create workflow request with args %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		workflow.logger.Errorf("unauthenticated request to create workflow")
		return utils.AuthenticateError[web_api.GetWorkflowResponse]()
	}
	return workflow.workflowClient.CreateWorkflow(ctx, iAuth, iRequest)
}

// CreateEndpointTag implements lexatic_backend.EndpointServiceServer.
func (workflow *webWorkflowGRPCApi) CreateWorkflowTag(ctx context.Context, iRequest *web_api.CreateWorkflowTagRequest) (*web_api.CreateWorkflowTagResponse, error) {
	workflow.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		workflow.logger.Errorf("unauthenticated request to create endpoint tag")
		return utils.AuthenticateError[web_api.CreateWorkflowTagResponse]()
	}
	return workflow.workflowClient.CreateWorkflowTag(ctx, iAuth, iRequest)
}

func (workflow *webWorkflowGRPCApi) UpdateWorkflow(ctx context.Context, iRequest *web_api.UpdateWorkflowRequest) (*web_api.GetWorkflowResponse, error) {
	workflow.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		workflow.logger.Errorf("unauthenticated request to create endpoint tag")
		return utils.AuthenticateError[web_api.GetWorkflowResponse]()
	}
	return workflow.workflowClient.UpdateWorkflow(ctx, iAuth, iRequest)
}
