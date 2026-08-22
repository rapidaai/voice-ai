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
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/protos"
)

type preparedSession struct {
	stage    sip_infra.SessionEstablishedPipeline
	setup    *CallSetupResult
	observer observability.Recorder
	runtime  PreparedCallRuntime
}

type sessionPreparationError struct {
	reason sip_infra.LifecycleReason
	err    error
}

func (e *sessionPreparationError) Error() string {
	return e.err.Error()
}

func newSessionPreparationError(reason sip_infra.LifecycleReason, err error) *sessionPreparationError {
	return &sessionPreparationError{reason: reason, err: err}
}

func (d *Dispatcher) handleSessionEstablished(ctx context.Context, v sip_infra.SessionEstablishedPipeline) {
	prepared, err := d.prepareSession(ctx, v)
	if err != nil {
		d.logger.Error("Pipeline: session preparation failed", "call_id", v.ID, "error", err)
		d.endCall(v.Session, sessionPreparationReason(err))
		return
	}
	d.startPreparedSession(ctx, prepared)
}

func (d *Dispatcher) PrepareSession(ctx context.Context, v sip_infra.SessionEstablishedPipeline) error {
	prepared, err := d.prepareSession(ctx, v)
	if err != nil {
		return err
	}
	d.preparedMu.Lock()
	d.preparedSessions[v.ID] = prepared
	d.preparedMu.Unlock()
	return nil
}

