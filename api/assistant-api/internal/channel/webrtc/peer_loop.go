// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"fmt"
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/protos"
)

func (s *webrtcStreamer) runPeerEventLoop() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		case event := <-s.peerEventCh:
			switch event.Kind {
			case webrtc_internal.SignalEventClientMessage:
				s.handleClientSignal(event.SignalClientMessage)
			case webrtc_internal.PeerEventStateChanged:
				s.handlePeerState(event.MediaSessionID, event.PeerState, event.PeerStateChangedAt)
			case webrtc_internal.PeerEventICEConnectionStateChanged:
				s.handlePeerICEConnectionState(event.MediaSessionID, event.PeerICEState, event.PeerICEStateChangedAt)
			}
		}
	}
}

func (s *webrtcStreamer) enqueuePeerEvent(event webrtc_internal.PeerEvent) {
	if s.peerEventCh == nil {
		switch event.Kind {
		case webrtc_internal.SignalEventClientMessage:
			s.handleClientSignal(event.SignalClientMessage)
		case webrtc_internal.PeerEventStateChanged:
			s.handlePeerState(event.MediaSessionID, event.PeerState, event.PeerStateChangedAt)
		case webrtc_internal.PeerEventICEConnectionStateChanged:
			s.handlePeerICEConnectionState(event.MediaSessionID, event.PeerICEState, event.PeerICEStateChangedAt)
		}
		return
	}

	select {
	case s.peerEventCh <- event:
	case <-s.Ctx.Done():
	}
}

func (s *webrtcStreamer) handlePeerState(mediaSessionID uint64, state pionwebrtc.PeerConnectionState, peerStateChangedAt time.Time) {
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	s.Mu.Lock()
	switch state {
	case pionwebrtc.PeerConnectionStateConnected:
		s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
		s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioConnected)
		s.mediaHealthState.RecordPeerConnected(peerStateChangedAt)
	case pionwebrtc.PeerConnectionStateFailed,
		pionwebrtc.PeerConnectionStateDisconnected:
		s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioNegotiating)
		if state == pionwebrtc.PeerConnectionStateFailed {
			s.mediaHealthState.RecordPeerFailed(peerStateChangedAt)
		} else {
			s.mediaHealthState.RecordPeerDisconnected(peerStateChangedAt)
		}
	case pionwebrtc.PeerConnectionStateClosed:
		s.currentMode = protos.StreamMode_STREAM_MODE_TEXT
		s.sessionState.SetMediaState(webrtc_internal.MediaStateText)
		s.mediaHealthState.RecordPeerClosed(peerStateChangedAt)
	}
	iceLatencyMs := peerStateChangedAt.Sub(s.mediaHealthState.ICEStartedAt).Milliseconds()
	peerConnection := s.peerConnection
	s.Mu.Unlock()

	switch state {
	case pionwebrtc.PeerConnectionStateConnected:
		s.sessionState.SetPeerConnected(true)
		s.sessionState.ResetICERestartAttempts()
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope,
			observability.RecordEvent{
				Component: observability.ComponentWebRTC,
				Event:     observability.WebRTCConnected,
				Attributes: observability.Attributes{
					"component":                      observability.ComponentWebRTC.String(),
					webrtc_internal.DataType:         "peer_connected",
					webrtc_internal.DataSessionID:    s.sessionID,
					webrtc_internal.DataICELatencyMs: fmt.Sprintf("%d", iceLatencyMs),
				},
			}, observability.RecordWebhook{
				Event: observability.WebRTCConnected,
				Payload: observability.WebRTCConnectedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					SessionID:            s.sessionID,
					MediaSessionID:       mediaSessionID,
					ICELatencyMs:         iceLatencyMs,
					PeerConnectionState:  state.String(),
				},
			}, observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricICELatencyMs,
					Value:       fmt.Sprintf("%d", iceLatencyMs),
					Description: "WebRTC ICE connection latency in milliseconds",
				}},
			})
		s.reportSelectedICECandidatePair(peerConnection, peerStateChangedAt)
		s.signalReady()

	case pionwebrtc.PeerConnectionStateFailed:
		s.sessionState.SetPeerConnected(false)
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope,
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "WebRTC peer failed, restarting ICE",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataType:      "peer_failed",
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataReason:    webrtc_internal.ReasonPeerFailed,
				},
			},
			observability.RecordEvent{
				Component: observability.ComponentWebRTC,
				Event:     observability.WebRTCFailed,
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataType:      "peer_failed",
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataReason:    webrtc_internal.ReasonPeerFailed,
				},
			},
			observability.RecordWebhook{
				Event: observability.WebRTCFailed,
				Payload: observability.WebRTCFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Type:                 "peer_failed",
					SessionID:            s.sessionID,
					MediaSessionID:       mediaSessionID,
					Reason:               webrtc_internal.ReasonPeerFailed,
					PeerConnectionState:  state.String(),
				},
			})
		s.queueMediaSessionRecovery(mediaSessionID, webrtc_internal.ReasonPeerFailed, peerStateChangedAt)

	case pionwebrtc.PeerConnectionStateDisconnected:
		s.sessionState.SetPeerConnected(false)
		s.Logger.Debugw("webrtc webhook record call site",
			"call_site", "peer_state_disconnected",
			"event", observability.WebRTCDisconnected.String(),
			webrtc_internal.DataSessionID, s.sessionID,
			webrtc_internal.DataMediaSessionID, mediaSessionID,
			webrtc_internal.DataPeerConnectionState, state.String(),
		)
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope,
			observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "WebRTC peer disconnected, restarting ICE",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataType:      "peer_disconnected",
					webrtc_internal.DataSessionID: s.sessionID,
				},
			},
			observability.RecordEvent{
				Component: observability.ComponentWebRTC,
				Event:     observability.WebRTCDisconnected,
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataType:      "peer_disconnected",
					webrtc_internal.DataSessionID: s.sessionID,
				},
			},
			observability.RecordWebhook{
				Event: observability.WebRTCDisconnected,
				Payload: observability.WebRTCDisconnectedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Type:                 "peer_disconnected",
					SessionID:            s.sessionID,
					MediaSessionID:       mediaSessionID,
					Reason:               webrtc_internal.ReasonPeerDisconnected,
					PeerConnectionState:  state.String(),
				},
			},
		)
		s.queueMediaSessionRecovery(mediaSessionID, webrtc_internal.ReasonPeerDisconnected, peerStateChangedAt)

	case pionwebrtc.PeerConnectionStateClosed:
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "WebRTC peer closed, resetting audio",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
			},
		})
		s.stopMediaSessionAndFallbackToText()
	}
}

