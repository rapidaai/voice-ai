// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors"
	observability_collector_requestlog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/requestlog"
	observability_collector_toollog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/toollog"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (d *Dispatcher) runInboundCall(ctx context.Context, v CallReceivedPipeline) *PipelineResult {
	v.Observer.
		AddCollectors(
			observability_collector_requestlog.New(observability_collector_requestlog.Config{
				Logger:         d.logger,
				HTTPLogService: d.httpLogService,
			}),
			observability_collector_toollog.New(observability_collector_toollog.Config{
				Logger:      d.logger,
				ToolService: d.assistantToolService,
			}),
			collectors.NewWithWebhookConfiguration(ctx, d.logger, v.Auth, v.AssistantID, d.configurationService, d.httpLogService),
		)

	callInfo, err := d.inboundDispatcher.ReceiveCall(v.GinContext, v.Provider)
	if err != nil {
		_ = v.Observer.Record(ctx,
			observability.AssistantScope{AssistantID: v.AssistantID},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "inbound call receive failed",
				Attributes: observability.Attributes{
					"provider": v.Provider,
					"error":    err.Error(),
				},
			})
		return &PipelineResult{Error: err}
	}
	if callInfo == nil {
		_ = v.Observer.Record(ctx,
			observability.AssistantScope{AssistantID: v.AssistantID},
			observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "inbound call ignored",
				Attributes: observability.Attributes{
					"provider": v.Provider,
				},
			})
		return &PipelineResult{}
	}
	_ = v.Observer.Record(ctx,
		observability.AssistantScope{AssistantID: v.AssistantID},
		observability.RecordEvent{
			Event: observability.CallReceived,
			Attributes: observability.Attributes{
				"provider": v.Provider,
				"to":       callInfo.CallerNumber,
				"from":     callInfo.FromNumber,
				"call_id":  callInfo.ChannelUUID,
			},
		},
		observability.RecordWebhook{
			Event: observability.CallReceived,
			Payload: observability.CallReceivedWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				Provider:             v.Provider,
				CallID:               callInfo.ChannelUUID,
				To:                   callInfo.CallerNumber,
				From:                 callInfo.FromNumber,
				Direction:            observability.WebhookCallDirectionInbound,
				Status:               observability.WebhookCallStatusInProgress,
			},
		})

	assistant, err := d.assistantService.Get(ctx, v.Auth, v.AssistantID, utils.GetVersionDefinition("latest"), &internal_services.GetAssistantOption{InjectPhoneDeployment: true})
	if err != nil {
		_ = v.Observer.Record(ctx,
			observability.AssistantScope{AssistantID: v.AssistantID},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "inbound assistant load failed",
				Attributes: observability.Attributes{
					"provider": v.Provider,
					"to":       callInfo.CallerNumber,
					"from":     callInfo.FromNumber,
					"call_id":  callInfo.ChannelUUID,
				},
			}, observability.RecordWebhook{
				Event: observability.CallFailed,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             v.Provider,
					CallID:               callInfo.ChannelUUID,
					To:                   callInfo.CallerNumber,
					From:                 callInfo.FromNumber,
					Direction:            observability.WebhookCallDirectionInbound,
					Stage:                "assistant_load",
					Error:                err.Error(),
					Status:               observability.WebhookCallStatusFailed,
					DisconnectReason:     observability.WebhookCallDisconnectReasonInternalError,
				},
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, err.Error()),
			})
		return &PipelineResult{Error: err}
	}

	_ = v.Observer.Record(ctx,
		observability.AssistantScope{AssistantID: assistant.Id},
		observability.RecordEvent{
			Event: observability.CallAssistantLoaded,
			Attributes: observability.Attributes{
				"provider": v.Provider,
				"caller":   callInfo.CallerNumber,
				"call_id":  callInfo.ChannelUUID,
			},
		})

	conversation, err := d.conversationService.CreateConversation(ctx, v.Auth, callInfo.CallerNumber, assistant.Id, assistant.AssistantProviderId, type_enums.DIRECTION_INBOUND, utils.PhoneCall)
	if err != nil {
		_ = v.Observer.Record(ctx,
			observability.AssistantScope{AssistantID: v.AssistantID},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "inbound conversation create failed",
				Attributes: observability.Attributes{
					"provider": v.Provider,
					"to":       callInfo.CallerNumber,
					"from":     callInfo.FromNumber,
					"call_id":  callInfo.ChannelUUID,
				},
			},
			observability.RecordWebhook{
				Event: observability.CallFailed,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             v.Provider,
					CallID:               callInfo.ChannelUUID,
					To:                   callInfo.CallerNumber,
					From:                 callInfo.FromNumber,
					Direction:            observability.WebhookCallDirectionInbound,
					Stage:                "conversation_create",
					Error:                err.Error(),
					Status:               observability.WebhookCallStatusFailed,
					DisconnectReason:     observability.WebhookCallDisconnectReasonInternalError,
				},
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, err.Error()),
			})
		return &PipelineResult{Error: err}
	}

	_ = v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		},
		observability.RecordEvent{
			Event: observability.CallConversationCreated,
			Attributes: observability.Attributes{
				"provider": v.Provider,
				"to":       callInfo.CallerNumber,
				"from":     callInfo.FromNumber,
				"call_id":  callInfo.ChannelUUID,
			},
		})

	contextID, err := d.inboundDispatcher.SaveCallContext(ctx, v.Auth, assistant, conversation.Id, callInfo, v.Provider)
	if err != nil {
		_ = v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "inbound call context save failed",
				Attributes: observability.Attributes{
					"provider":   v.Provider,
					"to":         callInfo.CallerNumber,
					"from":       callInfo.FromNumber,
					"call_id":    callInfo.ChannelUUID,
					"context_id": contextID,
				},
			},
			observability.RecordWebhook{
				Event:     observability.CallFailed,
				ContextID: contextID,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             v.Provider,
					CallID:               callInfo.ChannelUUID,
					To:                   callInfo.CallerNumber,
					From:                 callInfo.FromNumber,
					Direction:            observability.WebhookCallDirectionInbound,
					ContextID:            contextID,
					Stage:                "call_context_save",
					Error:                err.Error(),
					Status:               observability.WebhookCallStatusFailed,
					DisconnectReason:     observability.WebhookCallDisconnectReasonInternalError,
				},
			},
			observability.RecordMetadata{
				Metadata: observability.DisconnectMetadata(
					protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
					err.Error(),
				),
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, err.Error()),
			})
		return &PipelineResult{Error: err}
	}
	_ = v.Observer.Record(ctx, observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
		ConversationID: conversation.Id,
	}, observability.RecordEvent{
		Event: observability.CallContextSaved,
		Attributes: observability.Attributes{
			"provider":   v.Provider,
			"to":         callInfo.CallerNumber,
			"from":       callInfo.FromNumber,
			"call_id":    callInfo.ChannelUUID,
			"context_id": contextID,
		},
	})

	if len(callInfo.Extra) > 0 {
		metadata := make([]*protos.Metadata, 0, len(callInfo.Extra))
		for key, value := range callInfo.Extra {
			metadata = append(metadata, &protos.Metadata{Key: key, Value: value})
		}
		_ = v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordMetadata{
				Metadata: metadata,
			})
	}

	if callInfo.StatusInfo.Event != "" {
		_ = v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordEvent{
				Event: observability.CallStatus,
				Attributes: observability.Attributes{
					"provider":     v.Provider,
					"to":           callInfo.CallerNumber,
					"from":         callInfo.FromNumber,
					"call_id":      callInfo.ChannelUUID,
					"context_id":   contextID,
					"status_event": callInfo.StatusInfo.Event.String(),
				},
			})
		switch callInfo.StatusInfo.Event {
		case internal_type.TelephonyEventRinging:
			_ = v.Observer.Record(ctx,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
					ConversationID: conversation.Id,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusRinging, callInfo.StatusInfo.Event.String()),
				})
		case internal_type.TelephonyEvent("cancelled"), internal_type.TelephonyEvent("canceled"):
			_ = v.Observer.Record(ctx,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
					ConversationID: conversation.Id,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusCancelled, callInfo.StatusInfo.Event.String()),
				})
		case internal_type.TelephonyEvent("failed"):
			_ = v.Observer.Record(ctx,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
					ConversationID: conversation.Id,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, callInfo.StatusInfo.Event.String()),
				})
		case internal_type.TelephonyEventCompleted, internal_type.TelephonyEvent("complete"):
			_ = v.Observer.Record(ctx,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
					ConversationID: conversation.Id,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusComplete, callInfo.StatusInfo.Event.String()),
				})
		default:
			_ = v.Observer.Record(ctx,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
					ConversationID: conversation.Id,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusInProgress, callInfo.StatusInfo.Event.String()),
				})
		}
	}

	v.GinContext.Set("contextId", contextID)
	if err := d.inboundDispatcher.AnswerProvider(v.GinContext, v.Auth, v.Provider, v.AssistantID, callInfo.CallerNumber, conversation.Id); err != nil {
		_ = v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "inbound provider answer failed",
				Attributes: observability.Attributes{
					"provider":   v.Provider,
					"to":         callInfo.CallerNumber,
					"from":       callInfo.FromNumber,
					"call_id":    callInfo.ChannelUUID,
					"context_id": contextID,
					"error":      err.Error(),
				},
			},
			observability.RecordWebhook{
				Event:     observability.CallFailed,
				ContextID: contextID,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             v.Provider,
					CallID:               callInfo.ChannelUUID,
					To:                   callInfo.CallerNumber,
					From:                 callInfo.FromNumber,
					Direction:            observability.WebhookCallDirectionInbound,
					ContextID:            contextID,
					Stage:                "provider_answer",
					Error:                err.Error(),
					Status:               observability.WebhookCallStatusFailed,
					DisconnectReason:     observability.WebhookCallDisconnectReasonInternalError,
				},
			},
			observability.RecordMetadata{
				Metadata: observability.DisconnectMetadata(
					protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
					err.Error(),
				),
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, err.Error()),
			})
		return &PipelineResult{Error: err}
	}
	_ = v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		},
		observability.RecordEvent{
			Event: observability.CallProviderAnswered,
			Attributes: observability.Attributes{
				"provider":   v.Provider,
				"to":         callInfo.CallerNumber,
				"from":       callInfo.FromNumber,
				"call_id":    callInfo.ChannelUUID,
				"context_id": contextID,
			},
		},
		observability.RecordWebhook{
			Event:     observability.CallProviderAnswered,
			ContextID: contextID,
			Payload: observability.CallProviderAnsweredWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				Provider:             v.Provider,
				CallID:               callInfo.ChannelUUID,
				To:                   callInfo.CallerNumber,
				From:                 callInfo.FromNumber,
				Direction:            observability.WebhookCallDirectionInbound,
				ContextID:            contextID,
				Status:               observability.WebhookCallStatusInProgress,
			},
		})

	return &PipelineResult{ContextID: contextID, ConversationID: conversation.Id}
}
