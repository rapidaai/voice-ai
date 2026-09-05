// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package integration_client

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type IntegrationServiceClient interface {
	Chat(c context.Context,
		auth *types.Authentication,
		providerName string,
		request *protos.ChatRequest) (*protos.ChatResponse, error)
	StreamChat(c context.Context, auth *types.Authentication, providerName string) (grpc.BidiStreamingClient[protos.StreamChatRequest, protos.StreamChatResponse], error)
	Embedding(ctx context.Context, auth *types.Authentication, providerName string, in *protos.EmbeddingRequest) (*protos.EmbeddingResponse, error)
	Reranking(ctx context.Context, auth *types.Authentication, providerName string, in *protos.RerankingRequest) (*protos.RerankingResponse, error)
	VerifyCredential(ctx context.Context, auth *types.Authentication, providerName string, in *protos.Credential) (*protos.VerifyCredentialResponse, error)
}

type integrationServiceClient struct {
	clients.InternalClient
	cfg           *config.AppConfig
	logger        commons.Logger
	unifiedClient protos.UnifiedProviderServiceClient
}

func NewIntegrationServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) IntegrationServiceClient {
	lightConnection, err := grpc.NewClient(config.Integration.Host, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}...)
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return NewIntegrationServiceClientWithClient(config, logger, redis, protos.NewUnifiedProviderServiceClient(lightConnection))
}

// NewIntegrationServiceClientWithClient creates an integration client using the provided gRPC client.
func NewIntegrationServiceClientWithClient(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector, unifiedClient protos.UnifiedProviderServiceClient) IntegrationServiceClient {
	return &integrationServiceClient{
		InternalClient: clients.NewInternalClient(config, logger, redis),
		cfg:            config,
		logger:         logger,
		unifiedClient:  unifiedClient,
	}
}

func (client *integrationServiceClient) Embedding(c context.Context,
	auth *types.Authentication,
	providerName string,
	request *protos.EmbeddingRequest) (*protos.EmbeddingResponse, error) {
	request.ProviderName = strings.ToLower(providerName)
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	return client.unifiedClient.Embedding(authContext, request)
}

func (client *integrationServiceClient) Reranking(c context.Context,
	auth *types.Authentication,
	providerName string,
	request *protos.RerankingRequest) (*protos.RerankingResponse, error) {
	request.ProviderName = strings.ToLower(providerName)
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	return client.unifiedClient.Reranking(authContext, request)
}

func (client *integrationServiceClient) Chat(c context.Context,
	auth *types.Authentication,
	providerName string,
	request *protos.ChatRequest) (*protos.ChatResponse, error) {
	request.ProviderName = strings.ToLower(providerName)
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	return client.unifiedClient.Chat(authContext, request)
}

// StreamChat opens a bidirectional stream via the unified provider service.
// The caller must set ProviderName on each ChatRequest before sending.
func (client *integrationServiceClient) StreamChat(c context.Context, auth *types.Authentication, providerName string) (grpc.BidiStreamingClient[protos.StreamChatRequest, protos.StreamChatResponse], error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	return client.unifiedClient.StreamChat(authContext)
}

func (client *integrationServiceClient) VerifyCredential(c context.Context,
	auth *types.Authentication,
	providerName string,
	cr *protos.Credential) (*protos.VerifyCredentialResponse, error) {
	request := &protos.VerifyCredentialRequest{
		Credential:   cr,
		ProviderName: strings.ToLower(providerName),
	}
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	return client.unifiedClient.VerifyCredential(authContext, request)
}
