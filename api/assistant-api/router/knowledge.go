// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_router

import (
	knowledgeApi "github.com/rapidaai/api/assistant-api/api/knowledge"
	"github.com/rapidaai/api/assistant-api/config"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
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
	rapidaClient *rapida_client.RapidaClient,
) {
	workflow_api.RegisterKnowledgeServiceServer(S,
		knowledgeApi.NewKnowledgeGRPCApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
			rapidaClient,
		))
}

func DocumentApiRoute(
	Cfg *config.AssistantConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Opensearch connectors.OpenSearchConnector,
	rapidaClient *rapida_client.RapidaClient,

) {
	workflow_api.RegisterDocumentServiceServer(S,
		knowledgeApi.NewDocumentGRPCApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
			rapidaClient,
		))
}
