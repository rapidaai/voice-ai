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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type AuditServiceClient interface {
	GetAuditLog(c context.Context, auth *types.Authentication, auditId uint64) (*protos.GetAuditLogResponse, error)
	GetAllAuditLog(c context.Context, auth *types.Authentication, req *protos.GetAllAuditLogRequest) (*protos.GetAllAuditLogResponse, error)
}

type auditServiceClient struct {
	clients.InternalClient
	cfg                *config.AppConfig
	logger             commons.Logger
	auditLoggingClient protos.AuditLoggingServiceClient
}

func NewAuditServiceClient(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) AuditServiceClient {
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt64),
			grpc.MaxCallSendMsgSize(math.MaxInt64),
		),
	}
	conn, err := grpc.NewClient(config.Integration.Host, grpcOpts...)
	if err != nil {
		logger.Errorf("Unable to create connection %v", err)
	}
	return &auditServiceClient{
		InternalClient:     clients.NewInternalClient(config, logger, redis),
		cfg:                config,
		logger:             logger,
		auditLoggingClient: protos.NewAuditLoggingServiceClient(conn),
	}
}

func (client *auditServiceClient) GetAuditLog(c context.Context, auth *types.Authentication, auditId uint64) (*protos.GetAuditLogResponse, error) {
	client.logger.Debugf("Calling to get audit log with org and project")
	start := time.Now()
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.auditLoggingClient.GetAuditLog(authContext, &protos.GetAuditLogRequest{
		Id: auditId,
	})
	if err != nil {
		client.logger.Errorf("error while getting audit log error %v", err)
		return nil, err
	}
	client.logger.Debugf("Benchmarking: auditServiceClient.GetAuditLog time taken %v", time.Since(start))
	return res, nil
}
func (client *auditServiceClient) GetAllAuditLog(c context.Context, auth *types.Authentication, req *protos.GetAllAuditLogRequest) (*protos.GetAllAuditLogResponse, error) {
	client.logger.Debugf("Calling to get audit log with org and project")
	start := time.Now()
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.auditLoggingClient.GetAllAuditLog(authContext, req)
	if err != nil {
		client.logger.Errorf("error while getting audit log error %v", err)
		return nil, err
	}
	client.logger.Debugf("Benchmarking: auditServiceClient.GetAllAuditLog time taken %v", time.Since(start))
	return res, nil
}
