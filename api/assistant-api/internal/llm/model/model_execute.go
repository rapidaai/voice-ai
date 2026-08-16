// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/variable"
	internal_namespace "github.com/rapidaai/api/assistant-api/internal/variable/namespace"
	"github.com/rapidaai/pkg/parsers"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (e *modelAssistantExecutor) Execute(ctx context.Context, communication internal_type.Communication, pctk internal_type.Packet) error {
	switch p := pctk.(type) {
	case internal_type.UserInputPacket:
		if supersededCtx := e.history.SupersedePending(); supersededCtx != "" {
			assistant, err := communication.Assistant()
			if err != nil {
				return err
			}
			communication.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				ContextID: supersededCtx,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "tool block superseded",
					Attributes: observability.Attributes{
						"component":             observability.ComponentTool.String(),
						"operation":             "supersede_tool_block",
						"provider":              assistant.AssistantProviderModel.ModelProviderName,
						"context_id":            supersededCtx,
						"reason":                "user_interrupted",
						"superseded_context_id": supersededCtx,
					},
					OccurredAt: time.Now(),
				},
			})
		}
		e.mu.Lock()
		e.currentPacket = &p
		e.mu.Unlock()
		e.Run(ctx, communication, UserTurnPipeline{Packet: p})

	case internal_type.InjectMessagePacket:
		e.Run(ctx, communication, InjectMessagePipeline{Packet: p})

	case internal_type.LLMToolCallPacket:
		// no-op: dispatch handles logging/notification

	case internal_type.LLMToolResultPacket:
		e.Run(ctx, communication, ToolResultPipeline{Packet: p})

	case internal_type.LLMInterruptPacket:
		e.Run(ctx, communication, InterruptionPipeline{Packet: p})

	default:
		e.logger.Errorf("unsupported packet type: %T", pctk)
	}
	return nil
}

func (e *modelAssistantExecutor) Run(ctx context.Context, communication internal_type.Communication, p AgentPipeline) {
	switch v := p.(type) {
	case UserTurnPipeline:
		e.handleUserTurn(ctx, communication, v.Packet)
	case InjectMessagePipeline:
		e.history.AppendInjected(v.Packet.Text)
	case ToolResultPipeline:
		e.handleToolResult(ctx, communication, v.Packet)
	case InterruptionPipeline:
		e.handleInterruption()
	case ResponsePipeline:
		e.handleResponse(ctx, communication, v.Response)
	case ToolFollowUpPipeline:
		e.handleToolFollowUp(ctx, communication, v.ContextID)
	default:
		e.logger.Errorf("unknown pipeline type: %T", p)
	}
}

