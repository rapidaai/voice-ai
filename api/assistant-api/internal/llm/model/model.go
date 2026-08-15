// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_model

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_agent_tool "github.com/rapidaai/api/assistant-api/internal/tool"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"golang.org/x/sync/errgroup"
)

type modelAssistantExecutor struct {
	logger             commons.Logger
	toolExecutor       internal_agent_tool.ToolExecutor
	providerCredential *protos.VaultCredential
	inputBuilder       integration_client_builders.InputChatBuilder
	history            *ConversationHistory
	connection         *ModelConnection
	providerOptions    utils.Option

	currentPacket *internal_type.UserInputPacket
	mu            sync.RWMutex
	ctx           context.Context
	ctxCancel     context.CancelFunc
}

type options struct {
	ctx           context.Context
	logger        commons.Logger
	communication internal_type.Communication
	configuration *protos.ConversationInitialization
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

func WithCommunication(communication internal_type.Communication) Option {
	return func(options *options) {
		options.communication = communication
	}
}

func WithConfiguration(configuration *protos.ConversationInitialization) Option {
	return func(options *options) {
		options.configuration = configuration
	}
}

func New(opts ...Option) (*modelAssistantExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if validator.NonNil(opt) {
			opt(options)
		}
	}
	if !validator.NonNil(options.ctx) {
		options.ctx = context.Background()
	}
	if !validator.NonNil(options.communication) {
		return nil, errors.New("model: communication is required")
	}
	if !validator.NonNil(options.configuration) {
		return nil, errors.New("model: configuration is required")
	}
	assistant, err := options.communication.Assistant()
	if err != nil || !validator.NonNil(assistant) {
		return nil, errors.New("model: assistant is required")
	}
	if !validator.NonNil(assistant.AssistantProviderModel) {
		return nil, errors.New("model: provider configuration is required")
	}

	start := time.Now()
	providerConfig := assistant.AssistantProviderModel
	providerOptions := withModelOverrides(providerConfig.GetOptions(), options.communication.GetOptions())
	provider := providerConfig.ModelProviderName
	executorCtx, cancel := context.WithCancel(context.Background())
	executor := &modelAssistantExecutor{
		logger:          options.logger,
		history:         NewConversationHistory(),
		inputBuilder:    integration_client_builders.NewChatInputBuilder(options.logger),
		connection:      NewModelConnection(provider),
		providerOptions: providerOptions,
		ctx:             executorCtx,
		ctxCancel:       cancel,
	}

	g, gCtx := errgroup.WithContext(options.ctx)
	var providerCredential *protos.VaultCredential
	var toolExecutor internal_agent_tool.ToolExecutor
	g.Go(func() error {
		credentialID, err := providerOptions.GetUint64(modelOptionCredentialID)
		if err != nil {
			return fmt.Errorf("failed to get credential ID: %w", err)
		}
		cred, err := options.communication.VaultCaller().GetCredential(gCtx, options.communication.Auth(), credentialID)
		if err != nil {
			return fmt.Errorf("failed to get provider credential: %w", err)
		}
		providerCredential = cred
		return nil
	})
	g.Go(func() error {
		initializedToolExecutor, err := internal_agent_tool.New(
			internal_agent_tool.WithContext(gCtx),
			internal_agent_tool.WithLogger(executor.logger),
			internal_agent_tool.WithCommunication(options.communication),
		)
		if err != nil {
			return err
		}
		toolExecutor = initializedToolExecutor
		return nil
	})
	if err := g.Wait(); err != nil {
		options.communication.OnPacket(options.ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("%s: error while initialization %s", executor.Name(), err.Error()),
				Attributes: observability.Attributes{
					"component":  observability.ComponentAgent.String(),
					"provider":   provider,
					"options":    observability.AttributeValue(providerOptions),
					"error":      err.Error(),
					"error_type": fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		executor.Close(options.ctx)
		return nil, err
	}
	executor.providerCredential = providerCredential
	executor.toolExecutor = toolExecutor

	if err := executor.connection.OpenStream(executor.ctx, options.communication); err != nil {
		options.communication.OnPacket(options.ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("%s: error while initialization %s", executor.Name(), err.Error()),
				Attributes: observability.Attributes{
					"component":  observability.ComponentAgent.String(),
					"provider":   provider,
					"options":    observability.AttributeValue(providerOptions),
					"error":      err.Error(),
					"error_type": fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		executor.Close(options.ctx)
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	utils.Go(executor.ctx, func() {
		executor.listen(executor.ctx, options.communication)
	})
	if err := executor.sendStreamConfiguration(options.ctx, executor.connection, options.communication); err != nil {
		options.communication.OnPacket(options.ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: fmt.Sprintf("%s: error while initialization %s", executor.Name(), err.Error()),
				Attributes: observability.Attributes{
					"component":  observability.ComponentAgent.String(),
					"provider":   provider,
					"options":    observability.AttributeValue(providerOptions),
					"error":      err.Error(),
					"error_type": fmt.Sprintf("%T", err),
				},
				OccurredAt: time.Now(),
			},
		})
		executor.Close(options.ctx)
		return nil, err
	}

	options.communication.OnPacket(options.ctx,
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": provider},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricLLMInitLatencyMs,
					Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
					Description: "LLM initialization latency in milliseconds",
				}},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: fmt.Sprintf("%s: initialization completed", executor.Name()),
				Attributes: observability.Attributes{
					"component": observability.ComponentAgent.String(),
					"provider":  provider,
					"options":   observability.AttributeValue(providerOptions),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return executor, nil
}

func (e *modelAssistantExecutor) Close(ctx context.Context) error {
	if validator.NonNil(e.ctxCancel) {
		e.ctxCancel()
	}
	e.mu.Lock()
	activeConnection := e.connection
	e.currentPacket = nil
	e.connection = nil
	e.mu.Unlock()
	if validator.NonNil(e.history) {
		e.history.Reset()
	}
	if validator.NonNil(activeConnection) {
		if err := activeConnection.Close("session ended"); err != nil {
			e.logger.Warnf("failed to close model connection: %v", err)
		}
	}
	if validator.NonNil(e.toolExecutor) {
		if err := e.toolExecutor.Close(ctx); err != nil {
			e.logger.Errorf("error closing tool executor: %v", err)
		}
	}
	return nil
}

func (e *modelAssistantExecutor) Name() string { return "model" }
