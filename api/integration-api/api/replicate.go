package integration_api

import (
	"context"

	"github.com/gin-gonic/gin"
	config "github.com/rapidaai/api/integration-api/config"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_replicate_callers "github.com/rapidaai/api/integration-api/internal/caller/replicate"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type replicateIntegrationApi struct {
	integrationApi
}

type replicateIntegrationRPCApi struct {
	replicateIntegrationApi
}

type replicateIntegrationGRPCApi struct {
	replicateIntegrationApi
}

// StreamChat implements lexatic_backend.ReplicateServiceServer.
func (*replicateIntegrationGRPCApi) StreamChat(*integration_api.ChatRequest, integration_api.ReplicateService_StreamChatServer) error {
	panic("unimplemented")
}

func NewReplicateRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *replicateIntegrationRPCApi {
	return &replicateIntegrationRPCApi{
		replicateIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewReplicateGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.ReplicateServiceServer {
	return &replicateIntegrationGRPCApi{
		replicateIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// all the rpc handler
func (replicateRPC *replicateIntegrationRPCApi) Generate(c *gin.Context) {
	replicateRPC.logger.Debugf("Generate from rpc with gin context %v", c)
}
func (replicateRPC *replicateIntegrationRPCApi) Chat(c *gin.Context) {
	replicateRPC.logger.Debugf("Chat from rpc with gin context %v", c)
}

// all grpc handler
func (replicateGRPC *replicateIntegrationGRPCApi) Chat(c context.Context, irRequest *integration_api.ChatRequest) (*integration_api.ChatResponse, error) {
	return replicateGRPC.integrationApi.Chat(c, irRequest, "REPLICATE", internal_replicate_callers.NewLargeLanguageCaller(replicateGRPC.logger, irRequest.GetCredential()))

}

func (replicateGRPC *replicateIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	replicateCaller := internal_replicate_callers.NewVerifyCredentialCaller(replicateGRPC.logger, irRequest.Credential)
	st, err := replicateCaller.CredentialVerifier(
		c,
		&internal_callers.CredentialVerifierOptions{},
	)
	if err != nil {
		replicateGRPC.logger.Errorf("verify credential response with error %v", err)
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
