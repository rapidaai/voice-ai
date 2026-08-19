// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_resembleai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	resembleai_internal "github.com/rapidaai/api/assistant-api/internal/transformer/resembleai/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type resembleaiTTS struct {
	*resembleaiOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	contextId      string
	ttsConnectedAt time.Time

	ttsStartedAt  time.Time
	ttsMetricSent bool

	logger     commons.Logger
	connection *websocket.Conn
	onPacket   func(pkt ...internal_type.Packet) error
}

func NewResembleAITextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	resembleaiOpts, err := NewResembleAIOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("resembleai-tts: initializing resembleai failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &resembleaiTTS{
		ctx:              ctx2,
		ctxCancel:        contextCancel,
		onPacket:         onPacket,
		logger:           logger,
		resembleaiOption: resembleaiOpts,
	}, nil
}

// Initialize opens a fresh WebSocket connection to ResembleAI and starts the
// read goroutine. Called at session start and after each interruption so the
// connection is warm before the first text delta arrives.
func (ct *resembleaiTTS) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+ct.GetKey())
	conn, resp, err := websocket.DefaultDialer.Dial(RESEMBLEAI_WS_URL, header)
	if err != nil {
		ct.logger.Errorf("resembleai-tts: error while connecting to resembleai %s with response %v", err, resp)
		ct.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "resembleai-tts: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(RESEMBLEAI_WS_URL),
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
				Message: "resembleai-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(RESEMBLEAI_WS_URL),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

func (*resembleaiTTS) Name() string {
	return "resembleai-tts"
}

// handleFlushComplete is called when ResembleAI signals audio_end. It emits
// TextToSpeechEndPacket — correctly ordered after the last audio chunk — and
// closes the per-turn connection.
func (rt *resembleaiTTS) handleFlushComplete(conn *websocket.Conn) {
	rt.mu.Lock()
	ctxId := rt.contextId
	rt.connection = nil // mark before Close so readLoop error handler sees intentional
	rt.mu.Unlock()

	rt.onPacket(
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
// It exits when the connection closes — intentionally (interrupt / audio_end)
// or unexpectedly (network drop).
func (rt *resembleaiTTS) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-rt.ctx.Done():
			return
		default:
		}

		_, audioChunk, err := conn.ReadMessage()
		if err != nil {
			rt.mu.Lock()
			if rt.connection != conn {
				rt.mu.Unlock()
				return
			}
			// Active connection dropped; next text packet reconnects.
			rt.connection = nil
			rt.mu.Unlock()
			rt.logger.Errorf("resembleai-tts: connection lost: %v", err)
			return
		}

		var audioData resembleai_internal.ResembleAITextToSpeechResponse
		if err := json.Unmarshal(audioChunk, &audioData); err != nil {
			rt.logger.Errorf("resembleai-tts: error parsing audio chunk: %v", err)
			continue
		}

		switch audioData.Type {
		case "audio":
			if rawAudioData, err := base64.StdEncoding.DecodeString(audioData.AudioContent); err == nil {
				var shouldEmitFirstAudioLatencyMetric bool
				rt.mu.Lock()
				ttsStartedAt := rt.ttsStartedAt
				ctxId := rt.contextId
				if !rt.ttsMetricSent && !ttsStartedAt.IsZero() {
					rt.ttsMetricSent = true
					shouldEmitFirstAudioLatencyMetric = true
				}
				rt.mu.Unlock()
				if ctxId != "" {
					if shouldEmitFirstAudioLatencyMetric {
						rt.onPacket(internal_type.ObservabilityMetricRecordPacket{
							ContextID: ctxId,
							Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
							Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": rt.Name()}),
						})
					}
					rt.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: ctxId, AudioChunk: rawAudioData})
				}
			} else {
				rt.logger.Errorf("resembleai-tts: error decoding base64 audio: %v", err)
			}
		case "audio_end":
			rt.handleFlushComplete(conn)
			return
		case "error":
			rt.logger.Errorf("resembleai-tts: server error: %s", string(audioChunk))
		default:
			rt.logger.Debugf("resembleai-tts: unhandled message type: %s", audioData.Type)
		}
	}
}

func (t *resembleaiTTS) Transform(ctx context.Context, in internal_type.Packet) error {
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
			t.logger.Errorf("resembleai-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.TextToSpeechTextPacket:
		// Fallback reconnect: handles Initialize() failure or an unintentional drop.
		if connection == nil {
			if err := t.Initialize(); err != nil {
				t.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("resembleai-tts: failed to connect: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				})
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
		if err := connection.WriteJSON(map[string]interface{}{
			"voice_uuid":      t.GetVoiceUUID(),
			"data":            input.Text,
			"output_format":   "wav",
			"sample_rate":     t.GetSampleRate(),
			"precision":       "PCM_16",
			"no_audio_header": true,
		}); err != nil {
			t.logger.Errorf("resembleai-tts: unable to write json for text to speech: %v", err)
			t.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("resembleai-tts: failed to write text: %w", err),
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
					"text": input.Text,
				},
				OccurredAt: time.Now(),
			},
		})
		return nil

	case internal_type.TextToSpeechDonePacket:
		// TextToSpeechEndPacket is emitted by handleFlushComplete once audio_end received.
		return nil

	default:
		return fmt.Errorf("resembleai-tts: unsupported input type %T", in)
	}
}

func (t *resembleaiTTS) Close(ctx context.Context) error {
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
