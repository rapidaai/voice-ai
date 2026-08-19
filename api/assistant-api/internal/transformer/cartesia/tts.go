// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_cartesia

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	cartesia_internal "github.com/rapidaai/api/assistant-api/internal/transformer/cartesia/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type cartesiaTTS struct {
	*cartesiaOption
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

func NewCartesiaTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	cartesiaOpts, err := NewCartesiaOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("intializing cartesia failed %+v", err)
		return nil, err
	}

	ct, ctxCancel := context.WithCancel(ctx)
	return &cartesiaTTS{
		cartesiaOption: cartesiaOpts,
		logger:         logger,
		ctx:            ct,
		ctxCancel:      ctxCancel,
		onPacket:       onPacket,
		normalizer:     cartesia_internal.NewCartesiaNormalizer(logger, opts),
	}, nil
}

// Initialize opens a fresh WebSocket connection to Cartesia and starts the
// read goroutine. Called at session start and after each interruption so the
// connection is warm before the first text delta arrives.
func (ct *cartesiaTTS) Initialize() error {
	start := time.Now()
	conn, _, err := websocket.DefaultDialer.Dial(ct.GetTextToSpeechConnectionString(), nil)
	if err != nil {
		ct.logger.Errorf("cartesia-tts: unable to dial %v", err)
		ct.onPacket(internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "cartesia-tts: error while performing connect",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(ct.GetTextToSpeechConnectionString()),
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
				Message: "cartesia-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  ct.Name(),
					"path":      observability.AttributeValue(ct.GetTextToSpeechConnectionString()),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

// Name returns the name of this transformer.
func (*cartesiaTTS) Name() string {
	return "cartesia-tts"
}

// handleFlushComplete is called when Cartesia signals done. It emits
// TextToSpeechEndPacket — correctly ordered after the last audio chunk — and
// closes the per-turn connection.
func (cst *cartesiaTTS) handleFlushComplete(conn *websocket.Conn) {
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

// readLoop owns a single WebSocket connection for the duration of one TTS turn.
// It exits when the connection closes — intentionally (interrupt / flush complete)
// or unexpectedly (network drop).
func (cst *cartesiaTTS) readLoop(conn *websocket.Conn) {
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

			cst.logger.Errorf("cartesia-tts: connection lost: %v", err)
			return
		}

		var payload cartesia_internal.TextToSpeechOuput
		if err := json.Unmarshal(msg, &payload); err != nil {
			cst.logger.Errorf("cartesia-tts: invalid json from cartesia error : %v", err)
			continue
		}

		if payload.Done {
			cst.handleFlushComplete(conn)
			return
		}

		if payload.Data == "" {
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(payload.Data)
		if err != nil {
			cst.logger.Errorf("cartesia-tts: failed to decode audio payload error: %v", err)
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

func (ct *cartesiaTTS) Transform(ctx context.Context, in internal_type.Packet) error {
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
			ct.logger.Errorf("cartesia-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.TextToSpeechTextPacket:
		if connection == nil {
			if err := ct.Initialize(); err != nil {
				ct.onPacket(
					internal_type.TextToSpeechErrorPacket{
						ContextID: input.ContextID,
						Error:     fmt.Errorf("cartesia-tts: failed to connect: %w", err),
						Type:      internal_type.TTSNetworkTimeout,
					},
					internal_type.ObservabilityLogRecordPacket{
						ContextID: input.ContextID,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordLog{
							Level:   observability.LevelError,
							Message: "cartesia-tts: failed to connect",
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
		message := ct.GetTextToSpeechInput(normalized, map[string]interface{}{"continue": true, "context_id": contextID, "max_buffer_delay_ms": "0ms"})
		if err := connection.WriteJSON(message); err != nil {
			ct.logger.Errorf("cartesia-tts: failed to write text: %v", err)
			ct.onPacket(
				internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("cartesia-tts: failed to write text: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: input.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "cartesia-tts: failed to write text",
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
		// Signal end of text stream; Cartesia will respond with done:true.
		message := ct.GetTextToSpeechInput("", map[string]interface{}{"continue": false, "flush": true, "context_id": contextID})
		if err := connection.WriteJSON(message); err != nil {
			ct.logger.Errorf("cartesia-tts: flush failed: %v", err)
			ct.onPacket(
				internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("cartesia-tts: flush failed: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: input.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "cartesia-tts: flush failed",
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
		// TextToSpeechEndPacket is emitted by handleFlushComplete once done received.

	default:
		return fmt.Errorf("cartesia-tts: unsupported input type %T", in)
	}
	return nil
}

func (ct *cartesiaTTS) Close(ctx context.Context) error {
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
