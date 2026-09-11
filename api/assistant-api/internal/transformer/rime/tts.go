// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_rime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"

	"github.com/gorilla/websocket"

	rime_internal "github.com/rapidaai/api/assistant-api/internal/transformer/rime/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type rimeTTS struct {
	*rimeOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu              sync.Mutex
	connection      *websocket.Conn
	contextId       string
	ttsConnectedAt  time.Time
	ttsStartedAt    time.Time
	ttsMetricSent   bool
	textClosed      bool
	synthesisFailed bool
	endSent         bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewRimeTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	rimeOpts, err := NewRimeOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("rime-tts: failed to initialize options: %v", err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(ctx)
	return &rimeTTS{
		ctx:        ct,
		ctxCancel:  ctxCancel,
		onPacket:   onPacket,
		logger:     logger,
		rimeOption: rimeOpts,
	}, nil
}

func (*rimeTTS) Name() string {
	return "rime-tts"
}

// Initialize opens a fresh WebSocket connection to Rime and starts the read
// goroutine for that connection. Called at session start and after each
// interruption so the connection is warm before the first text delta arrives.
func (rt *rimeTTS) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+rt.GetKey())
	connectionString := rt.GetTextToSpeechConnectionString()
	conn, _, err := websocket.DefaultDialer.Dial(connectionString, header)
	if err != nil {
		rt.logger.Errorf("rime-tts: dial failed: %v", err)
		rt.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "rime-tts: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  rt.Name(),
					"path":      observability.AttributeValue(connectionString),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	rt.mu.Lock()
	rt.connection = conn
	if rt.ttsConnectedAt.IsZero() {
		rt.ttsConnectedAt = time.Now()
	}
	rt.mu.Unlock()
	go rt.readLoop(conn)

	rt.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": rt.Name()},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "rime-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  rt.Name(),
					"path":      observability.AttributeValue(connectionString),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}

// readLoop owns a single WebSocket connection for the duration of one TTS turn.
// It exits when the connection closes intentionally (interrupt / flush complete)
// or unexpectedly (network drop / server error).
func (rt *rimeTTS) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-rt.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			rt.mu.Lock()
			if rt.connection != conn {
				rt.mu.Unlock()
				return
			}
			canComplete := rt.ctx.Err() == nil && rt.textClosed && !rt.synthesisFailed && !rt.endSent &&
				websocket.IsCloseError(err, websocket.CloseNormalClosure)
			contextID := rt.contextId
			if canComplete {
				rt.endSent = true
			}
			rt.connection = nil
			rt.mu.Unlock()
			if canComplete {
				rt.onPacket(
					internal_type.TextToSpeechEndPacket{ContextID: contextID},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: contextID,
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
				return
			}
			// Active connection dropped; next text packet reconnects.
			rt.logger.Errorf("rime-tts: connection lost: %v", err)
			return
		}

		var response rime_internal.RimeTextToSpeechResponse
		if err := json.Unmarshal(msg, &response); err != nil {
			rt.logger.Errorf("rime-tts: failed to parse message: %v", err)
			continue
		}

		switch response.Type {
		case "chunk":
			rt.handleAudio(response)
		case "done":
			if rt.handleFlushComplete(conn) {
				return
			}
		case "error":
			rt.handleServerError(conn, response)
			return
		default:
			// rt.logger.Debugf("rime-tts: unhandled message type: %s", response.Type)
		}
	}
}

// handleAudio decodes an audio chunk and forwards it downstream.
// Chunks arriving with no active contextId are discarded (post-interrupt drain).
func (rt *rimeTTS) handleAudio(response rime_internal.RimeTextToSpeechResponse) {
	raw, err := base64.StdEncoding.DecodeString(response.Data)
	if err != nil {
		rt.logger.Errorf("rime-tts: base64 decode failed: %v", err)
		return
	}

	rt.mu.Lock()
	contextId := rt.contextId
	ttsStartedAt := rt.ttsStartedAt
	shouldEmitFirstAudioLatencyMetric := !rt.ttsMetricSent && !ttsStartedAt.IsZero()
	if shouldEmitFirstAudioLatencyMetric {
		rt.ttsMetricSent = true
	}
	rt.mu.Unlock()

	if contextId == "" {
		rt.logger.Debugf("rime-tts: discarding audio, no active context")
		return
	}

	if shouldEmitFirstAudioLatencyMetric {
		rt.onPacket(internal_type.ObservabilityMetricRecordPacket{
			ContextID: contextId,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": rt.Name()}),
		})
	}
	rt.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: contextId, AudioChunk: raw})
}

