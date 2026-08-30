// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type ProductUsageClient interface {
	CreateProductUsage(context.Context, *types.Authentication, *protos.CreateProductUsageRequest) (*protos.GetProductUsageResponse, error)
}

type productUsageServiceClient struct {
	clients.InternalClient
	cfg                *config.AppConfig
	logger             commons.Logger
	productUsageClient protos.ProductUsageServiceClient
}

func NewProductUsageServiceClientGRPC(config *config.AppConfig, logger commons.Logger) ProductUsageClient {
	connection, err := grpc.NewClient(config.Web.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return &productUsageServiceClient{
		InternalClient:     clients.NewInternalClient(config, logger, nil),
		cfg:                config,
		logger:             logger,
		productUsageClient: protos.NewProductUsageServiceClient(connection),
	}
}

func (client productUsageServiceClient) CreateProductUsage(ctx context.Context, auth *types.Authentication, request *protos.CreateProductUsageRequest) (*protos.GetProductUsageResponse, error) {
	authContext, err := client.WithAuth(ctx, auth)
	if err != nil {
		return nil, err
	}
	response, err := client.productUsageClient.CreateProductUsage(authContext, request)
	if err != nil {
		client.logger.Errorf("Unable to create product usage %+v", err)
	}
	return response, err
}
