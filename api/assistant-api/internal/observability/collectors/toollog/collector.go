// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_toollog

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
	ErrToolLogServiceRequired  = errors.New("toollog: service is required")
	ErrToolLogAuthRequired     = errors.New("toollog: auth is required")
	ErrToolLogScopeRequired    = errors.New("toollog: assistant_id and conversation_id are required")
	ErrToolLogScopeUnsupported = errors.New("toollog: scope is not supported")
)

type Config struct {
	Logger      commons.Logger
	ToolService internal_services.AssistantToolService
}

type Collector struct {
	logger      commons.Logger
	toolService internal_services.AssistantToolService
}

func New(config Config) observability.Collector {
	return &Collector{
		logger:      config.Logger,
		toolService: config.ToolService,
	}
}

func (c *Collector) Key() string {
	return "toollog"
}

func (c *Collector) Collect(ctx context.Context, scope observability.Scope, observationContext observability.Context, record observability.Record) error {
	toolLogRecord, ok := record.(observability.RecordToolLog)
	if !ok {
		return nil
	}
	if !validator.NonNil(c) || !validator.NonNil(c.toolService) {
		return ErrToolLogServiceRequired
	}
	if !validator.NonZero(observationContext.Auth.Subject().ID) {
		return ErrToolLogAuthRequired
	}

	assistantID := uint64(0)
	conversationID := uint64(0)
	messageID := ""
	switch typedScope := scope.(type) {
	case observability.MessageScope:
		assistantID = typedScope.AssistantScopeID()
		conversationID = typedScope.ConversationScopeID()
		messageID = typedScope.ContextID()
	case observability.ConversationScope:
		assistantID = typedScope.AssistantScopeID()
		conversationID = typedScope.ConversationScopeID()
	default:
		return fmt.Errorf("%w: %T", ErrToolLogScopeUnsupported, scope)
	}
	if assistantID == 0 || conversationID == 0 {
		return fmt.Errorf("%w: assistant_id=%d conversation_id=%d", ErrToolLogScopeRequired, assistantID, conversationID)
	}

	switch toolLogRecord.Operation {
	case observability.ToolLogOperationCreate:
		_, err := c.toolService.CreateLog(
			ctx,
			observationContext.Auth,
			assistantID,
			conversationID,
			messageID,
			toolLogRecord.ToolCallID,
			toolLogRecord.ToolName,
			toolLogRecord.Status,
			toolLogRecord.RequestPayload,
		)
		return err
	case observability.ToolLogOperationUpdate:
		_, err := c.toolService.UpdateLog(
			ctx,
			observationContext.Auth,
			toolLogRecord.ToolCallID,
			conversationID,
			toolLogRecord.Status,
			toolLogRecord.ResponsePayload,
		)
		return err
	default:
		return nil
	}
}

func (c *Collector) Close(context.Context) error {
	return nil
}
