// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (e *modelAssistantExecutor) sendStreamConfiguration(
	initCtx context.Context,
	connection *ModelConnection,
	communication internal_type.Communication,
) error {
	assistant, err := communication.Assistant()
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- connection.Send(&protos.StreamChatRequest{
			Request: &protos.StreamChatRequest_Configuration{
				Configuration: &protos.StreamChatConfiguration{
					Credential:        &protos.Credential{Id: e.providerCredential.GetId(), Value: e.providerCredential.GetValue()},
					ProviderName:      strings.ToLower(assistant.AssistantProviderModel.ModelProviderName),
					ConnectionOptions: modelConnectionOptions(e.providerOptions),
				},
			},
		})
	}()
	select {
	case <-initCtx.Done():
		return initCtx.Err()
	case err := <-done:
		return err
	}
}

func (e *modelAssistantExecutor) handleToolFollowUp(ctx context.Context, communication internal_type.Communication, contextID string) {
	snapshot := e.history.Snapshot()

	e.mu.RLock()
	connection := e.connection
	e.mu.RUnlock()
	if !validator.NonNil(connection) {
		e.logger.Errorf("stream not connected for tool follow-up")
		return
	}
	if err := e.validateHistorySequence(snapshot); err != nil {
		e.logger.Errorf("history integrity failed, blocking tool follow-up: %v", err)
		return
	}
	promptArgs := e.buildBasePromptArgs(communication)
	request, err := e.chatStreamRequest(communication, contextID, promptArgs, snapshot...)
	if err != nil {
		e.logger.Errorf("tool follow-up request build failed: %v", err)
		return
	}
	if err := connection.Send(&protos.StreamChatRequest{Request: &protos.StreamChatRequest_Chat{Chat: request}}); err != nil {
		e.logger.Errorf("tool follow-up send failed: %v", err)
	}
}

func (e *modelAssistantExecutor) sendChat(
	communication internal_type.Communication,
	contextID string,
	promptArgs map[string]interface{},
	messages ...*protos.Message,
) error {
	e.mu.RLock()
	connection := e.connection
	e.mu.RUnlock()
	if !validator.NonNil(connection) {
		return fmt.Errorf("stream not connected")
	}
	request, err := e.chatStreamRequest(communication, contextID, promptArgs, messages...)
	if err != nil {
		return err
	}
	return connection.Send(&protos.StreamChatRequest{
		Request: &protos.StreamChatRequest_Chat{Chat: request},
	})
}

func (e *modelAssistantExecutor) listen(ctx context.Context, communication internal_type.Communication) {
	for {
		e.mu.RLock()
		connection := e.connection
		e.mu.RUnlock()
		if !validator.NonNil(connection) {
			return
		}
		resp, err := connection.Recv()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			contextID := e.currentContextID()
			providerName := "unknown"
			if assistant, assistantErr := communication.Assistant(); assistantErr == nil {
				providerName = assistant.AssistantProviderModel.ModelProviderName
			}
			communication.OnPacket(ctx,
				internal_type.LLMErrorPacket{
					ContextID: contextID,
					Error:     err,
					Type:      internal_type.LLMSystemPanic,
				},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.NewMessageRecord(contextID, observability.ComponentAgent, observability.AgentError, observability.MessageRoleAssistant, observability.Attributes{
						"provider":   providerName,
						"context_id": contextID,
						"error":      err.Error(),
						"error_type": fmt.Sprintf("%T", err),
					}),
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "llm stream receive failed",
						Attributes: observability.Attributes{
							"component":  observability.ComponentAgent.String(),
							"operation":  "listen",
							"provider":   providerName,
							"context_id": contextID,
							"error":      err.Error(),
							"error_type": fmt.Sprintf("%T", err),
						},
						OccurredAt: time.Now(),
					},
				},
			)
			return
		}
		switch v := resp.GetResponse().(type) {
		case *protos.StreamChatResponse_Chat:
			e.Run(ctx, communication, ResponsePipeline{Response: v.Chat})
		case *protos.StreamChatResponse_Close:
			communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
				ContextID: e.currentContextID(),
				Name:      "end_conversation",
				Action:    protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
				Arguments: map[string]string{"reason": v.Close.GetReason()},
			})
			return
		case *protos.StreamChatResponse_Configuration:
			// Late configuration response (we already handled it during initialization). Ignore.
		default:
			e.logger.Warnf("unknown stream response variant: %T", v)
		}
	}
}
