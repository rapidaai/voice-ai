// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors"
	observability_collector_requestlog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/requestlog"
	observability_collector_toollog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/toollog"
	"github.com/rapidaai/protos"
)

// runSession handles telephony media setup and keeps the Talk lifecycle synchronous.
func (d *Dispatcher) runSession(ctx context.Context, v SessionConnectedPipeline) (result *PipelineResult) {
	startTime := time.Now()
	contextID := v.ContextID
	if contextID == "" {
		contextID = v.ID
	}
	v.Observer.AddCollectors(
		observability_collector_requestlog.New(observability_collector_requestlog.Config{
			Logger:         d.logger,
			HTTPLogService: d.httpLogService,
		}),
		observability_collector_toollog.New(observability_collector_toollog.Config{
			Logger:      d.logger,
			ToolService: d.assistantToolService,
		}),
		collectors.NewWithWebhookConfiguration(ctx, d.logger, v.CallContext.ToAuth(), v.CallContext.AssistantID, d.configurationService, d.httpLogService),
	)

	v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.CallContext.AssistantID},
			ConversationID: v.CallContext.ConversationID,
		},
		observability.RecordMetadata{
			Metadata: observability.ClientMetadata(
				v.CallContext.CallerNumber, v.CallContext.FromNumber, v.CallContext.Direction, v.CallContext.Provider,
				v.CallContext.ChannelUUID, contextID, "", "", // codec/sampleRate set by streamer
			),
		},
		observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallStarted,
			Attributes: observability.Attributes{
				"provider":   v.CallContext.Provider,
				"to":         v.CallContext.CallerNumber,
				"from":       v.CallContext.FromNumber,
				"context_id": contextID,
				"direction":  v.CallContext.Direction,
			},
		},
		observability.RecordWebhook{
			Event:     observability.CallStarted,
			ContextID: contextID,
			Payload: observability.CallStartedWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				Provider:             v.CallContext.Provider,
				CallID:               v.CallContext.ChannelUUID,
				To:                   v.CallContext.CallerNumber,
				From:                 v.CallContext.FromNumber,
				Direction:            v.CallContext.Direction,
				ContextID:            contextID,
			},
		},
		observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Pipeline session connected",
			Attributes: observability.Attributes{
				"provider":   v.CallContext.Provider,
				"direction":  v.CallContext.Direction,
				"to":         v.CallContext.CallerNumber,
				"from":       v.CallContext.FromNumber,
				"context_id": contextID,
				"call_id":    v.CallContext.ChannelUUID,
			},
		})

	// recovering from panic in case talker panic
	defer func() {
		if r := recover(); r != nil {
			v.Observer.Record(ctx,
				observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: v.CallContext.AssistantID},
					ConversationID: v.CallContext.ConversationID,
				},
				observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Error while starting session",
					Attributes: observability.Attributes{
						"context_id": contextID,
						"provider":   v.CallContext.Provider,
						"direction":  v.CallContext.Direction,
						"error":      fmt.Sprintf("%v", r),
					},
				},
				observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: fmt.Sprintf("%v", r),
					}, {
						Name:        observability.MetricCallDurationMs,
						Value:       fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
						Description: "Call duration in milliseconds",
					}},
				})
			result = &PipelineResult{ContextID: contextID, Error: fmt.Errorf("%v", r)}
		}
	}()
	err := v.Talker.Talk(ctx, v.CallContext.ToAuth())
	if err != nil {
		v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.CallContext.AssistantID},
				ConversationID: v.CallContext.ConversationID,
			},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Pipeline talk failed",
				Attributes: observability.Attributes{
					"provider":   v.CallContext.Provider,
					"direction":  v.CallContext.Direction,
					"error":      err.Error(),
					"context_id": contextID,
				},
			})

		v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.CallContext.AssistantID},
				ConversationID: v.CallContext.ConversationID,
			},
			observability.RecordWebhook{
				Event:     observability.CallFailed,
				ContextID: contextID,
				Payload: observability.CallFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             v.CallContext.Provider,
					CallID:               v.CallContext.ChannelUUID,
					To:                   v.CallContext.CallerNumber,
					From:                 v.CallContext.FromNumber,
					Direction:            v.CallContext.Direction,
					ContextID:            contextID,
					Error:                err.Error(),
					DurationMs:           fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
				},
			})
		v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.CallContext.AssistantID},
				ConversationID: v.CallContext.ConversationID,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusFailed,
					Description: err.Error(),
				}, {
					Name:        observability.MetricCallDurationMs,
					Value:       fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
					Description: "Call duration in milliseconds",
				}},
			})
		return &PipelineResult{ContextID: contextID, Error: err}
	}
	v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.CallContext.AssistantID},
			ConversationID: v.CallContext.ConversationID,
		},
		observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallEnded,
			Attributes: observability.Attributes{
				"provider":    v.CallContext.Provider,
				"to":          v.CallContext.CallerNumber,
				"from":        v.CallContext.FromNumber,
				"call_id":     v.CallContext.ChannelUUID,
				"context_id":  contextID,
				"direction":   v.CallContext.Direction,
				"duration_ms": fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
			},
		},
		observability.RecordWebhook{
			Event:     observability.CallEnded,
			ContextID: contextID,
			Payload: observability.CallEndedWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				Provider:             v.CallContext.Provider,
				CallID:               v.CallContext.ChannelUUID,
				To:                   v.CallContext.CallerNumber,
				From:                 v.CallContext.FromNumber,
				Direction:            v.CallContext.Direction,
				ContextID:            contextID,
				DurationMs:           fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
			},
		},
		observability.RecordMetric{
			Metrics: []*protos.Metric{
				{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: "Call talk return with success",
				},
				{
					Name:        observability.MetricCallDurationMs,
					Value:       fmt.Sprintf("%d", time.Since(startTime).Milliseconds()),
					Description: "Call duration in milliseconds",
				},
			},
		},
	)
	return &PipelineResult{
		ContextID:      contextID,
		ConversationID: v.CallContext.ConversationID,
		Provider:       v.CallContext.Provider,
		CallStatus:     "COMPLETE",
	}
}
