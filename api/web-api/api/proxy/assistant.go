package web_proxy_api

import (
	"context"
	"errors"

	assistant_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
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
	assistant.logger.Debugf("GetAllAssistantConversation from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return exceptions.AuthenticationError[protos.GetAllAssistantConversationResponse]()
	}

	_page, _assistant, err := assistant.assistantClient.GetAllAssistantConversation(c, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate(), nil)
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllAssistantConversationResponse](err, "Unable to get all the assistant sessions")
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantConversationResponse, []*protos.AssistantConversation](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) GetAllConversationMessage(c context.Context, iRequest *protos.GetAllConversationMessageRequest) (*protos.GetAllConversationMessageResponse, error) {
	assistant.logger.Debugf("GetAllConversationMessage from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return exceptions.AuthenticationError[protos.GetAllConversationMessageResponse]()
	}

	_page, _assistant, err := assistant.assistantClient.GetAllConversationMessage(c, iAuth, iRequest.GetAssistantId(), iRequest.GetAssistantConversationId(), iRequest.GetCriterias(), iRequest.GetPaginate(), nil)
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllConversationMessageResponse](err, "Unable to get all the assistant sessions")
	}

	return utils.PaginatedSuccess[protos.GetAllConversationMessageResponse, []*protos.AssistantConversationMessage](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

// GetAllAssistantMessage implements protos.AssistantServiceServer.
func (assistant *webAssistantGRPCApi) GetAllAssistantMessage(c context.Context, iRequest *protos.GetAllAssistantMessageRequest) (*protos.GetAllAssistantMessageResponse, error) {
	assistant.logger.Debugf("GetAllAssistantMessage from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return exceptions.AuthenticationError[protos.GetAllAssistantMessageResponse]()
	}

	_page, _assistant, err := assistant.assistantClient.GetAllAssistantMessage(c, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate(), iRequest.GetOrder(), iRequest.GetSelectors())
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllAssistantMessageResponse](err, "Unable to get all the assistant messages")
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantMessageResponse, []*protos.AssistantConversationMessage](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) GetAllMessage(c context.Context, iRequest *protos.GetAllMessageRequest) (*protos.GetAllMessageResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return exceptions.AuthenticationError[protos.GetAllMessageResponse]()
	}

	_page, _assistant, err := assistant.assistantClient.GetAllMessage(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate(), iRequest.GetOrder(), iRequest.GetSelectors())
	if err != nil {
		return exceptions.InternalServerError[protos.GetAllMessageResponse](err, "Unable to get all the assistant messages")
	}

	return utils.PaginatedSuccess[protos.GetAllMessageResponse, []*protos.AssistantConversationMessage](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) GetAssistant(c context.Context, iRequest *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
	assistant.logger.Debugf("GetAssistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	_assistant, err := assistant.assistantClient.GetAssistant(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[protos.GetAssistantResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	if _assistant.GetSuccess() {
		providerModel := _assistant.GetData().GetAssistantProviderModel()
		if providerModel != nil {
			user := assistant.GetUser(c, iAuth, providerModel.GetCreatedBy())
			providerModel.CreatedUser = user
			_assistant.GetData().AssistantProviderModel = providerModel
		}

		agentKit := _assistant.GetData().GetAssistantProviderAgentkit()
		if agentKit != nil {
			user := assistant.GetUser(c, iAuth, agentKit.GetCreatedBy())
			agentKit.CreatedUser = user
			_assistant.GetData().AssistantProviderAgentkit = agentKit
		}

		websocket := _assistant.GetData().GetAssistantProviderWebsocket()
		if websocket != nil {
			user := assistant.GetUser(c, iAuth, websocket.GetCreatedBy())
			websocket.CreatedUser = user
			_assistant.GetData().AssistantProviderWebsocket = websocket
		}

	}
	return _assistant, nil
}

/*
 */

/*
 */
