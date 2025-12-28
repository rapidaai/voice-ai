// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"errors"

	"github.com/rapidaai/api/assistant-api/config"
	internal_factory "github.com/rapidaai/api/assistant-api/internal/factory"
	internal_grpc "github.com/rapidaai/api/assistant-api/internal/grpc"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

type ConversationApi struct {
	cfg        *config.AssistantConfig
	logger     commons.Logger
	postgres   connectors.PostgresConnector
	redis      connectors.RedisConnector
	opensearch connectors.OpenSearchConnector
	storage    storages.Storage

	assistantConversationService internal_services.AssistantConversationService
	assistantService             internal_services.AssistantService
	vaultClient                  web_client.VaultClient
}

type ConversationGrpcApi struct {
	ConversationApi
}

func NewConversationGRPCApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	vectordb connectors.VectorConnector,
) assistant_api.TalkServiceServer {
	return &ConversationGrpcApi{
		ConversationApi{
			cfg:                          config,
			logger:                       logger,
			postgres:                     postgres,
			redis:                        redis,
			opensearch:                   opensearch,
			assistantConversationService: internal_assistant_service.NewAssistantConversationService(logger, postgres, storage_files.NewStorage(config.AssetStoreConfig, logger)),
			assistantService:             internal_assistant_service.NewAssistantService(config, logger, postgres, opensearch),
			storage:                      storage_files.NewStorage(config.AssetStoreConfig, logger),
			vaultClient:                  web_client.NewVaultClientGRPC(&config.AppConfig, logger, redis),
		},
	}
}

func NewConversationApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	vectordb connectors.VectorConnector,
) *ConversationApi {
	return &ConversationApi{
		cfg:                          config,
		logger:                       logger,
		postgres:                     postgres,
		redis:                        redis,
		opensearch:                   opensearch,
		assistantConversationService: internal_assistant_service.NewAssistantConversationService(logger, postgres, storage_files.NewStorage(config.AssetStoreConfig, logger)),
		assistantService:             internal_assistant_service.NewAssistantService(config, logger, postgres, opensearch),
		storage:                      storage_files.NewStorage(config.AssetStoreConfig, logger),
	}

}

// AssistantTalk handles incoming assistant talk requests.
// It establishes a connection with the client and processes the incoming requests.
//
// Parameters:
// - stream: A server stream for handling bidirectional communication with the client.
//
// Returns:
// - An error if any error occurs during the processing of the request.
func (cApi *ConversationGrpcApi) AssistantTalk(stream assistant_api.TalkService_AssistantTalkServer) error {
	auth, isAuthenticated := types.GetSimplePrincipleGRPC(stream.Context())
	if !isAuthenticated {
		cApi.logger.Errorf("unable to resolve the authentication object, please check the parameter for authentication")
		return errors.New("unauthenticated request for messaging")
	}

	source, ok := utils.GetClientSource(stream.Context())
	if !ok {
		return errors.New("illegal source")
	}
	talker, err := internal_factory.GetTalker(
		source,
		stream.Context(),
		cApi.cfg,
		cApi.logger,
		cApi.postgres,
		cApi.opensearch,
		cApi.redis,
		cApi.storage,
		internal_grpc.NewGrpcUnidirectionalStreamer(stream),
	)
	if err != nil {
		return err
	}

	return talker.
		Talk(
			stream.Context(),
			auth,
			internal_factory.Identifier(source, stream.Context(), auth, ""),
		)
}
