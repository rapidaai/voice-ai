// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_deepgram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	deepgram_internal "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	utils "github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

/*
Deepgram Continuous Streaming TTS
Reference: https://developers.deepgram.com/reference/text-to-speech/speak-streaming
*/

type deepgramTTS struct {
	*deepgram_internal.DeepgramOption
	ctx            context.Context
	ctxCancel      context.CancelFunc
	contextId      string
	ttsConnectedAt time.Time
	mu             sync.Mutex

	ttsStartedAt  time.Time
	ttsMetricSent bool

	logger     commons.Logger
	connection *websocket.Conn
	onPacket   func(pkt ...internal_type.Packet) error
	normalizer internal_type.TextNormalizer
}

func NewDeepgramTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {

	dGoptions, err := deepgram_internal.NewDeepgramOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("deepgram-tts: error while intializing deepgram text to speech")
		return nil, err
	}
	ctx2, cancel := context.WithCancel(ctx)
	return &deepgramTTS{
		DeepgramOption: dGoptions,
		ctx:            ctx2,
		ctxCancel:      cancel,
		logger:         logger,
		onPacket:       onPacket,
		normalizer:     deepgram_internal.NewDeepgramNormalizer(logger, opts),
	}, nil
}

// Initialize opens a fresh WebSocket connection to Deepgram and starts the
// read goroutine. Called at session start and after each interruption so the
// connection is warm before the first text delta arrives.
func (t *deepgramTTS) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("token %s", t.GetKey()))
	conn, resp, err := websocket.DefaultDialer.Dial(t.GetTextToSpeechConnectionString(), header)
	if err != nil {
		t.logger.Errorf("deepgram-tts: websocket dial failed err=%v resp=%v", err, resp)
		t.onPacket(
			internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "deepgram-tts: error while performing connect",
					Attributes: observability.Attributes{
						"component": observability.ComponentTTS.String(),
						"provider":  t.Name(),
						"path":      observability.AttributeValue(t.GetTextToSpeechConnectionString()),
					},
					OccurredAt: time.Now(),
				},
			})
		return err
	}

	t.mu.Lock()
	t.connection = conn
	if t.ttsConnectedAt.IsZero() {
		t.ttsConnectedAt = time.Now()
	}
	t.mu.Unlock()
	go t.readLoop(conn)
	t.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope:  internal_type.ObservabilityRecordScopeConversation,
			Record: observability.NewMetricTTSInitLatencyMs(time.Since(start), observability.Attributes{"provider": t.Name()}),
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "deepgram-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  t.Name(),
					"path":      observability.AttributeValue(t.GetTextToSpeechConnectionString()),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

func (*deepgramTTS) Name() string {
	return deepgram_internal.DeepgramTextToSpeechTransformerName
}

// handleFlushComplete is called when Deepgram signals Flushed. It emits
// TextToSpeechEndPacket — correctly ordered after the last audio chunk — and
// closes the per-turn connection.
func (t *deepgramTTS) handleFlushComplete(conn *websocket.Conn) {
	t.mu.Lock()
	ctxId := t.contextId
	t.connection = nil // mark before Close so readLoop error handler sees intentional
	t.mu.Unlock()
	t.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: ctxId},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxId,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSCompleted,
				Attributes: observability.Attributes{"type": "completed"},
				OccurredAt: time.Now(),
			},
		},
	)
	conn.Close()
}

