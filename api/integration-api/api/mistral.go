package integration_api

import (
	"context"

	"github.com/gin-gonic/gin"
	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_mistral_callers "github.com/rapidaai/api/integration-api/internal/caller/mistral"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type mistralIntegrationApi struct {
	integrationApi
}

type mistralIntegrationRPCApi struct {
	mistralIntegrationApi
}

type mistralIntegrationGRPCApi struct {
	mistralIntegrationApi
}

// StreamChat implements lexatic_backend.MistralServiceServer.
func (*mistralIntegrationGRPCApi) StreamChat(*integration_api.ChatRequest, integration_api.MistralService_StreamChatServer) error {
	panic("unimplemented")
}

// Embedding implements lexatic_backend.mistralServiceServer.
func (mistral *mistralIntegrationGRPCApi) Embedding(c context.Context, irRequest *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {
	return mistral.integrationApi.Embedding(c, irRequest, "MISTRAL", internal_mistral_callers.NewEmbeddingCaller(mistral.logger, irRequest.GetCredential()))
}

func NewMistralRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *mistralIntegrationRPCApi {
	return &mistralIntegrationRPCApi{
		mistralIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewMistralGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.MistralServiceServer {
	return &mistralIntegrationGRPCApi{
		mistralIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// all the rpc handler
func (mistralRPC *mistralIntegrationRPCApi) Generate(c *gin.Context) {
	mistralRPC.logger.Debugf("Generate from rpc with gin context %v", c)
}
func (mistralRPC *mistralIntegrationRPCApi) Chat(c *gin.Context) {
	mistralRPC.logger.Debugf("Chat from rpc with gin context %v", c)
}

// all grpc handler
func (mistral *mistralIntegrationGRPCApi) Chat(c context.Context, irRequest *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	return mistral.integrationApi.Chat(c, irRequest, "MISTRAL", internal_mistral_callers.NewLargeLanguageCaller(mistral.logger, irRequest.GetCredential()))

}

func (mistralGRPC *mistralIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	mistralCaller := internal_mistral_callers.NewVerifyCredentialCaller(mistralGRPC.logger, irRequest.Credential)
	st, err := mistralCaller.CredentialVerifier(
		c,
		&internal_callers.CredentialVerifierOptions{},
	)
	if err != nil {
		mistralGRPC.logger.Errorf("verify credential response with error %v", err)
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
