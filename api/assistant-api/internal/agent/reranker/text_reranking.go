package internal_agent_rerankers

import (
	"context"

	"github.com/rapidaai/api/assistant-api/config"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type textReranker struct {
	logger            commons.Logger
	integrationCaller integration_client.IntegrationServiceClient
	inputBuilder      integration_client_builders.InputRerankingBuilder
}

func NewTextReranker(logger commons.Logger, cfg *config.AssistantConfig, redis connectors.RedisConnector) Reranking[string] {
	return &textReranker{
		logger:            logger,
		integrationCaller: integration_client.NewIntegrationServiceClientGRPC(&cfg.AppConfig, logger, redis),
		inputBuilder:      integration_client_builders.NewRerankingInputBuilder(logger),
	}
}

func (qe *textReranker) Rerank(ctx context.Context,
	auth types.SimplePrinciple,
	config *RerankingOption,
	in []string, query string, additionalData map[string]string) (map[int32]string, error) {

	contents := make(map[int32]*protos.Content)
	for idx, s := range in {
		contents[int32(idx)] = &protos.Content{
			ContentType:   commons.TEXT_CONTENT.String(),
			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
			Content:       []byte(s),
		}
	}

	res, err := qe.integrationCaller.Reranking(ctx,
		auth,
		config.ModelProviderName,
		qe.inputBuilder.Reranking(
			qe.
				inputBuilder.
				Credential(config.ProviderCredential.GetId(), config.ProviderCredential.GetValue()),
			qe.
				inputBuilder.
				Options(config.Options, nil),
			additionalData,
			contents,
		))
	if err != nil {
		qe.logger.Errorf("Error while building embedding request for text query %v", err)
		return nil, err
	}

	reranked := res.GetData()
	output := make(map[int32]string, len(reranked))
	for _, rk := range reranked {
		output[rk.GetIndex()] = types.ContentString(rk.GetContent())
	}
	return output, nil
}