func (s *webrtcStreamer) handlePeerICEConnectionState(mediaSessionID uint64, state pionwebrtc.ICEConnectionState, iceStateChangedAt time.Time) {
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	stateName := state.String()
	s.Mu.Lock()
	s.mediaHealthState.RecordICEConnectionState(stateName, iceStateChangedAt)
	s.Mu.Unlock()

	eventType := webrtc_internal.EventICEConnectionState
	eventName := observability.WebRTCICEConnectionState
	if state == pionwebrtc.ICEConnectionStateConnected || state == pionwebrtc.ICEConnectionStateCompleted {
		eventType = "ice_connected"
		eventName = observability.WebRTCICEConnected
	}
	if state == pionwebrtc.ICEConnectionStateFailed {
		eventType = "ice_failed"
		eventName = observability.WebRTCICEFailed
	}

	_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordEvent{
		Component: observability.ComponentWebRTC,
		Event:     eventName,
		Attributes: observability.Attributes{
			"component":                            observability.ComponentWebRTC.String(),
			webrtc_internal.DataType:               eventType,
			webrtc_internal.DataSessionID:          s.sessionID,
			webrtc_internal.DataICEConnectionState: stateName,
		},
	})

	if state == pionwebrtc.ICEConnectionStateConnected || state == pionwebrtc.ICEConnectionStateCompleted {
		s.sessionState.ResetICERestartAttempts()
	}
	if state == pionwebrtc.ICEConnectionStateFailed {
		s.sessionState.SetPeerConnected(false)
		s.queueMediaSessionRecovery(mediaSessionID, webrtc_internal.ReasonICEFailed, iceStateChangedAt)
	}
}

func (s *webrtcStreamer) reportSelectedICECandidatePair(peerConnection *pionwebrtc.PeerConnection, selectedAt time.Time) {
	if peerConnection == nil {
		return
	}
	pair, ok := selectedICECandidatePairFromStats(peerConnection.GetStats())
	if !ok {
		return
	}

	s.Mu.Lock()
	changed := s.mediaHealthState.RecordSelectedICECandidatePair(pair, selectedAt)
	qualityState := s.mediaHealthState.QualityState(selectedAt)
	s.Mu.Unlock()
	if !changed {
		return
	}

	_ = s.observer.Record(s.Ctx, s.sessionState.Scope,
		observability.RecordEvent{
			Component: observability.ComponentWebRTC,
			Event:     observability.WebRTCSelectedICECandidatePair,
			Attributes: observability.Attributes{
				"component":                                     observability.ComponentWebRTC.String(),
				webrtc_internal.DataType:                        webrtc_internal.EventSelectedICECandidatePair,
				webrtc_internal.DataSessionID:                   s.sessionID,
				webrtc_internal.DataCandidatePairID:             pair.ID,
				webrtc_internal.DataLocalCandidateType:          pair.LocalCandidateType,
				webrtc_internal.DataLocalProtocol:               pair.LocalProtocol,
				webrtc_internal.DataRemoteCandidateType:         pair.RemoteCandidateType,
				webrtc_internal.DataRemoteProtocol:              pair.RemoteProtocol,
				webrtc_internal.DataCandidatePairRTTMs:          fmt.Sprintf("%d", pair.CurrentRoundTripTimeMs),
				webrtc_internal.DataAvailableOutgoingBitrateBps: fmt.Sprintf("%d", pair.AvailableOutgoingBitrateBps),
				webrtc_internal.DataQualityState:                qualityState,
			},
		},
		observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "WebRTC selected ICE candidate pair changed; RTT and outgoing bitrate help diagnose network path changes that can affect audio latency or quality.",
			Attributes: observability.Attributes{
				"component":                                     observability.ComponentWebRTC.String(),
				webrtc_internal.DataType:                        webrtc_internal.EventSelectedICECandidatePair,
				webrtc_internal.DataSessionID:                   s.sessionID,
				webrtc_internal.DataCandidatePairID:             pair.ID,
				webrtc_internal.DataCandidatePairRTTMs:          fmt.Sprintf("%d", pair.CurrentRoundTripTimeMs),
				webrtc_internal.DataAvailableOutgoingBitrateBps: fmt.Sprintf("%d", pair.AvailableOutgoingBitrateBps),
				webrtc_internal.DataQualityState:                qualityState,
			},
		})
}

