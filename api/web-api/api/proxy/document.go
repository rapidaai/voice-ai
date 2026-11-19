package web_proxy_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	config "github.com/rapidaai/api/web-api/config"
	document_client "github.com/rapidaai/pkg/clients/document"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	knowledge_api "github.com/rapidaai/protos"
)

type indexerApi struct {
	cfg                  *config.WebAppConfig
	logger               commons.Logger
	postgres             connectors.PostgresConnector
	redis                connectors.RedisConnector
	indexerServiceClient document_client.IndexerServiceClient
}

type indexerGrpcApi struct {
	indexerApi
}

func NewDocumentGRPCApi(config *config.WebAppConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector) knowledge_api.DocumentServiceServer {
	return &indexerGrpcApi{
		indexerApi{
			cfg:                  config,
			logger:               logger,
			postgres:             postgres,
			redis:                redis,
			indexerServiceClient: document_client.NewIndexerServiceClient(&config.AppConfig, logger, redis),
		},
	}
}

func (iApi *indexerApi) IndexKnowledgeDocument(ctx context.Context, cer *knowledge_api.IndexKnowledgeDocumentRequest) (*knowledge_api.IndexKnowledgeDocumentResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	return iApi.indexerServiceClient.IndexKnowledgeDocument(ctx, principal, cer)
}
