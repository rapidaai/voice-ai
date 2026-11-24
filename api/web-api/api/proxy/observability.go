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
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
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

func NewObservabilityGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector, observabilityClient workflow_client.ObservabilityServiceClient) protos.ObservabilityServiceServer {
	return &webObservabilityGRPCApi{
		webObservabilityApi{
			WebApi:              web_api.NewWebApi(config, logger, postgres, redis),
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			redis:               redis,
			observabilityClient: observabilityClient,
		},
	}
}

func (api *webObservabilityGRPCApi) GetAllTelemetry(ctx context.Context, request *protos.GetAllTelemetryRequest) (*protos.GetAllTelemetryResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return api.observabilityClient.GetAllTelemetry(ctx, principal, request)
}
