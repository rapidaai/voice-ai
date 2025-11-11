package integration_api

import (
	"context"
	"errors"

	config "github.com/rapidaai/api/integration-api/config"
	callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	internal_voyageai_callers "github.com/rapidaai/api/integration-api/internal/caller/voyageai"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type voyageaiIntegrationApi struct {
	integrationApi
	caller callers.Caller
}

type voyageaiIntegrationRPCApi struct {
	voyageaiIntegrationApi
}

type voyageaiIntegrationGRPCApi struct {
	voyageaiIntegrationApi
}

func NewVoyageAiRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) *voyageaiIntegrationRPCApi {
	return &voyageaiIntegrationRPCApi{
		voyageaiIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func NewVoyageAiGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.VoyageAiServiceServer {
	return &voyageaiIntegrationGRPCApi{
		voyageaiIntegrationApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

// Embedding implements lexatic_backend.VoyageAiServiceServer.
func (oiGRPC *voyageaiIntegrationGRPCApi) Embedding(c context.Context, irRequest *integration_api.EmbeddingRequest) (*integration_api.EmbeddingResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		oiGRPC.logger.Errorf("unauthenticated request for embedding")
		return utils.Error[integration_api.EmbeddingResponse](
			errors.New("unauthenticated request for chat"),
			"Please provider valid service credentials to perfom embedding, read docs @ docs.rapida.ai",
		)
	}
	return oiGRPC.integrationApi.Embedding(
		c, irRequest,
		"VOYAGEAI",
		internal_voyageai_callers.NewEmbeddingCaller(oiGRPC.logger, irRequest.GetCredential()),
	)
}

// Reranking implements lexatic_backend.VoyageAiServiceServer.
func (oiGRPC *voyageaiIntegrationGRPCApi) Reranking(c context.Context, irRequest *integration_api.RerankingRequest) (*integration_api.RerankingResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		oiGRPC.logger.Errorf("unauthenticated request for embedding")
		return utils.Error[integration_api.RerankingResponse](
			errors.New("unauthenticated request for reranking"),
			"Please provider valid service credentials to perfom embedding, read docs @ docs.rapida.ai",
		)
	}
	return oiGRPC.integrationApi.Reranking(
		c, irRequest,
		"VOYAGEAI",
		internal_voyageai_callers.NewRerankingCaller(oiGRPC.logger, irRequest.GetCredential()),
	)
}

func (dgGRPC *voyageaiIntegrationGRPCApi) VerifyCredential(c context.Context, irRequest *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
	deepgramCaller := internal_voyageai_callers.NewVerifyCredentialCaller(dgGRPC.logger, irRequest.Credential)
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
