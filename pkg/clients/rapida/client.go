// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package rapida_client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rapidaai/config"
	document_client "github.com/rapidaai/pkg/clients/document"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RapidaClient owns the internal service clients and their connections.
type RapidaClient struct {
	Authentication     web_client.AuthClient
	Vault              web_client.VaultClient
	Project            web_client.ProjectClient
	ProductUsageClient web_client.ProductUsageClient
	Integration        integration_client.IntegrationServiceClient
	Deployment         endpoint_client.DeploymentServiceClient
	Indexer            document_client.IndexerServiceClient

	connections []*grpc.ClientConn
	closeOnce   sync.Once
	closeErr    error
}

// New creates every internal service client with an independent connection.
func New(cfg *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) (*RapidaClient, error) {
	client := &RapidaClient{connections: make([]*grpc.ClientConn, 0, 6)}
	var err error
	defer func() {
		if err != nil {
			_ = client.Close(context.Background())
		}
	}()

	newConnection := func(name, target string, options ...grpc.DialOption) (*grpc.ClientConn, error) {
		connection, connectionErr := grpc.NewClient(target, options...)
		if connectionErr != nil {
			return nil, fmt.Errorf("connect %s: %w", name, connectionErr)
		}
		client.connections = append(client.connections, connection)
		return connection, nil
	}

	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())
	authenticationConnection, err := newConnection("web authentication client", cfg.Web.Host, transportCredentials)
	if err != nil {
		return nil, err
	}
	vaultConnection, err := newConnection("web vault client", cfg.Web.Host, transportCredentials)
	if err != nil {
		return nil, err
	}
	projectConnection, err := newConnection("web project client", cfg.Web.Host, transportCredentials)
	if err != nil {
		return nil, err
	}
	productUsageConnection, err := newConnection("web product usage client", cfg.Web.Host, transportCredentials)
	if err != nil {
		return nil, err
	}
	integrationConnection, err := newConnection("integration client", cfg.Integration.Host, transportCredentials)
	if err != nil {
		return nil, err
	}
	deploymentConnection, err := newConnection(
		"endpoint deployment client",
		cfg.Endpoint.Host,
		transportCredentials,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(commons.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(commons.MaxSendMsgSize),
		),
	)
	if err != nil {
		return nil, err
	}

	client.Authentication = web_client.NewAuthenticatorWithClient(
		cfg,
		logger,
		redis,
		protos.NewAuthenticationServiceClient(authenticationConnection),
	)
	client.Vault = web_client.NewVaultClientWithClient(
		cfg,
		logger,
		redis,
		protos.NewVaultServiceClient(vaultConnection),
	)
	client.Project = web_client.NewProjectClientWithClient(
		cfg,
		logger,
		redis,
		protos.NewProjectServiceClient(projectConnection),
	)
	client.ProductUsageClient = web_client.NewProductUsageClientWithClient(
		cfg,
		logger,
		protos.NewProductUsageServiceClient(productUsageConnection),
	)
	client.Integration = integration_client.NewIntegrationServiceClientWithClient(
		cfg,
		logger,
		redis,
		protos.NewUnifiedProviderServiceClient(integrationConnection),
	)
	client.Deployment = endpoint_client.NewDeploymentServiceClientWithClient(
		cfg,
		logger,
		redis,
		protos.NewDeploymentClient(deploymentConnection),
	)
	client.Indexer = document_client.NewIndexerServiceClient(cfg, logger, redis)
	return client, nil
}

// Close closes every owned connection once.
func (client *RapidaClient) Close(context.Context) error {
	if client == nil {
		return nil
	}
	client.closeOnce.Do(func() {
		var closeErrors []error
		for _, connection := range client.connections {
			if err := connection.Close(); err != nil {
				closeErrors = append(closeErrors, err)
			}
		}
		client.closeErr = errors.Join(closeErrors...)
	})
	return client.closeErr
}
