package integration_api

import (
	"context"

	"github.com/gin-gonic/gin"
	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_google_callers "github.com/rapidaai/api/integration-api/internal/caller/google"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type googleIntegrationApi struct {
	integrationApi
}

type googleIntegrationRPCApi struct {
	googleIntegrationApi
}

type googleIntegrationGRPCApi struct {
	googleIntegrationApi
}

// Embedding implements lexatic_backend.GoogleServiceServer.
func (googAi *googleIntegrationGRPCApi) Embedding(c context.Context, irRequest *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {
	return googAi.integrationApi.Embedding(
		c, irRequest,

		"GOOGLE",
		internal_google_callers.NewEmbeddingCaller(googAi.logger, irRequest.GetCredential()),
	)
}

func NewGoogleRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *googleIntegrationRPCApi {
	return &googleIntegrationRPCApi{
		googleIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewGoogleGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.GoogleServiceServer {
	return &googleIntegrationGRPCApi{
		googleIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// all the rpc handler
func (googleRPC *googleIntegrationRPCApi) Generate(c *gin.Context) {
	googleRPC.logger.Debugf("Generate from rpc with gin context %v", c)
}
func (googleRPC *googleIntegrationRPCApi) Chat(c *gin.Context) {
	googleRPC.logger.Debugf("Chat from rpc with gin context %v", c)
}

// StreamChat implements lexatic_backend.GoogleServiceServer.
func (googleGRPc *googleIntegrationGRPCApi) StreamChat(irRequest *integration_api.ChatRequest, stream integration_api.GoogleService_StreamChatServer) error {
	googleGRPc.logger.Debugf("request for streaming chat google with request %+v", irRequest)
	return googleGRPc.integrationApi.StreamChat(
		irRequest,

		stream.Context(),
		"GOOGLE",
		internal_google_callers.NewLargeLanguageCaller(googleGRPc.logger, irRequest.GetCredential()),
		stream.Send,
	)
}

// all grpc handler
func (googAi *googleIntegrationGRPCApi) Chat(c context.Context, irRequest *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	return googAi.integrationApi.Chat(
		c, irRequest,
		"GOOGLE",
		internal_google_callers.NewLargeLanguageCaller(googAi.logger, irRequest.GetCredential()),
	)
}

func (googleGRPC *googleIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	googleCaller := internal_google_callers.NewVerifyCredentialCaller(googleGRPC.logger, irRequest.Credential)
	st, err := googleCaller.CredentialVerifier(
		c,
		&internal_callers.CredentialVerifierOptions{},
	)
	if err != nil {
		googleGRPC.logger.Errorf("verify credential response with error %v", err)
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
