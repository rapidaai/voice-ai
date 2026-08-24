// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"fmt"
	"time"

	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// =============================================================================
// Talk - Main Entry Point
// =============================================================================

// Talk handles the main conversation loop for different streamer types.
// It processes incoming messages and manages the connection lifecycle.
//
// Shutdown relies on Recv() returning an error (EOF or context-cancelled)
// or a ConversationDisconnection message. All streamer implementations
// guarantee one of these when the connection ends.
func (t *genericRequestor) Talk(_ context.Context, auth *types.Authentication) error {
	totalTime := time.Now()
	for {
		req, err := t.streamer.Recv()
		if err != nil {
			t.OnCallCompletion(totalTime)
			t.OnDisconnect(context.Background())
			return nil
		}
		switch payload := req.(type) {
		case *protos.ConversationInitialization:
			t.OnConnect(t.streamer.Context(), auth, payload)
		case *protos.ConversationConfiguration:
			t.OnStreamModeSwitch(t.streamer.Context(), payload)
		case *protos.ConversationUserMessage:
			t.OnStreamUserMessage(t.streamer.Context(), payload)
		case *protos.ConversationToolCallResult:
			t.OnPacket(t.streamer.Context(), internal_type.LLMToolResultPacket{
				ToolID:    payload.GetToolId(),
				Name:      payload.GetName(),
				ContextID: payload.GetId(),
				Action:    payload.GetAction(),
				Result:    payload.GetResult(),
			})
		case *protos.ConversationBridgeUserAudio:
			t.OnPacket(t.streamer.Context(), internal_type.RecordUserAudioPacket{ContextID: t.GetID(), Audio: payload.Audio, Timestamp: payload.Time.AsTime()})
		case *protos.ConversationBridgeOperatorAudio:
			t.OnPacket(t.streamer.Context(), internal_type.RecordAssistantAudioPacket{ContextID: t.GetID(), Audio: payload.Audio, Timestamp: payload.Time.AsTime()})
		case *protos.ConversationMetadata:
			if t.metadata == nil {
				t.metadata = make(map[string]interface{})
			}
			for _, metadata := range payload.GetMetadata() {
				if metadata == nil {
					continue
				}
				t.metadata[metadata.GetKey()] = metadata.GetValue()
			}
			t.OnPacket(t.streamer.Context(), internal_type.ObservabilityMetadataRecordPacket{
				ContextID: fmt.Sprintf("%d", payload.GetAssistantConversationId()),
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewConversationMetadataRecord(payload.GetMetadata()),
			})
		case *protos.ConversationMetric:
			if t.metrics == nil {
				t.metrics = make(map[string]*protos.Metric)
			}
			for _, metric := range payload.GetMetrics() {
				if metric == nil {
					continue
				}
				t.metrics[metric.GetName()] = &protos.Metric{
					Name:        metric.GetName(),
					Value:       metric.GetValue(),
					Description: metric.GetDescription(),
				}
			}
			t.OnPacket(t.streamer.Context(), internal_type.ObservabilityMetricRecordPacket{
				ContextID: fmt.Sprintf("%d", payload.GetAssistantConversationId()),
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewConversationMetricRecord(payload.GetMetrics()),
			})
		case *protos.ConversationEvent:
			eventTime := time.Now()
			if payload.Time != nil {
				eventTime = payload.Time.AsTime()
			}
			t.OnPacket(t.streamer.Context(), internal_type.ObservabilityEventRecordPacket{
				ContextID: payload.GetId(),
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordEvent{
					Component:  observability.ComponentConversation,
					Event:      observability.EventName(payload.Name),
					Attributes: observability.Attributes(payload.Data),
					OccurredAt: eventTime,
				},
			})
		case *protos.ConversationDisconnection:
			ctx := context.Background()
			if t.metrics == nil {
				t.metrics = make(map[string]*protos.Metric)
			}
			if payload.GetType() == protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR {
				t.metrics[type_enums.CONVERSATION_STATUS.String()] = &protos.Metric{
					Name:        type_enums.CONVERSATION_STATUS.String(),
					Value:       "error",
					Description: payload.GetType().String(),
				}
			} else {
				t.metrics[type_enums.CONVERSATION_STATUS.String()] = &protos.Metric{
					Name:        type_enums.CONVERSATION_STATUS.String(),
					Value:       type_enums.CONVERSATION_COMPLETE.String(),
					Description: payload.GetType().String(),
				}
			}
			if conversation, err := t.Conversation(); err == nil {
				t.OnPacket(ctx,
					internal_type.ObservabilityEventRecordPacket{
						ContextID: t.GetID(),
						Scope:     internal_type.ObservabilityRecordScopeConversation,
						Record: observability.RecordEvent{
							Component: observability.ComponentConversation,
							Event:     observability.ConversationCompleted,
							Attributes: observability.Attributes{
								"reason": payload.GetType().String(),
							},
							OccurredAt: time.Now(),
						},
					},
					internal_type.ObservabilityMetadataRecordPacket{
						ContextID: fmt.Sprintf("%d", conversation.Id),
						Scope:     internal_type.ObservabilityRecordScopeConversation,
						Record:    observability.NewConversationMetadataRecord(observability.DisconnectMetadata(payload.GetType().String(), "")),
					},
					internal_type.ObservabilityMetricRecordPacket{
						ContextID: fmt.Sprintf("%d", conversation.Id),
						Scope:     internal_type.ObservabilityRecordScopeConversation,
						Record: observability.NewConversationMetricRecord([]*protos.Metric{{
							Name:        type_enums.CONVERSATION_STATUS.String(),
							Value:       t.metrics[type_enums.CONVERSATION_STATUS.String()].GetValue(),
							Description: t.metrics[type_enums.CONVERSATION_STATUS.String()].GetDescription(),
						}}),
					},
				)
			}
			// client calling disconnect, we can safely cleanup the session and return
			t.OnCallCompletion(totalTime)
			t.OnDisconnect(ctx)
			return nil

		}

	}
}