// handleFlushComplete keeps Rime done as a per-batch drain hint only.
// The eos completion boundary is the provider's clean remote close.
func (rt *rimeTTS) handleFlushComplete(conn *websocket.Conn) bool {
	rt.mu.Lock()
	if rt.connection != conn {
		rt.mu.Unlock()
		conn.Close()
		return true
	}
	rt.mu.Unlock()
	return false
}

// handleServerError logs the Rime error, surfaces it downstream, and closes
// the connection. The next delta will trigger a fresh reconnect via lazy fallback.
func (rt *rimeTTS) handleServerError(conn *websocket.Conn, response rime_internal.RimeTextToSpeechResponse) {
	rt.logger.Errorf("rime-tts: server error: %s", response.Message)
	rt.mu.Lock()
	rt.connection = nil
	rt.synthesisFailed = true
	ctxID := rt.contextId
	rt.mu.Unlock()
	rt.onPacket(
		internal_type.TextToSpeechErrorPacket{
			ContextID: ctxID,
			Error:     fmt.Errorf("rime-tts: server error: %s", response.Message),
			Type:      internal_type.TTSInvalidInput,
		},
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSError,
				Attributes: observability.Attributes{"type": "error", "message": response.Message},
				OccurredAt: time.Now(),
			},
		},
	)
	conn.Close()
}

func (rt *rimeTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	rt.mu.Lock()
	if in.ContextId() != rt.contextId {
		rt.contextId = in.ContextId()
		rt.ttsStartedAt = time.Time{}
		rt.ttsMetricSent = false
		rt.textClosed = false
		rt.synthesisFailed = false
		rt.endSent = false
	}
	connection := rt.connection
	rt.mu.Unlock()

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		// Close the current connection so any in-flight Rime audio is discarded.
		// The old readLoop goroutine will exit. Reconnect now so the fresh
		// connection is ready before the next text delta arrives.
		rt.mu.Lock()
		rt.contextId = ""
		rt.ttsStartedAt = time.Time{}
		rt.ttsMetricSent = false
		rt.textClosed = false
		rt.synthesisFailed = false
		rt.endSent = false
		conn := rt.connection
		rt.connection = nil
		rt.mu.Unlock()
		if conn != nil {
			conn.Close()
		}
		rt.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"},
				OccurredAt: time.Now(),
			},
		})
		if err := rt.Initialize(); err != nil {
			rt.logger.Errorf("rime-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.TextToSpeechTextPacket:
		// Fallback reconnect: handles Initialize() failure during interrupt or
		// an unintentional connection drop between turns.
		if connection == nil {
			if err := rt.Initialize(); err != nil {
				rt.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("rime-tts: failed to connect: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				})
				return nil
			}
			rt.mu.Lock()
			connection = rt.connection
			if rt.ttsStartedAt.IsZero() {
				rt.ttsStartedAt = time.Now()
			}
			rt.mu.Unlock()
		} else {
			rt.mu.Lock()
			if rt.ttsStartedAt.IsZero() {
				rt.ttsStartedAt = time.Now()
			}
			rt.mu.Unlock()
		}
		if err := connection.WriteJSON(map[string]interface{}{"text": input.Text}); err != nil {
			rt.mu.Lock()
			rt.synthesisFailed = true
			rt.mu.Unlock()
			rt.logger.Errorf("rime-tts: write failed: %v", err)
			rt.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("rime-tts: failed to write text: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		rt.onPacket(internal_type.ObservabilityEventRecordPacket{
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
		// Interrupted before done arrived. Nothing to flush.
		if connection == nil {
			return nil
		}
		rt.mu.Lock()
		rt.textClosed = true
		rt.mu.Unlock()
		if err := connection.WriteJSON(map[string]interface{}{"operation": "eos"}); err != nil {
			rt.mu.Lock()
			rt.synthesisFailed = true
			rt.mu.Unlock()
			rt.logger.Errorf("rime-tts: flush failed: %v", err)
			rt.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("rime-tts: flush failed: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		// Rime drains eos and then closes the socket, which is the final boundary.

	default:
		return fmt.Errorf("rime-tts: unsupported packet type %T", in)
	}
	return nil
}

func (rt *rimeTTS) Close(ctx context.Context) error {
	rt.ctxCancel()
	rt.mu.Lock()
	connectedAt := rt.ttsConnectedAt
	rt.ttsConnectedAt = time.Time{}
	if rt.connection != nil {
		conn := rt.connection
		rt.connection = nil // mark before Close so readLoop sees intentional
		conn.Close()
	}
	rt.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		rt.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": rt.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				Scope:  internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewTTSDurationUsageRecord(rt.Name(), duration, observability.Attributes{}),
			},
		)
	}
	rt.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": rt.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
