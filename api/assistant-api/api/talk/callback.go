// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (cApi *ConversationApi) UnviersalCallback(c *gin.Context) {
	provider := c.Param("telephony")
	if !validator.NotBlank(provider) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing telephony provider"})
		return
	}
	assistantID, err := utils.StringToUint64(c.Param("assistantId"))
	if err != nil || !validator.NonZero(assistantID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistantId"})
		return
	}

	statusInfo, err := cApi.inboundDispatcher.HandleCatchAllStatusCallback(c, provider)
	if err != nil {
		cApi.logger.Errorf("catch-all status callback failed for provider %s: %v", provider, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}
	if statusInfo == nil {
		c.Status(http.StatusCreated)
		return
	}

	cc, err := cApi.callContextStore.GetByChannelUUID(c, provider, assistantID, statusInfo.ChannelUUID)
	if err != nil {
		cApi.logger.Errorf("failed to resolve call context for provider %s assistant %d uuid %s: %v", provider, assistantID, statusInfo.ChannelUUID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}

	auth, err := cc.ToAuthentication()
	if err != nil {
		cApi.logger.Error("Failed to reconstruct call authentication")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}
	observer := cApi.Observability(c, auth, observability.WithGracePeriod())
	if err := observer.AddCollectors(collectors.NewWithWebhookConfiguration(c, cApi.logger, auth, cc.AssistantID, cApi.configurationService, cApi.httpLogService)); err != nil {
		cApi.logger.Warnw("observability collector registration failed",
			"component", "callback",
			"operation", "add_assistant_collectors",
			"assistant_id", cc.AssistantID,
			"context_id", cc.ContextID,
			"error", err,
		)
	}

	observer.Record(c,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
			ConversationID: cc.ConversationID,
		},
		observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "telephony provider callback received",
			Attributes: observability.Attributes{
				"provider":     cc.Provider,
				"status_event": statusInfo.Event.String(),
				"context_id":   cc.ContextID,
				"direction":    cc.Direction,
				"channel_uuid": statusInfo.ChannelUUID,
				"raw_payload":  statusInfo.RawPayload,
			},
		},
		observability.RecordEvent{
			Event: observability.CallStatus,
			Attributes: observability.Attributes{
				"provider":     cc.Provider,
				"status_event": statusInfo.Event.String(),
				"context_id":   cc.ContextID,
				"direction":    cc.Direction,
				"channel_uuid": statusInfo.ChannelUUID,
			},
		})
	if statusInfo.Error != nil {
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordWebhook{
				Event:     observability.CallFailed,
				ContextID: cc.ContextID,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
						"raw_payload": statusInfo.RawPayload,
						"payload":     statusInfo.Payload,
					}),
					Provider:         cc.Provider,
					CallID:           statusInfo.ChannelUUID,
					To:               cc.CallerNumber,
					From:             cc.FromNumber,
					Direction:        observability.WebhookCallDirection(cc.Direction),
					ContextID:        cc.ContextID,
					Source:           "provider_callback",
					Error:            statusInfo.Error.Error,
					Status:           observability.WebhookCallStatusFailed,
					DisconnectReason: observability.WebhookCallDisconnectReasonProviderFailed,
				},
			})
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus:       callcontext.CallStatusFailed,
			CallError:        statusInfo.Error.Error,
			FailureClass:     "provider_response",
			FailureReason:    statusInfo.Error.Reason,
			DisconnectReason: statusInfo.Error.Reason,
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from failed callback: %v", cc.ContextID, err)
		}
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordMetric{
				Metrics: append(observability.CallStatusMetric(observability.MetricCallStatusFailed, statusInfo.Error.Reason), &protos.Metric{
					Name:        observability.MetricConversationStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: statusInfo.Error.Reason,
				}),
			})
		if validator.NotBlank(statusInfo.Error.Reason) {
			observer.Record(c,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
					ConversationID: cc.ConversationID,
				},
				observability.RecordMetadata{
					Metadata: observability.DisconnectMetadata(protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(), statusInfo.RawPayload),
				})
		}
	} else if statusInfo.Completed {
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus:       callcontext.CallStatusCompleted,
			DisconnectReason: statusInfo.Event.String(),
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from completed callback: %v", cc.ContextID, err)
		}
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusComplete, statusInfo.Event.String()),
			})
	} else if statusInfo.Event != "" {
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus: statusInfo.Event.String(),
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from callback event %s: %v", cc.ContextID, statusInfo.Event, err)
		}
		switch statusInfo.Event {
		case internal_type.TelephonyEventRinging:
			observer.Record(c,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
					ConversationID: cc.ConversationID,
				},
				observability.RecordWebhook{
					Event:     observability.CallRinging,
					ContextID: cc.ContextID,
					Payload: observability.CallRingingWebhookPayload{
						V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
							"raw_payload": statusInfo.RawPayload,
							"payload":     statusInfo.Payload,
						}),
						Provider:  cc.Provider,
						CallID:    statusInfo.ChannelUUID,
						To:        cc.CallerNumber,
						From:      cc.FromNumber,
						Direction: observability.WebhookCallDirection(cc.Direction),
						ContextID: cc.ContextID,
						Source:    "provider_callback",
						Status:    observability.WebhookCallStatusRinging,
					},
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusRinging, statusInfo.Event.String()),
				})
		case internal_type.TelephonyEvent("cancelled"), internal_type.TelephonyEvent("canceled"):
			observer.Record(c,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
					ConversationID: cc.ConversationID,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusCancelled, statusInfo.Event.String()),
				})
		default:
			observer.Record(c,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
					ConversationID: cc.ConversationID,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusInProgress, statusInfo.Event.String()),
				})
		}
	}
	if validator.NonNil(statusInfo.Duration) {
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordMetric{Metrics: []*protos.Metric{
				{Name: observability.MetricCallDurationMs, Value: strconv.FormatInt(statusInfo.Duration.Milliseconds(), 10), Description: "Call duration in milliseconds"},
			}},
		)
	}
	if validator.NotBlank(statusInfo.Price) {
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordMetric{Metrics: []*protos.Metric{
				{Name: observability.MetricCallPrice, Value: statusInfo.Price, Description: "Call price"},
			}},
		)
	}

	observer.Close(context.Background())
	c.Status(http.StatusCreated)
}