func (t *genericRequestor) OnStreamModeSwitch(ctx context.Context, payload *protos.ConversationConfiguration) {
	t.OnPacket(ctx, internal_type.ModeSwitchRequestedPacket{
		ContextID:   t.GetID(),
		StreamMode:  payload.GetStreamMode(),
		RequestedAt: time.Now(),
	})
}

func (t *genericRequestor) OnStreamUserMessage(ctx context.Context, payload *protos.ConversationUserMessage) {
	switch msg := payload.GetMessage().(type) {
	case *protos.ConversationUserMessage_Audio:
		t.OnPacket(ctx, internal_type.UserAudioReceivedPacket{ContextID: t.GetID(), Audio: msg.Audio})
	case *protos.ConversationUserMessage_Text:
		t.OnPacket(ctx, internal_type.UserTextReceivedPacket{ContextID: t.GetID(), Text: msg.Text})
	default:
		t.logger.Errorf("illegal input from the user %+v", msg)
	}
}

// OnCallCompletion emits final metrics + an EventCompleted event when the talk
// loop exits. Persistence and telemetry collection happen in the existing
// background-channel handlers, so this function only enqueues packets.
func (t *genericRequestor) OnCallCompletion(startTime time.Time) {
	conv, err := t.Conversation()
	if err != nil {
		return
	}
	duration := time.Since(startTime)
	if t.metrics == nil {
		t.metrics = make(map[string]*protos.Metric)
	}
	if t.metrics[type_enums.CONVERSATION_STATUS.String()] == nil {
		t.metrics[type_enums.CONVERSATION_STATUS.String()] = &protos.Metric{
			Name:        type_enums.CONVERSATION_STATUS.String(),
			Value:       type_enums.CONVERSATION_COMPLETE.String(),
			Description: "Status of current conversation",
		}
	}

	t.OnPacket(context.Background(),
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: fmt.Sprintf("%d", conv.Id),
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewConversationMetricRecord([]*protos.Metric{
				t.metrics[type_enums.CONVERSATION_STATUS.String()],
				{
					Name:        observability.MetricConversationDuration,
					Value:       fmt.Sprintf("%d", duration.Milliseconds()),
					Description: "Conversation duration from first message to end",
				},
			}),
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: t.GetID(),
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentConversation,
				Event:     observability.ConversationCleanup,
				Attributes: observability.Attributes{
					"duration_ms": fmt.Sprintf("%d", duration.Milliseconds()),
					"messages":    fmt.Sprintf("%d", len(t.GetHistories())),
				},
				OccurredAt: time.Now(),
			},
		},
	)
}

// Notify sends notifications to websocket for various events.
func (t *genericRequestor) Notify(ctx context.Context, actionDatas ...internal_type.Stream) error {
	for _, actionData := range actionDatas {
		if err := t.streamer.Send(actionData); err != nil {
			t.logger.Errorf("error while notifing client %v", err)
		}
	}
	return nil
}

