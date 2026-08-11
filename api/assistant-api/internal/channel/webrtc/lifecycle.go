// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"context"
	"fmt"
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/protos"
)

func (s *webrtcStreamer) watchCallerContext(callerCtx context.Context) {
	select {
	case <-callerCtx.Done():
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "Caller context cancelled, closing WebRTC streamer gracefully",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
			},
		})
		if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
			s.Input(disc)
		}
		s.Close()
	case <-s.Ctx.Done():
	}
}

func (s *webrtcStreamer) runMediaLifecycleLoop() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		case event := <-s.mediaLifecycleCh:
			switch event.Kind {
			case webrtc_internal.MediaLifecycleEventRestart:
				s.restartMediaSessionOrFallbackToText(event.MediaSessionID, event.Reason, event.RequestedAt)
			case webrtc_internal.MediaLifecycleEventRecover:
				s.restartICEOrMediaSessionFallback(event.MediaSessionID, event.Reason, event.RequestedAt)
			}
		}
	}
}

func (s *webrtcStreamer) queueMediaSessionRecovery(mediaSessionID uint64, reason string, requestedAt time.Time) {
	event := webrtc_internal.MediaLifecycleEvent{
		Kind:           webrtc_internal.MediaLifecycleEventRecover,
		MediaSessionID: mediaSessionID,
		Reason:         reason,
		RequestedAt:    requestedAt,
	}
	if s.mediaLifecycleCh == nil {
		go s.restartICEOrMediaSessionFallback(mediaSessionID, reason, requestedAt)
		return
	}

	select {
	case s.mediaLifecycleCh <- event:
	case <-s.Ctx.Done():
	default:
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC media lifecycle queue full, dropping recovery request",
			Attributes: observability.Attributes{
				"component":                        observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
				webrtc_internal.DataReason:         reason,
			},
		})
	}
}

func (s *webrtcStreamer) queueMediaSessionRestart(mediaSessionID uint64, reason string, requestedAt time.Time) {
	event := webrtc_internal.MediaLifecycleEvent{
		Kind:           webrtc_internal.MediaLifecycleEventRestart,
		MediaSessionID: mediaSessionID,
		Reason:         reason,
		RequestedAt:    requestedAt,
	}
	if s.mediaLifecycleCh == nil {
		go s.restartMediaSessionOrFallbackToText(mediaSessionID, reason, requestedAt)
		return
	}

	select {
	case s.mediaLifecycleCh <- event:
	case <-s.Ctx.Done():
	default:
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC media lifecycle queue full, dropping restart request",
			Attributes: observability.Attributes{
				"component":                        observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
				webrtc_internal.DataReason:         reason,
			},
		})
	}
}

func (s *webrtcStreamer) runMediaSessionDeadlines(mediaSessionID uint64) {
	s.Mu.Lock()
	mediaCtx := s.mediaCtx
	s.Mu.Unlock()
	if mediaCtx == nil {
		return
	}

	ticker := time.NewTicker(webrtc_internal.HandshakeDeadlineCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.Ctx.Done():
			return
		case <-mediaCtx.Done():
			return
		case deadlineCheckedAt := <-ticker.C:
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) || s.sessionState.PeerConnected() {
				return
			}

			s.Mu.Lock()
			mediaHealthState := s.mediaHealthState
			s.Mu.Unlock()

			reason, deadline, elapsed, exceeded := mediaHealthState.HandshakeDeadlineExceeded(deadlineCheckedAt)
			if !exceeded {
				continue
			}

			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelError,
				Message: "WebRTC handshake deadline exceeded",
				Attributes: observability.Attributes{
					"component":                    observability.ComponentWebRTC.String(),
					webrtc_internal.DataType:       webrtc_internal.EventHandshakeDeadlineExceeded,
					webrtc_internal.DataSessionID:  s.sessionID,
					webrtc_internal.DataReason:     reason,
					webrtc_internal.DataDeadline:   reason,
					webrtc_internal.DataDeadlineMs: fmt.Sprintf("%d", deadline.Milliseconds()),
					webrtc_internal.DataElapsedMs:  fmt.Sprintf("%d", elapsed.Milliseconds()),
				},
			})
			s.queueMediaSessionRestart(mediaSessionID, reason, deadlineCheckedAt)
			return
		}
	}
}

