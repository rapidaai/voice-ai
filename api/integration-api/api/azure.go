package integration_api

import (
	"context"

	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_azure_callers "github.com/rapidaai/api/integration-api/internal/caller/azure"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type azureIntegrationApi struct {
	integrationApi
}

type azureIntegrationRPCApi struct {
	azureIntegrationApi
}

type azureIntegrationGRPCApi struct {
	azureIntegrationApi
}

func (az *azureIntegrationGRPCApi) StreamChat(irRequest *integration_api.ChatRequest, stream integration_api.AzureService_StreamChatServer) error {
	return az.integrationApi.StreamChat(
		irRequest,
		stream.Context(),
		"AZURE",
		internal_azure_callers.NewLargeLanguageCaller(az.logger, irRequest.GetCredential()),
		func(cr *integration_api.ChatResponse) error {
			return stream.Send(cr)
		},
	)
}

func NewAzureRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *azureIntegrationRPCApi {
	return &azureIntegrationRPCApi{
		azureIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewAzureGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.AzureServiceServer {
	return &azureIntegrationGRPCApi{
		azureIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// Embedding implements lexatic_backend.AzureServiceServer.
func (oiGRPC *azureIntegrationGRPCApi) Embedding(c context.Context, irRequest *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {
	return oiGRPC.integrationApi.Embedding(c, irRequest, "AZURE", internal_azure_callers.NewEmbeddingCaller(oiGRPC.logger, irRequest.GetCredential()))
}

// all grpc handler
func (oiGRPC *azureIntegrationGRPCApi) Chat(c context.Context, irRequest *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	oiGRPC.logger.Debugf("request for chat azure with request %+v", irRequest)
	return oiGRPC.integrationApi.Chat(c, irRequest, "AZURE", internal_azure_callers.NewLargeLanguageCaller(oiGRPC.logger, irRequest.GetCredential()))
}

func (dgGRPC *azureIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	deepgramCaller := internal_azure_callers.NewVerifyCredentialCaller(dgGRPC.logger, irRequest.GetCredential())
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

func (*azureIntegrationGRPCApi) GetModeration(context.Context, *integration_api.GetModerationRequest) (*integration_api.GetModerationResponse, error) {
	panic("unimplemented")
}
