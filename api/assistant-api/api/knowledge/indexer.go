// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	"errors"

	"github.com/rapidaai/api/assistant-api/config"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_knowledge_service "github.com/rapidaai/api/assistant-api/internal/services/knowledge"
	document_client "github.com/rapidaai/pkg/clients/document"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

type indexerApi struct {
	cfg                      *config.AssistantConfig
	logger                   commons.Logger
	postgres                 connectors.PostgresConnector
	redis                    connectors.RedisConnector
	knowledgeService         internal_services.KnowledgeService
	indexerServiceClient     document_client.IndexerServiceClient
	knowledgeDocumentService internal_services.KnowledgeDocumentService
}

type indexerGrpcApi struct {
	indexerApi
}

func NewDocumentGRPCApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
) knowledge_api.DocumentServiceServer {
	return &indexerGrpcApi{
		indexerApi{
			cfg:                      config,
			logger:                   logger,
			postgres:                 postgres,
			redis:                    redis,
			knowledgeService:         internal_knowledge_service.NewKnowledgeService(config, logger, postgres, storage_files.NewStorage(config.AssetStoreConfig, logger)),
			knowledgeDocumentService: internal_knowledge_service.NewKnowledgeDocumentService(config, logger, postgres, opensearch),
			indexerServiceClient:     document_client.NewIndexerServiceClient(&config.AppConfig, logger, redis),
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

	return iApi.indexerServiceClient.IndexKnowledgeDocument(ctx, iAuth,
		&knowledge_api.IndexKnowledgeDocumentRequest{
			KnowledgeId:         cer.GetKnowledgeId(),
			KnowledgeDocumentId: cer.GetKnowledgeDocumentId(),
		})
}