func (e *modelAssistantExecutor) handleUserTurn(ctx context.Context, communication internal_type.Communication, p internal_type.UserInputPacket) {
	assistant, err := communication.Assistant()
	if err != nil {
		communication.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: p.ContextID, Error: err})
		return
	}
	snapshot := e.history.Snapshot()
	promptArgs := e.buildPromptArgs(communication, p)
	providerName := assistant.AssistantProviderModel.ModelProviderName

	if err := e.validateHistorySequence(snapshot); err != nil {
		err = fmt.Errorf("history integrity: %w", err)
		communication.OnPacket(ctx,
			internal_type.LLMErrorPacket{ContextID: p.ContextID, Error: err},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: p.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(p.ContextID, observability.ComponentAgent, observability.AgentError, observability.MessageRoleAssistant, observability.Attributes{
					"provider":   providerName,
					"context_id": p.ContextID,
					"error":      err.Error(),
				}),
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: p.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "llm request failed",
					Attributes: observability.Attributes{
						"component":  observability.ComponentAgent.String(),
						"operation":  "execute",
						"provider":   providerName,
						"context_id": p.ContextID,
						"error":      err.Error(),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}

	e.mu.Lock()
	e.requestStartedAt = time.Now()
	e.waitingForFirstResponse = true
	e.mu.Unlock()

	communication.OnPacket(ctx,
		internal_type.ObservabilityEventRecordPacket{
			ContextID: p.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.NewMessageRecord(p.ContextID, observability.ComponentAgent, observability.AgentStarted, observability.MessageRoleAssistant, observability.Attributes{
				"provider":         providerName,
				"context_id":       p.ContextID,
				"input_char_count": fmt.Sprintf("%d", len(p.Text)),
				"history_count":    fmt.Sprintf("%d", len(snapshot)),
			}),
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: p.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "llm request started",
				Attributes: observability.Attributes{
					"component":        observability.ComponentAgent.String(),
					"operation":        "execute",
					"provider":         providerName,
					"context_id":       p.ContextID,
					"input_char_count": fmt.Sprintf("%d", len(p.Text)),
					"history_count":    fmt.Sprintf("%d", len(snapshot)),
				},
				OccurredAt: time.Now(),
			},
		},
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: p.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{"provider": providerName},
				Metrics: []*protos.Metric{
					{Name: observability.MetricAgentMessageCharCount, Value: fmt.Sprintf("%d", len(p.Text)), Description: "Input character count sent to agent"},
					{Name: observability.MetricAgentMessageCount, Value: fmt.Sprintf("%d", len(snapshot)), Description: "History message count sent to agent"},
				},
			},
		},
	)

	userMsg := &protos.Message{
		Role:    "user",
		Message: &protos.Message_User{User: &protos.UserMessage{Content: p.Text}},
	}
	if err := e.sendChat(communication, p.ContextID, promptArgs, append(snapshot, userMsg)...); err != nil {
		e.mu.Lock()
		e.requestStartedAt = time.Time{}
		e.waitingForFirstResponse = false
		e.mu.Unlock()
		communication.OnPacket(ctx,
			internal_type.LLMErrorPacket{ContextID: p.ContextID, Error: err},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: p.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(p.ContextID, observability.ComponentAgent, observability.AgentError, observability.MessageRoleAssistant, observability.Attributes{
					"provider":   providerName,
					"context_id": p.ContextID,
					"error":      err.Error(),
				}),
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: p.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "llm request failed",
					Attributes: observability.Attributes{
						"component":  observability.ComponentAgent.String(),
						"operation":  "execute",
						"provider":   providerName,
						"context_id": p.ContextID,
						"error":      err.Error(),
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}
	e.history.AppendUser(p.Text)
}

func (e *modelAssistantExecutor) handleToolResult(ctx context.Context, communication internal_type.Communication, p internal_type.LLMToolResultPacket) {
	assistant, err := communication.Assistant()
	if err != nil {
		communication.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: p.ContextID, Error: err})
		return
	}
	providerName := assistant.AssistantProviderModel.ModelProviderName
	resultJSON, _ := json.Marshal(p.Result)
	accepted, resolved := e.history.AcceptToolResult(p.ContextID, p.ToolID, p.Name, string(resultJSON))
	if !accepted {
		pendingCtx := e.history.PendingContextID()
		reason := "no_pending_block"
		data := map[string]string{"type": "tool_result_ignored", "reason": reason, "tool_id": p.ToolID}
		if pendingCtx != "" {
			reason = "context_or_id_mismatch"
			data["reason"] = reason
			data["pending_context"] = pendingCtx
		}
		communication.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			ContextID: p.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "tool result ignored",
				Attributes: observability.Attributes{
					"component":       observability.ComponentTool.String(),
					"operation":       "ignore_tool_result",
					"provider":        providerName,
					"context_id":      p.ContextID,
					"reason":          data["reason"],
					"tool_id":         p.ToolID,
					"name":            p.Name,
					"pending_context": data["pending_context"],
				},
				OccurredAt: time.Now(),
			},
		})
		return
	}
	if !resolved {
		return
	}

	contextID, followUp := e.history.FlushToolBlock()
	if !followUp {
		communication.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "tool block discarded",
				Attributes: observability.Attributes{
					"component":  observability.ComponentTool.String(),
					"operation":  "discard_tool_block",
					"provider":   providerName,
					"context_id": contextID,
					"reason":     "superseded",
				},
				OccurredAt: time.Now(),
			},
		})
		return
	}
	e.Run(ctx, communication, ToolFollowUpPipeline{ContextID: contextID})
}

