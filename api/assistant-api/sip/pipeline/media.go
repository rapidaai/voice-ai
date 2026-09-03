// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/protos"
)

type preparedSession struct {
	stage    sip_runtime.SessionEstablishedPipeline
	setup    *CallSetupResult
	observer observability.Recorder
	runtime  PreparedCallRuntime
}

type sessionPreparationError struct {
	reason sip_runtime.LifecycleReason
	err    error
}

func (e *sessionPreparationError) Error() string {
	return e.err.Error()
}

func newSessionPreparationError(reason sip_runtime.LifecycleReason, err error) *sessionPreparationError {
	return &sessionPreparationError{reason: reason, err: err}
}

func (d *Dispatcher) handleSessionEstablished(ctx context.Context, stage sip_runtime.SessionEstablishedPipeline) {
	prepared, err := d.prepareSession(ctx, stage)
	if err != nil {
		d.logger.Error("Pipeline: session preparation failed", "call_id", stage.ID, "error", err)
		d.endCall(stage.Session, sessionPreparationReason(err))
		return
	}
	d.startPreparedSession(ctx, prepared)
}

func (d *Dispatcher) PrepareSession(ctx context.Context, stage sip_runtime.SessionEstablishedPipeline) error {
	prepared, err := d.prepareSession(ctx, stage)
	if err != nil {
		return err
	}
	d.preparedMu.Lock()
	d.preparedSessions[stage.ID] = prepared
	d.preparedMu.Unlock()
	return nil
}

func (d *Dispatcher) StartPreparedSession(ctx context.Context, stage sip_runtime.SessionEstablishedPipeline) error {
	prepared := d.popPreparedSession(stage.ID)
	if prepared == nil {
		return fmt.Errorf("prepared SIP session not found for call %s", stage.ID)
	}
	d.startPreparedSession(ctx, prepared)
	return nil
}

func (d *Dispatcher) DiscardPreparedSession(ctx context.Context, callID string) {
	prepared := d.popPreparedSession(callID)
	if prepared == nil {
		return
	}
	prepared.Close(ctx)
}

func (d *Dispatcher) popPreparedSession(callID string) *preparedSession {
	d.preparedMu.Lock()
	defer d.preparedMu.Unlock()
	prepared := d.preparedSessions[callID]
	delete(d.preparedSessions, callID)
	return prepared
}

func (d *Dispatcher) prepareSession(ctx context.Context, stage sip_runtime.SessionEstablishedPipeline) (*preparedSession, error) {
	d.logger.Infow("Pipeline: SessionEstablished",
		"call_id", stage.ID,
		"direction", stage.Direction,
		"assistant_id", stage.AssistantID,
		"conversation_id", stage.ConversationID)

	conversationID := stage.ConversationID
	if conversationID == 0 {
		var err error
		conversationID, err = d.createConversation(ctx, stage)
		if err != nil {
			d.logger.Error("Pipeline: create conversation failed", "call_id", stage.ID, "error", err)
			return nil, newSessionPreparationError(sip_runtime.LifecycleReasonPipelineConversationFailed, err)
		}
		stage.Session.SetConversationID(conversationID)
	}

	cc, err := d.ensureCallContext(ctx, stage, conversationID)
	if err != nil {
		d.logger.Error("Pipeline: ensure call context failed", "call_id", stage.ID, "error", err)
		return nil, newSessionPreparationError(sip_runtime.LifecycleReasonPipelineSetupFailed, err)
	}

	setup, err := d.setupCall(ctx, stage, conversationID, cc)
	if err != nil {
		d.logger.Error("Pipeline: call setup failed", "call_id", stage.ID, "error", err)
		return nil, newSessionPreparationError(sip_runtime.LifecycleReasonPipelineSetupFailed, err)
	}

	observer := d.createObserver(ctx, setup, stage.Auth)
	codec := ""
	sampleRate := ""
	if negotiated := stage.Session.GetNegotiatedCodec(); negotiated != nil {
		codec = negotiated.Name
		sampleRate = fmt.Sprintf("%d", negotiated.ClockRate)
	}

	observer.Record(
		ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
			ConversationID: setup.ConversationID,
		},
		observability.RecordMetadata{
			Metadata: observability.ClientMetadata("", "", string(stage.Direction), "sip", stage.ID, "", codec, sampleRate),
		},
		observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallSessionConnected,
			Attributes: observability.Attributes{
				"provider":   "sip",
				"direction":  string(stage.Direction),
				"call_id":    stage.ID,
				"context_id": stage.ID,
			},
		},
		observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricCallStatus,
				Value:       observability.MetricCallStatusInProgress,
				Description: "SIP session connected",
			}},
		},
	)
	var runtime PreparedCallRuntime
	if stage.Direction == sip_runtime.CallDirectionInbound {
		var err error
		preparedRuntime, err := d.prepareSIPCallRuntime(ctx, stage, setup, observer)
		if err != nil {
			observer.Close(ctx)
			d.logger.Error("Pipeline: runtime preparation failed", "call_id", stage.ID, "error", err)
			return nil, newSessionPreparationError(sip_runtime.LifecycleReasonPipelineSetupFailed, err)
		}
		if err := preparedRuntime.StartBeforeAnswer(ctx, inboundRuntimeReadyTimeout(stage.Config)); err != nil {
			preparedRuntime.Close(ctx)
			observer.Close(ctx)
			d.logger.Error("Pipeline: runtime pre-answer start failed", "call_id", stage.ID, "error", err)
			return nil, newSessionPreparationError(sip_runtime.LifecycleReasonPipelineSetupFailed, err)
		}
		runtime = preparedRuntime
	}
	return &preparedSession{stage: stage, setup: setup, observer: observer, runtime: runtime}, nil
}

