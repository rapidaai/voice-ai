// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_elevenlabs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"

	elevenlabs_internal "github.com/rapidaai/api/assistant-api/internal/transformer/elevenlabs/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type elevenlabsTTS struct {
	*elevenLabsOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	connection     *websocket.Conn
	contextId      string
	ttsConnectedAt time.Time
	ttsStartedAt   time.Time
	ttsMetricSent  bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewElevenlabsTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	eleOpts, err := NewElevenLabsOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("elevenlabs-tts: initializing elevenlabs failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &elevenlabsTTS{
		ctx:              ctx2,
		ctxCancel:        contextCancel,
		onPacket:         onPacket,
		logger:           logger,
		elevenLabsOption: eleOpts,
	}, nil
}

func (*elevenlabsTTS) Name() string {
	return "elevenlabs-tts"
}

// Initialize opens a fresh WebSocket connection to ElevenLabs and starts the
// read goroutine. Called at session start and after each interruption so the
// connection is warm before the first text delta arrives.
func (ct *elevenlabsTTS) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("xi-api-key", ct.GetKey())
	conn, resp, err := websocket.DefaultDialer.Dial(ct.GetTextToSpeechConnectionString(), header)
	if err != nil {
		ct.logger.Errorf("elevenlabs-tts: dial failed %s with response %v", err, resp)
		ct.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "elevenlabs-tts: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(ct.GetTextToSpeechConnectionString()),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	ct.mu.Lock()
	ct.connection = conn
	if ct.ttsConnectedAt.IsZero() {
		ct.ttsConnectedAt = time.Now()
	}
	ct.mu.Unlock()

	go ct.readLoop(conn)
	ct.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": ct.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "elevenlabs-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(ct.GetTextToSpeechConnectionString()),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

// readLoop owns a single WebSocket connection for the duration of one TTS turn.
// It exits when the connection closes — intentionally (interrupt / flush complete)
// or unexpectedly (network drop).
func (elt *elevenlabsTTS) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-elt.ctx.Done():
			return
		default:
		}
		_, audioChunk, err := conn.ReadMessage()
		if err != nil {
			elt.mu.Lock()
			if elt.connection != conn {
				elt.mu.Unlock()
				return
			}
			// Active connection dropped; next text packet reconnects.
			elt.connection = nil
			elt.mu.Unlock()
			elt.logger.Errorf("elevenlabs-tts: connection lost: %v", err)
			return
		}
		var audioData elevenlabs_internal.ElevenlabTextToSpeechResponse
		if err := json.Unmarshal(audioChunk, &audioData); err != nil {
			elt.logger.Errorf("elevenlabs-tts: error parsing audio chunk: %v", err)
			continue
		}

		if audioData.Audio != "" {
			if rawAudioData, err := base64.StdEncoding.DecodeString(audioData.Audio); err == nil {
				var shouldEmitFirstAudioLatencyMetric bool
				elt.mu.Lock()
				contextId := elt.contextId
				ttsStartedAt := elt.ttsStartedAt
				if !elt.ttsMetricSent && !ttsStartedAt.IsZero() {
					elt.ttsMetricSent = true
					shouldEmitFirstAudioLatencyMetric = true
				}
				elt.mu.Unlock()
				if shouldEmitFirstAudioLatencyMetric {
					elt.onPacket(internal_type.ObservabilityMetricRecordPacket{
						ContextID: contextId,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": elt.Name()}),
					})
				}
				elt.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: contextId, AudioChunk: rawAudioData})
			} else {
				elt.logger.Errorf("elevenlabs-tts: base64 decode failed: %v", err)
			}
		}

		if audioData.IsFinal != nil && *audioData.IsFinal {
			elt.handleFlushComplete(conn)
			return
		}
	}
}

// handleFlushComplete is called when ElevenLabs signals isFinal. It emits
// TextToSpeechEndPacket — correctly ordered after the last audio chunk — and
// closes the per-turn connection.
func (elt *elevenlabsTTS) handleFlushComplete(conn *websocket.Conn) {
	elt.mu.Lock()
	ctxId := elt.contextId
	elt.connection = nil // mark before Close so readLoop error handler sees intentional
	elt.mu.Unlock()

	elt.onPacket(
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

func (t *elevenlabsTTS) Transform(ctx context.Context, in internal_type.Packet) error {
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
			t.logger.Errorf("elevenlabs-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.TextToSpeechTextPacket:
		// Fallback reconnect: handles Initialize() failure or an unintentional drop.
		if connection == nil {
			if err := t.Initialize(); err != nil {
				t.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: input.ContextID, Error: fmt.Errorf("elevenlabs-tts: failed to connect: %w", err), Type: internal_type.TTSNetworkTimeout})
				return nil
			}
			t.mu.Lock()
			connection = t.connection
			if t.ttsStartedAt.IsZero() {
				t.ttsStartedAt = time.Now()
			}
			t.mu.Unlock()
		} else {
			t.mu.Lock()
			if t.ttsStartedAt.IsZero() {
				t.ttsStartedAt = time.Now()
			}
			t.mu.Unlock()
		}
		t.mu.Lock()
		ctxId := t.contextId
		t.mu.Unlock()
		if err := connection.WriteJSON(map[string]interface{}{
			"text":       input.Text,
			"context_id": ctxId,
			"flush":      true,
		}); err != nil {
			t.logger.Errorf("elevenlabs-tts: write failed: %v", err)
			t.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: input.ContextID, Error: fmt.Errorf("elevenlabs-tts: send failed: %w", err), Type: internal_type.TTSNetworkTimeout})
			return nil
		}
		t.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSSpeaking,
				Attributes: observability.Attributes{"type": "speaking", "text": input.Text},
				OccurredAt: time.Now(),
			},
		})

	case internal_type.TextToSpeechDonePacket:
		// Interrupted before done arrived — nothing to flush.
		if connection == nil {
			return nil
		}
		t.mu.Lock()
		ctxId := t.contextId
		t.mu.Unlock()
		// Signal end of text stream; ElevenLabs will respond with isFinal:true.
		if err := connection.WriteJSON(map[string]interface{}{
			"text":       " ",
			"context_id": ctxId,
			"flush":      true,
		}); err != nil {
			t.logger.Errorf("elevenlabs-tts: flush signal failed: %v", err)
			t.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: input.ContextID, Error: fmt.Errorf("elevenlabs-tts: flush failed: %w", err), Type: internal_type.TTSNetworkTimeout})
			return nil
		}
		// TextToSpeechEndPacket is emitted by handleFlushComplete once isFinal received.

	default:
		return fmt.Errorf("elevenlabs-tts: unsupported input type %T", in)
	}
	return nil
}

func (t *elevenlabsTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.mu.Lock()
	connectedAt := t.ttsConnectedAt
	t.ttsConnectedAt = time.Time{}

	if t.connection != nil {
		conn := t.connection
		t.connection = nil // mark before Close so readLoop sees intentional
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
