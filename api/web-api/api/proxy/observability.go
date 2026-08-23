// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package web_proxy_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	workflow_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type webObservabilityApi struct {
	web_api.WebApi
	cfg                 *config.WebAppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	redis               connectors.RedisConnector
	observabilityClient workflow_client.ObservabilityServiceClient
}

type webObservabilityGRPCApi struct {
	webObservabilityApi
}

func NewObservabilityGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.ObservabilityServiceServer {
	return &webObservabilityGRPCApi{
		webObservabilityApi{
			WebApi:              web_api.NewWebApi(config, logger, postgres, redis),
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			redis:               redis,
			observabilityClient: workflow_client.NewObservabilityServiceClientGRPC(&config.AppConfig, logger, redis),
		},
	}
}

func (api *webObservabilityGRPCApi) GetAllTelemetry(ctx context.Context, request *protos.GetAllTelemetryRequest) (*protos.GetAllTelemetryResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return api.observabilityClient.GetAllTelemetry(ctx, iAuth, request)
}