func (d *Dispatcher) startPreparedSession(ctx context.Context, prepared *preparedSession) {
	stage := prepared.stage
	setup := prepared.setup
	observer := prepared.observer
	go func() {
		startTime := time.Now()
		contextID := stage.Session.GetContextID()
		if contextID == "" && setup.CallContext != nil {
			contextID = setup.CallContext.ContextID
		}
		if contextID == "" {
			contextID = stage.ID
		}

		observer.Record(
			ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
				ConversationID: setup.ConversationID,
			},
			observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallStarted,
				Attributes: observability.Attributes{
					"provider":  "sip",
					"direction": string(stage.Direction),
					"call_id":   stage.ID,
				},
			},
			observability.RecordWebhook{
				Event:     observability.CallStarted,
				ContextID: contextID,
				Payload: observability.CallStartedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             "sip",
					CallID:               stage.ID,
					To:                   setup.CallContext.CallerNumber,
					From:                 setup.CallContext.FromNumber,
					Direction:            observability.WebhookCallDirection(stage.Direction),
					ContextID:            contextID,
					Status:               observability.WebhookCallStatusInProgress,
				},
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusInProgress,
					Description: "SIP call started",
				}},
			},
		)

		defer func() {
			if r := recover(); r != nil {
				reason := fmt.Sprintf("panic: %v", r)
				d.logger.Error("Pipeline: onCallStart panicked", "call_id", stage.ID, "panic", r)
				observer.Record(ctx, observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
					ConversationID: setup.ConversationID,
				}, observability.RecordLog{
					Level:   observability.LevelError,
					Message: "SIP pipeline call start panicked",
					Attributes: observability.Attributes{
						"provider":  "sip",
						"direction": string(stage.Direction),
						"call_id":   stage.ID,
						"panic":     fmt.Sprintf("%v", r),
					},
				})
				durationMs := time.Since(startTime).Milliseconds()
				observer.Record(ctx, observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
					ConversationID: setup.ConversationID,
				},
					observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallFailed,
						Attributes: observability.Attributes{
							"provider":    "sip",
							"direction":   string(stage.Direction),
							"call_id":     stage.ID,
							"reason":      reason,
							"status":      observability.MetricCallStatusFailed,
							"duration_ms": fmt.Sprintf("%d", durationMs),
						},
					},
					observability.RecordWebhook{
						Event:     observability.CallFailed,
						ContextID: contextID,
						Payload: observability.CallFailedWebhookPayload{
							V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
								"status": observability.MetricCallStatusFailed,
							}),
							Provider:         "sip",
							CallID:           stage.ID,
							To:               setup.CallContext.CallerNumber,
							From:             setup.CallContext.FromNumber,
							Direction:        observability.WebhookCallDirection(stage.Direction),
							ContextID:        contextID,
							DurationMs:       fmt.Sprintf("%d", durationMs),
							Status:           observability.WebhookCallStatusFailed,
							DisconnectReason: observability.WebhookCallDisconnectReasonInternalError,
						},
					},
					observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: reason,
						}, {
							Name:        observability.MetricCallDurationMs,
							Value:       fmt.Sprintf("%d", durationMs),
							Description: "SIP call duration in milliseconds",
						}},
					})
				observer.Close(ctx)
				d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
			}
		}()
		if prepared.runtime != nil {
			if err := prepared.runtime.Start(ctx); err != nil {
				if targetVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferTarget); ok {
					if target, ok := targetVal.(string); ok && target != "" {
						transferStatus := "failed"
						if statusVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferStatus); ok {
							if s, ok := statusVal.(string); ok {
								transferStatus = s
							}
						}
						reason := "transfer_" + transferStatus
						observer.Record(ctx, observability.ConversationScope{
							AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
							ConversationID: setup.ConversationID,
						}, observability.RecordEvent{
							Component: observability.ComponentSIP,
							Event:     observability.SIPTransferRequested,
							Attributes: observability.Attributes{
								"provider":  "sip",
								"direction": string(stage.Direction),
								"call_id":   stage.ID,
								"target":    target,
								"reason":    transferStatus,
							},
						})
						durationMs := time.Since(startTime).Milliseconds()
						observer.Record(ctx, observability.ConversationScope{
							AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
							ConversationID: setup.ConversationID,
						},
							observability.RecordEvent{
								Component: observability.ComponentCall,
								Event:     observability.CallFailed,
								Attributes: observability.Attributes{
									"provider":    "sip",
									"direction":   string(stage.Direction),
									"call_id":     stage.ID,
									"reason":      reason,
									"status":      observability.MetricCallStatusFailed,
									"duration_ms": fmt.Sprintf("%d", durationMs),
								},
							},
							observability.RecordWebhook{
								Event:     observability.CallFailed,
								ContextID: contextID,
								Payload: observability.CallFailedWebhookPayload{
									V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
										"status": observability.MetricCallStatusFailed,
									}),
									Provider:         "sip",
									CallID:           stage.ID,
									To:               setup.CallContext.CallerNumber,
									From:             setup.CallContext.FromNumber,
									Direction:        observability.WebhookCallDirection(stage.Direction),
									ContextID:        contextID,
									DurationMs:       fmt.Sprintf("%d", durationMs),
									Status:           observability.WebhookCallStatusFailed,
									DisconnectReason: observability.WebhookCallDisconnectReasonMediaFailed,
								},
							},
							observability.RecordMetric{
								Metrics: []*protos.Metric{{
									Name:        observability.MetricCallStatus,
									Value:       observability.MetricCallStatusFailed,
									Description: reason,
								}, {
									Name:        observability.MetricCallDurationMs,
									Value:       fmt.Sprintf("%d", durationMs),
									Description: "SIP call duration in milliseconds",
								}},
							})
						observer.Close(ctx)
						d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
						return
					}
				}
				reason := err.Error()
				durationMs := time.Since(startTime).Milliseconds()
				observer.Record(ctx, observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
					ConversationID: setup.ConversationID,
				},
					observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallFailed,
						Attributes: observability.Attributes{
							"provider":    "sip",
							"direction":   string(stage.Direction),
							"call_id":     stage.ID,
							"reason":      reason,
							"status":      observability.MetricCallStatusFailed,
							"duration_ms": fmt.Sprintf("%d", durationMs),
						},
					},
					observability.RecordWebhook{
						Event:     observability.CallFailed,
						ContextID: contextID,
						Payload: observability.CallFailedWebhookPayload{
							V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
								"status": observability.MetricCallStatusFailed,
							}),
							Provider:         "sip",
							CallID:           stage.ID,
							To:               setup.CallContext.CallerNumber,
							From:             setup.CallContext.FromNumber,
							Direction:        observability.WebhookCallDirection(stage.Direction),
							ContextID:        contextID,
							DurationMs:       fmt.Sprintf("%d", durationMs),
							Status:           observability.WebhookCallStatusFailed,
							DisconnectReason: observability.WebhookCallDisconnectReasonMediaFailed,
						},
					},
					observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusFailed,
							Description: reason,
						}, {
							Name:        observability.MetricCallDurationMs,
							Value:       fmt.Sprintf("%d", durationMs),
							Description: "SIP call duration in milliseconds",
						}},
					})
				observer.Close(ctx)
				d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
				return
			}
			if targetVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferTarget); ok {
				if target, ok := targetVal.(string); ok && target != "" {
					transferStatus := "failed"
					if statusVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferStatus); ok {
						if s, ok := statusVal.(string); ok {
							transferStatus = s
						}
					}
					durationMs := time.Since(startTime).Milliseconds()
					observer.Record(ctx, observability.ConversationScope{
						AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
						ConversationID: setup.ConversationID,
					},
						observability.RecordEvent{
							Component: observability.ComponentSIP,
							Event:     observability.SIPTransferRequested,
							Attributes: observability.Attributes{
								"provider":  "sip",
								"direction": string(stage.Direction),
								"call_id":   stage.ID,
								"target":    target,
								"reason":    transferStatus,
							},
						},
						observability.RecordEvent{
							Component: observability.ComponentCall,
							Event:     observability.CallEnded,
							Attributes: observability.Attributes{
								"provider":    "sip",
								"direction":   string(stage.Direction),
								"call_id":     stage.ID,
								"reason":      "transfer_" + transferStatus,
								"status":      observability.MetricCallStatusComplete,
								"duration_ms": fmt.Sprintf("%d", durationMs),
							},
						},
						observability.RecordWebhook{
							Event:     observability.CallEnded,
							ContextID: contextID,
							Payload: observability.CallEndedWebhookPayload{
								V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
								Provider:             "sip",
								CallID:               stage.ID,
								To:                   setup.CallContext.CallerNumber,
								From:                 setup.CallContext.FromNumber,
								Direction:            observability.WebhookCallDirection(stage.Direction),
								ContextID:            contextID,
								DurationMs:           fmt.Sprintf("%d", durationMs),
								Status:               observability.WebhookCallStatusCompleted,
								DisconnectReason:     observability.WebhookCallDisconnectReasonTransferred,
							},
						},
						observability.RecordMetric{
							Metrics: []*protos.Metric{{
								Name:        observability.MetricCallStatus,
								Value:       observability.MetricCallStatusComplete,
								Description: "transfer_" + transferStatus,
							}, {
								Name:        observability.MetricCallDurationMs,
								Value:       fmt.Sprintf("%d", durationMs),
								Description: "SIP call duration in milliseconds",
							}},
						})
					observer.Close(ctx)
					d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
					return
				}
			}
			durationMs := time.Since(startTime).Milliseconds()
			observer.Record(ctx, observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
				ConversationID: setup.ConversationID,
			},
				observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallEnded,
					Attributes: observability.Attributes{
						"provider":    "sip",
						"direction":   string(stage.Direction),
						"call_id":     stage.ID,
						"reason":      "talk_completed",
						"status":      observability.MetricCallStatusComplete,
						"duration_ms": fmt.Sprintf("%d", durationMs),
					},
				},
				observability.RecordWebhook{
					Event:     observability.CallEnded,
					ContextID: contextID,
					Payload: observability.CallEndedWebhookPayload{
						V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
						Provider:             "sip",
						CallID:               stage.ID,
						To:                   setup.CallContext.CallerNumber,
						From:                 setup.CallContext.FromNumber,
						Direction:            observability.WebhookCallDirection(stage.Direction),
						ContextID:            contextID,
						DurationMs:           fmt.Sprintf("%d", durationMs),
						Status:               observability.WebhookCallStatusCompleted,
						DisconnectReason:     observability.WebhookCallDisconnectReasonAssistantEnded,
					},
				},
				observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusComplete,
						Description: "talk_completed",
					}, {
						Name:        observability.MetricCallDurationMs,
						Value:       fmt.Sprintf("%d", durationMs),
						Description: "SIP call duration in milliseconds",
					}},
				})
			observer.Close(ctx)
			d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
			return
		}

		runtime, err := d.prepareSIPCallRuntime(ctx, stage, setup, observer)
		if err != nil {
			if stage.Session.GetInfo().Direction == sip_runtime.CallDirectionOutbound && !stage.Session.IsEnded() {
				state := stage.Session.GetState()
				if state != sip_runtime.CallStateTransferring && state != sip_runtime.CallStateBridgeConnected {
					d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineTalkCompleted)
				}
			}
			if targetVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferTarget); ok {
				if target, ok := targetVal.(string); ok && target != "" {
					transferStatus := "failed"
					if statusVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferStatus); ok {
						if s, ok := statusVal.(string); ok {
							transferStatus = s
						}
					}
					reason := "transfer_" + transferStatus
					observer.Record(ctx, observability.ConversationScope{
						AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
						ConversationID: setup.ConversationID,
					}, observability.RecordEvent{
						Component: observability.ComponentSIP,
						Event:     observability.SIPTransferRequested,
						Attributes: observability.Attributes{
							"provider":  "sip",
							"direction": string(stage.Direction),
							"call_id":   stage.ID,
							"target":    target,
							"reason":    transferStatus,
						},
					})
					durationMs := time.Since(startTime).Milliseconds()
					observer.Record(ctx, observability.ConversationScope{
						AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
						ConversationID: setup.ConversationID,
					},
						observability.RecordEvent{
							Component: observability.ComponentCall,
							Event:     observability.CallFailed,
							Attributes: observability.Attributes{
								"provider":    "sip",
								"direction":   string(stage.Direction),
								"call_id":     stage.ID,
								"reason":      reason,
								"status":      observability.MetricCallStatusFailed,
								"duration_ms": fmt.Sprintf("%d", durationMs),
							},
						},
						observability.RecordWebhook{
							Event:     observability.CallFailed,
							ContextID: contextID,
							Payload: observability.CallFailedWebhookPayload{
								V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
									"status": observability.MetricCallStatusFailed,
								}),
								Provider:         "sip",
								CallID:           stage.ID,
								To:               setup.CallContext.CallerNumber,
								From:             setup.CallContext.FromNumber,
								Direction:        observability.WebhookCallDirection(stage.Direction),
								ContextID:        contextID,
								DurationMs:       fmt.Sprintf("%d", durationMs),
								Status:           observability.WebhookCallStatusFailed,
								DisconnectReason: observability.WebhookCallDisconnectReasonMediaFailed,
							},
						},
						observability.RecordMetric{
							Metrics: []*protos.Metric{{
								Name:        observability.MetricCallStatus,
								Value:       observability.MetricCallStatusFailed,
								Description: reason,
							}, {
								Name:        observability.MetricCallDurationMs,
								Value:       fmt.Sprintf("%d", durationMs),
								Description: "SIP call duration in milliseconds",
							}},
						})
					observer.Close(ctx)
					d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
					return
				}
			}
			reason := err.Error()
			durationMs := time.Since(startTime).Milliseconds()
			observer.Record(ctx, observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
				ConversationID: setup.ConversationID,
			},
				observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallFailed,
					Attributes: observability.Attributes{
						"provider":    "sip",
						"direction":   string(stage.Direction),
						"call_id":     stage.ID,
						"reason":      reason,
						"status":      observability.MetricCallStatusFailed,
						"duration_ms": fmt.Sprintf("%d", durationMs),
					},
				},
				observability.RecordWebhook{
					Event:     observability.CallFailed,
					ContextID: contextID,
					Payload: observability.CallFailedWebhookPayload{
						V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
							"status": observability.MetricCallStatusFailed,
						}),
						Provider:         "sip",
						CallID:           stage.ID,
						To:               setup.CallContext.CallerNumber,
						From:             setup.CallContext.FromNumber,
						Direction:        observability.WebhookCallDirection(stage.Direction),
						ContextID:        contextID,
						DurationMs:       fmt.Sprintf("%d", durationMs),
						Status:           observability.WebhookCallStatusFailed,
						DisconnectReason: observability.WebhookCallDisconnectReasonMediaFailed,
					},
				},
				observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: reason,
					}, {
						Name:        observability.MetricCallDurationMs,
						Value:       fmt.Sprintf("%d", durationMs),
						Description: "SIP call duration in milliseconds",
					}},
				})
			observer.Close(ctx)
			d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
			return
		}
		if err := runtime.Start(ctx); err != nil {
			if stage.Session.GetInfo().Direction == sip_runtime.CallDirectionOutbound && !stage.Session.IsEnded() {
				state := stage.Session.GetState()
				if state != sip_runtime.CallStateTransferring && state != sip_runtime.CallStateBridgeConnected {
					d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineTalkCompleted)
				}
			}
			if targetVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferTarget); ok {
				if target, ok := targetVal.(string); ok && target != "" {
					transferStatus := "failed"
					if statusVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferStatus); ok {
						if s, ok := statusVal.(string); ok {
							transferStatus = s
						}
					}
					reason := "transfer_" + transferStatus
					observer.Record(ctx, observability.ConversationScope{
						AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
						ConversationID: setup.ConversationID,
					}, observability.RecordEvent{
						Component: observability.ComponentSIP,
						Event:     observability.SIPTransferRequested,
						Attributes: observability.Attributes{
							"provider":  "sip",
							"direction": string(stage.Direction),
							"call_id":   stage.ID,
							"target":    target,
							"reason":    transferStatus,
						},
					})
					durationMs := time.Since(startTime).Milliseconds()
					observer.Record(ctx, observability.ConversationScope{
						AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
						ConversationID: setup.ConversationID,
					},
						observability.RecordEvent{
							Component: observability.ComponentCall,
							Event:     observability.CallFailed,
							Attributes: observability.Attributes{
								"provider":    "sip",
								"direction":   string(stage.Direction),
								"call_id":     stage.ID,
								"reason":      reason,
								"status":      observability.MetricCallStatusFailed,
								"duration_ms": fmt.Sprintf("%d", durationMs),
							},
						},
						observability.RecordWebhook{
							Event:     observability.CallFailed,
							ContextID: contextID,
							Payload: observability.CallFailedWebhookPayload{
								V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
									"status": observability.MetricCallStatusFailed,
								}),
								Provider:         "sip",
								CallID:           stage.ID,
								To:               setup.CallContext.CallerNumber,
								From:             setup.CallContext.FromNumber,
								Direction:        observability.WebhookCallDirection(stage.Direction),
								ContextID:        contextID,
								DurationMs:       fmt.Sprintf("%d", durationMs),
								Status:           observability.WebhookCallStatusFailed,
								DisconnectReason: observability.WebhookCallDisconnectReasonMediaFailed,
							},
						},
						observability.RecordMetric{
							Metrics: []*protos.Metric{{
								Name:        observability.MetricCallStatus,
								Value:       observability.MetricCallStatusFailed,
								Description: reason,
							}, {
								Name:        observability.MetricCallDurationMs,
								Value:       fmt.Sprintf("%d", durationMs),
								Description: "SIP call duration in milliseconds",
							}},
						})
					observer.Close(ctx)
					d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
					return
				}
			}
			reason := err.Error()
			durationMs := time.Since(startTime).Milliseconds()
			observer.Record(ctx, observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
				ConversationID: setup.ConversationID,
			},
				observability.RecordEvent{
					Component: observability.ComponentCall,
					Event:     observability.CallFailed,
					Attributes: observability.Attributes{
						"provider":    "sip",
						"direction":   string(stage.Direction),
						"call_id":     stage.ID,
						"reason":      reason,
						"status":      observability.MetricCallStatusFailed,
						"duration_ms": fmt.Sprintf("%d", durationMs),
					},
				},
				observability.RecordWebhook{
					Event:     observability.CallFailed,
					ContextID: contextID,
					Payload: observability.CallFailedWebhookPayload{
						V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{
							"status": observability.MetricCallStatusFailed,
						}),
						Provider:         "sip",
						CallID:           stage.ID,
						To:               setup.CallContext.CallerNumber,
						From:             setup.CallContext.FromNumber,
						Direction:        observability.WebhookCallDirection(stage.Direction),
						ContextID:        contextID,
						DurationMs:       fmt.Sprintf("%d", durationMs),
						Status:           observability.WebhookCallStatusFailed,
						DisconnectReason: observability.WebhookCallDisconnectReasonMediaFailed,
					},
				},
				observability.RecordMetric{
					Metrics: []*protos.Metric{{
						Name:        observability.MetricCallStatus,
						Value:       observability.MetricCallStatusFailed,
						Description: reason,
					}, {
						Name:        observability.MetricCallDurationMs,
						Value:       fmt.Sprintf("%d", durationMs),
						Description: "SIP call duration in milliseconds",
					}},
				})
			observer.Close(ctx)
			d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
			return
		}
		if stage.Session.GetInfo().Direction == sip_runtime.CallDirectionOutbound && !stage.Session.IsEnded() {
			state := stage.Session.GetState()
			if state != sip_runtime.CallStateTransferring && state != sip_runtime.CallStateBridgeConnected {
				d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineTalkCompleted)
			}
		}
		if targetVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferTarget); ok {
			if target, ok := targetVal.(string); ok && target != "" {
				transferStatus := "failed"
				if statusVal, ok := stage.Session.GetMetadata(sip_runtime.MetadataBridgeTransferStatus); ok {
					if s, ok := statusVal.(string); ok {
						transferStatus = s
					}
				}
				reason := "transfer_" + transferStatus
				observer.Record(ctx, observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
					ConversationID: setup.ConversationID,
				}, observability.RecordEvent{
					Component: observability.ComponentSIP,
					Event:     observability.SIPTransferRequested,
					Attributes: observability.Attributes{
						"provider":  "sip",
						"direction": string(stage.Direction),
						"call_id":   stage.ID,
						"target":    target,
						"reason":    transferStatus,
					},
				})
				durationMs := time.Since(startTime).Milliseconds()
				observer.Record(ctx, observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
					ConversationID: setup.ConversationID,
				},
					observability.RecordEvent{
						Component: observability.ComponentCall,
						Event:     observability.CallEnded,
						Attributes: observability.Attributes{
							"provider":    "sip",
							"direction":   string(stage.Direction),
							"call_id":     stage.ID,
							"reason":      reason,
							"status":      observability.MetricCallStatusComplete,
							"duration_ms": fmt.Sprintf("%d", durationMs),
						},
					},
					observability.RecordWebhook{
						Event:     observability.CallEnded,
						ContextID: contextID,
						Payload: observability.CallEndedWebhookPayload{
							V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
							Provider:             "sip",
							CallID:               stage.ID,
							To:                   setup.CallContext.CallerNumber,
							From:                 setup.CallContext.FromNumber,
							Direction:            observability.WebhookCallDirection(stage.Direction),
							ContextID:            contextID,
							DurationMs:           fmt.Sprintf("%d", durationMs),
							Status:               observability.WebhookCallStatusCompleted,
							DisconnectReason:     observability.WebhookCallDisconnectReasonTransferred,
						},
					},
					observability.RecordMetric{
						Metrics: []*protos.Metric{{
							Name:        observability.MetricCallStatus,
							Value:       observability.MetricCallStatusComplete,
							Description: reason,
						}, {
							Name:        observability.MetricCallDurationMs,
							Value:       fmt.Sprintf("%d", durationMs),
							Description: "SIP call duration in milliseconds",
						}},
					})
				observer.Close(ctx)
				d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
				return
			}
		}
		reason := "talk_completed"
		durationMs := time.Since(startTime).Milliseconds()
		observer.Record(ctx, observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
			ConversationID: setup.ConversationID,
		},
			observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallEnded,
				Attributes: observability.Attributes{
					"provider":    "sip",
					"direction":   string(stage.Direction),
					"call_id":     stage.ID,
					"reason":      reason,
					"status":      observability.MetricCallStatusComplete,
					"duration_ms": fmt.Sprintf("%d", durationMs),
				},
			},
			observability.RecordWebhook{
				Event:     observability.CallEnded,
				ContextID: contextID,
				Payload: observability.CallEndedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             "sip",
					CallID:               stage.ID,
					To:                   setup.CallContext.CallerNumber,
					From:                 setup.CallContext.FromNumber,
					Direction:            observability.WebhookCallDirection(stage.Direction),
					ContextID:            contextID,
					DurationMs:           fmt.Sprintf("%d", durationMs),
					Status:               observability.WebhookCallStatusCompleted,
					DisconnectReason:     observability.WebhookCallDisconnectReasonAssistantEnded,
				},
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricCallStatus,
					Value:       observability.MetricCallStatusComplete,
					Description: reason,
				}, {
					Name:        observability.MetricCallDurationMs,
					Value:       fmt.Sprintf("%d", durationMs),
					Description: "SIP call duration in milliseconds",
				}},
			})
		observer.Close(ctx)
		d.endCall(stage.Session, sip_runtime.LifecycleReasonPipelineCallEnd)
	}()
}

func (p *preparedSession) Close(ctx context.Context) {
	if p == nil {
		return
	}
	if p.runtime != nil {
		p.runtime.Close(ctx)
	}
	p.observer.Close(ctx)
}

func sessionPreparationReason(err error) sip_runtime.LifecycleReason {
	var preparationErr *sessionPreparationError
	if errors.As(err, &preparationErr) {
		return preparationErr.reason
	}
	return sip_runtime.LifecycleReasonPipelineSetupFailed
}