func (assistant *webAssistantGRPCApi) GetAllAssistant(c context.Context, iRequest *protos.GetAllAssistantRequest) (*protos.GetAllAssistantResponse, error) {
	assistant.logger.Debugf("GetAllAssistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _assistant, err := assistant.assistantClient.GetAllAssistant(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	for _, ast := range _assistant {
		providerModel := ast.GetAssistantProviderModel()
		if providerModel != nil {
			user := assistant.GetUser(c, iAuth, providerModel.GetCreatedBy())
			providerModel.CreatedUser = user
			ast.AssistantProviderModel = providerModel
		}

		agentKit := ast.GetAssistantProviderAgentkit()
		if agentKit != nil {
			user := assistant.GetUser(c, iAuth, agentKit.GetCreatedBy())
			agentKit.CreatedUser = user
			ast.AssistantProviderAgentkit = agentKit
		}

		websocket := ast.GetAssistantProviderWebsocket()
		if websocket != nil {
			user := assistant.GetUser(c, iAuth, websocket.GetCreatedBy())
			websocket.CreatedUser = user
			ast.AssistantProviderWebsocket = websocket
		}
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantResponse, []*protos.Assistant](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) CreateAssistant(c context.Context, iRequest *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error) {
	assistant.logger.Debugf("Create assistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return assistant.assistantClient.CreateAssistant(c, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantProvider(ctx context.Context, iRequest *protos.GetAllAssistantProviderRequest) (*protos.GetAllAssistantProviderResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _assistantProviders, err := assistantGRPCApi.assistantClient.GetAllAssistantProvider(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantProviderResponse](
			err,
			"Unable to get your assistant provider models, please try again in sometime.")
	}

	for _, ast := range _assistantProviders {
		if ast.GetAssistantProvider() != nil {
			switch assistantProvider := ast.GetAssistantProvider().(type) {
			case *protos.GetAllAssistantProviderResponse_AssistantProvider_AssistantProviderAgentkit:
				user := assistantGRPCApi.GetUser(ctx, iAuth, assistantProvider.AssistantProviderAgentkit.GetCreatedBy())
				assistantProvider.AssistantProviderAgentkit.CreatedUser = user
				ast.AssistantProvider = assistantProvider
			case *protos.GetAllAssistantProviderResponse_AssistantProvider_AssistantProviderModel:
				user := assistantGRPCApi.GetUser(ctx, iAuth, assistantProvider.AssistantProviderModel.GetCreatedBy())
				assistantProvider.AssistantProviderModel.CreatedUser = user
				ast.AssistantProvider = assistantProvider
			case *protos.GetAllAssistantProviderResponse_AssistantProvider_AssistantProviderWebsocket:
				user := assistantGRPCApi.GetUser(ctx, iAuth, assistantProvider.AssistantProviderWebsocket.GetCreatedBy())
				assistantProvider.AssistantProviderWebsocket.CreatedUser = user
				ast.AssistantProvider = assistantProvider
			}
		}
	}
	return &protos.GetAllAssistantProviderResponse{
		Code:      200,
		Success:   true,
		Paginated: _page,
		Data:      _assistantProviders,
	}, nil

}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantVersion(ctx context.Context, iRequest *protos.UpdateAssistantVersionRequest) (*protos.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantVersion(
		ctx,
		iAuth,
		iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantProvider(ctx context.Context, iRequest *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant provider model")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantProvider(ctx, iAuth, iRequest)
}

// CreateAssistantTag implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantTag(ctx context.Context, iRequest *protos.CreateAssistantTagRequest) (*protos.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantTag(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantDetail(ctx context.Context, iRequest *protos.UpdateAssistantDetailRequest) (*protos.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantDetail(ctx, iAuth, iRequest)
}

// CreateAssistantWebhook implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantWebhook(ctx context.Context, iRequest *protos.CreateAssistantWebhookRequest) (*protos.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantWebhook(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantWebhook(ctx context.Context, iRequest *protos.UpdateAssistantWebhookRequest) (*protos.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Update assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantWebhook(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistantWebhook(ctx context.Context, iRequest *protos.DeleteAssistantWebhookRequest) (*protos.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Delete assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistantWebhook(ctx, iAuth, iRequest)

}

// GetAllAssistantWebhook implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantWebhook(ctx context.Context, iRequest *protos.GetAllAssistantWebhookRequest) (*protos.GetAllAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantWebhook(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantWebhookResponse](
			err,
			"Unable to get all the webhooks, please try again later.",
		)
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantWebhookResponse, []*protos.AssistantWebhook](
		page.GetTotalItem(), page.GetCurrentPage(),
		tls)
}

