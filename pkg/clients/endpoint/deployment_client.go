// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_client

import (
	"context"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	endpoint_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DeploymentServiceClient interface {
	Invoke(ctx context.Context, auth types.SimplePrinciple, iRequest *endpoint_api.InvokeRequest) (*endpoint_api.InvokeResponse, error)
}

type deploymentServiceClient struct {
	clients.InternalClient
	cfg              *config.AppConfig
	logger           commons.Logger
	deploymentClient endpoint_api.DeploymentClient
}

func NewDeploymentServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) DeploymentServiceClient {

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(commons.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(commons.MaxSendMsgSize),
		),
	}
	conn, err := grpc.NewClient(config.EndpointHost,
		grpcOpts...)

	if err != nil {
		logger.Errorf("Unable to create connection %v", err)
	}
	return &deploymentServiceClient{
		InternalClient:   clients.NewInternalClient(config, logger, redis),
		cfg:              config,
		logger:           logger,
		deploymentClient: endpoint_api.NewDeploymentClient(conn),
	}
}

func (dsc *deploymentServiceClient) Invoke(ctx context.Context, auth types.SimplePrinciple, iRequest *endpoint_api.InvokeRequest) (*endpoint_api.InvokeResponse, error) {
	dsc.logger.Debugf("invoke api for endpoint")
	res, err := dsc.deploymentClient.Invoke(dsc.WithAuth(ctx, auth), iRequest)
	if err != nil {
		dsc.logger.Errorf("error while calling invoke endpoint %v", err)
		return nil, err
	}

	return res, nil
}
