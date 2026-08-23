// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_conversationmetric

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

type conversationMetricServiceStub struct {
	internal_services.AssistantConversationService

	metricAuth           *types.Authentication
	metricAssistantID    uint64
	metricConversationID uint64
	metrics              []*protos.Metric

	messageMetricAuth           *types.Authentication
	messageMetricConversationID uint64
	messageMetricMessageID      string
	messageMetrics              []*protos.Metric
}

func (s *conversationMetricServiceStub) CreateOrUpdateConversationMetrics(
	_ context.Context,
	auth *types.Authentication,
	assistantID uint64,
	conversationID uint64,
	metrics []*protos.Metric,
) ([]*internal_conversation_entity.AssistantConversationMetric, error) {
	s.metricAuth = auth
	s.metricAssistantID = assistantID
	s.metricConversationID = conversationID
	s.metrics = metrics
	return nil, nil
}

func (s *conversationMetricServiceStub) CreateOrUpdateMessageMetrics(
	_ context.Context,
	auth *types.Authentication,
	conversationID uint64,
	messageID string,
	metrics []*protos.Metric,
) ([]*internal_message_gorm.AssistantConversationMessageMetric, error) {
	s.messageMetricAuth = auth
	s.messageMetricConversationID = conversationID
	s.messageMetricMessageID = messageID
	s.messageMetrics = metrics
	return nil, nil
}

func TestCollectMetric_RequiresValidConversationScope(t *testing.T) {
	collector := New(Config{ConversationService: &conversationMetricServiceStub{}})
	auth := testMetricAuth()

	err := collector.Collect(context.Background(), observability.ConversationScope{
		AssistantScope: observability.AssistantScope{
			AssistantID: 10,
		},
	}, observability.Context{Auth: auth}, observability.RecordMetric{
		Metrics: []*protos.Metric{{Name: observability.MetricConversationDuration, Value: "1000"}},
	})
	if !errors.Is(err, ErrMetricScopeRequired) {
		t.Fatalf("expected scope error, got %v", err)
	}
}

func TestCollectMetric_EmptyMetricsAreNoop(t *testing.T) {
	collector := New(Config{ConversationService: &conversationMetricServiceStub{}})
	if err := collector.Collect(context.Background(), observability.ConversationScope{}, observability.Context{}, observability.RecordMetric{}); err != nil {
		t.Fatalf("empty metrics should be noop, got %v", err)
	}
}

func TestCollectMetric_ForwardsConversationScopedRecords(t *testing.T) {
	service := &conversationMetricServiceStub{}
	collector := New(Config{ConversationService: service})
	auth := testMetricAuth()
	scope := observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}

	err := collector.Collect(context.Background(), scope, observability.Context{Auth: auth}, observability.RecordMetric{
		Metrics: []*protos.Metric{{
			Name:        observability.MetricConversationDuration,
			Value:       "1000",
			Description: "duration",
		}},
	})
	if err != nil {
		t.Fatalf("CollectMetric returned error: %v", err)
	}
	if service.metricAuth != auth || service.metricAssistantID != 10 || service.metricConversationID != 20 {
		t.Fatalf("unexpected call: auth=%v assistant=%d conversation=%d", service.metricAuth, service.metricAssistantID, service.metricConversationID)
	}
	if len(service.metrics) != 1 || service.metrics[0].Name != observability.MetricConversationDuration {
		t.Fatalf("unexpected metrics: %+v", service.metrics)
	}
}

func TestCollectMetric_ForwardsMessageScopedRecords(t *testing.T) {
	service := &conversationMetricServiceStub{}
	collector := New(Config{ConversationService: service})
	auth := testMetricAuth()
	scope := observability.MessageScope{
		ConversationScope: observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		MessageID: "assistant-ctx-1",
		Role:      observability.MessageRoleAssistant,
	}

	err := collector.Collect(context.Background(), scope, observability.Context{Auth: auth}, observability.RecordMetric{
		Metrics: []*protos.Metric{{Name: observability.MetricConversationDuration, Value: "1000"}},
	})
	if err != nil {
		t.Fatalf("CollectMetric returned error: %v", err)
	}
	if service.messageMetricAuth != auth || service.messageMetricConversationID != 20 || service.messageMetricMessageID != "assistant-assistant-ctx-1" {
		t.Fatalf("unexpected message metric call: auth=%v conversation=%d message=%s", service.messageMetricAuth, service.messageMetricConversationID, service.messageMetricMessageID)
	}
}

func testMetricAuth() *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeService,
		UserValue:         &types.UserContext{UserID: 99},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 1},
		ProjectValue:      &types.ProjectContext{OrganizationID: 1, ProjectID: 2},
	}
}
