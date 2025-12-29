// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package integration_client

import (
	"context"
	"errors"
	"strings"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	integration_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type IntegrationServiceClient interface {
	Chat(c context.Context,
		auth types.SimplePrinciple,
		providerName string,
		request *integration_api.ChatRequest) (*integration_api.ChatResponse, error)
	StreamChat(c context.Context, auth types.SimplePrinciple,
		providerName string,
		request *integration_api.ChatRequest) (integration_api.OpenAiService_StreamChatClient, error)
	Embedding(ctx context.Context, auth types.SimplePrinciple, providerName string, in *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error)
	Reranking(ctx context.Context, auth types.SimplePrinciple, providerName string, in *integration_api.RerankingRequest) (*integration_api.RerankingResponse, error)
	VerifyCredential(ctx context.Context, auth types.SimplePrinciple, providerName string, in *integration_api.Credential) (*integration_api.VerifyCredentialResponse, error)
}

type integrationServiceClient struct {
	clients.InternalClient
	cfg               *config.AppConfig
	logger            commons.Logger
	cohereClient      integration_api.CohereServiceClient
	replicateClient   integration_api.ReplicateServiceClient
	openAiClient      integration_api.OpenAiServiceClient
	voyageAiClient    integration_api.VoyageAiServiceClient
	bedrockClient     integration_api.BedrockServiceClient
	azureAiClient     integration_api.AzureServiceClient
	anthropicClient   integration_api.AnthropicServiceClient
	googleClient      integration_api.GoogleServiceClient
	mistralClient     integration_api.MistralServiceClient
	togetherAiClient  integration_api.TogetherAiServiceClient
	deepInfraCLient   integration_api.DeepInfraServiceClient
	huggingfaceClient integration_api.HuggingfaceServiceClient
	awsbedrockClient  integration_api.BedrockServiceClient
}

func NewIntegrationServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) IntegrationServiceClient {
	logger.Debugf("conntecting to integration client with %s", config.IntegrationHost)

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(commons.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(commons.MaxSendMsgSize),
		),
	}
	conn, err := grpc.NewClient(config.IntegrationHost,
		grpcOpts...)

	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}

	return &integrationServiceClient{
		InternalClient:    clients.NewInternalClient(config, logger, redis),
		cfg:               config,
		logger:            logger,
		cohereClient:      integration_api.NewCohereServiceClient(conn),
		replicateClient:   integration_api.NewReplicateServiceClient(conn),
		openAiClient:      integration_api.NewOpenAiServiceClient(conn),
		anthropicClient:   integration_api.NewAnthropicServiceClient(conn),
		googleClient:      integration_api.NewGoogleServiceClient(conn),
		mistralClient:     integration_api.NewMistralServiceClient(conn),
		togetherAiClient:  integration_api.NewTogetherAiServiceClient(conn),
		deepInfraCLient:   integration_api.NewDeepInfraServiceClient(conn),
		voyageAiClient:    integration_api.NewVoyageAiServiceClient(conn),
		bedrockClient:     integration_api.NewBedrockServiceClient(conn),
		azureAiClient:     integration_api.NewAzureServiceClient(conn),
		huggingfaceClient: integration_api.NewHuggingfaceServiceClient(conn),
		awsbedrockClient:  integration_api.NewBedrockServiceClient(conn),
	}
}

