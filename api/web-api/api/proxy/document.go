package web_proxy_api

import (
	"context"
	"errors"

	config "github.com/rapidaai/api/web-api/config"
	document_client "github.com/rapidaai/pkg/clients/document"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
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
	iApi.logger.Debugf("index document request %v, %v", cer, ctx)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		iApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.IndexKnowledgeDocumentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	return iApi.indexerServiceClient.IndexKnowledgeDocument(ctx, iAuth, cer)
}