func (d *Dispatcher) StartPreparedSession(ctx context.Context, v sip_infra.SessionEstablishedPipeline) error {
	prepared := d.popPreparedSession(v.ID)
	if prepared == nil {
		return fmt.Errorf("prepared SIP session not found for call %s", v.ID)
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

func (d *Dispatcher) prepareSession(ctx context.Context, v sip_infra.SessionEstablishedPipeline) (*preparedSession, error) {
	d.logger.Infow("Pipeline: SessionEstablished",
		"call_id", v.ID,
		"direction", v.Direction,
		"assistant_id", v.AssistantID,
		"conversation_id", v.ConversationID)

	conversationID := v.ConversationID
	if conversationID == 0 {
		var err error
		conversationID, err = d.createConversation(ctx, v)
		if err != nil {
			d.logger.Error("Pipeline: create conversation failed", "call_id", v.ID, "error", err)
			return nil, newSessionPreparationError(sip_infra.LifecycleReasonPipelineConversationFailed, err)
		}
		v.Session.SetConversationID(conversationID)
	}

	cc, err := d.ensureCallContext(ctx, v, conversationID)
	if err != nil {
		d.logger.Error("Pipeline: ensure call context failed", "call_id", v.ID, "error", err)
		return nil, newSessionPreparationError(sip_infra.LifecycleReasonPipelineSetupFailed, err)
	}

	setup, err := d.setupCall(ctx, v, conversationID, cc)
	if err != nil {
		d.logger.Error("Pipeline: call setup failed", "call_id", v.ID, "error", err)
		return nil, newSessionPreparationError(sip_infra.LifecycleReasonPipelineSetupFailed, err)
	}

	observer := d.createObserver(ctx, setup, v.Auth)
	codec := ""
	sampleRate := ""
	if negotiated := v.Session.GetNegotiatedCodec(); negotiated != nil {
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
			Metadata: observability.ClientMetadata("", "", string(v.Direction), "sip", v.ID, "", codec, sampleRate),
		},
		observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallSessionConnected,
			Attributes: observability.Attributes{
				"provider":   "sip",
				"direction":  string(v.Direction),
				"call_id":    v.ID,
				"context_id": v.ID,
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
	if v.Direction == sip_infra.CallDirectionInbound {
		var err error
		preparedRuntime, err := d.prepareSIPCallRuntime(ctx, v.Session, setup, observer, v.VaultCredential, v.Config, string(v.Direction), v.FromIdentity, v.ToIdentity)
		if err != nil {
			observer.Close(ctx)
			d.logger.Error("Pipeline: runtime preparation failed", "call_id", v.ID, "error", err)
			return nil, newSessionPreparationError(sip_infra.LifecycleReasonPipelineSetupFailed, err)
		}
		if err := preparedRuntime.StartBeforeAnswer(ctx, inboundRuntimeReadyTimeout(v.Config)); err != nil {
			preparedRuntime.Close(ctx)
			observer.Close(ctx)
			d.logger.Error("Pipeline: runtime pre-answer start failed", "call_id", v.ID, "error", err)
			return nil, newSessionPreparationError(sip_infra.LifecycleReasonPipelineSetupFailed, err)
		}
		runtime = preparedRuntime
	}
	return &preparedSession{stage: v, setup: setup, observer: observer, runtime: runtime}, nil
}

func (d *Dispatcher) startPreparedSession(ctx context.Context, prepared *preparedSession) {
	v := prepared.stage
	setup := prepared.setup
	observer := prepared.observer
	go func() {
		startTime := time.Now()
		contextID := v.Session.GetContextID()
		if contextID == "" && setup.CallContext != nil {
			contextID = setup.CallContext.ContextID
		}
		if contextID == "" {
			contextID = v.ID
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
					"direction": string(v.Direction),
					"call_id":   v.ID,
				},
			},
			observability.RecordWebhook{
				Event:     observability.CallStarted,
				ContextID: contextID,
				Payload: observability.CallStartedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Provider:             "sip",
					CallID:               v.ID,
					To:                   setup.CallContext.CallerNumber,
					From:                 setup.CallContext.FromNumber,
					Direction:            string(v.Direction),
					ContextID:            contextID,
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
				d.logger.Error("Pipeline: onCallStart panicked", "call_id", v.ID, "panic", r)
				observer.Record(ctx, observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: setup.AssistantID},
					ConversationID: setup.ConversationID,
				}, observability.RecordLog{
					Level:   observability.LevelError,
					Message: "SIP pipeline call start panicked",
					Attributes: observability.Attributes{
						"provider":  "sip",
						"direction": string(v.Direction),
						"call_id":   v.ID,
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
							"direction":   string(v.Direction),
							"call_id":     v.ID,
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
							Provider:   "sip",
							CallID:     v.ID,
							To:         setup.CallContext.CallerNumber,
							From:       setup.CallContext.FromNumber,
							Direction:  string(v.Direction),
							ContextID:  contextID,
							Reason:     reason,
							DurationMs: fmt.Sprintf("%d", durationMs),
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
				d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
			}
		}()
		if prepared.runtime != nil {
			if err := prepared.runtime.Start(ctx); err != nil {
				if targetVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferTarget); ok {
					if target, ok := targetVal.(string); ok && target != "" {
						transferStatus := "failed"
						if statusVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferStatus); ok {
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
								"direction": string(v.Direction),
								"call_id":   v.ID,
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
									"direction":   string(v.Direction),
									"call_id":     v.ID,
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
									Provider:   "sip",
									CallID:     v.ID,
									To:         setup.CallContext.CallerNumber,
									From:       setup.CallContext.FromNumber,
									Direction:  string(v.Direction),
									ContextID:  contextID,
									Reason:     reason,
									DurationMs: fmt.Sprintf("%d", durationMs),
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
						d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
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
							"direction":   string(v.Direction),
							"call_id":     v.ID,
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
							Provider:   "sip",
							CallID:     v.ID,
							To:         setup.CallContext.CallerNumber,
							From:       setup.CallContext.FromNumber,
							Direction:  string(v.Direction),
							ContextID:  contextID,
							Reason:     reason,
							DurationMs: fmt.Sprintf("%d", durationMs),
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
				d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
				return
			}
			if targetVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferTarget); ok {
				if target, ok := targetVal.(string); ok && target != "" {
					transferStatus := "failed"
					if statusVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferStatus); ok {
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
								"direction": string(v.Direction),
								"call_id":   v.ID,
								"target":    target,
								"reason":    transferStatus,
							},
						},
						observability.RecordEvent{
							Component: observability.ComponentCall,
							Event:     observability.CallEnded,
							Attributes: observability.Attributes{
								"provider":    "sip",
								"direction":   string(v.Direction),
								"call_id":     v.ID,
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
								CallID:               v.ID,
								To:                   setup.CallContext.CallerNumber,
								From:                 setup.CallContext.FromNumber,
								Direction:            string(v.Direction),
								ContextID:            contextID,
								DurationMs:           fmt.Sprintf("%d", durationMs),
								Reason:               "transfer_" + transferStatus,
								Status:               observability.MetricCallStatusComplete,
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
					d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
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
						"direction":   string(v.Direction),
						"call_id":     v.ID,
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
						CallID:               v.ID,
						To:                   setup.CallContext.CallerNumber,
						From:                 setup.CallContext.FromNumber,
						Direction:            string(v.Direction),
						ContextID:            contextID,
						DurationMs:           fmt.Sprintf("%d", durationMs),
						Reason:               "talk_completed",
						Status:               observability.MetricCallStatusComplete,
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
			d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
			return
		}

		runtime, err := d.prepareSIPCallRuntime(ctx, v.Session, setup, observer, v.VaultCredential, v.Config, string(v.Direction), v.FromIdentity, v.ToIdentity)
		if err != nil {
			if v.Session.GetInfo().Direction == sip_infra.CallDirectionOutbound && !v.Session.IsEnded() {
				state := v.Session.GetState()
				if state != sip_infra.CallStateTransferring && state != sip_infra.CallStateBridgeConnected {
					d.endCall(v.Session, sip_infra.LifecycleReasonPipelineTalkCompleted)
				}
			}
			if targetVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferTarget); ok {
				if target, ok := targetVal.(string); ok && target != "" {
					transferStatus := "failed"
					if statusVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferStatus); ok {
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
							"direction": string(v.Direction),
							"call_id":   v.ID,
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
								"direction":   string(v.Direction),
								"call_id":     v.ID,
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
								Provider:   "sip",
								CallID:     v.ID,
								To:         setup.CallContext.CallerNumber,
								From:       setup.CallContext.FromNumber,
								Direction:  string(v.Direction),
								ContextID:  contextID,
								Reason:     reason,
								DurationMs: fmt.Sprintf("%d", durationMs),
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
					d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
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
						"direction":   string(v.Direction),
						"call_id":     v.ID,
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
						Provider:   "sip",
						CallID:     v.ID,
						To:         setup.CallContext.CallerNumber,
						From:       setup.CallContext.FromNumber,
						Direction:  string(v.Direction),
						ContextID:  contextID,
						Reason:     reason,
						DurationMs: fmt.Sprintf("%d", durationMs),
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
			d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
			return
		}
		if err := runtime.Start(ctx); err != nil {
			if v.Session.GetInfo().Direction == sip_infra.CallDirectionOutbound && !v.Session.IsEnded() {
				state := v.Session.GetState()
				if state != sip_infra.CallStateTransferring && state != sip_infra.CallStateBridgeConnected {
					d.endCall(v.Session, sip_infra.LifecycleReasonPipelineTalkCompleted)
				}
			}
			if targetVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferTarget); ok {
				if target, ok := targetVal.(string); ok && target != "" {
					transferStatus := "failed"
					if statusVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferStatus); ok {
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
							"direction": string(v.Direction),
							"call_id":   v.ID,
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
								"direction":   string(v.Direction),
								"call_id":     v.ID,
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
								Provider:   "sip",
								CallID:     v.ID,
								To:         setup.CallContext.CallerNumber,
								From:       setup.CallContext.FromNumber,
								Direction:  string(v.Direction),
								ContextID:  contextID,
								Reason:     reason,
								DurationMs: fmt.Sprintf("%d", durationMs),
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
					d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
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
						"direction":   string(v.Direction),
						"call_id":     v.ID,
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
						Provider:   "sip",
						CallID:     v.ID,
						To:         setup.CallContext.CallerNumber,
						From:       setup.CallContext.FromNumber,
						Direction:  string(v.Direction),
						ContextID:  contextID,
						Reason:     reason,
						DurationMs: fmt.Sprintf("%d", durationMs),
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
			d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
			return
		}
		if v.Session.GetInfo().Direction == sip_infra.CallDirectionOutbound && !v.Session.IsEnded() {
			state := v.Session.GetState()
			if state != sip_infra.CallStateTransferring && state != sip_infra.CallStateBridgeConnected {
				d.endCall(v.Session, sip_infra.LifecycleReasonPipelineTalkCompleted)
			}
		}
		if targetVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferTarget); ok {
			if target, ok := targetVal.(string); ok && target != "" {
				transferStatus := "failed"
				if statusVal, ok := v.Session.GetMetadata(sip_infra.MetadataBridgeTransferStatus); ok {
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
						"direction": string(v.Direction),
						"call_id":   v.ID,
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
							"direction":   string(v.Direction),
							"call_id":     v.ID,
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
							CallID:               v.ID,
							To:                   setup.CallContext.CallerNumber,
							From:                 setup.CallContext.FromNumber,
							Direction:            string(v.Direction),
							ContextID:            contextID,
							DurationMs:           fmt.Sprintf("%d", durationMs),
							Reason:               reason,
							Status:               observability.MetricCallStatusComplete,
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
				d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
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
					"direction":   string(v.Direction),
					"call_id":     v.ID,
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
					CallID:               v.ID,
					To:                   setup.CallContext.CallerNumber,
					From:                 setup.CallContext.FromNumber,
					Direction:            string(v.Direction),
					ContextID:            contextID,
					DurationMs:           fmt.Sprintf("%d", durationMs),
					Reason:               reason,
					Status:               observability.MetricCallStatusComplete,
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
		d.endCall(v.Session, sip_infra.LifecycleReasonPipelineCallEnd)
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

func sessionPreparationReason(err error) sip_infra.LifecycleReason {
	var preparationErr *sessionPreparationError
	if errors.As(err, &preparationErr) {
		return preparationErr.reason
	}
	return sip_infra.LifecycleReasonPipelineSetupFailed
}
