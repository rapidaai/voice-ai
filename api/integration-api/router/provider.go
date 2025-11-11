package integration_routers

import (
	integrationApi "github.com/rapidaai/api/integration-api/api"
	"github.com/rapidaai/api/integration-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

func ProviderApiRoute(
	Cfg *config.IntegrationConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector) {
	integration_api.RegisterCohereServiceServer(S, integrationApi.NewCohereGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterOpenAiServiceServer(S, integrationApi.NewOpenAiGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterGoogleServiceServer(S, integrationApi.NewGoogleGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterAzureServiceServer(S, integrationApi.NewAzureGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterAnthropicServiceServer(S, integrationApi.NewAnthropicGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterVoyageAiServiceServer(S, integrationApi.NewVoyageAiGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterHuggingfaceServiceServer(S, integrationApi.NewHuggingfaceGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterMistralServiceServer(S, integrationApi.NewMistralGRPC(Cfg, Logger, Postgres))
	integration_api.RegisterReplicateServiceServer(S, integrationApi.NewReplicateGRPC(Cfg, Logger, Postgres))
}

func SendgridApiRoute(Cfg *config.IntegrationConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector) {
	integration_api.RegisterSendgridServiceServer(S, integrationApi.NewSendgridGRPC(Cfg, Logger, Postgres))
}

func AuditLoggingApiRoute(
	Cfg *config.IntegrationConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
) {
	integration_api.RegisterAuditLoggingServiceServer(S, integrationApi.NewAuditLoggingGRPC(Cfg, Logger, Postgres))
}
