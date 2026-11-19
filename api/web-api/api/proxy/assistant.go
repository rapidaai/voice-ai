package web_proxy_api

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	assistant_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
)

type webAssistantApi struct {
	web_api.WebApi
	cfg             *config.WebAppConfig
	logger          commons.Logger
	postgres        connectors.PostgresConnector
	redis           connectors.RedisConnector
	assistantClient assistant_client.AssistantServiceClient
}

type webAssistantGRPCApi struct {
	webAssistantApi
}

func NewAssistantGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.AssistantServiceServer {
	return &webAssistantGRPCApi{
		webAssistantApi{
			WebApi:          web_api.NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			assistantClient: assistant_client.NewAssistantServiceClientGRPC(&config.AppConfig, logger, redis),
		},
	}
}

func (assistant *webAssistantGRPCApi) GetAllAssistantConversation(c context.Context, iRequest *protos.GetAllAssistantConversationRequest) (*protos.GetAllAssistantConversationResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _assistant, err := assistant.assistantClient.GetAllAssistantConversation(c, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate(), nil)
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllAssistantConversationResponse](err, "Unable to get all the assistant sessions")
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantConversationResponse, []*protos.AssistantConversation](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) GetAllConversationMessage(c context.Context, iRequest *protos.GetAllConversationMessageRequest) (*protos.GetAllConversationMessageResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _assistant, err := assistant.assistantClient.GetAllConversationMessage(c, principal, iRequest.GetAssistantId(), iRequest.GetAssistantConversationId(), iRequest.GetCriterias(), iRequest.GetPaginate(), nil)
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllConversationMessageResponse](err, "Unable to get all the assistant sessions")
	}

	return utils.PaginatedSuccess[protos.GetAllConversationMessageResponse, []*protos.AssistantConversationMessage](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

// GetAllAssistantMessage implements protos.AssistantServiceServer.
func (assistant *webAssistantGRPCApi) GetAllAssistantMessage(c context.Context, iRequest *protos.GetAllAssistantMessageRequest) (*protos.GetAllAssistantMessageResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _assistant, err := assistant.assistantClient.GetAllAssistantMessage(c, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate(), iRequest.GetOrder(), iRequest.GetSelectors())
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllAssistantMessageResponse](err, "Unable to get all the assistant messages")
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantMessageResponse, []*protos.AssistantConversationMessage](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) GetAllMessage(c context.Context, iRequest *protos.GetAllMessageRequest) (*protos.GetAllMessageResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _assistant, err := assistant.assistantClient.GetAllMessage(c, principal, iRequest.GetCriterias(), iRequest.GetPaginate(), iRequest.GetOrder(), iRequest.GetSelectors())
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllMessageResponse](err, "Unable to get all the assistant messages")
	}

	return utils.PaginatedSuccess[protos.GetAllMessageResponse, []*protos.AssistantConversationMessage](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) GetAssistantDashboard(c context.Context, iRequest *protos.GetAssistantDashboardRequest) (*protos.GetAssistantDashboardResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	assistantDashboardResponse, err := assistant.assistantClient.GetAssistantDashboard(c, principal, iRequest)
	if err != nil {
		return exceptions.InternalServerError[protos.GetAssistantDashboardResponse](err, "Unable to get the assistant dashboard")
	}
	return assistantDashboardResponse, nil
}

