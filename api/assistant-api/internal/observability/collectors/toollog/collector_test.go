// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_toollog

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestCollectorCreatesAndUpdatesToolLog(t *testing.T) {
	toolService := &recordingToolService{}
	collector := New(Config{ToolService: toolService})
	auth := &types.Authentication{}

	err := collector.Collect(
		context.Background(),
		observability.MessageScope{
			ConversationScope: observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: 10},
				ConversationID: 20,
			},
			MessageID: "message-1",
			Role:      observability.MessageRoleAssistant,
		},
		observability.Context{Auth: auth},
		observability.RecordToolLog{
			Operation:      observability.ToolLogOperationCreate,
			ToolCallID:     "call-1",
			ToolName:       "lookup",
			Status:         type_enums.RECORD_COMPLETE,
			RequestPayload: []byte(`{"request":true}`),
		},
	)
	if err != nil {
		t.Fatalf("Collect create returned error: %v", err)
	}

	err = collector.Collect(
		context.Background(),
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		observability.Context{Auth: auth},
		observability.RecordToolLog{
			Operation:       observability.ToolLogOperationUpdate,
			ToolCallID:      "call-1",
			Status:          type_enums.RECORD_FAILED,
			ResponsePayload: []byte(`{"error":true}`),
		},
	)
	if err != nil {
		t.Fatalf("Collect update returned error: %v", err)
	}

	if len(toolService.createCalls) != 1 {
		t.Fatalf("expected one create call, got %d", len(toolService.createCalls))
	}
	createCall := toolService.createCalls[0]
	if createCall.assistantID != 10 || createCall.conversationID != 20 || createCall.messageID != "message-1" {
		t.Fatalf("unexpected create scope: %+v", createCall)
	}
	if createCall.toolCallID != "call-1" || createCall.toolName != "lookup" || createCall.status != type_enums.RECORD_COMPLETE {
		t.Fatalf("unexpected create fields: %+v", createCall)
	}

	if len(toolService.updateCalls) != 1 {
		t.Fatalf("expected one update call, got %d", len(toolService.updateCalls))
	}
	updateCall := toolService.updateCalls[0]
	if updateCall.toolCallID != "call-1" || updateCall.conversationID != 20 || updateCall.status != type_enums.RECORD_FAILED {
		t.Fatalf("unexpected update fields: %+v", updateCall)
	}
}

type toolCreateCall struct {
	assistantID    uint64
	conversationID uint64
	messageID      string
	toolCallID     string
	toolName       string
	status         type_enums.RecordState
}

type toolUpdateCall struct {
	toolCallID     string
	conversationID uint64
	status         type_enums.RecordState
}

type recordingToolService struct {
	createCalls []toolCreateCall
	updateCalls []toolUpdateCall
}

func (s *recordingToolService) Get(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantTool, error) {
	return nil, nil
}

func (s *recordingToolService) GetAll(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_assistant_entity.AssistantTool, error) {
	return 0, nil, nil
}

func (s *recordingToolService) Create(context.Context, *types.Authentication, uint64, string, *string, map[string]interface{}, string, []*protos.Metadata) (*internal_assistant_entity.AssistantTool, error) {
	return nil, nil
}

func (s *recordingToolService) Update(context.Context, *types.Authentication, uint64, uint64, string, *string, map[string]interface{}, string, []*protos.Metadata) (*internal_assistant_entity.AssistantTool, error) {
	return nil, nil
}

func (s *recordingToolService) Delete(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantTool, error) {
	return nil, nil
}

func (s *recordingToolService) CreateLog(
	_ context.Context,
	_ *types.Authentication,
	assistantID uint64,
	conversationID uint64,
	messageID string,
	toolCallID string,
	toolName string,
	status type_enums.RecordState,
	_ []byte,
) (*internal_assistant_entity.AssistantToolLog, error) {
	s.createCalls = append(s.createCalls, toolCreateCall{
		assistantID:    assistantID,
		conversationID: conversationID,
		messageID:      messageID,
		toolCallID:     toolCallID,
		toolName:       toolName,
		status:         status,
	})
	return &internal_assistant_entity.AssistantToolLog{}, nil
}

func (s *recordingToolService) UpdateLog(
	_ context.Context,
	_ *types.Authentication,
	toolCallID string,
	conversationID uint64,
	status type_enums.RecordState,
	_ []byte,
) (*internal_assistant_entity.AssistantToolLog, error) {
	s.updateCalls = append(s.updateCalls, toolUpdateCall{
		toolCallID:     toolCallID,
		conversationID: conversationID,
		status:         status,
	})
	return &internal_assistant_entity.AssistantToolLog{}, nil
}

func (s *recordingToolService) GetLog(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantToolLog, error) {
	return nil, nil
}

func (s *recordingToolService) GetAllLog(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate, *protos.Ordering) (int64, []*internal_assistant_entity.AssistantToolLog, error) {
	return 0, nil, nil
}

func (s *recordingToolService) GetLogObject(context.Context, *types.Authentication, uint64) ([]byte, []byte, error) {
	return nil, nil, nil
}
