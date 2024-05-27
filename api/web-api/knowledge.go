package web_api

import (
	"context"
	"errors"

	knowledge_client "github.com/lexatic/web-backend/pkg/clients/workflow"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webKnowledgeApi struct {
	WebApi
	cfg             *config.AppConfig
	logger          commons.Logger
	postgres        connectors.PostgresConnector
	redis           connectors.RedisConnector
	knowledgeClient knowledge_client.KnowledgeServiceClient
}

type webKnowledgeGRPCApi struct {
	webKnowledgeApi
}

// CreateKnowledgeDocument implements lexatic_backend.KnowledgeServiceServer.

func NewKnowledgeGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.KnowledgeServiceServer {
	return &webKnowledgeGRPCApi{
		webKnowledgeApi{
			WebApi:          NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			knowledgeClient: knowledge_client.NewKnowledgeServiceClientGRPC(config, logger, redis),
		},
	}
}

func (knowledge *webKnowledgeGRPCApi) GetKnowledge(c context.Context, iRequest *web_api.GetKnowledgeRequest) (*web_api.GetKnowledgeResponse, error) {
	knowledge.logger.Debugf("GetKnowledge from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		knowledge.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	_knowledge, err := knowledge.knowledgeClient.GetKnowledge(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[web_api.GetKnowledgeResponse](
			err,
			"Unable to get your knowledge, please try again in sometime.")
	}

	_knowledge.CreatedUser = knowledge.GetUser(c, iAuth, _knowledge.GetCreatedBy())
	_knowledge.EmbeddingProviderModel = knowledge.GetProviderModel(c, iAuth, _knowledge.GetEmbeddingProviderModelId())

	return utils.Success[web_api.GetKnowledgeResponse, *web_api.Knowledge](_knowledge)

}

/*
 */

/*
 */
func (knowledge *webKnowledgeGRPCApi) GetAllKnowledge(c context.Context, iRequest *web_api.GetAllKnowledgeRequest) (*web_api.GetAllKnowledgeResponse, error) {
	knowledge.logger.Debugf("GetAllKnowledge from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		knowledge.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _knowledge, err := knowledge.knowledgeClient.GetAllKnowledge(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllKnowledgeResponse](
			err,
			"Unable to get your knowledge, please try again in sometime.")
	}

	for _, _ep := range _knowledge {
		_ep.CreatedUser = knowledge.GetUser(c, iAuth, _ep.GetCreatedBy())
		_ep.EmbeddingProviderModel = knowledge.GetProviderModel(c, iAuth, _ep.GetEmbeddingProviderModelId())
	}
	return utils.PaginatedSuccess[web_api.GetAllKnowledgeResponse, []*web_api.Knowledge](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_knowledge)
}

func (knowledge *webKnowledgeGRPCApi) CreateKnowledge(c context.Context, iRequest *web_api.CreateKnowledgeRequest) (*web_api.CreateKnowledgeResponse, error) {
	knowledge.logger.Debugf("Create knowledge from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		knowledge.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return knowledge.knowledgeClient.CreateKnowledge(c, iAuth, iRequest)
}

// CreateKnowledgeTag implements lexatic_backend.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeTag(ctx context.Context, iRequest *web_api.CreateKnowledgeTagRequest) (*web_api.CreateKnowledgeTagResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Create knowledge provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to create knowledge tag")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeTag(ctx, iAuth, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeDocument(ctx context.Context, iRequest *web_api.CreateKnowledgeDocumentRequest) (*web_api.CreateKnowledgeDocumentResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Create knowledge document request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to create knowledge document")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeDocument(ctx, iAuth, iRequest)
}

// GetAllKnowledgeDocument implements lexatic_backend.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetAllKnowledgeDocument(ctx context.Context, iRequest *web_api.GetAllKnowledgeDocumentRequest) (*web_api.GetAllKnowledgeDocumentResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Get all knowledge document request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to get all knowledge document")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.GetAllKnowledgeDocument(ctx, iAuth, iRequest)
}
