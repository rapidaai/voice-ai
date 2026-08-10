// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentkit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (e *agentkitExecutor) Read(ctx context.Context, comm internal_type.Communication, connection *AgentkitConnection) {
	for {
		if ctx.Err() != nil {
			return
		}
		resp, err := connection.Recv()
		if err != nil {
			switch {
			case ctx.Err() != nil:
				return
			case errors.Is(err, io.EOF):
				return
			case status.Code(err) == codes.Canceled:
				return
			case status.Code(err) == codes.Unavailable:
				comm.OnPacket(ctx, internal_type.LLMErrorPacket{
					ContextID: e.getActiveContextID(),
					Error:     fmt.Errorf("%w: server unavailable", ErrAgentkitConnectionRecv),
					Type:      internal_type.LLMSystemPanic,
				})
			default:
				comm.OnPacket(ctx, internal_type.LLMErrorPacket{
					ContextID: e.getActiveContextID(),
					Error:     fmt.Errorf("%w: %s", ErrAgentkitConnectionRecv, err.Error()),
					Type:      internal_type.LLMSystemPanic,
				})
			}
			return
		}
		e.Write(ctx, comm, resp)
	}
}

func (e *agentkitExecutor) Write(ctx context.Context, comm internal_type.Communication, resp *protos.TalkOutput) {
	switch data := resp.GetData().(type) {
	case *protos.TalkOutput_Initialization:
		comm.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "agentkit initialization acknowledged",
				Attributes: observability.Attributes{
					"component":       observability.ComponentLLM.String(),
					"operation":       "initialization_ack",
					"provider":        e.Name(),
					"conversation_id": fmt.Sprintf("%d", data.Initialization.GetAssistantConversationId()),
				},
				OccurredAt: time.Now(),
			},
		})

	case *protos.TalkOutput_Interruption:
		if !e.isCurrentContext(data.Interruption.GetId()) {
			return
		}
		comm.OnPacket(ctx,
			internal_type.InterruptionDetectedPacket{ContextID: data.Interruption.Id, Source: internal_type.InterruptionSourceWord},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: data.Interruption.Id,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(data.Interruption.Id, observability.ComponentLLM, observability.LLMDiscarded, observability.MessageRoleAssistant, observability.Attributes{
					"provider":   e.Name(),
					"context_id": data.Interruption.Id,
					"reason":     "interruption",
					"source":     "word",
				}),
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: data.Interruption.Id,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "agentkit response interrupted",
					Attributes: observability.Attributes{
						"component":  observability.ComponentLLM.String(),
						"operation":  "interrupt",
						"provider":   e.Name(),
						"context_id": data.Interruption.Id,
						"source":     "word",
					},
					OccurredAt: time.Now(),
				},
			},
		)

	case *protos.TalkOutput_User:
		if !validator.NotBlank(data.User.GetText()) {
			return
		}
		comm.OnPacket(ctx, internal_type.UserInputPacket{
			ContextID: data.User.Id,
			Text:      data.User.GetText(),
		})

	case *protos.TalkOutput_Control:
		e.logger.Debugf("controling packet with %+v", data)
		if data.Control == nil {
			return
		}
		if !e.isCurrentContext(data.Control.GetId()) {
			return
		}
		switch data.Control.GetAction() {
		case protos.ConversationControl_CONTROL_ACTION_BLOCK:
			e.logger.Debugf("controling packet with %+v", data)
			for _, controlType := range data.Control.GetTypes() {
				switch controlType {
				case protos.ConversationControl_CONTROL_TYPE_USER_AUDIO:
					comm.OnPacket(ctx, internal_type.DispatchPolicyPacket{
						ContextID: data.Control.GetId(),
						Policy: internal_type.DispatchPolicy{
							Target: internal_type.PacketNameUserAudioReceived,
							Action: internal_type.DispatchActionIgnore,
						},
					},
						internal_type.DispatchPolicyPacket{
							ContextID: data.Control.GetId(),
							Policy: internal_type.DispatchPolicy{
								Target: internal_type.PacketNameInterruptionDetected,
								Action: internal_type.DispatchActionIgnore,
							},
						})
				case protos.ConversationControl_CONTROL_TYPE_USER_TEXT:
					comm.OnPacket(ctx,
						internal_type.DispatchPolicyPacket{
							ContextID: data.Control.GetId(),
							Policy: internal_type.DispatchPolicy{
								Target: internal_type.PacketNameUserTextReceived,
								Action: internal_type.DispatchActionIgnore,
							},
						},
						internal_type.DispatchPolicyPacket{
							ContextID: data.Control.GetId(),
							Policy: internal_type.DispatchPolicy{
								Target: internal_type.PacketNameInterruptionDetected,
								Action: internal_type.DispatchActionIgnore,
							},
						})
				case protos.ConversationControl_CONTROL_TYPE_BARGE_IN:
					comm.OnPacket(ctx,
						internal_type.DispatchPolicyPacket{
							ContextID: data.Control.GetId(),
							Policy: internal_type.DispatchPolicy{
								Target: internal_type.PacketNameUserAudioReceived,
								Action: internal_type.DispatchActionIgnore,
							},
						},
						internal_type.DispatchPolicyPacket{
							ContextID: data.Control.GetId(),
							Policy: internal_type.DispatchPolicy{
								Target: internal_type.PacketNameUserTextReceived,
								Action: internal_type.DispatchActionIgnore,
							},
						},
						internal_type.DispatchPolicyPacket{
							ContextID: data.Control.GetId(),
							Policy: internal_type.DispatchPolicy{
								Target: internal_type.PacketNameInterruptionDetected,
								Action: internal_type.DispatchActionIgnore,
							},
						})
				}
			}
		case protos.ConversationControl_CONTROL_ACTION_UNBLOCK:
			for _, controlType := range data.Control.GetTypes() {
				switch controlType {
				case protos.ConversationControl_CONTROL_TYPE_USER_AUDIO:
					comm.OnPacket(ctx, internal_type.DispatchPolicyPacket{
						ContextID: data.Control.GetId(),
						Policy: internal_type.DispatchPolicy{
							Target: internal_type.PacketNameUserAudioReceived,
							Action: internal_type.DispatchActionPassthrough,
						},
					})
				case protos.ConversationControl_CONTROL_TYPE_USER_TEXT:
					comm.OnPacket(ctx, internal_type.DispatchPolicyPacket{
						ContextID: data.Control.GetId(),
						Policy: internal_type.DispatchPolicy{
							Target: internal_type.PacketNameUserTextReceived,
							Action: internal_type.DispatchActionPassthrough,
						},
					})
				case protos.ConversationControl_CONTROL_TYPE_BARGE_IN:
					comm.OnPacket(ctx, internal_type.DispatchPolicyPacket{
						ContextID: data.Control.GetId(),
						Policy: internal_type.DispatchPolicy{
							Target: internal_type.PacketNameInterruptionDetected,
							Action: internal_type.DispatchActionPassthrough,
						},
					})
				}
			}
		}

	case *protos.TalkOutput_Assistant:
		if !e.isCurrentContext(data.Assistant.GetId()) {
			return
		}

		switch msg := data.Assistant.GetMessage().(type) {
		case *protos.ConversationAssistantMessage_Text:
			now := time.Now()
			if data.Assistant.GetCompleted() {
				e.stateMu.Lock()
				requestStartedAt := e.requestStartedAt
				publishTTFT := e.waitingForFirstResponse
				e.waitingForFirstResponse = false
				e.requestStartedAt = time.Time{}
				e.stateMu.Unlock()

				metrics := []*protos.Metric{{
					Name:        "llm_response_char_count",
					Value:       fmt.Sprintf("%d", len(msg.Text)),
					Description: "AgentKit response character count",
				}}
				if !requestStartedAt.IsZero() {
					if publishTTFT {
						metrics = append(metrics, &protos.Metric{
							Name:        observability.MetricAgentTTFTMs,
							Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
							Description: "AgentKit time to first token in milliseconds",
						})
					}
					metrics = append(metrics, &protos.Metric{
						Name:        observability.MetricAgentTRTMs,
						Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
						Description: "AgentKit total response time in milliseconds",
					})
				}

				packets := []internal_type.Packet{
					internal_type.LLMResponseDonePacket{ContextID: data.Assistant.GetId(), Text: msg.Text},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: data.Assistant.GetId(),
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.NewMessageRecord(data.Assistant.GetId(), observability.ComponentLLM, observability.LLMCompleted, observability.MessageRoleAssistant, observability.Attributes{
							"provider":            e.Name(),
							"context_id":          data.Assistant.GetId(),
							"response_char_count": fmt.Sprintf("%d", len(msg.Text)),
						}),
					},
					internal_type.ObservabilityMetricRecordPacket{
						ContextID: data.Assistant.GetId(),
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordMetric{
							Attributes: observability.Attributes{"provider": e.Name()},
							Metrics:    metrics,
						},
					},
				}
				if !requestStartedAt.IsZero() {
					packets = append(packets, internal_type.ObservabilityUsageRecordPacket{
						ContextID: data.Assistant.GetId(),
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.NewLLMDurationUsageRecord(e.Name(), now.Sub(requestStartedAt), observability.Attributes{
							"context_id":          data.Assistant.GetId(),
							"response_char_count": fmt.Sprintf("%d", len(msg.Text)),
						}),
					})
				}
				comm.OnPacket(ctx, packets...)
			} else {
				e.stateMu.Lock()
				requestStartedAt := e.requestStartedAt
				publishTTFT := e.waitingForFirstResponse
				e.waitingForFirstResponse = false
				e.stateMu.Unlock()

				if publishTTFT && !requestStartedAt.IsZero() {
					comm.OnPacket(ctx,
						internal_type.LLMResponseDeltaPacket{ContextID: data.Assistant.GetId(), Text: msg.Text},
						internal_type.ObservabilityMetricRecordPacket{
							ContextID: data.Assistant.GetId(),
							Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
							Record: observability.RecordMetric{
								Attributes: observability.Attributes{"provider": e.Name()},
								Metrics: []*protos.Metric{{
									Name:        observability.MetricAgentTTFTMs,
									Value:       fmt.Sprintf("%d", now.Sub(requestStartedAt).Milliseconds()),
									Description: "AgentKit time to first token in milliseconds",
								}},
							},
						},
					)
				} else {
					comm.OnPacket(ctx, internal_type.LLMResponseDeltaPacket{ContextID: data.Assistant.GetId(), Text: msg.Text})
				}
			}
		}

	case *protos.TalkOutput_ToolCall:
		if !e.isCurrentContext(data.ToolCall.GetId()) {
			return
		}
		toolID := data.ToolCall.GetToolId()
		if !validator.NotBlank(toolID) {
			toolID = AgentkitToolIDPrefix + uuid.NewString()
		}
		comm.OnPacket(ctx,
			internal_type.LLMToolCallPacket{
				ContextID: data.ToolCall.GetId(),
				ToolID:    toolID,
				Name:      data.ToolCall.GetName(),
				Action:    data.ToolCall.GetAction(),
				Arguments: data.ToolCall.GetArgs(),
			})

	case *protos.TalkOutput_ToolCallResult:
		if !e.isCurrentContext(data.ToolCallResult.GetId()) {
			return
		}
		comm.OnPacket(ctx,
			internal_type.LLMToolResultPacket{
				ToolID:    data.ToolCallResult.GetToolId(),
				Name:      data.ToolCallResult.GetName(),
				ContextID: data.ToolCallResult.GetId(),
				Action:    data.ToolCallResult.GetAction(),
				Result:    data.ToolCallResult.GetResult(),
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: data.ToolCallResult.GetId(),
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelDebug,
					Message: "agentkit tool result received",
					Attributes: observability.Attributes{
						"component":  observability.ComponentTool.String(),
						"operation":  "tool_result",
						"provider":   e.Name(),
						"context_id": data.ToolCallResult.GetId(),
						"tool_id":    data.ToolCallResult.GetToolId(),
						"name":       data.ToolCallResult.GetName(),
						"action":     data.ToolCallResult.GetAction().String(),
					},
					OccurredAt: time.Now(),
				},
			})

	case *protos.TalkOutput_Error:
		comm.OnPacket(ctx,
			internal_type.LLMErrorPacket{
				ContextID: e.getActiveContextID(),
				Error:     fmt.Errorf("%w %d: %s", ErrAgentkitResponse, data.Error.GetErrorCode(), data.Error.GetErrorMessage()),
				Type:      internal_type.LLMSystemPanic,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: e.getActiveContextID(),
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.NewMessageRecord(e.getActiveContextID(), observability.ComponentLLM, observability.LLMError, observability.MessageRoleAssistant, observability.Attributes{
					"provider":   e.Name(),
					"context_id": e.getActiveContextID(),
					"error":      data.Error.GetErrorMessage(),
					"code":       fmt.Sprintf("%d", data.Error.GetErrorCode()),
				}),
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: e.getActiveContextID(),
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "agentkit response failed",
					Attributes: observability.Attributes{
						"component":  observability.ComponentLLM.String(),
						"operation":  "response",
						"provider":   e.Name(),
						"context_id": e.getActiveContextID(),
						"error":      data.Error.GetErrorMessage(),
						"code":       fmt.Sprintf("%d", data.Error.GetErrorCode()),
					},
					OccurredAt: time.Now(),
				},
			},
			internal_type.LLMToolCallPacket{
				ContextID: e.getActiveContextID(),
				Name:      "end_conversation",
				Action:    protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
				Arguments: map[string]string{"reason": data.Error.GetErrorMessage()},
			},
		)

	case *protos.TalkOutput_Observability:
		e.logger.Debugf("agentkit observability record received: %v", data.Observability)
		switch record := data.Observability.GetRecord().(type) {
		case *protos.ObservabilityRecord_Log:
			occurredAt := time.Now()
			if record.Log.GetOccurredAt() != nil {
				occurredAt = record.Log.GetOccurredAt().AsTime()
			}
			comm.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordLog{
					ID:         record.Log.GetId(),
					Message:    record.Log.GetMessage(),
					Level:      observability.Level(record.Log.GetLevel()),
					Attributes: observability.Attributes(record.Log.GetAttributes()),
					OccurredAt: occurredAt,
				},
			})
		case *protos.ObservabilityRecord_Event:
			occurredAt := time.Now()
			if record.Event.GetOccurredAt() != nil {
				occurredAt = record.Event.GetOccurredAt().AsTime()
			}
			event := record.Event.GetEvent()
			if !strings.HasPrefix(event, AgentkitObservabilityPrefix) {
				event = AgentkitObservabilityPrefix + event
			}
			component := record.Event.GetComponent()
			if !strings.HasPrefix(component, AgentkitObservabilityPrefix) {
				component = AgentkitObservabilityPrefix + component
			}
			comm.OnPacket(ctx, internal_type.ObservabilityEventRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					ID:         record.Event.GetId(),
					Component:  observability.ComponentName(component),
					Event:      observability.EventName(event),
					Attributes: observability.Attributes(record.Event.GetAttributes()),
					OccurredAt: occurredAt,
				},
			})
		case *protos.ObservabilityRecord_Metric:
			occurredAt := time.Now()
			if record.Metric.GetOccurredAt() != nil {
				occurredAt = record.Metric.GetOccurredAt().AsTime()
			}
			name := record.Metric.GetName()
			if !strings.HasPrefix(name, AgentkitObservabilityPrefix) {
				name = AgentkitObservabilityPrefix + name
			}
			comm.OnPacket(ctx, internal_type.ObservabilityMetricRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordMetric{
					ID: record.Metric.GetId(),
					Metrics: []*protos.Metric{{
						Name:        name,
						Value:       record.Metric.GetValue(),
						Description: record.Metric.GetDescription(),
					}},
					Attributes: observability.Attributes(record.Metric.GetAttributes()),
					OccurredAt: occurredAt,
				},
			})
		}
	}
}

func (e *agentkitExecutor) isCurrentContext(id string) bool {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	if !validator.NotBlank(e.activeContextID) {
		return true
	}
	if !validator.NotBlank(id) {
		return true
	}
	return id == e.activeContextID
}

func (e *agentkitExecutor) getActiveContextID() string {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.activeContextID
}
