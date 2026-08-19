// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_smallest

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
	smallest_internal "github.com/rapidaai/api/assistant-api/internal/transformer/smallest/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type smallestTTS struct {
	*smallestOption
	mu        sync.Mutex
	ctx       context.Context
	ctxCancel context.CancelFunc

	contextId      string
	ttsConnectedAt time.Time

	ttsStartedAt  time.Time
	ttsMetricSent bool

	logger     commons.Logger
	connection *websocket.Conn
	onPacket   func(pkt ...internal_type.Packet) error
	normalizer internal_type.TextNormalizer
}

func NewSmallestTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	smallestOpts, err := NewSmallestOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("intializing smallest failed %+v", err)
		return nil, err
	}

	ct, ctxCancel := context.WithCancel(ctx)
	return &smallestTTS{
		smallestOption: smallestOpts,
		logger:         logger,
		ctx:            ct,
		ctxCancel:      ctxCancel,
		onPacket:       onPacket,
		normalizer:     smallest_internal.NewSmallestNormalizer(logger, opts),
	}, nil
}

// Initialize opens a fresh WebSocket connection to Smallest's Lightning
// streaming endpoint and starts the read goroutine. Called at session start
// and after each interruption so the connection is warm before the first
// text delta arrives.
func (ct *smallestTTS) Initialize() error {
	start := time.Now()
	connectionString := ct.GetTextToSpeechConnectionString()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+ct.GetKey())
	setSourceHeaders(header)

	conn, _, err := websocket.DefaultDialer.Dial(connectionString, header)
	if err != nil {
		err = fmt.Errorf("smallest-tts: dial %s: %w", connectionString, err)
		ct.logger.Errorf("smallest-tts: unable to dial %v", err)
		ct.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "smallest-tts: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(connectionString),
					"error":     observability.AttributeValue(err.Error()),
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
				Message: "smallest-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(connectionString),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

// Name returns the name of this transformer.
func (*smallestTTS) Name() string {
	return "smallest-tts"
}

// handleFlushComplete is called when Smallest signals status:"complete". It
// emits TextToSpeechEndPacket — correctly ordered after the last audio chunk —
// and closes the per-turn connection.
func (cst *smallestTTS) handleFlushComplete(conn *websocket.Conn) {
	cst.mu.Lock()
	if cst.connection != conn {
		cst.mu.Unlock()
		conn.Close()
		return
	}
	contextID := cst.contextId
	cst.connection = nil // mark before Close so readLoop error handler sees intentional
	cst.mu.Unlock()
	if contextID == "" {
		conn.Close()
		return
	}

	cst.onPacket(
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
}

// handleServerError is called when Smallest signals status:"error" (e.g. a
// voice_id/model pairing it rejects, such as a Pro-only voice on
// lightning_v3.1). The server sends exactly one such frame and then holds
// the connection open without ever sending "complete", so this must be
// treated as terminal for the turn — otherwise the turn hangs forever
// waiting for audio that will never arrive.
func (cst *smallestTTS) handleServerError(conn *websocket.Conn, payload smallest_internal.TextToSpeechOutput) {
	cst.mu.Lock()
	if cst.connection != conn {
		cst.mu.Unlock()
		conn.Close()
		return
	}
	contextID := cst.contextId
	cst.connection = nil // mark before Close so the write-path sees this as intentional
	cst.mu.Unlock()

	msg := payload.Message
	if len(payload.Errors) > 0 && payload.Errors[0].Message != "" {
		msg = payload.Errors[0].Message
	}
	if msg == "" {
		msg = "unknown error"
	}

	cst.logger.Errorf("smallest-tts: server error: %s", msg)
	cst.onPacket(
		internal_type.TextToSpeechErrorPacket{
			ContextID: contextID,
			Error:     fmt.Errorf("smallest-tts: server error: %s", msg),
			Type:      internal_type.TTSInvalidInput,
		},
		internal_type.ObservabilityLogRecordPacket{
			ContextID: contextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "smallest-tts: server error",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  cst.Name(),
					"error":     observability.AttributeValue(msg),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	conn.Close()
}

// readLoop owns a single WebSocket connection for the duration of one TTS turn.
// It exits when the connection closes — intentionally (interrupt / flush complete)
// or unexpectedly (network drop).
func (cst *smallestTTS) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-cst.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			cst.mu.Lock()
			if cst.connection != conn {
				cst.mu.Unlock()
				return
			}
			cst.connection = nil
			cst.mu.Unlock()

			cst.logger.Errorf("smallest-tts: connection lost: %v", err)
			return
		}

		var payload smallest_internal.TextToSpeechOutput
		if err := json.Unmarshal(msg, &payload); err != nil {
			cst.logger.Errorf("smallest-tts: invalid json from smallest error: %v", err)
			continue
		}

		switch payload.Status {
		case "complete":
			cst.handleFlushComplete(conn)
			return
		case "error":
			cst.handleServerError(conn, payload)
			return
		case "word_timestamp":
			// Word-level timestamps are opt-in (word_timestamps=true) and not
			// yet consumed by Rapida's TTS pipeline — ignored for now.
			continue
		case "chunk":
			// handled below
		default:
			continue
		}

		if payload.Data.Audio == "" {
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(payload.Data.Audio)
		if err != nil {
			cst.logger.Errorf("smallest-tts: failed to decode audio payload error: %v", err)
			continue
		}

		var shouldEmitFirstAudioLatencyMetric bool
		cst.mu.Lock()
		ttsStartedAt := cst.ttsStartedAt
		contextID := cst.contextId
		if !cst.ttsMetricSent && !ttsStartedAt.IsZero() {
			cst.ttsMetricSent = true
			shouldEmitFirstAudioLatencyMetric = true
		}
		cst.mu.Unlock()
		if contextID == "" {
			continue
		}

		if shouldEmitFirstAudioLatencyMetric {
			_ = cst.onPacket(internal_type.ObservabilityMetricRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record:    observability.NewMetricTTSLatencyMs(time.Since(ttsStartedAt), observability.Attributes{"provider": cst.Name()}),
			})
		}
		_ = cst.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: contextID, AudioChunk: decoded})
	}
}

