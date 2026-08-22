package web_proxy_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	knowledge_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
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

// GetAllKnowledgeDocumentSegment implements protos.KnowledgeServiceServer.
func (knowledge *webKnowledgeGRPCApi) GetAllKnowledgeDocumentSegment(c context.Context, iRequest *protos.GetAllKnowledgeDocumentSegmentRequest) (*protos.GetAllKnowledgeDocumentSegmentResponse, error) {
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledge.knowledgeClient.GetAllKnowledgeDocumentSegment(c, iAuth, iRequest)
}

func (knowledge *webKnowledgeGRPCApi) GetKnowledge(c context.Context, iRequest *protos.GetKnowledgeRequest) (*protos.GetKnowledgeResponse, error) {
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	_knowledge, err := knowledge.knowledgeClient.GetKnowledge(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[protos.GetKnowledgeResponse](
			err,
			"Unable to get your knowledge, please try again in sometime.")
	}

	return utils.Success[protos.GetKnowledgeResponse, *protos.Knowledge](_knowledge)
}

/*
 */

/*
 */
func (knowledge *webKnowledgeGRPCApi) GetAllKnowledge(c context.Context, iRequest *protos.GetAllKnowledgeRequest) (*protos.GetAllKnowledgeResponse, error) {
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	_page, _knowledge, err := knowledge.knowledgeClient.GetAllKnowledge(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllKnowledgeResponse](
			err,
			"Unable to get your knowledge, please try again in sometime.")
	}

	return utils.PaginatedSuccess[protos.GetAllKnowledgeResponse, []*protos.Knowledge](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_knowledge)
}

func (knowledge *webKnowledgeGRPCApi) CreateKnowledge(c context.Context, iRequest *protos.CreateKnowledgeRequest) (*protos.CreateKnowledgeResponse, error) {
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledge.knowledgeClient.CreateKnowledge(c, iAuth, iRequest)
}

// CreateKnowledgeTag implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeTag(ctx context.Context, iRequest *protos.CreateKnowledgeTagRequest) (*protos.GetKnowledgeResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Create knowledge provider model request %v, %v", iRequest, ctx)
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeTag(ctx, iAuth, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) UpdateKnowledgeDetail(ctx context.Context, iRequest *protos.UpdateKnowledgeDetailRequest) (*protos.GetKnowledgeResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.UpdateKnowledgeDetail(ctx, iAuth, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeDocument(ctx context.Context, iRequest *protos.CreateKnowledgeDocumentRequest) (*protos.CreateKnowledgeDocumentResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeDocument(ctx, iAuth, iRequest)
}

// GetAllKnowledgeDocument implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetAllKnowledgeDocument(ctx context.Context, iRequest *protos.GetAllKnowledgeDocumentRequest) (*protos.GetAllKnowledgeDocumentResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.GetAllKnowledgeDocument(ctx, iAuth, iRequest)
}

// GetAllKnowledgeDocument implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) DeleteKnowledgeDocumentSegment(ctx context.Context, iRequest *protos.DeleteKnowledgeDocumentSegmentRequest) (*protos.BaseResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.DeleteKnowledgeDocumentSegment(ctx, iAuth, iRequest)
}

// GetAllKnowledgeDocument implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) UpdateKnowledgeDocumentSegment(ctx context.Context, iRequest *protos.UpdateKnowledgeDocumentSegmentRequest) (*protos.BaseResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.UpdateKnowledgeDocumentSegment(ctx, iAuth, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetAllKnowledgeLog(ctx context.Context, iRequest *protos.GetAllKnowledgeLogRequest) (*protos.GetAllKnowledgeLogResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.GetAllKnowledgeLog(ctx, iAuth, iRequest)
}
func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetKnowledgeLog(ctx context.Context, iRequest *protos.GetKnowledgeLogRequest) (*protos.GetKnowledgeLogResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return knowledgeGRPCApi.knowledgeClient.GetKnowledgeLog(ctx, iAuth, iRequest)
}