func (e *modelAssistantExecutor) handleInterruption() {
	e.mu.Lock()
	e.requestStartedAt = time.Time{}
	e.waitingForFirstResponse = false
	e.mu.Unlock()
	e.history.SupersedePending()
}

func (e *modelAssistantExecutor) handleResponse(ctx context.Context, communication internal_type.Communication, resp *protos.StreamChatOutput) {
	if e.isStaleResponse(resp.GetRequestId()) {
		return
	}
	contextID := resp.GetRequestId()
	assistant, err := communication.Assistant()
	if err != nil {
		communication.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: contextID, Error: err})
		return
	}
	providerName := assistant.AssistantProviderModel.ModelProviderName

	if resp.GetError() != nil {
		e.mu.Lock()
		e.requestStartedAt = time.Time{}
		e.waitingForFirstResponse = false
		e.mu.Unlock()
		errMsg := resp.GetError().GetErrorMessage()
		communication.OnPacket(ctx,
			internal_type.LLMErrorPacket{ContextID: contextID, Error: errors.New(errMsg)},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(contextID, observability.ComponentAgent, observability.AgentError, observability.MessageRoleAssistant, observability.Attributes{
					"provider":   providerName,
					"context_id": contextID,
					"error":      errMsg,
				}),
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "llm response failed",
					Attributes: observability.Attributes{
						"component":  observability.ComponentAgent.String(),
						"operation":  "response",
						"provider":   providerName,
						"context_id": contextID,
						"error":      errMsg,
					},
					OccurredAt: time.Now(),
				},
			},
		)
		return
	}

	output := resp.GetData()
	if output == nil || output.GetAssistant() == nil {
		return
	}

	if len(resp.GetMetrics()) == 0 {
		e.onStreamingChunk(ctx, communication, contextID, output, providerName)
		return
	}
	e.onCompletion(ctx, communication, contextID, resp.GetFinishReason(), output, providerName)
}

func (e *modelAssistantExecutor) onStreamingChunk(ctx context.Context, communication internal_type.Communication, contextID string, output *protos.Message, providerName string) {
	text := strings.Join(output.GetAssistant().GetContents(), "")
	now := time.Now()
	e.mu.Lock()
	requestStartedAt := e.requestStartedAt
	publishTTFT := e.waitingForFirstResponse
	e.waitingForFirstResponse = false
	e.mu.Unlock()

	if publishTTFT && !requestStartedAt.IsZero() {
		communication.OnPacket(ctx,
			internal_type.LLMResponseDeltaPacket{ContextID: contextID, Text: text},
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordMetric{
					Attributes: observability.Attributes{"provider": providerName},
					Metrics: []*protos.Metric{{
						Name:        observability.MetricAgentTTFTMs,
						Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
						Description: "Agent time to first token in milliseconds",
					}},
				},
			},
		)
		return
	}
	communication.OnPacket(ctx, internal_type.LLMResponseDeltaPacket{ContextID: contextID, Text: text})
}