func (ct *smallestTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	ct.mu.Lock()
	if in.ContextId() != ct.contextId {
		ct.contextId = in.ContextId()
		ct.ttsStartedAt = time.Time{}
		ct.ttsMetricSent = false
	}
	connection := ct.connection
	ct.mu.Unlock()

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		ct.mu.Lock()
		ct.contextId = ""
		ct.ttsStartedAt = time.Time{}
		ct.ttsMetricSent = false
		conn := ct.connection
		ct.connection = nil
		ct.mu.Unlock()
		if conn != nil {
			conn.Close()
		}
		ct.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"},
				OccurredAt: time.Now(),
			},
		})
		if err := ct.Initialize(); err != nil {
			ct.logger.Errorf("smallest-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.TextToSpeechTextPacket:
		if connection == nil {
			if err := ct.Initialize(); err != nil {
				ct.onPacket(
					internal_type.TextToSpeechErrorPacket{
						ContextID: input.ContextID,
						Error:     fmt.Errorf("smallest-tts: failed to connect: %w", err),
						Type:      internal_type.TTSNetworkTimeout,
					},
					internal_type.ObservabilityLogRecordPacket{
						ContextID: input.ContextID,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordLog{
							Level:   observability.LevelError,
							Message: "smallest-tts: failed to connect",
							Attributes: observability.Attributes{
								"component": observability.ComponentTTS.String(),
								"provider":  ct.Name(),
								"error":     observability.AttributeValue(err.Error()),
							},
							OccurredAt: time.Now(),
						},
					},
				)
				return nil
			}
			ct.mu.Lock()
			connection = ct.connection
			if ct.ttsStartedAt.IsZero() {
				ct.ttsStartedAt = time.Now()
			}
			ct.mu.Unlock()
		} else {
			ct.mu.Lock()
			if ct.ttsStartedAt.IsZero() {
				ct.ttsStartedAt = time.Now()
			}
			ct.mu.Unlock()
		}
		ct.mu.Lock()
		contextID := ct.contextId
		ct.mu.Unlock()
		normalized := ct.normalizer.Normalize(input.Text)
		message := ct.GetTextToSpeechInput(normalized, map[string]interface{}{"continue": true, "context_id": contextID})
		if err := connection.WriteJSON(message); err != nil {
			ct.logger.Errorf("smallest-tts: failed to write text: %v", err)
			ct.onPacket(
				internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("smallest-tts: failed to write text: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: input.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "smallest-tts: failed to write text",
						Attributes: observability.Attributes{
							"component": observability.ComponentTTS.String(),
							"provider":  ct.Name(),
							"error":     observability.AttributeValue(err.Error()),
						},
						OccurredAt: time.Now(),
					},
				},
			)
			return nil
		}
		ct.onPacket(internal_type.ObservabilityEventRecordPacket{
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

	case internal_type.TextToSpeechDonePacket:
		// Interrupted before done arrived — nothing to flush.
		if connection == nil {
			return nil
		}
		ct.mu.Lock()
		contextID := ct.contextId
		ct.mu.Unlock()
		// Signal end of text stream; Smallest responds with status:"complete"
		// after complete_backoff_ms (server default ~4s) once all audio is flushed.
		message := ct.GetTextToSpeechInput("", map[string]interface{}{"continue": false, "flush": true, "context_id": contextID})
		if err := connection.WriteJSON(message); err != nil {
			ct.logger.Errorf("smallest-tts: flush failed: %v", err)
			ct.onPacket(
				internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("smallest-tts: flush failed: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: input.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "smallest-tts: flush failed",
						Attributes: observability.Attributes{
							"component": observability.ComponentTTS.String(),
							"provider":  ct.Name(),
							"error":     observability.AttributeValue(err.Error()),
						},
						OccurredAt: time.Now(),
					},
				},
			)
			return nil
		}
		// TextToSpeechEndPacket is emitted by handleFlushComplete once complete received.

	default:
		return fmt.Errorf("smallest-tts: unsupported input type %T", in)
	}
	return nil
}

func (ct *smallestTTS) Close(ctx context.Context) error {
	ct.ctxCancel()
	ct.mu.Lock()
	ctxID := ct.contextId
	connectedAt := ct.ttsConnectedAt
	ct.ttsConnectedAt = time.Time{}

	if ct.connection != nil {
		conn := ct.connection
		ct.connection = nil // mark before Close so readLoop sees intentional
		_ = conn.Close()
	}
	ct.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		ct.onPacket(
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewMetricTTSDuration(duration, observability.Attributes{"provider": ct.Name()}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record:    observability.NewTTSDurationUsageRecord(ct.Name(), duration, observability.Attributes{}),
			},
		)
	}
	ct.onPacket(
		internal_type.ObservabilityEventRecordPacket{
			ContextID: ctxID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS,
				Event:     observability.TTSClosed,
				Attributes: observability.Attributes{
					"type":     "closed",
					"provider": ct.Name(),
				},
				OccurredAt: time.Now(),
			},
		})
	return nil
}
