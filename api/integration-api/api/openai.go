package integration_api

import (
	"context"

	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_openai_callers "github.com/rapidaai/api/integration-api/internal/caller/openai"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type openaiIntegrationApi struct {
	integrationApi
}

type openaiIntegrationRPCApi struct {
	openaiIntegrationApi
}

type openaiIntegrationGRPCApi struct {
	openaiIntegrationApi
}

func NewOpenAiRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *openaiIntegrationRPCApi {
	return &openaiIntegrationRPCApi{
		openaiIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewOpenAiGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.OpenAiServiceServer {
	return &openaiIntegrationGRPCApi{
		openaiIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// Embedding implements lexatic_backend.OpenAiServiceServer.
func (oiGRPC *openaiIntegrationGRPCApi) Embedding(c context.Context, irRequest *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {
	return oiGRPC.integrationApi.Embedding(c, irRequest, "OPENAI", internal_openai_callers.NewEmbeddingCaller(oiGRPC.logger, irRequest.GetCredential()))
}

// all grpc handler
func (oiGRPC *openaiIntegrationGRPCApi) Chat(c context.Context, irRequest *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	return oiGRPC.integrationApi.Chat(c, irRequest, "OPENAI", internal_openai_callers.NewLargeLanguageCaller(oiGRPC.logger, irRequest.GetCredential()))
}

/*

Generate APi for openai
only supported for text prompt

*/

// StreamChat implements lexatic_backend.GoogleServiceServer.
func (oiGRPC *openaiIntegrationGRPCApi) StreamChat(irRequest *integration_api.ChatRequest, stream integration_api.OpenAiService_StreamChatServer) error {
	return oiGRPC.integrationApi.StreamChat(
		irRequest,

		stream.Context(),
		"OPENAI",
		internal_openai_callers.NewLargeLanguageCaller(oiGRPC.logger, irRequest.GetCredential()),
		stream.Send,
	)
}

func (dgGRPC *openaiIntegrationApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	deepgramCaller := internal_openai_callers.NewVerifyCredentialCaller(dgGRPC.logger, irRequest.GetCredential())
	st, err := deepgramCaller.CredentialVerifier(
		c,
		&internal_callers.CredentialVerifierOptions{},
	)
	if err != nil {
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

// GetModeration implements lexatic_backend.OpenAiServiceServer.
func (*openaiIntegrationGRPCApi) GetModeration(context.Context, *integration_api.GetModerationRequest) (*integration_api.GetModerationResponse, error) {
	panic("unimplemented")
}