func (client *integrationServiceClient) Embedding(c context.Context,
	auth types.SimplePrinciple,
	providerName string,
	request *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {

	switch providerName := strings.ToLower(providerName); providerName {
	case "cohere":
		return client.cohereClient.Embedding(client.WithAuth(c, auth), request)
	case "openai":
		return client.openAiClient.Embedding(client.WithAuth(c, auth), request)
	case "voyageai":
		return client.voyageAiClient.Embedding(client.WithAuth(c, auth), request)
	case "bedrock":
		return client.bedrockClient.Embedding(client.WithAuth(c, auth), request)
	case "azure":
		return client.azureAiClient.Embedding(client.WithAuth(c, auth), request)
	case "google", "gemini":
		return client.googleClient.Embedding(client.WithAuth(c, auth), request)
	// case "mistral":
	// return client.mistralClient.Embedding(client.WithAuth(c, auth), request)
	default:
		return nil, errors.New("illegal provider for chat request")
	}
}

func (client *integrationServiceClient) Reranking(c context.Context,
	auth types.SimplePrinciple,
	providerName string,
	request *integration_api.RerankingRequest) (*integration_api.RerankingResponse, error) {
	switch providerName := strings.ToLower(providerName); providerName {
	case "cohere":
		return client.cohereClient.Reranking(client.WithAuth(c, auth), request)
	default:
		return nil, errors.New("illegal provider for chat request")
	}
}

func (client *integrationServiceClient) Chat(c context.Context,
	auth types.SimplePrinciple,
	providerName string,
	request *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	switch providerName := strings.ToLower(providerName); providerName {
	case "cohere":
		return client.cohereClient.Chat(client.WithAuth(c, auth), request)
	case "anthropic":
		return client.anthropicClient.Chat(client.WithAuth(c, auth), request)
	case "replicate":
		return client.replicateClient.Chat(client.WithAuth(c, auth), request)
	case "google", "gemini":
		return client.googleClient.Chat(client.WithAuth(c, auth), request)
	case "mistral":
		return client.mistralClient.Chat(client.WithAuth(c, auth), request)
	case "togetherai":
		return client.togetherAiClient.Chat(client.WithAuth(c, auth), request)
	case "openai":
		return client.openAiClient.Chat(client.WithAuth(c, auth), request)
	case "aws-bedrock":
		return client.bedrockClient.Chat(client.WithAuth(c, auth), request)
	case "azure-openai", "azure":
		return client.azureAiClient.Chat(client.WithAuth(c, auth), request)
	default:
		return nil, errors.New("illegal provider for chat request")
	}
}

// StreamChat implements IntegrationServiceClient.
func (client *integrationServiceClient) StreamChat(c context.Context, auth types.SimplePrinciple, providerName string, request *integration_api.ChatRequest) (integration_api.OpenAiService_StreamChatClient, error) {

	switch providerName := strings.ToLower(providerName); providerName {
	case "openai":
		return client.openAiClient.StreamChat(client.WithAuth(c, auth), request)
	case "anthropic":
		return client.anthropicClient.StreamChat(client.WithAuth(c, auth), request)
	case "google", "gemini":
		return client.googleClient.StreamChat(client.WithAuth(c, auth), request)
	case "cohere":
		return client.cohereClient.StreamChat(client.WithAuth(c, auth), request)
	case "azure-openai", "azure":
		return client.azureAiClient.StreamChat(client.WithAuth(c, auth), request)
	default:
		return nil, errors.New("illegal provider for chat request")
	}
}

func (client *integrationServiceClient) VerifyCredential(c context.Context,
	auth types.SimplePrinciple,
	providerName string,
	cr *integration_api.Credential) (*integration_api.VerifyCredentialResponse, error) {

	request := &integration_api.VerifyCredentialRequest{
		Credential: cr,
	}
	switch providerName := strings.ToLower(providerName); providerName {
	case "cohere":
		return client.cohereClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "anthropic":
		return client.anthropicClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "replicate":
		return client.replicateClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "google", "gemini":
		return client.googleClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "mistral":
		return client.mistralClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "openai":
		return client.openAiClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "voyageai":
		return client.voyageAiClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "huggingface":
		return client.huggingfaceClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "aws-bedrock":
		return client.awsbedrockClient.VerifyCredential(client.WithAuth(c, auth), request)
	case "azure":
		return client.azureAiClient.VerifyCredential(client.WithAuth(c, auth), request)
	default:
		return nil, errors.New("illegal provider for chat request")
	}
}
