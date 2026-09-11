// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter

import (
	"context"

	"github.com/rapidaai/api/assistant-api/config"

	adapter_internal "github.com/rapidaai/api/assistant-api/internal/adapters/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/utils"
)

type TalkerOptions struct {
	Source       utils.RapidaSource
	Context      context.Context
	Config       *config.AssistantConfig
	RapidaClient *rapida_client.RapidaClient
	Logger       commons.Logger
	Postgres     connectors.PostgresConnector
	OpenSearch   connectors.OpenSearchConnector
	Redis        connectors.RedisConnector
	Storage      storages.Storage
	Streamer     internal_type.Streamer
	Observer     observability.Recorder
}

type FuncOption func(*TalkerOptions)

func WithSource(source utils.RapidaSource) FuncOption {
	return func(options *TalkerOptions) {
		options.Source = source
	}
}

func WithContext(ctx context.Context) FuncOption {
	return func(options *TalkerOptions) {
		options.Context = ctx
	}
}

func WithConfig(config *config.AssistantConfig) FuncOption {
	return func(options *TalkerOptions) {
		options.Config = config
	}
}

func WithRapidaClient(client *rapida_client.RapidaClient) FuncOption {
	return func(options *TalkerOptions) {
		options.RapidaClient = client
	}
}

func WithLogger(logger commons.Logger) FuncOption {
	return func(options *TalkerOptions) {
		options.Logger = logger
	}
}

func WithPostgres(postgres connectors.PostgresConnector) FuncOption {
	return func(options *TalkerOptions) {
		options.Postgres = postgres
	}
}

func WithOpenSearch(opensearch connectors.OpenSearchConnector) FuncOption {
	return func(options *TalkerOptions) {
		options.OpenSearch = opensearch
	}
}

func WithRedis(redis connectors.RedisConnector) FuncOption {
	return func(options *TalkerOptions) {
		options.Redis = redis
	}
}

func WithStorage(storage storages.Storage) FuncOption {
	return func(options *TalkerOptions) {
		options.Storage = storage
	}
}

func WithStreamer(streamer internal_type.Streamer) FuncOption {
	return func(options *TalkerOptions) {
		options.Streamer = streamer
	}
}

func WithObserver(observer observability.Recorder) FuncOption {
	return func(options *TalkerOptions) {
		options.Observer = observer
	}
}

func New(opts ...FuncOption) (internal_type.Talking, error) {
	var options TalkerOptions
	for _, opt := range opts {
		opt(&options)
	}

	return adapter_internal.NewGenericRequestor(
		options.Context,
		options.Config,
		options.RapidaClient,
		options.Logger,
		options.Source,
		options.Postgres,
		options.OpenSearch,
		options.Redis,
		options.Storage,
		options.Streamer,
		options.Observer,
	), nil
}
