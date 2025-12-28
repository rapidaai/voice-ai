// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package integration_api

import (
	"context"

	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_cohere_callers "github.com/rapidaai/api/integration-api/internal/caller/cohere"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type cohereIntegrationApi struct {
	integrationApi
}

type cohereIntegrationRPCApi struct {
	cohereIntegrationApi
}

type cohereIntegrationGRPCApi struct {
	cohereIntegrationApi
}

// StreamChat implements protos.CohereServiceServer.
func (cohere *cohereIntegrationGRPCApi) StreamChat(irRequest *integration_api.ChatRequest, stream integration_api.CohereService_StreamChatServer) error {
	cohere.logger.Debugf("request for streaming chat cohere with request %+v", irRequest)
	return cohere.integrationApi.StreamChat(
		irRequest,

		stream.Context(),
		"COHERE",
		internal_cohere_callers.NewLargeLanguageCaller(cohere.logger, irRequest.GetCredential()),
		stream.Send,
	)
}

func NewCohereRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *cohereIntegrationRPCApi {
	return &cohereIntegrationRPCApi{
		cohereIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewCohereGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.CohereServiceServer {
	return &cohereIntegrationGRPCApi{
		cohereIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// Embedding implements protos.CohereServiceServer.
func (cohere *cohereIntegrationGRPCApi) Embedding(c context.Context, irRequest *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {
	return cohere.integrationApi.Embedding(
		c, irRequest,
		"COHERE",
		internal_cohere_callers.NewEmbeddingCaller(cohere.logger, irRequest.GetCredential()),
	)

}

// all grpc handler
func (cohere *cohereIntegrationGRPCApi) Chat(ctx context.Context, irRequest *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	return cohere.integrationApi.Chat(
		ctx,
		irRequest,
		"COHERE",
		internal_cohere_callers.NewLargeLanguageCaller(cohere.logger, irRequest.GetCredential()))

}

func (cohereGRPC *cohereIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	antCaller := internal_cohere_callers.NewVerifyCredentialCaller(cohereGRPC.logger, irRequest.Credential)
	st, err := antCaller.CredentialVerifier(
		c,
		&internal_callers.CredentialVerifierOptions{},
	)
	if err != nil {
		cohereGRPC.logger.Errorf("verify credential response with error %v", err)
		return &integration_api.VerifyCredentialResponse{
			Code:         401,
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}
	return &integration_api.VerifyCredentialResponse{
		Code:     200,
		Success:  true,
		Response: st,
	}, nil
}

// Reranking implements protos.CohereServiceServer.
func (cohere *cohereIntegrationGRPCApi) Reranking(c context.Context, irRequest *integration_api.RerankingRequest) (*integration_api.RerankingResponse, error) {
	return cohere.integrationApi.Reranking(
		c,
		irRequest,
		"COHERE",
		internal_cohere_callers.NewRerankingCaller(cohere.logger, irRequest.GetCredential()))
}
