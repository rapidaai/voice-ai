// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_embedding

import (
	"context"

	"github.com/rapidaai/api/assistant-api/config"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type defaultQueryEmbedding struct {
	logger            commons.Logger
	integrationCaller integration_client.IntegrationServiceClient
	inputBuilder      integration_client_builders.InputEmbeddingBuilder
}

func NewQueryEmbedding(logger commons.Logger, cfg *config.AssistantConfig, redis connectors.RedisConnector) QueryEmbedding {
	return &defaultQueryEmbedding{
		logger:            logger,
		integrationCaller: integration_client.NewIntegrationServiceClientGRPC(&cfg.AppConfig, logger, redis),
		inputBuilder:      integration_client_builders.NewEmbeddingInputBuilder(logger),
	}
}

func (qe *defaultQueryEmbedding) TextQueryEmbedding(
	ctx context.Context,
	auth types.SimplePrinciple,
	query string, opts *TextEmbeddingOption,
) (*protos.EmbeddingResponse, error) {

	res, err := qe.integrationCaller.Embedding(ctx,
		auth,
		opts.ModelProviderName,
		qe.inputBuilder.Embedding(
			qe.
				inputBuilder.
				Credential(opts.ProviderCredential.GetId(), opts.ProviderCredential.GetValue()),
			qe.
				inputBuilder.
				Options(opts.Options, nil),
			opts.AdditionalData,
			map[int32]string{0: query},
		))
	if err != nil {
		qe.logger.Errorf("Error while building embedding request for text query %v", err)
		return nil, err
	}

	return res, nil
}
