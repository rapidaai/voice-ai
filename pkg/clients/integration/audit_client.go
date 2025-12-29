// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package integration_client

import (
	"context"
	"math"
	"time"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	integration_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuditServiceClient interface {
	GetAuditLog(c context.Context, auth types.SimplePrinciple, auditId uint64) (*integration_api.GetAuditLogResponse, error)
	GetAllAuditLog(c context.Context, auth types.SimplePrinciple, req *integration_api.GetAllAuditLogRequest) (*integration_api.GetAllAuditLogResponse, error)
}

type auditServiceClient struct {
	clients.InternalClient
	cfg                *config.AppConfig
	logger             commons.Logger
	auditLoggingClient integration_api.AuditLoggingServiceClient
}

func NewAuditServiceClient(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) AuditServiceClient {
	logger.Debugf("conntecting to integration client with %s", config.IntegrationHost)

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt64),
			grpc.MaxCallSendMsgSize(math.MaxInt64),
		),
	}
	conn, err := grpc.NewClient(config.IntegrationHost,
		grpcOpts...)

	if err != nil {
		logger.Errorf("Unable to create connection %v", err)
	}
	return &auditServiceClient{
		InternalClient:     clients.NewInternalClient(config, logger, redis),
		cfg:                config,
		logger:             logger,
		auditLoggingClient: integration_api.NewAuditLoggingServiceClient(conn),
	}
}

func (client *auditServiceClient) GetAuditLog(c context.Context, auth types.SimplePrinciple, auditId uint64) (*integration_api.GetAuditLogResponse, error) {
	client.logger.Debugf("Calling to get audit log with org and project")
	start := time.Now()
	res, err := client.auditLoggingClient.GetAuditLog(client.WithAuth(c, auth), &integration_api.GetAuditLogRequest{
		Id: auditId,
	})
	if err != nil {
		client.logger.Errorf("error while getting audit log error %v", err)
		return nil, err
	}
	client.logger.Debugf("Benchmarking: auditServiceClient.GetAuditLog time taken %v", time.Since(start))
	return res, nil
}
func (client *auditServiceClient) GetAllAuditLog(c context.Context, auth types.SimplePrinciple, req *integration_api.GetAllAuditLogRequest) (*integration_api.GetAllAuditLogResponse, error) {
	client.logger.Debugf("Calling to get audit log with org and project")
	start := time.Now()
	res, err := client.auditLoggingClient.GetAllAuditLog(client.WithAuth(c, auth), req)
	if err != nil {
		client.logger.Errorf("error while getting audit log error %v", err)
		return nil, err
	}
	client.logger.Debugf("Benchmarking: auditServiceClient.GetAllAuditLog time taken %v", time.Since(start))
	return res, nil
}
