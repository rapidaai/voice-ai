// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"fmt"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors"
	observability_collector_requestlog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/requestlog"
	observability_collector_toollog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/toollog"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (d *Dispatcher) runOutbound(ctx context.Context, v OutboundRequestedPipeline) *PipelineResult {
	v.Observer.AddCollectors(
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

	assistant, err := d.assistantService.Get(ctx,
		v.Auth,
		v.AssistantID,
		utils.GetVersionDefinition("latest"),
		&internal_services.GetAssistantOption{InjectPhoneDeployment: true})
	if err != nil {
		v.Observer.Record(ctx, observability.AssistantScope{AssistantID: v.AssistantID}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Outbound call pipeline failed to load assistant",
			Attributes: observability.Attributes{
				"assistant_id": fmt.Sprintf("%d", v.AssistantID),
				"to":           v.ToPhone,
				"from":         v.FromPhone,
				"error":        err.Error(),
			},
		}, observability.RecordWebhook{
			Event: observability.CallFailed,
			Payload: observability.CallFailedWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				To:                   v.ToPhone,
				From:                 v.FromPhone,
				Direction:            "outbound",
				Stage:                "assistant_load",
				Error:                err.Error(),
				Status:               observability.MetricCallStatusFailed,
				DisconnectReason:     protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
			},
		}, observability.RecordMetric{
			Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, err.Error()),
		})
		return &PipelineResult{Error: fmt.Errorf("invalid assistant: %w", err)}
	}
	if assistant.AssistantPhoneDeployment == nil {
		v.Observer.Record(ctx,
			observability.AssistantScope{AssistantID: v.AssistantID},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Outbound call pipeline failed because phone deployment is not enabled",
				Attributes: observability.Attributes{
					"assistant_id": fmt.Sprintf("%d", assistant.Id),
					"to":           v.ToPhone,
					"from":         v.FromPhone,
				},
			},
			observability.RecordWebhook{
				Event: observability.CallFailed,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					To:                   v.ToPhone,
					From:                 v.FromPhone,
					Direction:            "outbound",
					Stage:                "phone_deployment",
					Error:                "Please check phone deployment not enabled",
					Status:               observability.MetricCallStatusFailed,
					DisconnectReason:     protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
				},
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, "Please check phone deployment not enabled"),
			})
		return &PipelineResult{Error: fmt.Errorf("phone deployment not enabled")}
	}

	fromPhone := v.FromPhone
	if !validator.NotBlank(fromPhone) {
		fn, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone")
		if err != nil {
			v.Observer.Record(ctx,
				observability.AssistantScope{AssistantID: v.AssistantID},
				observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Outbound call pipeline failed to resolve from phone number",
					Attributes: observability.Attributes{
						"provider":     assistant.AssistantPhoneDeployment.TelephonyProvider,
						"to":           v.ToPhone,
						"from":         v.FromPhone,
						"error":        err.Error(),
						"assistant_id": fmt.Sprintf("%d", assistant.Id),
					},
				},
				observability.RecordWebhook{
					Event: observability.CallFailed,
					Payload: observability.CallFailedWebhookPayload{
						V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
						Provider:             assistant.AssistantPhoneDeployment.TelephonyProvider,
						To:                   v.ToPhone,
						From:                 v.FromPhone,
						Direction:            "outbound",
						Stage:                "from_phone_resolve",
						Error:                err.Error(),
						Status:               observability.MetricCallStatusFailed,
						DisconnectReason:     protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
					},
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, err.Error()),
				})
			return &PipelineResult{Error: fmt.Errorf("no phone number configured: %w", err)}
		}
		fromPhone = fn
	}
	conversation, err := d.conversationService.CreateConversation(ctx, v.Auth, v.ToPhone, assistant.Id, assistant.AssistantProviderId, type_enums.DIRECTION_OUTBOUND, utils.PhoneCall)
	if err != nil {
		v.Observer.Record(ctx,
			observability.AssistantScope{AssistantID: v.AssistantID},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Outbound call pipeline failed to create conversation",
				Attributes: observability.Attributes{
					"assistant_id": fmt.Sprintf("%d", assistant.Id),
					"provider":     assistant.AssistantPhoneDeployment.TelephonyProvider,
					"to":           v.ToPhone,
					"from":         fromPhone,
					"error":        err.Error(),
				},
			},
			observability.RecordWebhook{
				Event: observability.CallFailed,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             assistant.AssistantPhoneDeployment.TelephonyProvider,
					To:                   v.ToPhone,
					From:                 fromPhone,
					Direction:            "outbound",
					Stage:                "conversation_create",
					Error:                err.Error(),
					Status:               observability.MetricCallStatusFailed,
					DisconnectReason:     protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
				},
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusFailed, err.Error()),
			})
		return &PipelineResult{Error: fmt.Errorf("failed to create conversation: %w", err)}
	}

	v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		},
		observability.RecordEvent{
			Event: observability.CallConversationCreated,
			Attributes: observability.Attributes{
				"provider": assistant.AssistantPhoneDeployment.TelephonyProvider,
				"to":       v.ToPhone,
				"from":     fromPhone,
			},
		})

	if len(v.Options) > 0 {
		if _, err := d.conversationService.CreateOrUpdateConversationOption(ctx, v.Auth, assistant.Id, conversation.Id, v.Options); err != nil {
			d.logger.Warnw("Failed to CreateOrUpdate conversation extras", "error", err)
		}
	}
	if len(v.Args) > 0 {
		if _, err := d.conversationService.CreateOrUpdateConversationArgument(ctx, v.Auth, assistant.Id, conversation.Id, v.Args); err != nil {
			d.logger.Warnw("Failed to CreateOrUpdate conversation extras", "error", err)
		}
	}
	if len(v.Metadata) > 0 {
		conversationMetadata := make([]*protos.Metadata, 0, len(v.Metadata))
		for key, value := range v.Metadata {
			conversationMetadata = append(conversationMetadata, &protos.Metadata{Key: key, Value: fmt.Sprintf("%v", value)})
		}
		v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordMetadata{
				Metadata: conversationMetadata,
			})
	}

	//
	contextID, err := d.inboundDispatcher.SaveCallContext(ctx, v.Auth, assistant, conversation.Id,
		&internal_type.CallInfo{
			CallerNumber: v.ToPhone,
			FromNumber:   fromPhone,
			Direction:    "outbound",
			Provider:     assistant.AssistantPhoneDeployment.TelephonyProvider,
			Status:       callcontext.CallStatusNew,
		}, assistant.AssistantPhoneDeployment.TelephonyProvider)
	if err != nil {
		v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Outbound call pipeline failed to save call context",
				Attributes: observability.Attributes{
					"provider":   assistant.AssistantPhoneDeployment.TelephonyProvider,
					"to":         v.ToPhone,
					"from":       fromPhone,
					"context_id": contextID,
					"error":      err.Error(),
				},
			},
			observability.RecordWebhook{
				ContextID: contextID,
				Event:     observability.CallFailed,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             assistant.AssistantPhoneDeployment.TelephonyProvider,
					To:                   v.ToPhone,
					From:                 fromPhone,
					Direction:            "outbound",
					ContextID:            contextID,
					Stage:                "call_context_save",
					Error:                err.Error(),
					Status:               observability.MetricCallStatusFailed,
					DisconnectReason:     protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
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
		return &PipelineResult{Error: fmt.Errorf("failed to save call context: %w", err)}
	}
	v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		},
		observability.RecordEvent{
			Event: observability.CallContextSaved,
			Attributes: observability.Attributes{
				"provider":   assistant.AssistantPhoneDeployment.TelephonyProvider,
				"to":         v.ToPhone,
				"from":       fromPhone,
				"context_id": contextID,
			},
		},
		observability.RecordMetadata{
			Metadata: observability.ClientMetadata(
				v.ToPhone,
				fromPhone,
				"outbound",
				assistant.AssistantPhoneDeployment.TelephonyProvider,
				"", contextID, "", "",
			),
		},
		observability.RecordEvent{
			Event: observability.CallOutboundRequested,
			Attributes: observability.Attributes{
				"provider":   assistant.AssistantPhoneDeployment.TelephonyProvider,
				"to":         v.ToPhone,
				"from":       fromPhone,
				"context_id": contextID,
			},
		},
		observability.RecordWebhook{
			Event:     observability.CallOutboundRequested,
			ContextID: contextID,
			Payload: observability.CallOutboundRequestedWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				Provider:             assistant.AssistantPhoneDeployment.TelephonyProvider,
				To:                   v.ToPhone,
				From:                 fromPhone,
				Direction:            "outbound",
				ContextID:            contextID,
				Status:               observability.MetricCallStatusInProgress,
			},
		})

	callInfo, err := d.outboundDispatcher.Dispatch(ctx, contextID)
	if err != nil {
		v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Outbound call pipeline failed to dispatch provider call",
				Attributes: observability.Attributes{
					"provider":   assistant.AssistantPhoneDeployment.TelephonyProvider,
					"to":         v.ToPhone,
					"from":       fromPhone,
					"context_id": contextID,
					"error":      err.Error(),
				},
			},
			observability.RecordEvent{
				Event: observability.CallOutboundDispatchFailed,
				Attributes: observability.Attributes{
					"provider":   assistant.AssistantPhoneDeployment.TelephonyProvider,
					"context_id": contextID,
					"error":      err.Error(),
				},
			},
			observability.RecordWebhook{
				Event:     observability.CallFailed,
				ContextID: contextID,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             assistant.AssistantPhoneDeployment.TelephonyProvider,
					To:                   v.ToPhone,
					From:                 fromPhone,
					Direction:            "outbound",
					ContextID:            contextID,
					Stage:                "provider_dispatch",
					Error:                err.Error(),
					Status:               observability.MetricCallStatusFailed,
					DisconnectReason:     protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
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
		return &PipelineResult{ContextID: contextID, ConversationID: conversation.Id, Error: err}
	}

	v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		},
		observability.RecordEvent{
			Event: observability.CallOutboundDispatched,
			Attributes: observability.Attributes{
				"provider":          assistant.AssistantPhoneDeployment.TelephonyProvider,
				"to":                v.ToPhone,
				"from":              fromPhone,
				"context_id":        contextID,
				"call_id":           callInfo.ChannelUUID,
				"status_event":      callInfo.StatusInfo.Event.String(),
				"provider_response": observability.AttributeValue(callInfo.StatusInfo.Payload),
			},
		},
		observability.RecordWebhook{
			Event:     observability.CallOutboundDispatched,
			ContextID: contextID,
			Payload: observability.CallOutboundDispatchedWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
					"provider_response": callInfo.StatusInfo.Payload,
				}),
				Provider:    assistant.AssistantPhoneDeployment.TelephonyProvider,
				CallID:      callInfo.ChannelUUID,
				To:          v.ToPhone,
				From:        fromPhone,
				Direction:   "outbound",
				ContextID:   contextID,
				StatusEvent: callInfo.StatusInfo.Event.String(),
				Status:      observability.MetricCallStatusInProgress,
			},
		})

	return &PipelineResult{ContextID: contextID, ConversationID: conversation.Id}
}
