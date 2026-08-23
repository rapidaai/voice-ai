// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package workflow_client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type ObservabilityServiceClient interface {
	GetAllTelemetry(ctx context.Context, auth *types.Authentication, in *protos.GetAllTelemetryRequest) (*protos.GetAllTelemetryResponse, error)
}

type observabilityServiceClient struct {
	clients.InternalClient
	cfg                 *config.AppConfig
	logger              commons.Logger
	observabilityClient protos.ObservabilityServiceClient
}

func NewObservabilityServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) ObservabilityServiceClient {
	conn, err := grpc.NewClient(config.Assistant.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Errorf("Unable to create connection %v", err)
	}
	return &observabilityServiceClient{
		cfg:                 config,
		logger:              logger,
		InternalClient:      clients.NewInternalClient(config, logger, redis),
		observabilityClient: protos.NewObservabilityServiceClient(conn),
	}
}

func (client *observabilityServiceClient) GetAllTelemetry(ctx context.Context, auth *types.Authentication, in *protos.GetAllTelemetryRequest) (*protos.GetAllTelemetryResponse, error) {
	start := time.Now()
	authContext, err := client.WithAuth(ctx, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.observabilityClient.GetAllTelemetry(authContext, in)
	if err != nil {
		client.logger.Benchmark("Benchmarking: observabilityClient.GetAllTelemetry", time.Since(start))
		client.logger.Errorf("error while calling GetAllTelemetry %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling GetAllTelemetry %v", err)
	}
	client.logger.Benchmark("Benchmarking: observabilityClient.GetAllTelemetry", time.Since(start))
	return res, nil
}