// readLoop owns a single WebSocket connection for the duration of one TTS turn.
// It exits when the connection closes — intentionally (interrupt / flush complete)
// or unexpectedly (network drop).
func (t *deepgramTTS) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		msgType, data, err := conn.ReadMessage()
		if err != nil {
			t.mu.Lock()
			if t.connection != conn {
				t.mu.Unlock()
				return
			}
			// Active connection dropped; next text packet reconnects.
			t.connection = nil
			t.mu.Unlock()
			t.logger.Errorf("deepgram-tts: connection lost: %v", err)
			return
		}

		if msgType == websocket.BinaryMessage {
			var shouldEmitFirstAudioLatencyMetric bool
			t.mu.Lock()
			ttsStartedAt := t.ttsStartedAt
			contextId := t.contextId
			if !t.ttsMetricSent && !ttsStartedAt.IsZero() {
				t.ttsMetricSent = true
				shouldEmitFirstAudioLatencyMetric = true
			}
			t.mu.Unlock()
			if shouldEmitFirstAudioLatencyMetric {
				t.onPacket(internal_type.ObservabilityMetricRecordPacket{
					ContextID: contextId,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": t.Name()}),
				})
			}
			t.onPacket(internal_type.TextToSpeechAudioPacket{
				ContextID:  contextId,
				AudioChunk: data,
			})
			continue
		}

		var envelope *deepgram_internal.DeepgramTextToSpeechResponse
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}

		switch envelope.Type {
		case "Metadata":
			continue
		case "Flushed":
			t.handleFlushComplete(conn)
			return
		case "Cleared":
			continue
		case "Warning":
			t.logger.Warnf("deepgram-tts warning code=%s message=%s", envelope.Code, envelope.Message)
		default:
			t.logger.Debugf("deepgram-tts: unhandled message type: %s", envelope.Type)
		}
	}
}

// Transform streams text into Deepgram
func (t *deepgramTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	t.mu.Lock()
	if in.ContextId() != t.contextId {
		t.contextId = in.ContextId()
		t.ttsStartedAt = time.Time{}
		t.ttsMetricSent = false
	}
	connection := t.connection
	t.mu.Unlock()

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		t.mu.Lock()
		t.contextId = ""
		t.ttsStartedAt = time.Time{}
		t.ttsMetricSent = false
		conn := t.connection
		t.connection = nil
		t.mu.Unlock()
		if conn != nil {
			conn.Close()
		}
		t.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"},
				OccurredAt: time.Now(),
			},
		})
		if err := t.Initialize(); err != nil {
			t.logger.Errorf("deepgram-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.TextToSpeechTextPacket:
		if connection == nil {
			if err := t.Initialize(); err != nil {
				t.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("deepgram-tts: failed to connect: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				})
				return nil
			}
			t.mu.Lock()
			connection = t.connection
			t.mu.Unlock()
		}
		t.mu.Lock()
		if t.ttsStartedAt.IsZero() {
			t.ttsStartedAt = time.Now()
		}
		t.mu.Unlock()
		normalized := t.normalizer.Normalize(input.Text)
		if err := connection.WriteJSON(map[string]interface{}{
			"type": "Speak",
			"text": normalized,
		}); err != nil {
			t.logger.Errorf("deepgram-tts: failed to send Speak message %v", err)
			t.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("deepgram-tts: failed to send Speak message: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		t.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSSpeaking,
				Attributes: observability.Attributes{
					"type": "speaking",
					"text": normalized,
				},
				OccurredAt: time.Now(),
			},
		})
		return nil

	case internal_type.TextToSpeechDonePacket:
		// Interrupted before done arrived — nothing to flush.
		if connection == nil {
			return nil
		}
		// Signal end of text stream; Deepgram will respond with Flushed.
		if err := connection.WriteJSON(map[string]string{"type": "Flush"}); err != nil {
			t.logger.Errorf("deepgram-tts: failed to send Flush %v", err)
			t.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("deepgram-tts: failed to send Flush: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		// TextToSpeechEndPacket is emitted by handleFlushComplete once Flushed received.
		return nil

	default:
		return fmt.Errorf("deepgram-tts: unsupported input type %T", in)
	}
}

// Close gracefully closes the Deepgram connection
func (t *deepgramTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.mu.Lock()
	connectedAt := t.ttsConnectedAt
	t.ttsConnectedAt = time.Time{}

	if t.connection != nil {
		conn := t.connection
		t.connection = nil
		_ = conn.WriteJSON(map[string]string{"type": "Close"})
		conn.Close()
	}
	t.mu.Unlock()
	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		t.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": t.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewTTSDurationUsageRecord(t.Name(), duration, observability.Attributes{}),
			},
		)
	}
	t.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": t.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