// =============================================================================
// Session Lifecycle
// =============================================================================

// Connect starts bootstrap/background dispatchers and enqueues the init chain.
// Runtime dispatchers (critical/ingress/egress) are started after
// InitializationCompleted. Connect always returns nil because initialization
// runs asynchronously on the bootstrap dispatcher goroutine.
// The gRPC stream is already open by the time Connect is called; any init errors
// are delivered to the client via InitializationFailedPacket → ConversationError
// proto on the stream, not via this return value.
func (r *genericRequestor) OnConnect(ctx context.Context, auth *types.Authentication, config *protos.ConversationInitialization) {
	if err := r.sessionLifecycle.Transition(adapter_lifecycle.EventConnectRequested); err != nil {
		r.logger.Tracef(ctx, "connect ignored due to session lifecycle transition: %v", err)
		return
	}
	r.setAuth(auth)
	utils.WithDeadline(r.sessionCtx, connectDeadline, func() {
		if r.sessionLifecycle.Current() != adapter_lifecycle.StateInitializing {
			return
		}
		if conversation, err := r.Conversation(); err == nil {
			if r.metrics == nil {
				r.metrics = make(map[string]*protos.Metric)
			}
			r.metrics[type_enums.CONVERSATION_STATUS.String()] = &protos.Metric{
				Name:        type_enums.CONVERSATION_STATUS.String(),
				Value:       "error",
				Description: "initialization timeout",
			}
			r.OnPacket(r.sessionCtx,
				internal_type.ObservabilityEventRecordPacket{
					ContextID: r.GetID(),
					Scope:     internal_type.ObservabilityRecordScopeConversation,
					Record: observability.RecordEvent{
						Component: observability.ComponentConversation,
						Event:     observability.ConversationCompleted,
						Attributes: observability.Attributes{
							"reason": protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
						},
						OccurredAt: time.Now(),
					},
				},
				internal_type.ObservabilityMetadataRecordPacket{
					ContextID: fmt.Sprintf("%d", conversation.Id),
					Scope:     internal_type.ObservabilityRecordScopeConversation,
					Record: observability.NewConversationMetadataRecord(observability.DisconnectMetadata(
						protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR.String(),
						"initialization timeout",
					)),
				},
				internal_type.ObservabilityMetricRecordPacket{
					ContextID: fmt.Sprintf("%d", conversation.Id),
					Scope:     internal_type.ObservabilityRecordScopeConversation,
					Record: observability.NewConversationMetricRecord([]*protos.Metric{{
						Name:        type_enums.CONVERSATION_STATUS.String(),
						Value:       "error",
						Description: "initialization timeout",
					}}),
				},
			)
		}
		r.Notify(r.sessionCtx,
			&protos.ConversationError{Message: "initialization timeout"},
			&protos.ConversationDisconnection{Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_ERROR},
		)
		r.cancelSession()
	}, func(connectCtx context.Context) {
		r.OnPacket(r.sessionCtx,
			internal_type.ObservabilityEventRecordPacket{
				ContextID: r.GetID(),
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordEvent{
					Component: observability.ComponentConversation,
					Event:     observability.ConversationInitializing,
					Attributes: observability.Attributes{
						"mode": config.GetStreamMode().String(),
					},
					OccurredAt: time.Now(),
				},
			}, internal_type.InitializeAssistantPacket{
				ContextID: r.GetID(),
				Config:    config,
			})
	})
}

// OnDisconnect enqueues the disconnect chain. sessionCtx is cancelled either by
// HandleFinalizationCompleted (normal completion) or by the watchdog if the
// chain exceeds disconnectDeadline.
func (r *genericRequestor) OnDisconnect(ctx context.Context) {
	if err := r.sessionLifecycle.Transition(adapter_lifecycle.EventDisconnectRequested); err != nil {
		r.logger.Tracef(ctx, "disconnect ignored due to session lifecycle transition: %v", err)
		return
	}
	utils.WithDeadline(r.sessionCtx, disconnectDeadline, func() {
		r.logger.Warnf("disconnect deadline %v exceeded, force-cancelling session", disconnectDeadline)
		r.cancelSession()
	}, func(disconnectCtx context.Context) {
		r.OnPacket(disconnectCtx, internal_type.FinalizeInboundDispatcherPacket{ContextID: r.GetID()})
	})
}
