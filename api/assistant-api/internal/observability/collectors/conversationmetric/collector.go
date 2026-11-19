// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_conversationmetric

import (
	"context"
	"errors"
	"fmt"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/validator"
)

var (
	ErrMetricServiceRequired  = errors.New("conversationmetric: service is required")
	ErrMetricAuthRequired     = errors.New("conversationmetric: auth is required")
	ErrMetricScopeRequired    = errors.New("conversationmetric: assistant_id and conversation_id are required")
	ErrMetricMessageRequired  = errors.New("conversationmetric: conversation_id and message_id are required")
	ErrMetricScopeUnsupported = errors.New("conversationmetric: scope is not supported")
)

type Config struct {
	Logger              commons.Logger
	ConversationService internal_services.AssistantConversationService
}

type Collector struct {
	logger              commons.Logger
	conversationService internal_services.AssistantConversationService
}

func New(config Config) observability.Collector {
	return &Collector{
		logger:              config.Logger,
		conversationService: config.ConversationService,
	}
}

func (c *Collector) Key() string {
	return "conversationmetric"
}

func (c *Collector) Collect(ctx context.Context, scope observability.Scope, observationContext observability.Context, record observability.Record) error {
	metricRecord, ok := record.(observability.RecordMetric)
	if !ok {
		return nil
	}
	if !validator.NotEmpty(metricRecord.Metrics) {
		return nil
	}
	if !validator.NonNil(c) || !validator.NonNil(c.conversationService) {
		return ErrMetricServiceRequired
	}
	if !validator.NonZero(observationContext.Auth.Subject().ID) {
		return ErrMetricAuthRequired
	}

	switch typedScope := scope.(type) {
	case observability.MessageScope:
		if typedScope.ConversationScopeID() == 0 || !validator.NotBlank(typedScope.MessageScopeID()) {
			return fmt.Errorf("%w: conversation_id=%d message_id=%q", ErrMetricMessageRequired, typedScope.ConversationScopeID(), typedScope.MessageScopeID())
		}
		_, err := c.conversationService.CreateOrUpdateMessageMetrics(
			ctx,
			observationContext.Auth,
			typedScope.ConversationScopeID(),
			typedScope.MessageScopeID(),
			metricRecord.Metrics,
		)
		return err
	case observability.ConversationScope:
		if typedScope.AssistantScopeID() == 0 || typedScope.ConversationScopeID() == 0 {
			return fmt.Errorf("%w: assistant_id=%d conversation_id=%d", ErrMetricScopeRequired, typedScope.AssistantScopeID(), typedScope.ConversationScopeID())
		}
		_, err := c.conversationService.CreateOrUpdateConversationMetrics(
			ctx,
			observationContext.Auth,
			typedScope.AssistantScopeID(),
			typedScope.ConversationScopeID(),
			metricRecord.Metrics,
		)
		return err
	case observability.AssistantScope:
		return fmt.Errorf("%w: %s", ErrMetricScopeUnsupported, observability.ScopeAssistant)
	default:
		return fmt.Errorf("%w: %T", ErrMetricScopeUnsupported, scope)
	}
}

func (c *Collector) Close(context.Context) error {
	return nil
}