func selectedICECandidatePairFromStats(report pionwebrtc.StatsReport) (webrtc_internal.SelectedICECandidatePair, bool) {
	candidates := make(map[string]pionwebrtc.ICECandidateStats)
	var selectedPair pionwebrtc.ICECandidatePairStats
	selected := false

	for _, stat := range report {
		switch typed := stat.(type) {
		case pionwebrtc.ICECandidateStats:
			candidates[typed.ID] = typed
		case pionwebrtc.ICECandidatePairStats:
			if typed.Nominated && typed.State == pionwebrtc.StatsICECandidatePairStateSucceeded {
				selectedPair = typed
				selected = true
			}
		}
	}
	if !selected {
		return webrtc_internal.SelectedICECandidatePair{}, false
	}

	localCandidate := candidates[selectedPair.LocalCandidateID]
	remoteCandidate := candidates[selectedPair.RemoteCandidateID]
	return webrtc_internal.SelectedICECandidatePair{
		ID:                          selectedPair.ID,
		LocalCandidateType:          localCandidate.CandidateType.String(),
		LocalProtocol:               localCandidate.Protocol,
		RemoteCandidateType:         remoteCandidate.CandidateType.String(),
		RemoteProtocol:              remoteCandidate.Protocol,
		CurrentRoundTripTimeMs:      int64(selectedPair.CurrentRoundTripTime * float64(webrtc_internal.MillisecondsPerSecond)),
		AvailableOutgoingBitrateBps: int64(selectedPair.AvailableOutgoingBitrate),
	}, true
}

// handleClientSignal applies already-ordered SDP and ICE updates from the browser.
func (s *webrtcStreamer) handleClientSignal(signaling *protos.ClientSignaling) {
	if signaling == nil {
		return
	}

	s.Mu.Lock()
	signalingSessionID := s.signalingSessionID
	mediaSessionID := s.sessionState.ActiveMediaSessionID()
	s.Mu.Unlock()

	switch msg := signaling.GetMessage().(type) {
	case *protos.ClientSignaling_Sdp:
		if signalingSessionID != "" && signaling.GetSessionId() != signalingSessionID {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Received SDP for stale WebRTC signaling session, ignoring",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					"signaling_session_id":        signaling.GetSessionId(),
					"current_signaling_session":   signalingSessionID,
				},
			})
			return
		}
		if msg.Sdp.GetType() == protos.WebRTCSDP_ANSWER {
			s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
				Kind:            webrtc_internal.WebRTCOperationApplyRemoteAnswer,
				MediaSessionID:  mediaSessionID,
				RemoteAnswerSDP: msg.Sdp.GetSdp(),
			})
		}

	case *protos.ClientSignaling_IceCandidate:
		if signalingSessionID != "" && signaling.GetSessionId() != signalingSessionID {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Received ICE candidate for stale WebRTC signaling session, ignoring",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					"signaling_session_id":        signaling.GetSessionId(),
					"current_signaling_session":   signalingSessionID,
				},
			})
			return
		}
		ice := msg.IceCandidate
		if ice == nil || ice.GetCandidate() == "" {
			return
		}
		idx := uint16(ice.GetSdpMLineIndex())
		sdpMid := ice.GetSdpMid()
		usernameFragment := ice.GetUsernameFragment()
		candidate := pionwebrtc.ICECandidateInit{
			Candidate:        ice.GetCandidate(),
			SDPMid:           &sdpMid,
			SDPMLineIndex:    &idx,
			UsernameFragment: &usernameFragment,
		}
		s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
			Kind:               webrtc_internal.WebRTCOperationAddRemoteICECandidate,
			MediaSessionID:     mediaSessionID,
			RemoteICECandidate: candidate,
		})

	case *protos.ClientSignaling_Disconnect:
		if msg.Disconnect {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordMetadata{
				Metadata: observability.DisconnectMetadata(
					protos.ConversationDisconnection_DISCONNECTION_TYPE_USER.String(),
					"client_signaling_disconnect",
				),
			})
			if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
				s.Input(disc)
			}
			s.Close()
		}
	}
}

func (s *webrtcStreamer) addRemoteICECandidate(peerConnection *pionwebrtc.PeerConnection, candidate pionwebrtc.ICECandidateInit) {
	if err := peerConnection.AddICECandidate(candidate); err != nil {
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Failed to add ICE candidate",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				"error":                       err.Error(),
			},
		})
	}
}