func (e *modelAssistantExecutor) onCompletion(ctx context.Context, communication internal_type.Communication, contextID, finishReason string, output *protos.Message, providerName string) {
	now := time.Now()
	e.mu.Lock()
	requestStartedAt := e.requestStartedAt
	publishTTFT := e.waitingForFirstResponse
	e.waitingForFirstResponse = false
	e.requestStartedAt = time.Time{}
	e.mu.Unlock()

	assistant := output.GetAssistant()
	responseText := strings.Join(assistant.GetContents(), "")
	toolCalls := assistant.GetToolCalls()

	supersededCtx := e.history.AppendAssistant(contextID, output)
	if supersededCtx != "" {
		e.logger.Errorf("new tool block while previous unresolved (context=%s), superseding", supersededCtx)
		communication.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			ContextID: supersededCtx,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "tool block superseded",
				Attributes: observability.Attributes{
					"component":             observability.ComponentTool.String(),
					"operation":             "supersede_tool_block",
					"provider":              providerName,
					"context_id":            supersededCtx,
					"reason":                "new_tool_block",
					"superseded_context_id": supersededCtx,
				},
				OccurredAt: time.Now(),
			},
		})
	}
	if len(toolCalls) > 0 {
		e.toolExecutor.ExecuteAll(ctx, contextID, toolCalls, communication)
	}
	packets := []internal_type.Packet{
		internal_type.LLMResponseDonePacket{ContextID: contextID, Text: responseText},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.NewMessageRecord(contextID, observability.ComponentAgent, observability.AgentCompleted, observability.MessageRoleAssistant, observability.Attributes{
				"provider":            providerName,
				"context_id":          contextID,
				"response_char_count": fmt.Sprintf("%d", len(responseText)),
				"finish_reason":       finishReason,
				"tool_call_count":     fmt.Sprintf("%d", len(toolCalls)),
			}),
		},
	}
	metrics := []*protos.Metric{{
		Name:        observability.MetricAgentResponseCharCount,
		Value:       fmt.Sprintf("%d", len(responseText)),
		Description: "Agent response character count",
	}}
	if !requestStartedAt.IsZero() {
		if publishTTFT {
			metrics = append(metrics, &protos.Metric{
				Name:        observability.MetricAgentTTFTMs,
				Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
				Description: "Agent time to first token in milliseconds",
			})
		}
		metrics = append(metrics, &protos.Metric{
			Name:        observability.MetricAgentTRTMs,
			Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
			Description: "Agent total response time in milliseconds",
		})
	}
	packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
		ContextID: contextID,
		Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
		Record: observability.RecordMetric{
			Attributes: observability.Attributes{"provider": providerName},
			Metrics:    metrics,
		},
	})
	if !requestStartedAt.IsZero() {
		packets = append(packets, internal_type.ObservabilityUsageRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.NewLLMDurationUsageRecord(
				providerName,
				now.Sub(requestStartedAt),
				observability.Attributes{
					"context_id":          contextID,
					"finish_reason":       finishReason,
					"response_char_count": fmt.Sprintf("%d", len(responseText)),
					"tool_call_count":     fmt.Sprintf("%d", len(toolCalls)),
				},
			),
		})
	}
	communication.OnPacket(ctx, packets...)
}

func (e *modelAssistantExecutor) currentContextID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.currentPacket == nil {
		return ""
	}
	return e.currentPacket.ContextID
}

func (e *modelAssistantExecutor) isStaleResponse(requestID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.currentPacket == nil {
		return true
	}
	return requestID != e.currentPacket.ContextId()
}

func (e *modelAssistantExecutor) buildPromptArgs(communication internal_type.Communication, p internal_type.UserInputPacket) map[string]interface{} {
	return utils.MergeMaps(e.buildBasePromptArgs(communication), map[string]interface{}{"message": map[string]interface{}{
		"text": p.Text, "language_code": p.Language.ISO639_1, "language": p.Language.Name,
	}})
}

func (e *modelAssistantExecutor) buildBasePromptArgs(communication internal_type.Communication) map[string]interface{} {
	registry := internal_namespace.NewDefaultRegistry()
	src := variable.NewCommunicationSource(communication)
	out := registry.Expand(src, variable.ResolveContext{})
	out["message"] = map[string]interface{}{"language": "English"}
	return out
}

func (e *modelAssistantExecutor) chatStreamRequest(communication internal_type.Communication, contextID string, promptArgs map[string]interface{}, messages ...*protos.Message) (*protos.StreamChatInput, error) {
	assistant, err := communication.Assistant()
	if err != nil {
		return nil, err
	}
	template := assistant.AssistantProviderModel.Template.GetTextChatCompleteTemplate()
	defaultArgs := parsers.CanonicalizePromptArguments(e.inputBuilder.PromptArguments(template.Variables))
	runtimeArgs := parsers.CanonicalizePromptArguments(promptArgs)
	systemMessages := e.inputBuilder.Message(template.Prompt, utils.MergeMaps(defaultArgs, runtimeArgs))
	src, err := e.buildStreamChatInput(communication, contextID, append(systemMessages, messages...)...)
	if err != nil {
		return nil, err
	}
	return &protos.StreamChatInput{
		RequestId:       src.GetRequestId(),
		ProviderName:    strings.ToLower(assistant.AssistantProviderModel.ModelProviderName),
		Conversations:   src.GetConversations(),
		AdditionalData:  src.GetAdditionalData(),
		ModelParameters: src.GetModelParameters(),
		ToolDefinitions: src.GetToolDefinitions(),
	}, nil
}

