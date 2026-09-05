// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_client

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

const productUsageRequestTimeout = 5 * time.Second

type ProductUsageClient interface {
	CreateProductUsage(context.Context, *types.Authentication, *protos.CreateProductUsageRequest) (*protos.GetProductUsageResponse, error)
	Close() error
}

type productUsageServiceClient struct {
	clients.InternalClient
	cfg                *config.AppConfig
	logger             commons.Logger
	connection         io.Closer
	requestTimeout     time.Duration
	productUsageClient protos.ProductUsageServiceClient
}

func NewProductUsageServiceClientGRPC(config *config.AppConfig, logger commons.Logger) ProductUsageClient {
	connection, err := grpc.NewClient(config.Web.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	client := newProductUsageClientWithClient(config, logger, protos.NewProductUsageServiceClient(connection))
	client.connection = connection
	return client
}

// NewProductUsageClientWithClient creates a product usage client using the provided gRPC client.
func NewProductUsageClientWithClient(config *config.AppConfig, logger commons.Logger, productUsageClient protos.ProductUsageServiceClient) ProductUsageClient {
	return newProductUsageClientWithClient(config, logger, productUsageClient)
}

func newProductUsageClientWithClient(config *config.AppConfig, logger commons.Logger, productUsageClient protos.ProductUsageServiceClient) *productUsageServiceClient {
	return &productUsageServiceClient{
		InternalClient:     clients.NewInternalClient(config, logger, nil),
		cfg:                config,
		logger:             logger,
		requestTimeout:     productUsageRequestTimeout,
		productUsageClient: productUsageClient,
	}
}

func (client *productUsageServiceClient) CreateProductUsage(ctx context.Context, auth *types.Authentication, request *protos.CreateProductUsageRequest) (*protos.GetProductUsageResponse, error) {
	requestTimeout := client.requestTimeout
	if requestTimeout <= 0 {
		requestTimeout = productUsageRequestTimeout
	}
	requestContext, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	authContext, err := client.WithAuth(requestContext, auth)
	if err != nil {
		return nil, err
	}
	response, err := client.productUsageClient.CreateProductUsage(authContext, request)
	if err != nil {
		client.logger.Errorf("Unable to create product usage %+v", err)
	}
	return response, err
}

func (client *productUsageServiceClient) Close() error {
	if client.connection == nil {
		return nil
	}
	return client.connection.Close()
}