func (assistant *webAssistantGRPCApi) GetAssistant(c context.Context, iRequest *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	_assistant, err := assistant.assistantClient.GetAssistant(c, principal, iRequest)
	if err != nil {
		return utils.Error[protos.GetAssistantResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	return _assistant, nil
}

/*
 */

/*
 */
func (assistant *webAssistantGRPCApi) GetAllAssistant(c context.Context, iRequest *protos.GetAllAssistantRequest) (*protos.GetAllAssistantResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _assistant, err := assistant.assistantClient.GetAllAssistant(c, principal, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantResponse, []*protos.Assistant](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) CreateAssistant(c context.Context, iRequest *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistant.assistantClient.CreateAssistant(c, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantProvider(ctx context.Context, iRequest *protos.GetAllAssistantProviderRequest) (*protos.GetAllAssistantProviderResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _assistantProviders, err := assistantGRPCApi.assistantClient.GetAllAssistantProvider(ctx, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantProviderResponse](
			err,
			"Unable to get your assistant provider models, please try again in sometime.")
	}

	return &protos.GetAllAssistantProviderResponse{
		Code:      200,
		Success:   true,
		Paginated: _page,
		Data:      _assistantProviders,
	}, nil
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantVersion(ctx context.Context, iRequest *protos.UpdateAssistantVersionRequest) (*protos.GetAssistantResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantVersion(
		ctx,
		principal,
		iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantProvider(ctx context.Context, iRequest *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantProvider(ctx, principal, iRequest)
}

// CreateAssistantTag implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantTag(ctx context.Context, iRequest *protos.CreateAssistantTagRequest) (*protos.GetAssistantResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantTag(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantDetail(ctx context.Context, iRequest *protos.UpdateAssistantDetailRequest) (*protos.GetAssistantResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantDetail(ctx, principal, iRequest)
}

// GetAllAssistantHTTPLog implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantHTTPLog(ctx context.Context, iRequest *protos.GetAllAssistantHTTPLogRequest) (*protos.GetAllAssistantHTTPLogResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantHTTPLog(ctx, principal,
		iRequest.GetProjectId(),
		iRequest.GetCriterias(), iRequest.GetPaginate(), iRequest.GetOrder())
	if err != nil {
		return utils.Error[protos.GetAllAssistantHTTPLogResponse](
			err,
			"Unable to get all the HTTP logs, please try again later.",
		)
	}
	if page == nil {
		return utils.Error[protos.GetAllAssistantHTTPLogResponse](
			errors.New("assistant api returned empty pagination for http logs"),
			"Unable to get all the HTTP logs, please try again later.",
		)
	}
	if tls == nil {
		tls = make([]*protos.AssistantHTTPLog, 0)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantHTTPLogResponse, []*protos.AssistantHTTPLog](
		page.GetTotalItem(), page.GetCurrentPage(),
		tls)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantHTTPLog(ctx context.Context, iRequest *protos.GetAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	return assistantGRPCApi.assistantClient.GetAssistantHTTPLog(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) RetryAssistantHTTPLog(ctx context.Context, iRequest *protos.RetryAssistantHTTPLogRequest) (*protos.GetAssistantHTTPLogResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	return assistantGRPCApi.assistantClient.RetryAssistantHTTPLog(ctx, principal, iRequest)
}

// GetAssistantConversation implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantConversation(ctx context.Context, iRequest *protos.GetAssistantConversationRequest) (*protos.GetAssistantConversationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.GetAssistantConversation(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistant(ctx context.Context, iRequest *protos.DeleteAssistantRequest) (*protos.GetAssistantResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistant(ctx, principal, iRequest)
}

// CreateAssistantKnowledge implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantKnowledge(ctx context.Context, iRequest *protos.CreateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantKnowledge(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantKnowledge(ctx context.Context, iRequest *protos.UpdateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantKnowledge(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistantKnowledge(ctx context.Context, iRequest *protos.DeleteAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistantKnowledge(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantKnowledge(ctx context.Context, iRequest *protos.GetAllAssistantKnowledgeRequest) (*protos.GetAllAssistantKnowledgeResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantKnowledge(ctx, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantKnowledgeResponse](
			err,
			"Unable to get all the assistant knowledge, please try again later.",
		)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantKnowledgeResponse, []*protos.AssistantKnowledge](
		page.GetTotalItem(), page.GetCurrentPage(),
		tls)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantKnowledge(ctx context.Context, iRequest *protos.GetAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	return assistantGRPCApi.assistantClient.GetAssistantKnowledge(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantTool(ctx context.Context, iRequest *protos.CreateAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantTool(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantTool(ctx context.Context, iRequest *protos.UpdateAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantTool(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistantTool(ctx context.Context, iRequest *protos.DeleteAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistantTool(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantTool(ctx context.Context, iRequest *protos.GetAllAssistantToolRequest) (*protos.GetAllAssistantToolResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantTool(ctx, principal, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantToolResponse](
			err,
			"Unable to get all the webhook analysis, please try again later.",
		)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantToolResponse, []*protos.AssistantTool](
		page.GetTotalItem(), page.GetCurrentPage(),
		tls)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantTool(ctx context.Context, iRequest *protos.GetAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	return assistantGRPCApi.assistantClient.GetAssistantTool(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantToolLog(ctx context.Context, iRequest *protos.GetAssistantToolLogRequest) (*protos.GetAssistantToolLogResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.GetAssistantToolLog(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantToolLog(ctx context.Context, iRequest *protos.GetAllAssistantToolLogRequest) (*protos.GetAllAssistantToolLogResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.GetAllAssistantToolLog(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantConfiguration(ctx context.Context, iRequest *protos.CreateAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantConfiguration(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantConfiguration(ctx context.Context, iRequest *protos.UpdateAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantConfiguration(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistantConfiguration(ctx context.Context, iRequest *protos.DeleteAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistantConfiguration(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantConfiguration(ctx context.Context, iRequest *protos.GetAllAssistantConfigurationRequest) (*protos.GetAllAssistantConfigurationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.GetAllAssistantConfiguration(ctx, principal, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantConfiguration(ctx context.Context, iRequest *protos.GetAssistantConfigurationRequest) (*protos.GetAssistantConfigurationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return assistantGRPCApi.assistantClient.GetAssistantConfiguration(ctx, principal, iRequest)
}
