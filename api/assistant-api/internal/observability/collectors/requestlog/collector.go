// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_requestlog

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
	ErrRequestLogServiceRequired  = errors.New("requestlog: service is required")
	ErrRequestLogAuthRequired     = errors.New("requestlog: auth is required")
	ErrRequestLogScopeRequired    = errors.New("requestlog: assistant_id is required")
	ErrRequestLogScopeUnsupported = errors.New("requestlog: scope is not supported")
)

type Config struct {
	Logger         commons.Logger
	HTTPLogService internal_services.AssistantHTTPLogService
}

type Collector struct {
	logger         commons.Logger
	httpLogService internal_services.AssistantHTTPLogService
}

func New(config Config) observability.Collector {
	return &Collector{
		logger:         config.Logger,
		httpLogService: config.HTTPLogService,
	}
}

func (c *Collector) Key() string {
	return "requestlog"
}

func (c *Collector) Collect(ctx context.Context, scope observability.Scope, observationContext observability.Context, record observability.Record) error {
	requestLogRecord, ok := record.(observability.RecordRequestLog)
	if !ok {
		return nil
	}
	if !validator.NonNil(c) || !validator.NonNil(c.httpLogService) {
		return ErrRequestLogServiceRequired
	}
	if !validator.NonZero(observationContext.Auth.Subject().ID) {
		return ErrRequestLogAuthRequired
	}

	assistantID := uint64(0)
	var conversationID *uint64
	switch typedScope := scope.(type) {
	case observability.MessageScope:
		assistantID = typedScope.AssistantScopeID()
		scopeConversationID := typedScope.ConversationScopeID()
		conversationID = &scopeConversationID
	case observability.ConversationScope:
		assistantID = typedScope.AssistantScopeID()
		scopeConversationID := typedScope.ConversationScopeID()
		conversationID = &scopeConversationID
	case observability.AssistantScope:
		assistantID = typedScope.AssistantScopeID()
	default:
		return fmt.Errorf("%w: %T", ErrRequestLogScopeUnsupported, scope)
	}
	if assistantID == 0 {
		return fmt.Errorf("%w: assistant_id=%d", ErrRequestLogScopeRequired, assistantID)
	}

	_, err := c.httpLogService.CreateLog(
		ctx,
		observationContext.Auth,
		requestLogRecord.Source,
		requestLogRecord.SourceRefID,
		requestLogRecord.SourceEvent,
		requestLogRecord.ContextID,
		assistantID,
		conversationID,
		requestLogRecord.HTTPURL,
		requestLogRecord.HTTPMethod,
		requestLogRecord.ResponseStatus,
		requestLogRecord.TimeTaken,
		requestLogRecord.RetryCount,
		requestLogRecord.Status,
		requestLogRecord.ErrorMessage,
		requestLogRecord.RequestPayload,
		requestLogRecord.ResponsePayload,
	)
	return err
}

func (c *Collector) Close(context.Context) error {
	return nil
}
