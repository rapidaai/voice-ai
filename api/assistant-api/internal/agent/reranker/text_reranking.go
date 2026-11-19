// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_rerankers

import (
	"context"

	"github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/pkg/authentication"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/utils"
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
	auth authentication.Principal,
	config *RerankingOption,
	in []string, query string, additionalData map[string]string) (map[int32]string, error) {

	contents := make(map[int32]string)
	for idx, s := range in {
		contentIndex, err := utils.IntToInt32(idx)
		if err != nil {
			return nil, err
		}
		contents[contentIndex] = s
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
		output[rk.GetIndex()] = rk.GetContent()
	}
	return output, nil
}
