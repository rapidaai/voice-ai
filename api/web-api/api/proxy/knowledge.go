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
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
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

func NewKnowledgeGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector, knowledgeClient knowledge_client.KnowledgeServiceClient) protos.KnowledgeServiceServer {
	return &webKnowledgeGRPCApi{
		webKnowledgeApi{
			WebApi:          web_api.NewWebApi(config, logger, postgres, redis),
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			redis:           redis,
			knowledgeClient: knowledgeClient,
		},
	}
}

// GetAllKnowledgeDocumentSegment implements protos.KnowledgeServiceServer.
func (knowledge *webKnowledgeGRPCApi) GetAllKnowledgeDocumentSegment(c context.Context, iRequest *protos.GetAllKnowledgeDocumentSegmentRequest) (*protos.GetAllKnowledgeDocumentSegmentResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledge.knowledgeClient.GetAllKnowledgeDocumentSegment(c, principal, iRequest)
}

func (knowledge *webKnowledgeGRPCApi) GetKnowledge(c context.Context, iRequest *protos.GetKnowledgeRequest) (*protos.GetKnowledgeResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	_knowledge, err := knowledge.knowledgeClient.GetKnowledge(c, principal, iRequest)
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
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _knowledge, err := knowledge.knowledgeClient.GetAllKnowledge(c, principal, iRequest.GetCriterias(), iRequest.GetPaginate())
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
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledge.knowledgeClient.CreateKnowledge(c, principal, iRequest)
}

// CreateKnowledgeTag implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeTag(ctx context.Context, iRequest *protos.CreateKnowledgeTagRequest) (*protos.GetKnowledgeResponse, error) {
	knowledgeGRPCApi.logger.Debugf("Create knowledge provider model request %v, %v", iRequest, ctx)
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeTag(ctx, principal, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) UpdateKnowledgeDetail(ctx context.Context, iRequest *protos.UpdateKnowledgeDetailRequest) (*protos.GetKnowledgeResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.UpdateKnowledgeDetail(ctx, principal, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) CreateKnowledgeDocument(ctx context.Context, iRequest *protos.CreateKnowledgeDocumentRequest) (*protos.CreateKnowledgeDocumentResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.CreateKnowledgeDocument(ctx, principal, iRequest)
}

// GetAllKnowledgeDocument implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetAllKnowledgeDocument(ctx context.Context, iRequest *protos.GetAllKnowledgeDocumentRequest) (*protos.GetAllKnowledgeDocumentResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.GetAllKnowledgeDocument(ctx, principal, iRequest)
}

// GetAllKnowledgeDocument implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) DeleteKnowledgeDocumentSegment(ctx context.Context, iRequest *protos.DeleteKnowledgeDocumentSegmentRequest) (*protos.BaseResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.DeleteKnowledgeDocumentSegment(ctx, principal, iRequest)
}

// GetAllKnowledgeDocument implements protos.KnowledgeServiceServer.
func (knowledgeGRPCApi *webKnowledgeGRPCApi) UpdateKnowledgeDocumentSegment(ctx context.Context, iRequest *protos.UpdateKnowledgeDocumentSegmentRequest) (*protos.BaseResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.UpdateKnowledgeDocumentSegment(ctx, principal, iRequest)
}

func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetAllKnowledgeLog(ctx context.Context, iRequest *protos.GetAllKnowledgeLogRequest) (*protos.GetAllKnowledgeLogResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.GetAllKnowledgeLog(ctx, principal, iRequest)
}
func (knowledgeGRPCApi *webKnowledgeGRPCApi) GetKnowledgeLog(ctx context.Context, iRequest *protos.GetKnowledgeLogRequest) (*protos.GetKnowledgeLogResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return knowledgeGRPCApi.knowledgeClient.GetKnowledgeLog(ctx, principal, iRequest)
}
