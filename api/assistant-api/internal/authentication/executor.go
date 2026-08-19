// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_authentication

import (
	"context"
	"fmt"
	"time"

	internal_authentication_http "github.com/rapidaai/api/assistant-api/internal/authentication/http"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type options struct {
	logger        commons.Logger
	ctx           context.Context
	authenticator *internal_assistant_entity.AssistantConfiguration
	callback      internal_type.Callback
	caller        internal_type.InternalCaller
	onPacket      func(context.Context, ...internal_type.Packet) error
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

func WithConfiguration(authenticator *internal_assistant_entity.AssistantConfiguration) Option {
	return func(options *options) {
		options.authenticator = authenticator
	}
}

func WithCallback(callback internal_type.Callback) Option {
	return func(options *options) {
		options.callback = callback
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

// New is the factory that returns an authentication executor implementation.
// Currently only HTTP is supported; switch on the assistant authentication type
// when other modes (e.g., JWT, static) are added.
func New(opts ...Option) (internal_type.AuthenticationExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.authenticator == nil {
		return nil, fmt.Errorf("authentication: configuration is required")
	}

	start := time.Now()
	switch options.authenticator.Provider {
	case "http":
		return internal_authentication_http.New(
			internal_authentication_http.WithLogger(options.logger),
			internal_authentication_http.WithContext(options.ctx),
			internal_authentication_http.WithConfiguration(options.authenticator),
			internal_authentication_http.WithCallback(options.callback),
			internal_authentication_http.WithCaller(options.caller),
			internal_authentication_http.WithOnPacket(options.onPacket),
		)
	default:
		err := fmt.Errorf("authentication: unsupported executor type %q", options.authenticator.Provider)
		if options.onPacket != nil {
			_ = options.onPacket(options.ctx,
				internal_type.ObservabilityMetricRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordMetric{
						Attributes: observability.Attributes{
							"provider":         options.authenticator.Provider,
							"configuration_id": fmt.Sprintf("%d", options.authenticator.Id),
							"status":           "failed",
						},
						Metrics: []*protos.Metric{{
							Name:        observability.MetricAuthenticationInitLatencyMs,
							Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
							Description: "Authentication initialization latency in milliseconds",
						}},
					},
				},
				internal_type.ObservabilityLogRecordPacket{
					Scope: internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "authentication: initialization failed",
						Attributes: observability.Attributes{
							"component":        observability.ComponentAuthentication.String(),
							"operation":        "initialize_executor",
							"provider":         options.authenticator.Provider,
							"configuration_id": fmt.Sprintf("%d", options.authenticator.Id),
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
