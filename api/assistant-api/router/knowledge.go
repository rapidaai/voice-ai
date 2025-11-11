package assistant_router

import (
	knowledgeApi "github.com/rapidaai/api/assistant-api/api/knowledge"
	"github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	workflow_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

func KnowledgeApiRoute(
	Cfg *config.AssistantConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Opensearch connectors.OpenSearchConnector,
) {
	workflow_api.RegisterKnowledgeServiceServer(S,
		knowledgeApi.NewKnowledgeGRPCApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
		))
}

func DocumentApiRoute(
	Cfg *config.AssistantConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Opensearch connectors.OpenSearchConnector,

) {
	workflow_api.RegisterDocumentServiceServer(S,
		knowledgeApi.NewDocumentGRPCApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
		))
}
