// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"github.com/rapidaai/api/assistant-api/config"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"

	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	assistant_api "github.com/rapidaai/protos"
)

type assistantDeploymentApi struct {
	cfg               *config.AssistantConfig
	logger            commons.Logger
	postgres          connectors.PostgresConnector
	deploymentService internal_services.AssistantDeploymentService
	storage           storages.Storage
}

type assistantDeploymentGrpcApi struct {
	assistantDeploymentApi
}

func NewAssistantDeploymentGRPCApi(config *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
) assistant_api.AssistantDeploymentServiceServer {
	return &assistantDeploymentGrpcApi{
		assistantDeploymentApi{
			cfg:               config,
			logger:            logger,
			postgres:          postgres,
			deploymentService: internal_assistant_service.NewAssistantDeploymentService(config, logger, postgres),
			storage:           storage_files.NewStorage(config.AssetStoreConfig, logger),
		},
	}
}
