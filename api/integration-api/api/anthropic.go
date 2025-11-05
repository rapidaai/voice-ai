package integration_api

import (
	"context"

	"github.com/gin-gonic/gin"
	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_anthropic_callers "github.com/rapidaai/api/integration-api/internal/caller/anthropic"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	protos "github.com/rapidaai/protos"
)

type anthropicIntegrationApi struct {
	integrationApi
}

type anthropicIntegrationRPCApi struct {
	anthropicIntegrationApi
}

type anthropicIntegrationGRPCApi struct {
	anthropicIntegrationApi
}

// StreamChat implements lexatic_backend.AnthropicServiceServer.
func (anthropic *anthropicIntegrationGRPCApi) StreamChat(irRequest *protos.ChatRequest, stream protos.AnthropicService_StreamChatServer) error {
	// StreamChat implements lexatic_backend.CohereServiceServer.
	anthropic.logger.Debugf("request for streaming chat anthropic with request %+v", irRequest)
	return anthropic.integrationApi.StreamChat(
		irRequest,
		stream.Context(),
		"ANTHROPIC",
		internal_anthropic_callers.NewLargeLanguageCaller(anthropic.logger, irRequest.GetCredential()),
		stream.Send,
	)
}

func NewAnthropicRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *anthropicIntegrationRPCApi {
	return &anthropicIntegrationRPCApi{
		anthropicIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewAnthropicGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) protos.AnthropicServiceServer {
	return &anthropicIntegrationGRPCApi{
		anthropicIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// all the rpc handler
func (anthropicRPC *anthropicIntegrationRPCApi) Generate(c *gin.Context) {
	anthropicRPC.logger.Debugf("Generate from rpc with gin context %v", c)
}

func (anthropicRPC *anthropicIntegrationRPCApi) Chat(c *gin.Context) {
	anthropicRPC.logger.Debugf("Chat from rpc with gin context %v", c)
}

// all grpc handler
func (anthropicRPC *anthropicIntegrationGRPCApi) Chat(c context.Context, irRequest *protos.ChatRequest) (*protos.ChatResponse, error) {
	return anthropicRPC.integrationApi.Chat(c, irRequest, "ANTHROPIC", internal_anthropic_callers.NewLargeLanguageCaller(anthropicRPC.logger, irRequest.GetCredential()))
}

func (anthropicGRPC *anthropicIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *protos.VerifyCredentialRequest) (*protos.VerifyCredentialResponse, error) {
	antCaller := internal_anthropic_callers.NewVerifyCredentialCaller(anthropicGRPC.logger, irRequest.GetCredential())
	st, err := antCaller.CredentialVerifier(
		c,
		&internal_callers.CredentialVerifierOptions{},
	)
	if err != nil {
		anthropicGRPC.logger.Errorf("verify credential response with error %v", err)
		return &protos.VerifyCredentialResponse{
			Code:         401,
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}
	return &protos.VerifyCredentialResponse{
		Code:     200,
		Success:  true,
		Response: st,
	}, nil
}