// CallbackByContext handles status callback webhooks using a contextId stored in Postgres.
func (cApi *ConversationApi) CallbackByContext(c *gin.Context) {
	contextID := c.Param("contextId")
	if !validator.NotBlank(contextID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing contextId"})
		return
	}

	cc, err := cApi.callContextStore.Get(c, contextID)
	if err != nil {
		cApi.logger.Errorw("failed to resolve call context for event callback", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}

	auth, err := cc.ToAuthentication()
	if err != nil {
		cApi.logger.Error("Failed to reconstruct call authentication")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}
	statusInfo, err := cApi.inboundDispatcher.HandleStatusCallback(c, cc.Provider, auth, cc.AssistantID, cc.ConversationID)
	if err != nil {
		cApi.logger.Errorw("status callback failed", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}

	if !validator.NonNil(statusInfo) {
		c.Status(http.StatusBadRequest)
		return
	}

	observer := cApi.Observability(c, auth, observability.WithGracePeriod())
	if err := observer.AddCollectors(collectors.NewWithWebhookConfiguration(c, cApi.logger, auth, cc.AssistantID, cApi.configurationService, cApi.httpLogService)); err != nil {
		cApi.logger.Warnw("observability collector registration failed",
			"component", "callback",
			"operation", "add_assistant_collectors",
			"assistant_id", cc.AssistantID,
			"context_id", cc.ContextID,
			"error", err,
		)
	}

	observer.Record(c,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
			ConversationID: cc.ConversationID,
		},
		observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "telephony provider callback received",
			Attributes: observability.Attributes{
				"provider":     cc.Provider,
				"status_event": statusInfo.Event.String(),
				"context_id":   contextID,
				"direction":    cc.Direction,
				"channel_uuid": statusInfo.ChannelUUID,
				"raw_payload":  statusInfo.RawPayload,
			},
		},
		observability.RecordEvent{
			Event: observability.CallStatus,
			Attributes: observability.Attributes{
				"provider":     cc.Provider,
				"status_event": statusInfo.Event.String(),
				"context_id":   contextID,
				"direction":    cc.Direction,
				"channel_uuid": statusInfo.ChannelUUID,
			},
		})
	if statusInfo.Error != nil {
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordWebhook{
				Event:     observability.CallFailed,
				ContextID: cc.ContextID,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
						"raw_payload": statusInfo.RawPayload,
						"payload":     statusInfo.Payload,
					}),
					Provider:         cc.Provider,
					CallID:           statusInfo.ChannelUUID,
					To:               cc.CallerNumber,
					From:             cc.FromNumber,
					Direction:        observability.WebhookCallDirection(cc.Direction),
					ContextID:        cc.ContextID,
					Source:           "provider_callback",
					Error:            statusInfo.Error.Error,
					Status:           observability.WebhookCallStatusFailed,
					DisconnectReason: observability.WebhookCallDisconnectReasonProviderFailed,
				},
			})
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus:       callcontext.CallStatusFailed,
			CallError:        statusInfo.Error.Error,
			FailureClass:     "provider_response",
			FailureReason:    statusInfo.Error.Reason,
			DisconnectReason: statusInfo.Error.Reason,
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from failed callback: %v", cc.ContextID, err)
		}
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordMetric{
				Metrics: append(observability.CallStatusMetric(observability.MetricCallStatusFailed, statusInfo.Error.Reason), &protos.Metric{
					Name:        observability.MetricConversationStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: statusInfo.Error.Reason,
				}),
			})
		if validator.NotBlank(statusInfo.Error.Reason) {
			observer.Record(c, observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
				observability.RecordMetadata{
					Metadata: observability.DisconnectMetadata(protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(), statusInfo.RawPayload),
				})
		}
	} else if statusInfo.Completed {
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus:       callcontext.CallStatusCompleted,
			DisconnectReason: statusInfo.Event.String(),
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from completed callback: %v", cc.ContextID, err)
		}
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordMetric{
				Metrics: observability.CallStatusMetric(observability.MetricCallStatusComplete, statusInfo.Event.String()),
			})
	} else if statusInfo.Event != "" {
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus: statusInfo.Event.String(),
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from callback event %s: %v", cc.ContextID, statusInfo.Event, err)
		}
		switch statusInfo.Event {
		case internal_type.TelephonyEventRinging:
			observer.Record(c,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
					ConversationID: cc.ConversationID,
				},
				observability.RecordWebhook{
					Event:     observability.CallRinging,
					ContextID: cc.ContextID,
					Payload: observability.CallRingingWebhookPayload{
						V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
							"raw_payload": statusInfo.RawPayload,
							"payload":     statusInfo.Payload,
						}),
						Provider:  cc.Provider,
						CallID:    statusInfo.ChannelUUID,
						To:        cc.CallerNumber,
						From:      cc.FromNumber,
						Direction: observability.WebhookCallDirection(cc.Direction),
						ContextID: cc.ContextID,
						Source:    "provider_callback",
						Status:    observability.WebhookCallStatusRinging,
					},
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusRinging, statusInfo.Event.String()),
				})
		case internal_type.TelephonyEvent("cancelled"), internal_type.TelephonyEvent("canceled"):
			observer.Record(c,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
					ConversationID: cc.ConversationID,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusCancelled, statusInfo.Event.String()),
				})
		default:
			observer.Record(c,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
					ConversationID: cc.ConversationID,
				},
				observability.RecordMetric{
					Metrics: observability.CallStatusMetric(observability.MetricCallStatusInProgress, statusInfo.Event.String()),
				})
		}
	}

	if validator.NonNil(statusInfo.Duration) {
		observer.Record(c, observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
			ConversationID: cc.ConversationID,
		},
			observability.RecordMetric{Metrics: []*protos.Metric{{Name: observability.MetricCallDurationMs, Value: strconv.FormatInt(statusInfo.Duration.Milliseconds(), 10), Description: "Call duration in milliseconds"}}})
	}
	if validator.NotBlank(statusInfo.Price) {
		observer.Record(c,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
				ConversationID: cc.ConversationID,
			},
			observability.RecordMetric{Metrics: []*protos.Metric{{Name: observability.MetricCallPrice, Value: statusInfo.Price, Description: "Call price"}}})
	}

	if err := observer.Close(context.Background()); err != nil {
		cApi.logger.Warnf("failed to close callback observability recorder: %v", err)
	}
	c.Status(http.StatusCreated)
}
