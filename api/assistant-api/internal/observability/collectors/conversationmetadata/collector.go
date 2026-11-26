// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_conversationmetadata

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
	ErrMetadataServiceRequired  = errors.New("conversationmetadata: service is required")
	ErrMetadataAuthRequired     = errors.New("conversationmetadata: auth is required")
	ErrMetadataScopeRequired    = errors.New("conversationmetadata: assistant_id and conversation_id are required")
	ErrMetadataMessageRequired  = errors.New("conversationmetadata: conversation_id and message_id are required")
	ErrMetadataScopeUnsupported = errors.New("conversationmetadata: scope is not supported")
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
	return "conversationmetadata"
}

func (c *Collector) Collect(ctx context.Context, scope observability.Scope, observationContext observability.Context, record observability.Record) error {
	metadataRecord, ok := record.(observability.RecordMetadata)
	if !ok {
		return nil
	}
	if !validator.NotEmpty(metadataRecord.Metadata) {
		return nil
	}
	if !validator.NonNil(c) || !validator.NonNil(c.conversationService) {
		return ErrMetadataServiceRequired
	}
	if !validator.NonZero(observationContext.Auth.Subject().ID) {
		return ErrMetadataAuthRequired
	}

	switch typedScope := scope.(type) {
	case observability.MessageScope:
		if typedScope.ConversationScopeID() == 0 || !validator.NotBlank(typedScope.MessageScopeID()) {
			return fmt.Errorf("%w: conversation_id=%d message_id=%q", ErrMetadataMessageRequired, typedScope.ConversationScopeID(), typedScope.MessageScopeID())
		}
		_, err := c.conversationService.CreateOrUpdateMessageMetadata(
			ctx,
			observationContext.Auth,
			typedScope.ConversationScopeID(),
			typedScope.MessageScopeID(),
			metadataRecord.Metadata,
		)
		return err
	case observability.ConversationScope:
		if typedScope.AssistantScopeID() == 0 || typedScope.ConversationScopeID() == 0 {
			return fmt.Errorf("%w: assistant_id=%d conversation_id=%d", ErrMetadataScopeRequired, typedScope.AssistantScopeID(), typedScope.ConversationScopeID())
		}
		_, err := c.conversationService.CreateOrUpdateConversationMetadata(
			ctx,
			observationContext.Auth,
			typedScope.AssistantScopeID(),
			typedScope.ConversationScopeID(),
			metadataRecord.Metadata,
		)
		return err
	case observability.AssistantScope:
		return fmt.Errorf("%w: %s", ErrMetadataScopeUnsupported, observability.ScopeAssistant)
	default:
		return fmt.Errorf("%w: %T", ErrMetadataScopeUnsupported, scope)
	}
}

func (c *Collector) Close(context.Context) error {
	return nil
}
