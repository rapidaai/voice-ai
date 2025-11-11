package web_proxy_api

import (
	"context"
	"errors"

	knowledge_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type webKnowledgeApi struct {
	web_api.WebApi
	cfg             *config.WebAppConfig
	logger          commons.Logger
	postgres        connectors.PostgresConnector
	redis           connectors.RedisConnector
	knowledgeClient knowledge_client.KnowledgeServiceClient
}

type webKnowledgeGRPCApi struct {
	webKnowledgeApi
}

func NewKnowledgeGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.KnowledgeServiceServer {
	return &webKnowledgeGRPCApi{
		webKnowledgeApi{
			WebApi:          web_api.NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			knowledgeClient: knowledge_client.NewKnowledgeServiceClientGRPC(&config.AppConfig, logger, redis),
		},
	}
}

// GetAllKnowledgeDocumentSegment implements lexatic_backend.KnowledgeServiceServer.
func (knowledge *webKnowledgeGRPCApi) GetAllKnowledgeDocumentSegment(c context.Context, iRequest *protos.GetAllKnowledgeDocumentSegmentRequest) (*protos.GetAllKnowledgeDocumentSegmentResponse, error) {
	knowledge.logger.Debugf("GetAllKnowledgeDocumentSegment from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		knowledge.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return knowledge.knowledgeClient.GetAllKnowledgeDocumentSegment(c, iAuth, iRequest)
}

func (knowledge *webKnowledgeGRPCApi) GetKnowledge(c context.Context, iRequest *protos.GetKnowledgeRequest) (*protos.GetKnowledgeResponse, error) {
	knowledge.logger.Debugf("GetKnowledge from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		knowledge.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	_knowledge, err := knowledge.knowledgeClient.GetKnowledge(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[protos.GetKnowledgeResponse](
			err,
			"Unable to get your knowledge, please try again in sometime.")
	}

	_knowledge.CreatedUser = knowledge.GetUser(c, iAuth, _knowledge.GetCreatedBy())
	return utils.Success[protos.GetKnowledgeResponse, *protos.Knowledge](_knowledge)

}

/*
 */

/*
 */
func (knowledge *webKnowledgeGRPCApi) GetAllKnowledge(c context.Context, iRequest *protos.GetAllKnowledgeRequest) (*protos.GetAllKnowledgeResponse, error) {
	knowledge.logger.Debugf("GetAllKnowledge from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		knowledge.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _knowledge, err := knowledge.knowledgeClient.GetAllKnowledge(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllKnowledgeResponse](
			err,
			"Unable to get your knowledge, please try again in sometime.")
	}

	for _, _ep := range _knowledge {
		_ep.CreatedUser = knowledge.GetUser(c, iAuth, _ep.GetCreatedBy())
	}
	return utils.PaginatedSuccess[protos.GetAllKnowledgeResponse, []*protos.Knowledge](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_knowledge)
}

func (knowledge *webKnowledgeGRPCApi) CreateKnowledge(c context.Context, iRequest *protos.CreateKnowledgeRequest) (*protos.CreateKnowledgeResponse, error) {
	knowledge.logger.Debugf("Create knowledge from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		knowledge.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return knowledge.knowledgeClient.CreateKnowledge(c, iAuth, iRequest)
}

// CreateKnowledgeTag implements lexatic_backend.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeTag(ctx context.Context, iRequest *protos.CreateKnowledgeTagRequest) (*protos.GetKnowledgeResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Create knowledge provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to create knowledge tag")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeTag(ctx, iAuth, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) UpdateKnowledgeDetail(ctx context.Context, iRequest *protos.UpdateKnowledgeDetailRequest) (*protos.GetKnowledgeResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Create knowledge provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to create knowledge tag")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.UpdateKnowledgeDetail(ctx, iAuth, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeDocument(ctx context.Context, iRequest *protos.CreateKnowledgeDocumentRequest) (*protos.CreateKnowledgeDocumentResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Create knowledge document request request %+v", iRequest)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to create knowledge document")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeDocument(ctx, iAuth, iRequest)
}

// GetAllKnowledgeDocument implements lexatic_backend.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetAllKnowledgeDocument(ctx context.Context, iRequest *protos.GetAllKnowledgeDocumentRequest) (*protos.GetAllKnowledgeDocumentResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Get all knowledge document request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to get all knowledge document")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.GetAllKnowledgeDocument(ctx, iAuth, iRequest)
}

// GetAllKnowledgeDocument implements lexatic_backend.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) DeleteKnowledgeDocumentSegment(ctx context.Context, iRequest *protos.DeleteKnowledgeDocumentSegmentRequest) (*protos.BaseResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to delete knowledge document segment")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.DeleteKnowledgeDocumentSegment(ctx, iAuth, iRequest)
}

// GetAllKnowledgeDocument implements lexatic_backend.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) UpdateKnowledgeDocumentSegment(ctx context.Context, iRequest *protos.UpdateKnowledgeDocumentSegmentRequest) (*protos.BaseResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to update knowledge document segment")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.UpdateKnowledgeDocumentSegment(ctx, iAuth, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetAllKnowledgeLog(ctx context.Context, iRequest *protos.GetAllKnowledgeLogRequest) (*protos.GetAllKnowledgeLogResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to GetAllKnowledgeLog")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.GetAllKnowledgeLog(ctx, iAuth, iRequest)
}
func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetKnowledgeLog(ctx context.Context, iRequest *protos.GetKnowledgeLogRequest) (*protos.GetKnowledgeLogResponse, error) {
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		knowledgeGRPCApi.logger.Errorf("unauthenticated request to GetKnowledgeLog")
		return nil, errors.New("unauthenticated request")
	}
	return knowledgeGRPCApi.knowledgeClient.GetKnowledgeLog(ctx, iAuth, iRequest)
}
