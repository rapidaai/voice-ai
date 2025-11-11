package integration_api

import (
	"context"

	"github.com/gin-gonic/gin"
	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_huggingface_callers "github.com/rapidaai/api/integration-api/internal/caller/huggingface"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type huggingfaceIntegrationApi struct {
	integrationApi
}

type huggingfaceIntegrationRPCApi struct {
	huggingfaceIntegrationApi
}

type huggingfaceIntegrationGRPCApi struct {
	huggingfaceIntegrationApi
}

// Embedding implements lexatic_backend.huggingfaceServiceServer.
func (huggingf *huggingfaceIntegrationGRPCApi) Embedding(c context.Context, irRequest *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {
	return huggingf.integrationApi.Embedding(
		c, irRequest,

		"HUGGINGFACE",
		internal_huggingface_callers.NewEmbeddingCaller(huggingf.logger, irRequest.GetCredential()),
	)
}

func NewHuggingfaceRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *huggingfaceIntegrationRPCApi {
	return &huggingfaceIntegrationRPCApi{
		huggingfaceIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewHuggingfaceGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.HuggingfaceServiceServer {
	return &huggingfaceIntegrationGRPCApi{
		huggingfaceIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// all the rpc handler
func (huggingfaceRPC *huggingfaceIntegrationRPCApi) Generate(c *gin.Context) {
	huggingfaceRPC.logger.Debugf("Generate from rpc with gin context %v", c)
}
func (huggingfaceRPC *huggingfaceIntegrationRPCApi) Chat(c *gin.Context) {
	huggingfaceRPC.logger.Debugf("Chat from rpc with gin context %v", c)
}

// all grpc handler
func (huggingf *huggingfaceIntegrationGRPCApi) Chat(c context.Context, irRequest *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	return huggingf.integrationApi.Chat(
		c, irRequest,

		"HUGGINGFACE",
		internal_huggingface_callers.NewLargeLanguageCaller(huggingf.logger, irRequest.GetCredential()),
	)

}

func (huggingfaceGRPC *huggingfaceIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	antCaller := internal_huggingface_callers.NewVerifyCredentialCaller(huggingfaceGRPC.logger, irRequest.Credential)
	st, err := antCaller.CredentialVerifier(
		c,
		&internal_callers.CredentialVerifierOptions{},
	)
	if err != nil {
		huggingfaceGRPC.logger.Errorf("verify credential response with error %v", err)
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
