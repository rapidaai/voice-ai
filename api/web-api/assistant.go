package web_api

import (
	"context"
	"errors"
	"io"

	assistant_client "github.com/lexatic/web-backend/pkg/clients/workflow"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webAssistantApi struct {
	WebApi
	cfg                          *config.AppConfig
	logger                       commons.Logger
	postgres                     connectors.PostgresConnector
	redis                        connectors.RedisConnector
	assistantClient              assistant_client.AssistantServiceClient
	assistantConversactionClient assistant_client.AssistantConversactionServiceClient
}

type webAssistantGRPCApi struct {
	webAssistantApi
}

// CreateAssistantKnowledgeConfiguration implements lexatic_backend.AssistantServiceServer.
func (assistant *webAssistantGRPCApi) CreateAssistantKnowledgeConfiguration(c context.Context, iRequest *web_api.CreateAssistantKnowledgeConfigurationRequest) (*web_api.GetAssistantResponse, error) {
	assistant.logger.Debugf("GetAssistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return assistant.assistantClient.CreateAssistantKnowledgeConfiguration(c, iAuth, iRequest)

}

func NewAssistantGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.AssistantServiceServer {
	return &webAssistantGRPCApi{
		webAssistantApi{
			WebApi:          NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			assistantClient: assistant_client.NewAssistantServiceClientGRPC(config, logger, redis),
		},
	}
}

func NewAssistantConversactionGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.AssistantConversactionServiceServer {
	return &webAssistantGRPCApi{
		webAssistantApi{
			WebApi:                       NewWebApi(config, logger, postgres, redis),
			cfg:                          config,
			logger:                       logger,
			postgres:                     postgres,
			redis:                        redis,
			assistantConversactionClient: assistant_client.NewAssistantConversactionServiceClientGRPC(config, logger, redis),
		},
	}
}

//
//

// GetAllConversactionMessage implements lexatic_backend.AssistantConversactionServiceServer.
func (assistant *webAssistantGRPCApi) GetAllConversactionMessage(ctx context.Context, iRequest *web_api.GetAllConversactionMessageRequest) (*web_api.GetAllConversactionMessageResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _assistant, err := assistant.assistantConversactionClient.GetAllConversactionMessage(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetAssistantConversactionId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllConversactionMessageResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	return utils.PaginatedSuccess[web_api.GetAllConversactionMessageResponse, []*web_api.AssistantConversactionMessage](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)

}

func (assistant *webAssistantGRPCApi) CreateAssistantMessage(cer *web_api.CreateAssistantMessageRequest, stream web_api.AssistantConversactionService_CreateAssistantMessageServer) error {
	c := stream.Context()
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return errors.New("unauthenticated request")
	}
	out, err := assistant.assistantConversactionClient.CreateAssistantMessage(c, iAuth, cer)
	if err != nil {
		return err
	}

	// Channel to handle errors from the upstream stream
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		for {
			st, recvErr := out.Recv()
			if recvErr == io.EOF {
				return // End of upstream stream
			}
			if recvErr != nil {
				errCh <- recvErr
				return
			}
			// Forward message to downstream stream
			if err := stream.Send(st); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Wait for any errors from the upstream stream or the context cancellation
	select {
	case err := <-errCh:
		return err
	case <-c.Done():
		return c.Err()
	}

}

// CreateAssistantMessage implements lexatic_backend.AssistantConversactionServiceServer.
// func (assistant *webAssistantGRPCApi) CreateAssistantMessage(c context.Context, iRequest *web_api.CreateAssistantMessageRequest) (web_api.AssistantConversactionService_CreateAssistantMessageClient, error) {
// 	assistant.logger.Debugf("Create assistant from grpc with requestPayload %v, %v", iRequest, c)
// 	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
// 	if !isAuthenticated {
// 		assistant.logger.Errorf("unauthenticated request for get actvities")
// 		return nil, errors.New("unauthenticated request")
// 	}

// 	return assistant.assistantConversactionClient.CreateAssistantMessage(c, iAuth, iRequest)
// }

// GetAllAssistantConversaction implements lexatic_backend.AssistantConversactionServiceServer.
func (assistant *webAssistantGRPCApi) GetAllAssistantConversaction(c context.Context, iRequest *web_api.GetAllAssistantConversactionRequest) (*web_api.GetAllAssistantConversactionResponse, error) {
	assistant.logger.Debugf("GetAllAssistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _assistantConvo, err := assistant.assistantConversactionClient.GetAllAssistantConversaction(c, iAuth,
		iRequest.GetAssistantId(),
		iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllAssistantConversactionResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	for _, _ep := range _assistantConvo {
		_ep.User = assistant.GetUser(c, iAuth, _ep.GetUserId())
	}
	return utils.PaginatedSuccess[web_api.GetAllAssistantConversactionResponse, []*web_api.AssistantConversaction](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistantConvo)
}

//
//
//

func (assistant *webAssistantGRPCApi) GetAssistant(c context.Context, iRequest *web_api.GetAssistantRequest) (*web_api.GetAssistantResponse, error) {
	assistant.logger.Debugf("GetAssistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	_assistant, err := assistant.assistantClient.GetAssistant(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[web_api.GetAssistantResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	if _assistant.AssistantProviderModel != nil {
		_assistant.AssistantProviderModel.CreatedUser = assistant.GetUser(c, iAuth, _assistant.AssistantProviderModel.GetCreatedBy())
		_assistant.AssistantProviderModel.ProviderModel = assistant.GetProviderModel(c, iAuth, _assistant.AssistantProviderModel.GetProviderModelId())
	}

	return utils.Success[web_api.GetAssistantResponse, *web_api.Assistant](_assistant)

}

/*
 */

/*
 */
func (assistant *webAssistantGRPCApi) GetAllAssistant(c context.Context, iRequest *web_api.GetAllAssistantRequest) (*web_api.GetAllAssistantResponse, error) {
	assistant.logger.Debugf("GetAllAssistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _assistant, err := assistant.assistantClient.GetAllAssistant(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllAssistantResponse](
			err,
			"Unable to get your assistant, please try again in sometime.")
	}

	for _, _ep := range _assistant {
		if _ep.GetAssistantProviderModel() != nil {
			_ep.AssistantProviderModel.CreatedUser = assistant.GetUser(c, iAuth, _ep.AssistantProviderModel.GetCreatedBy())
			_ep.AssistantProviderModel.ProviderModel = assistant.GetProviderModel(c, iAuth, _ep.AssistantProviderModel.GetProviderModelId())
		}
	}
	return utils.PaginatedSuccess[web_api.GetAllAssistantResponse, []*web_api.Assistant](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistant *webAssistantGRPCApi) CreateAssistant(c context.Context, iRequest *web_api.CreateAssistantRequest) (*web_api.CreateAssistantResponse, error) {
	assistant.logger.Debugf("Create assistant from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		assistant.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return assistant.assistantClient.CreateAssistant(c, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) GetAllAssistantProviderModel(ctx context.Context, iRequest *web_api.GetAllAssistantProviderModelRequest) (*web_api.GetAllAssistantProviderModelResponse, error) {
	assistantGRPCApi.logger.Debugf("Create assistant from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _assistant, err := assistantGRPCApi.assistantClient.GetAllAssistantProviderModel(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllAssistantProviderModelResponse](
			err,
			"Unable to get your assistant provider models, please try again in sometime.")
	}

	for _, _ep := range _assistant {
		_ep.CreatedUser = assistantGRPCApi.GetUser(ctx, iAuth, _ep.GetCreatedBy())
		_ep.ProviderModel = assistantGRPCApi.GetProviderModel(ctx, iAuth, _ep.GetProviderModelId())
	}
	return utils.PaginatedSuccess[web_api.GetAllAssistantProviderModelResponse, []*web_api.AssistantProviderModel](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_assistant)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantVersion(ctx context.Context, iRequest *web_api.UpdateAssistantVersionRequest) (*web_api.UpdateAssistantVersionResponse, error) {
	assistantGRPCApi.logger.Debugf("Update assistant from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantVersion(ctx, iAuth, iRequest.GetAssistantId(), iRequest.GetAssistantProviderModelId())
}

func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantProviderModel(ctx context.Context, iRequest *web_api.CreateAssistantProviderModelRequest) (*web_api.CreateAssistantProviderModelResponse, error) {
	assistantGRPCApi.logger.Debugf("Create assistant provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant provider model")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantProviderModel(ctx, iAuth, iRequest)
}

// CreateAssistantTag implements lexatic_backend.AssistantServiceServer.
func (assistantGRPCApi *webAssistantGRPCApi) CreateAssistantTag(ctx context.Context, iRequest *web_api.CreateAssistantTagRequest) (*web_api.GetAssistantResponse, error) {
	assistantGRPCApi.logger.Debugf("Create assistant provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.CreateAssistantTag(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantDetail(ctx context.Context, iRequest *web_api.UpdateAssistantDetailRequest) (*web_api.GetAssistantResponse, error) {
	assistantGRPCApi.logger.Debugf("update assistant request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantDetail(ctx, iAuth, iRequest)
}

func (assistantGRPCApi *webAssistantGRPCApi) UpdateAssistantOpeningStatement(ctx context.Context, iRequest *web_api.UpdateAssistantOpeningStatementRequest) (*web_api.GetAssistantResponse, error) {
	assistantGRPCApi.logger.Debugf("update assistant request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		assistantGRPCApi.logger.Errorf("unauthenticated request to create assistant tag")
		return nil, errors.New("unauthenticated request")
	}
	return assistantGRPCApi.assistantClient.UpdateAssistantOpeningStatement(ctx, iAuth, iRequest)
}