func (s *webrtcStreamer) restartICEOrMediaSessionFallback(mediaSessionID uint64, reason string, restartedAt time.Time) {
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) || s.sessionState.PeerConnected() {
		return
	}

	if s.sessionState.ICERestartPending() || s.sessionState.DeferredICERestartPending(mediaSessionID) {
		return
	}

	iceRestartAttempt, ok := s.sessionState.TryBeginICERestart(webrtc_internal.ICERestartAttemptLimit)
	if !ok {
		s.restartMediaSessionOrFallbackToText(mediaSessionID, reason, restartedAt)
		return
	}
	s.Logger.Debugw("webrtc webhook record call site",
		"call_site", "ice_restart_reconnecting",
		"event", observability.WebRTCReconnecting.String(),
		webrtc_internal.DataSessionID, s.sessionID,
		webrtc_internal.DataMediaSessionID, mediaSessionID,
		webrtc_internal.DataReason, reason,
		webrtc_internal.DataRestartAttempt, iceRestartAttempt,
	)
	s.observer.Record(s.Ctx, s.sessionState.Scope,
		observability.RecordEvent{
			Component: observability.ComponentWebRTC,
			Event:     observability.WebRTCReconnecting,
			Attributes: observability.Attributes{
				"component":                        observability.ComponentWebRTC.String(),
				webrtc_internal.DataType:           webrtc_internal.EventICERestarting,
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataReason:         reason,
				webrtc_internal.DataRestartAttempt: fmt.Sprintf("%d", iceRestartAttempt),
				webrtc_internal.DataRestartLimit:   fmt.Sprintf("%d", webrtc_internal.ICERestartAttemptLimit),
			},
		},
		observability.RecordWebhook{
			Event: observability.WebRTCReconnecting,
			Payload: observability.WebRTCReconnectingWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				Type:                 webrtc_internal.EventICERestarting,
				SessionID:            s.sessionID,
				MediaSessionID:       mediaSessionID,
				Reason:               reason,
				RestartAttempt:       iceRestartAttempt,
				RestartLimit:         webrtc_internal.ICERestartAttemptLimit,
			},
		})

	s.clearBufferedOutputAudio()
	s.clearOutputAudio()

	s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioNegotiating)
	s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:           webrtc_internal.WebRTCOperationRestartICE,
		MediaSessionID: mediaSessionID,
		Reason:         reason,
		RequestedAt:    restartedAt,
		OfferOptions:   &pionwebrtc.OfferOptions{ICERestart: true},
	})
	go s.runMediaSessionDeadlines(mediaSessionID)
}

func (s *webrtcStreamer) restartMediaSessionOrFallbackToText(mediaSessionID uint64, reason string, restartedAt time.Time) {
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	restartAttempt, ok := s.sessionState.TryBeginMediaRestart(webrtc_internal.MediaRestartAttemptLimit)
	if !ok {
		s.observer.Record(s.Ctx, s.sessionState.Scope,
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "WebRTC media restart limit reached, falling back to text mode",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataReason:    reason,
				},
			},
			observability.RecordWebhook{
				Event: observability.WebRTCFailed,
				Payload: observability.WebRTCFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Type:                 "media_restart_limit_reached",
					SessionID:            s.sessionID,
					MediaSessionID:       mediaSessionID,
					Reason:               reason,
					Fallback:             "text",
				},
			})
		s.stopMediaSessionAndFallbackToText()
		return
	}

	s.observer.Record(s.Ctx, s.sessionState.Scope,
		observability.RecordEvent{
			Component: observability.ComponentWebRTC,
			Event:     observability.WebRTCReconnecting,
			Attributes: observability.Attributes{
				"component":                        observability.ComponentWebRTC.String(),
				webrtc_internal.DataType:           webrtc_internal.EventMediaSessionRestarting,
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataReason:         reason,
				webrtc_internal.DataRestartAttempt: fmt.Sprintf("%d", restartAttempt),
				webrtc_internal.DataRestartLimit:   fmt.Sprintf("%d", webrtc_internal.MediaRestartAttemptLimit),
			},
		},
		observability.RecordWebhook{
			Event: observability.WebRTCReconnecting,
			Payload: observability.WebRTCReconnectingWebhookPayload{
				V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
				Type:                 webrtc_internal.EventMediaSessionRestarting,
				SessionID:            s.sessionID,
				MediaSessionID:       mediaSessionID,
				Reason:               reason,
				RestartAttempt:       restartAttempt,
				RestartLimit:         webrtc_internal.MediaRestartAttemptLimit,
			},
		})

	s.clearBufferedOutputAudio()
	s.clearOutputAudio()
	if s.ambientMixer != nil {
		s.ambientMixer.Reset()
	}

	if err := s.startMediaSession(); err != nil {
		s.Logger.Debugw("webrtc webhook record call site",
			"call_site", "media_session_restart_failed",
			"event", observability.WebRTCFailed.String(),
			webrtc_internal.DataSessionID, s.sessionID,
			webrtc_internal.DataMediaSessionID, mediaSessionID,
			webrtc_internal.DataReason, reason,
			"error", err.Error(),
		)
		s.observer.Record(s.Ctx, s.sessionState.Scope,
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to restart WebRTC media session, falling back to text mode",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataReason:    reason,
					"error":                       err.Error(),
				},
			},
			observability.RecordWebhook{
				Event: observability.WebRTCFailed,
				Payload: observability.WebRTCFailedWebhookPayload{
					V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
					Type:                 "media_session_restart_failed",
					SessionID:            s.sessionID,
					MediaSessionID:       mediaSessionID,
					Reason:               reason,
					Error:                err.Error(),
					Fallback:             "text",
				},
			})
		s.stopMediaSessionAndFallbackToText()
	}
}
