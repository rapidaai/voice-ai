// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_api

import (
	"github.com/rapidaai/api/endpoint-api/config"
	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	internal_endpoint_service "github.com/rapidaai/api/endpoint-api/internal/service/endpoint"
	internal_log_service "github.com/rapidaai/api/endpoint-api/internal/service/log"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/protos"
)

type endpointApi struct {
	cfg                *config.EndpointConfig
	logger             commons.Logger
	postgres           connectors.PostgresConnector
	endpointService    internal_services.EndpointService
	endpointLogService internal_services.EndpointLogService
}

type endpointGRPCApi struct {
	endpointApi
}

func NewEndpointGRPCApi(config *config.EndpointConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
) protos.EndpointServiceServer {
	return &endpointGRPCApi{
		endpointApi{
			cfg:                config,
			logger:             logger,
			postgres:           postgres,
			endpointService:    internal_endpoint_service.NewEndpointService(config, logger, postgres),
			endpointLogService: internal_log_service.NewEndpointLogService(logger, postgres),
		},
	}
}