func (e *modelAssistantExecutor) buildStreamChatInput(
	communication internal_type.Communication,
	contextID string,
	conversations ...*protos.Message,
) (*protos.StreamChatInput, error) {
	assistant, err := communication.Assistant()
	if err != nil {
		return nil, err
	}
	conversation, err := communication.Conversation()
	if err != nil {
		return nil, err
	}
	modelOptions := make(map[string]interface{}, len(e.providerOptions))
	for key, value := range e.providerOptions {
		if key == modelOptionCredentialID || strings.HasPrefix(key, modelOptionConnectionPrefix) {
			continue
		}
		modelOptions[key] = value
	}

	functionDefinitions := e.toolExecutor.GetFunctionDefinitions()
	toolDefinitions := make([]*protos.ToolDefinition, 0, len(functionDefinitions))
	for _, definition := range functionDefinitions {
		toolDefinitions = append(toolDefinitions, &protos.ToolDefinition{
			Type:               "function",
			FunctionDefinition: definition,
		})
	}

	return &protos.StreamChatInput{
		RequestId:     contextID,
		ProviderName:  strings.ToLower(assistant.AssistantProviderModel.ModelProviderName),
		Conversations: conversations,
		AdditionalData: map[string]string{
			"assistant_id":                fmt.Sprintf("%d", conversation.AssistantId),
			"conversation_id":             fmt.Sprintf("%d", conversation.Id),
			"user_identifier":             fmt.Sprintf("%s", conversation.Identifier),
			"message_id":                  contextID,
			"assistant_provider_model_id": fmt.Sprintf("%d", assistant.AssistantProviderModel.Id),
		},
		ModelParameters: e.inputBuilder.Options(modelOptions, nil),
		ToolDefinitions: toolDefinitions,
	}, nil
}

func (e *modelAssistantExecutor) validateHistorySequence(messages []*protos.Message) error {
	for i, msg := range messages {
		if ast := msg.GetAssistant(); ast != nil && len(ast.GetToolCalls()) > 0 {
			if i+1 >= len(messages) || messages[i+1].GetTool() == nil {
				return fmt.Errorf("history: assistant tool_call at %d not followed by tool response", i)
			}
			if err := e.validateToolIDMatch(ast.GetToolCalls(), messages[i+1].GetTool().GetTools(), i); err != nil {
				return err
			}
		}
		if tool := msg.GetTool(); tool != nil {
			if i == 0 {
				return fmt.Errorf("history: orphan tool response at %d", i)
			}
			prev := messages[i-1].GetAssistant()
			if prev == nil || len(prev.GetToolCalls()) == 0 {
				return fmt.Errorf("history: orphan tool response at %d", i)
			}
		}
	}
	return nil
}

func (e *modelAssistantExecutor) validateToolIDMatch(calls []*protos.ToolCall, tools []*protos.ToolMessage_Tool, idx int) error {
	expected := make(map[string]struct{}, len(calls))
	for _, c := range calls {
		if id := strings.TrimSpace(c.GetId()); id != "" {
			expected[id] = struct{}{}
		}
	}
	for _, t := range tools {
		id := strings.TrimSpace(t.GetId())
		if _, ok := expected[id]; !ok {
			return fmt.Errorf("history: orphan tool result %q at assistant %d", id, idx)
		}
		delete(expected, id)
	}
	for id := range expected {
		return fmt.Errorf("history: missing tool result for %q at assistant %d", id, idx)
	}
	return nil
}