// GetAllAssistantWebhookLog implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantWebhookLog(ctx context.Context, iRequest *protos.GetAllAssistantWebhookLogRequest) (*protos.GetAllAssistantWebhookLogResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantWebhookLog(ctx, iAuth,
		iRequest.GetProjectId(),
		iRequest.GetCriterias(), iRequest.GetPaginate(), iRequest.GetOrder())
	if err != nil {
		return utils.Error[protos.GetAllAssistantWebhookLogResponse](
			err,
			"Unable to get all the webhook logs, please try again later.",
		)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantWebhookLogResponse, []*protos.AssistantWebhookLog](
		page.GetTotalItem(), page.GetCurrentPage(),
		tls)

}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantWebhookLog(ctx context.Context, iRequest *protos.GetAssistantWebhookLogRequest) (*protos.GetAssistantWebhookLogResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}

	return assistantGRPCApi.assistantClient.GetAssistantWebhookLog(ctx, iAuth, iRequest)
}

// GetAssistantWebhook implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantWebhook(ctx context.Context, iRequest *protos.GetAssistantWebhookRequest) (*protos.GetAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}

	return assistantGRPCApi.assistantClient.GetAssistantWebhook(ctx, iAuth, iRequest)
}

// GetAssistantWebhook implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantConversation(ctx context.Context, iRequest *protos.GetAssistantConversationRequest) (*protos.GetAssistantConversationResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.GetAssistantConversation(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistant(ctx context.Context, iRequest *protos.DeleteAssistantRequest) (*protos.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistant(ctx, iAuth, iRequest)

}

// CreateAssistantWebhook implements protos.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantKnowledge(ctx context.Context, iRequest *protos.CreateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantKnowledge(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantKnowledge(ctx context.Context, iRequest *protos.UpdateAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Update assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantKnowledge(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistantKnowledge(ctx context.Context, iRequest *protos.DeleteAssistantKnowledgeRequest) (*protos.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Delete assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistantKnowledge(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantKnowledge(ctx context.Context, iRequest *protos.GetAllAssistantKnowledgeRequest) (*protos.GetAllAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantKnowledge(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
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
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create knowledge tag")
		return nil, errors.New("unauthenticated request")
	}

	return assistantGRPCApi.assistantClient.GetAssistantKnowledge(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantTool(ctx context.Context, iRequest *protos.CreateAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantTool(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantTool(ctx context.Context, iRequest *protos.UpdateAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Update assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantTool(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistantTool(ctx context.Context, iRequest *protos.DeleteAssistantToolRequest) (*protos.GetAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Delete assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistantTool(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantTool(ctx context.Context, iRequest *protos.GetAllAssistantToolRequest) (*protos.GetAllAssistantToolResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantTool(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
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
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}

	return assistantGRPCApi.assistantClient.GetAssistantTool(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantAnalysis(ctx context.Context, iRequest *protos.CreateAssistantAnalysisRequest) (*protos.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantAnalysis(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantAnalysis(ctx context.Context, iRequest *protos.UpdateAssistantAnalysisRequest) (*protos.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Update assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantAnalysis(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) DeleteAssistantAnalysis(ctx context.Context, iRequest *protos.DeleteAssistantAnalysisRequest) (*protos.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to Delete assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.DeleteAssistantAnalysis(ctx, iAuth, iRequest)

}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantAnalysis(ctx context.Context, iRequest *protos.GetAllAssistantAnalysisRequest) (*protos.GetAllAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to GetAllAssistantAnalysis")
		return nil, errors.New("unauthenticated request")
	}

	page, tls, err := assistantGRPCApi.assistantClient.GetAllAssistantAnalysis(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllAssistantAnalysisResponse](
			err,
			"Unable to get all the webhook analysis, please try again later.",
		)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantAnalysisResponse, []*protos.AssistantAnalysis](
		page.GetTotalItem(), page.GetCurrentPage(),
		tls)

}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantAnalysis(ctx context.Context, iRequest *protos.GetAssistantAnalysisRequest) (*protos.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to GetAssistantAnalysis")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.GetAssistantAnalysis(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAssistantToolLog(ctx context.Context, iRequest *protos.GetAssistantToolLogRequest) (*protos.GetAssistantToolLogResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to GetAssistantToolLog")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.GetAssistantToolLog(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantToolLog(ctx context.Context, iRequest *protos.GetAllAssistantToolLogRequest) (*protos.GetAllAssistantToolLogResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to GetAllAssistantToolLog")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.GetAllAssistantToolLog(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantTelemetry(ctx context.Context, iRequest *protos.GetAllAssistantTelemetryRequest) (*protos.GetAllAssistantTelemetryResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to GetAllAssistantTelemetry")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.GetAllAssistantTelemetry(ctx, iAuth, iRequest)
}
