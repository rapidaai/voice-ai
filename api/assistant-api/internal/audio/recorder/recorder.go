// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_audio_recorder

import (
	"context"
	"fmt"
	"time"

	internal_recorder "github.com/rapidaai/api/assistant-api/internal/audio/recorder/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

type options struct {
	ctx       context.Context
	contextID string
	onPacket  func(context.Context, ...internal_type.Packet) error
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithContextID(contextID string) Option {
	return func(options *options) {
		options.contextID = contextID
	}
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func New(opts ...Option) (internal_type.ConversationRecordingExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.onPacket == nil {
		return nil, fmt.Errorf("conversation_recording: onPacket is required")
	}
	start := time.Now()
	executor, err := internal_recorder.New(
		internal_recorder.WithContextID(options.contextID),
		internal_recorder.WithOnPacket(options.onPacket),
	)
	if err != nil {
		return nil, err
	}

	if options.onPacket != nil {
		_ = options.onPacket(options.ctx,
			internal_type.ObservabilityEventRecordPacket{
				ContextID: options.contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewConversationEventRecord(observability.RecordingStarted, nil),
			},
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: options.contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Attributes: observability.Attributes{"provider": executor.Name()},
					Metrics: []*protos.Metric{{
						Name:        observability.MetricRecordingInitLatencyMs,
						Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
						Description: "Recording initialization latency in milliseconds",
					}},
				},
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: options.contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: fmt.Sprintf("%s: initialization completed", executor.Name()),
					Attributes: observability.Attributes{
						"component":  observability.ComponentRecording.String(),
						"operation":  "initialize_executor",
						"context_id": options.contextID,
						"provider":   executor.Name(),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return executor, nil
}
