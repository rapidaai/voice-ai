// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
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
	CreateProductUsages(context.Context, *types.Authentication, []*protos.ProductUsage) (*protos.CreateProductUsagesResponse, error)
	Close() error
}

type productUsageServiceClient struct {
	clients.InternalClient
	client     protos.ProductUsageServiceClient
	connection io.Closer
	timeout    time.Duration
}

func NewProductUsageServiceClientGRPC(cfg *config.AppConfig, logger commons.Logger) (ProductUsageClient, error) {
	if cfg == nil || strings.TrimSpace(cfg.Web.Host) == "" {
		return nil, errors.New("product usage client requires web service host")
	}
	connection, err := grpc.NewClient(cfg.Web.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create product usage connection: %w", err)
	}
	return &productUsageServiceClient{
		InternalClient: clients.NewInternalClient(cfg, logger, nil),
		client:         protos.NewProductUsageServiceClient(connection),
		connection:     connection,
		timeout:        productUsageRequestTimeout,
	}, nil
}

func (client *productUsageServiceClient) CreateProductUsages(ctx context.Context, auth *types.Authentication, usages []*protos.ProductUsage) (*protos.CreateProductUsagesResponse, error) {
	requestContext, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	authContext, err := client.WithAuth(requestContext, auth)
	if err != nil {
		return nil, fmt.Errorf("authorize product usage request: %w", err)
	}
	response, err := client.client.CreateProductUsages(authContext, &protos.CreateProductUsagesRequest{Usages: usages})
	if err != nil {
		return nil, fmt.Errorf("create product usages: %w", err)
	}
	if response == nil {
		return nil, errors.New("create product usages: empty response")
	}
	if !response.GetSuccess() {
		message := "request failed"
		if response.GetError() != nil && strings.TrimSpace(response.GetError().GetHumanMessage()) != "" {
			message = response.GetError().GetHumanMessage()
		}
		return nil, fmt.Errorf("create product usages: %s", message)
	}
	return response, nil
}

func (client *productUsageServiceClient) Close() error {
	if client == nil || client.connection == nil {
		return nil
	}
	return client.connection.Close()
}
