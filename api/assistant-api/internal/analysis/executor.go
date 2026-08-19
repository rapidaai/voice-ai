// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_analysis

import (
	"context"
	"fmt"
	"time"

	internal_analysis_endpoint "github.com/rapidaai/api/assistant-api/internal/analysis/endpoint"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type options struct {
	logger   commons.Logger
	ctx      context.Context
	analysis *internal_assistant_entity.AssistantConfiguration
	caller   internal_type.InternalCaller
	onPacket func(context.Context, ...internal_type.Packet) error
}

type Option func(*options)

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithConfiguration(analysis *internal_assistant_entity.AssistantConfiguration) Option {
	return func(options *options) {
		options.analysis = analysis
	}
}

func WithCaller(caller internal_type.InternalCaller) Option {
	return func(options *options) {
		options.caller = caller
	}
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

// New is the factory that returns an analysis executor implementation.
// Currently only the deployment-endpoint variant is supported; switch on the
// analysis artifact type when other transports are added.
func New(opts ...Option) (internal_type.AnalysisExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.analysis == nil {
		return nil, fmt.Errorf("analysis: configuration is required")
	}
	start := time.Now()
	switch options.analysis.Provider {
	case "endpoint":
		return internal_analysis_endpoint.New(
			internal_analysis_endpoint.WithLogger(options.logger),
			internal_analysis_endpoint.WithContext(options.ctx),
			internal_analysis_endpoint.WithConfiguration(options.analysis),
			internal_analysis_endpoint.WithCaller(options.caller),
			internal_analysis_endpoint.WithOnPacket(options.onPacket),
		)
	default:
		err := fmt.Errorf("analysis: unsupported executor type %q", options.analysis.Provider)
		if options.onPacket != nil {
			_ = options.onPacket(options.ctx,
				internal_type.ObservabilityMetricRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordMetric{
						Attributes: observability.Attributes{
							"provider":         options.analysis.Provider,
							"configuration_id": fmt.Sprintf("%d", options.analysis.Id),
							"status":           "failed",
						},
						Metrics: []*protos.Metric{{
							Name:        observability.MetricAnalysisInitLatencyMs,
							Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
							Description: "Analysis initialization latency in milliseconds",
						}},
					},
				},
				internal_type.ObservabilityLogRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "analysis: initialization failed",
						Attributes: observability.Attributes{
							"component":        observability.ComponentAnalysis.String(),
							"operation":        "initialize_executor",
							"provider":         options.analysis.Provider,
							"configuration_id": fmt.Sprintf("%d", options.analysis.Id),
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
