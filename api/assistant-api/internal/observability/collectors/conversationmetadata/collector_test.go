// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_conversationmetadata

import (
	"context"
	"errors"
	"testing"

	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_message_gorm "github.com/rapidaai/api/assistant-api/internal/entity/messages"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type conversationMetadataServiceStub struct {
	internal_services.AssistantConversationService

	metadataAuth           *types.Authentication
	metadataAssistantID    uint64
	metadataConversationID uint64
	metadata               []*protos.Metadata

	messageMetadataAuth           *types.Authentication
	messageMetadataConversationID uint64
	messageMetadataMessageID      string
	messageMetadata               []*protos.Metadata
}

func (s *conversationMetadataServiceStub) CreateOrUpdateConversationMetadata(
	_ context.Context,
	auth *types.Authentication,
	assistantID uint64,
	conversationID uint64,
	metadata []*protos.Metadata,
) ([]*internal_conversation_entity.AssistantConversationMetadata, error) {
	s.metadataAuth = auth
	s.metadataAssistantID = assistantID
	s.metadataConversationID = conversationID
	s.metadata = metadata
	return nil, nil
}

func (s *conversationMetadataServiceStub) CreateOrUpdateMessageMetadata(
	_ context.Context,
	auth *types.Authentication,
	conversationID uint64,
	messageID string,
	metadata []*protos.Metadata,
) ([]*internal_message_gorm.AssistantConversationMessageMetadata, error) {
	s.messageMetadataAuth = auth
	s.messageMetadataConversationID = conversationID
	s.messageMetadataMessageID = messageID
	s.messageMetadata = metadata
	return nil, nil
}

func TestCollectMetadata_RequiresValidConversationScope(t *testing.T) {
	collector := New(Config{ConversationService: &conversationMetadataServiceStub{}})
	auth := testMetadataAuth()

	err := collector.Collect(context.Background(), observability.ConversationScope{
		AssistantScope: observability.AssistantScope{
			AssistantID: 10,
		},
	}, observability.Context{Auth: auth}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{{Key: observability.MetadataLanguage, Value: "en"}},
	})
	if !errors.Is(err, ErrMetadataScopeRequired) {
		t.Fatalf("expected scope error, got %v", err)
	}
}

func TestCollectMetadata_EmptyMetadataIsNoop(t *testing.T) {
	collector := New(Config{ConversationService: &conversationMetadataServiceStub{}})
	if err := collector.Collect(context.Background(), observability.ConversationScope{}, observability.Context{}, observability.RecordMetadata{}); err != nil {
		t.Fatalf("empty metadata should be noop, got %v", err)
	}
}

func TestCollectMetadata_ForwardsConversationScopedRecords(t *testing.T) {
	service := &conversationMetadataServiceStub{}
	collector := New(Config{ConversationService: service})
	auth := testMetadataAuth()
	scope := observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}

	err := collector.Collect(context.Background(), scope, observability.Context{Auth: auth}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{{
			Key:   observability.MetadataLanguage,
			Value: "en",
		}},
	})
	if err != nil {
		t.Fatalf("CollectMetadata returned error: %v", err)
	}
	if service.metadataAuth != auth || service.metadataAssistantID != 10 || service.metadataConversationID != 20 {
		t.Fatalf("unexpected call: auth=%v assistant=%d conversation=%d", service.metadataAuth, service.metadataAssistantID, service.metadataConversationID)
	}
	if len(service.metadata) != 1 || service.metadata[0].Key != observability.MetadataLanguage {
		t.Fatalf("unexpected metadata: %+v", service.metadata)
	}
}

func TestCollectMetadata_ForwardsMessageScopedRecords(t *testing.T) {
	service := &conversationMetadataServiceStub{}
	collector := New(Config{ConversationService: service})
	auth := testMetadataAuth()
	scope := observability.MessageScope{
		ConversationScope: observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		MessageID: "user-ctx-1",
		Role:      observability.MessageRoleUser,
	}

	err := collector.Collect(context.Background(), scope, observability.Context{Auth: auth}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{{
			Key:   observability.MetadataLanguage,
			Value: "en",
		}},
	})
	if err != nil {
		t.Fatalf("CollectMetadata returned error: %v", err)
	}
	if service.messageMetadataAuth != auth || service.messageMetadataConversationID != 20 || service.messageMetadataMessageID != "user-user-ctx-1" {
		t.Fatalf("unexpected message metadata call: auth=%v conversation=%d message=%s", service.messageMetadataAuth, service.messageMetadataConversationID, service.messageMetadataMessageID)
	}
}

func TestCollectMetadata_AssistantScopeUnsupported(t *testing.T) {
	collector := New(Config{ConversationService: &conversationMetadataServiceStub{}})
	auth := testMetadataAuth()
	err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 10}, observability.Context{Auth: auth}, observability.RecordMetadata{
		Metadata: []*protos.Metadata{{Key: observability.MetadataLanguage, Value: "en"}},
	})
	if !errors.Is(err, ErrMetadataScopeUnsupported) {
		t.Fatalf("expected unsupported scope error, got %v", err)
	}
}

func testMetadataAuth() *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeService,
		UserValue:         &types.UserContext{UserID: 99},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 1},
		ProjectValue:      &types.ProjectContext{OrganizationID: 1, ProjectID: 2},
	}
}
