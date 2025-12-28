// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"github.com/rapidaai/api/assistant-api/config"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	internal_knowledge_service "github.com/rapidaai/api/assistant-api/internal/services/knowledge"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/protos"
)

type assistantApi struct {
	cfg                       *config.AssistantConfig
	logger                    commons.Logger
	postgres                  connectors.PostgresConnector
	redis                     connectors.RedisConnector
	opensearch                connectors.OpenSearchConnector
	vectordb                  connectors.VectorConnector
	assistantService          internal_services.AssistantService
	knowledgeDocumentService  internal_services.KnowledgeDocumentService
	conversactionService      internal_services.AssistantConversationService
	assistantWebhookService   internal_services.AssistantWebhookService
	assistantAnalysisService  internal_services.AssistantAnalysisService
	assistantToolService      internal_services.AssistantToolService
	assistantKnowledgeService internal_services.AssistantKnowledgeService
}

type assistantGrpcApi struct {
	assistantApi
}

func NewAssistantGRPCApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	vectordb connectors.VectorConnector,

) protos.AssistantServiceServer {
	return &assistantGrpcApi{
		assistantApi{
			cfg:                      config,
			logger:                   logger,
			postgres:                 postgres,
			redis:                    redis,
			opensearch:               opensearch,
			vectordb:                 vectordb,
			assistantService:         internal_assistant_service.NewAssistantService(config, logger, postgres, opensearch),
			knowledgeDocumentService: internal_knowledge_service.NewKnowledgeDocumentService(config, logger, postgres, opensearch),
			conversactionService:     internal_assistant_service.NewAssistantConversationService(logger, postgres, storage_files.NewStorage(config.AssetStoreConfig, logger)),
			assistantWebhookService: internal_assistant_service.NewAssistantWebhookService(logger, postgres,
				storage_files.NewStorage(config.AssetStoreConfig, logger)),
			assistantAnalysisService:  internal_assistant_service.NewAssistantAnalysisService(logger, postgres),
			assistantToolService:      internal_assistant_service.NewAssistantToolService(logger, postgres, storage_files.NewStorage(config.AssetStoreConfig, logger)),
			assistantKnowledgeService: internal_assistant_service.NewAssistantKnowledgeService(logger, postgres, storage_files.NewStorage(config.AssetStoreConfig, logger)),
		},
	}
}
