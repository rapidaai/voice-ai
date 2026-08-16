// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentkit

import (
	"context"
	"fmt"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (e *agentkitExecutor) Execute(ctx context.Context, comm internal_type.Communication, pctk internal_type.Packet) error {
	switch p := pctk.(type) {
	case internal_type.UserInputPacket:
		return e.handleUserTurn(ctx, comm, p.ContextID, p.Text)
	case internal_type.InjectMessagePacket:
		return e.handleInjectMessage(p)
	case internal_type.LLMToolCallPacket:
		return e.handleToolCall(p)
	case internal_type.LLMToolResultPacket:
		return e.handleToolResult(p)
	case internal_type.LLMInterruptPacket:
		return e.handleInterrupt(p)
	default:
		return fmt.Errorf("%w: %T", ErrAgentkitExecuteUnsupportedPacket, pctk)
	}
}

func (e *agentkitExecutor) handleInjectMessage(packet internal_type.InjectMessagePacket) error {
	e.stateMu.RLock()
	activeConnection := e.connection
	e.stateMu.RUnlock()
	if !validator.NonNil(activeConnection) {
		return ErrAgentkitExecutorNotConnected
	}
	return activeConnection.Send(&protos.TalkInput{
		Request: &protos.TalkInput_Assistant{
			Assistant: &protos.ConversationAssistantMessage{
				Id:        packet.ContextID,
				Completed: true,
				Message:   &protos.ConversationAssistantMessage_Text{Text: packet.Text},
				Time:      timestamppb.Now(),
			},
		},
	})
}

func (e *agentkitExecutor) handleToolCall(packet internal_type.LLMToolCallPacket) error {
	e.stateMu.RLock()
	activeConnection := e.connection
	e.stateMu.RUnlock()
	if !validator.NonNil(activeConnection) {
		return ErrAgentkitExecutorNotConnected
	}
	return activeConnection.Send(&protos.TalkInput{
		Request: &protos.TalkInput_ToolCall{
			ToolCall: &protos.ConversationToolCall{
				Id:     packet.ContextID,
				ToolId: packet.ToolID,
				Name:   packet.Name,
				Action: packet.Action,
				Args:   packet.Arguments,
				Time:   timestamppb.Now(),
			},
		},
	})
}

func (e *agentkitExecutor) handleToolResult(packet internal_type.LLMToolResultPacket) error {
	e.stateMu.RLock()
	activeConnection := e.connection
	e.stateMu.RUnlock()
	if !validator.NonNil(activeConnection) {
		return ErrAgentkitExecutorNotConnected
	}
	return activeConnection.Send(&protos.TalkInput{
		Request: &protos.TalkInput_ToolCallResult{
			ToolCallResult: &protos.ConversationToolCallResult{
				Id:     packet.ContextID,
				ToolId: packet.ToolID,
				Name:   packet.Name,
				Action: packet.Action,
				Result: packet.Result,
				Time:   timestamppb.Now(),
			},
		},
	})
}

func (e *agentkitExecutor) handleInterrupt(packet internal_type.LLMInterruptPacket) error {
	e.stateMu.RLock()
	activeConnection := e.connection
	e.stateMu.RUnlock()
	if !validator.NonNil(activeConnection) {
		return ErrAgentkitExecutorNotConnected
	}
	e.stateMu.Lock()
	e.activeContextID = ""
	e.stateMu.Unlock()
	return activeConnection.Send(&protos.TalkInput{
		Request: &protos.TalkInput_Interruption{
			Interruption: &protos.ConversationInterruption{
				Id:   packet.ContextID,
				Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
				Time: timestamppb.Now(),
			},
		},
	})
}

func (e *agentkitExecutor) handleUserTurn(ctx context.Context, comm internal_type.Communication, contextID, text string) error {
	if !validator.NotBlank(contextID) {
		return nil
	}
	e.stateMu.RLock()
	activeConnection := e.connection
	e.stateMu.RUnlock()

	if !validator.NonNil(activeConnection) {
		return ErrAgentkitExecutorNotConnected
	}
	e.stateMu.Lock()
	e.activeContextID = contextID
	e.requestStartedAt = time.Now()
	e.waitingForFirstResponse = true
	e.stateMu.Unlock()

	comm.OnPacket(ctx,
		internal_type.ObservabilityEventRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.NewMessageRecord(contextID, observability.ComponentAgent, observability.AgentStarted, observability.MessageRoleAssistant, observability.Attributes{
				"provider":         e.Name(),
				"context_id":       contextID,
				"input_char_count": fmt.Sprintf("%d", len(text)),
			}),
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "agentkit request started",
				Attributes: observability.Attributes{
					"component":        observability.ComponentAgent.String(),
					"operation":        "execute",
					"provider":         e.Name(),
					"context_id":       contextID,
					"input_char_count": fmt.Sprintf("%d", len(text)),
				},
				OccurredAt: time.Now(),
			},
		},
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": e.Name()},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricAgentMessageCharCount,
					Value:       fmt.Sprintf("%d", len(text)),
					Description: "Input character count sent to AgentKit",
				}},
			},
		},
	)

	return activeConnection.Send(&protos.TalkInput{
		Request: &protos.TalkInput_User{
			User: &protos.ConversationUserMessage{
				Message:   &protos.ConversationUserMessage_Text{Text: text},
				Id:        contextID,
				Completed: true,
				Time:      timestamppb.Now(),
			},
		},
	})
}
