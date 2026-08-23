// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_artifact

import (
	"context"
	"fmt"
	"time"

	internal_artifact_storage "github.com/rapidaai/api/assistant-api/internal/artifact/storage"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

const (
	providerAWS   = "aws"
	providerAzure = "azure"
)

type options struct {
	ctx           context.Context
	logger        commons.Logger
	configuration *internal_assistant_entity.AssistantConfiguration
	caller        internal_type.InternalCaller
	auth          *types.Authentication
	onPacket      func(context.Context, ...internal_type.Packet) error
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithConfiguration(configuration *internal_assistant_entity.AssistantConfiguration) Option {
	return func(options *options) {
		options.configuration = configuration
	}
}

func WithCaller(caller internal_type.InternalCaller) Option {
	return func(options *options) {
		options.caller = caller
	}
}

func WithAuth(auth *types.Authentication) Option {
	return func(options *options) {
		options.auth = auth
	}
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func New(opts ...Option) (internal_type.ArtifactPushExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	start := time.Now()
	if options.configuration == nil {
		return nil, fmt.Errorf("artifact push storage: configuration is required")
	}
	switch options.configuration.Provider {
	case providerAWS:
		return internal_artifact_storage.NewAWS(
			internal_artifact_storage.WithAWSContext(options.ctx),
			internal_artifact_storage.WithAWSLogger(options.logger),
			internal_artifact_storage.WithAWSConfiguration(options.configuration),
			internal_artifact_storage.WithAWSCaller(options.caller),
			internal_artifact_storage.WithAWSAuth(options.auth),
			internal_artifact_storage.WithAWSOnPacket(options.onPacket),
		)
	case providerAzure:
		return internal_artifact_storage.NewAzureStorage(
			internal_artifact_storage.WithAzureStorageContext(options.ctx),
			internal_artifact_storage.WithAzureStorageLogger(options.logger),
			internal_artifact_storage.WithAzureStorageConfiguration(options.configuration),
			internal_artifact_storage.WithAzureStorageCaller(options.caller),
			internal_artifact_storage.WithAzureStorageAuth(options.auth),
			internal_artifact_storage.WithAzureStorageOnPacket(options.onPacket),
		)
	default:
		err := fmt.Errorf("artifact push storage: unsupported provider %q", options.configuration.Provider)
		if options.onPacket != nil {
			_ = options.onPacket(options.ctx,
				internal_type.ObservabilityMetricRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordMetric{
						Attributes: observability.Attributes{
							"provider":         options.configuration.Provider,
							"configuration_id": fmt.Sprintf("%d", options.configuration.Id),
							"status":           "failed",
						},
						Metrics: []*protos.Metric{{
							Name:        observability.MetricStorageInitLatencyMs,
							Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
							Description: "Storage initialization latency in milliseconds",
						}},
					},
				},
				internal_type.ObservabilityLogRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "External artifact storage executor initialization failed",
						Attributes: observability.Attributes{
							"component":        observability.ComponentStorage.String(),
							"operation":        "initialize_executor",
							"provider":         options.configuration.Provider,
							"configuration_id": fmt.Sprintf("%d", options.configuration.Id),
							"error":            err.Error(),
							"error_type":       fmt.Sprintf("%T", err),
						},
						OccurredAt: time.Now(),
					},
				},
			)
		}
		return nil, err
	}
}
