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
	"net"
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
	stateMu   sync.Mutex
	writeMu   sync.Mutex
	ctx       context.Context
	ctxCancel context.CancelFunc
	readers   sync.WaitGroup
	closed    bool
	retired   map[string]struct{}

	contextId      string
	ttsConnectedAt time.Time

	ttsStartedAt time.Time
	textClosed   bool

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
		retired:        make(map[string]struct{}),
		onPacket:       onPacket,
		normalizer:     cartesia_internal.NewCartesiaNormalizer(logger, opts),
	}, nil
}

// Initialize warms the session socket without replacing an existing connection.
func (ct *cartesiaTTS) Initialize() error {
	ct.writeMu.Lock()
	defer ct.writeMu.Unlock()
	return ct.connect(ct.ctx)
}

// connect requires writeMu so initialization and fresh text share one dial owner.
func (ct *cartesiaTTS) connect(ctx context.Context) error {
	if err := ct.ctx.Err(); err != nil {
		return err
	}
	ct.stateMu.Lock()
	connected := ct.connection != nil
	ct.stateMu.Unlock()
	if connected {
		return nil
	}
	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	stopSession := context.AfterFunc(ct.ctx, cancel)
	defer stopSession()
	dialer := *websocket.DefaultDialer
	dial := dialer.NetDialContext
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	if dialer.NetDialTLSContext != nil {
		dial = dialer.NetDialTLSContext
	}
	// DialContext alone does not cancel a stalled HTTP upgrade in Gorilla 1.5.3.
	var stopDial func() bool
	dialWithCancel := func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err == nil {
			stopDial = context.AfterFunc(dialCtx, func() { _ = conn.Close() })
		}
		return conn, err
	}
	if dialer.NetDialTLSContext != nil {
		dialer.NetDialTLSContext = dialWithCancel
	} else {
		dialer.NetDialContext = dialWithCancel
	}
	start := time.Now()
	conn, _, err := dialer.DialContext(dialCtx, ct.GetTextToSpeechConnectionString(), nil)
	if stopDial != nil {
		stopDial()
	}
	if err == nil {
		err = dialCtx.Err()
		if err == nil {
			err = ct.ctx.Err()
		}
		if err != nil {
			_ = conn.Close()
		}
	}
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
					"error":     observability.AttributeValue(err.Error()),
				},
				OccurredAt: time.Now(),
			},
		})
		return err
	}

	ct.stateMu.Lock()
	ct.connection = conn
	if ct.ttsConnectedAt.IsZero() {
		ct.ttsConnectedAt = time.Now()
	}
	ct.stateMu.Unlock()
	ct.readers.Go(func() { ct.readLoop(conn) })
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

// readLoop owns one session socket; provider completion does not close it.
func (ct *cartesiaTTS) readLoop(conn *websocket.Conn) {
	stop := context.AfterFunc(ct.ctx, func() { _ = conn.Close() })
	defer stop()
	defer conn.Close()
	ct.stateMu.Lock()
	if ct.retired == nil {
		ct.retired = make(map[string]struct{})
	}
	ct.stateMu.Unlock()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			ct.stateMu.Lock()
			if ct.connection != conn {
				ct.stateMu.Unlock()
				return
			}
			ct.connection = nil
			contextID := ct.contextId
			if contextID != "" {
				ct.retired[contextID] = struct{}{}
				ct.contextId = ""
			}
			ct.stateMu.Unlock()
			_ = conn.Close()
			if contextID != "" && ct.ctx.Err() == nil {
				ct.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: contextID,
					Error:     fmt.Errorf("cartesia-tts: connection lost: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				})
			}
			return
		}

		var payload cartesia_internal.TextToSpeechOuput
		if err := json.Unmarshal(msg, &payload); err != nil {
			ct.logger.Errorf("cartesia-tts: invalid json from cartesia error : %v", err)
			continue
		}

		ct.stateMu.Lock()
		if ct.connection != conn {
			ct.stateMu.Unlock()
			return
		}
		_, retired := ct.retired[payload.ContextID]
		if ct.ctx.Err() != nil || payload.ContextID == "" || payload.ContextID != ct.contextId || retired {
			ct.stateMu.Unlock()
			continue
		}
		var audio []byte
		if payload.Type == "chunk" {
			audio, err = base64.StdEncoding.DecodeString(payload.Data)
			if err != nil {
				err = fmt.Errorf("cartesia-tts: invalid audio: %w", err)
			}
		} else if payload.Type == "error" {
			err = fmt.Errorf("cartesia-tts: provider error status %d", payload.StatusCode)
		}
		if err != nil {
			ct.retired[payload.ContextID] = struct{}{}
			ct.contextId = ""
			// A blocked error callback must not keep a failed write alive.
			if ct.writeMu.TryLock() {
				ct.writeMu.Unlock()
			} else {
				ct.connection = nil
				_ = conn.Close()
			}
			ct.stateMu.Unlock()
			ct.onPacket(
				internal_type.TextToSpeechErrorPacket{
					ContextID: payload.ContextID,
					Error:     err,
					Type:      internal_type.TTSInvalidInput,
				},
				internal_type.ObservabilityLogRecordPacket{
					ContextID: payload.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordLog{
						Level:   observability.LevelError,
						Message: "cartesia-tts: synthesis failed",
						Attributes: observability.Attributes{
							"component": observability.ComponentTTS.String(),
							"provider":  ct.Name(),
							"error":     observability.AttributeValue(err.Error()),
						},
						OccurredAt: time.Now(),
					},
				},
			)
			continue
		}
		switch payload.Type {
		case "done":
			if ct.textClosed {
				ct.retired[payload.ContextID] = struct{}{}
				ct.contextId = ""
				ct.stateMu.Unlock()
				ct.onPacket(
					internal_type.TextToSpeechEndPacket{ContextID: payload.ContextID},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: payload.ContextID,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordEvent{
							Component:  observability.ComponentTTS,
							Event:      observability.TTSCompleted,
							Attributes: observability.Attributes{"type": "completed"},
							OccurredAt: time.Now(),
						},
					},
				)
				continue
			}
		case "chunk":
			if len(audio) == 0 {
				break
			}
			startedAt := ct.ttsStartedAt
			ct.ttsStartedAt = time.Time{}
			ct.stateMu.Unlock()
			if !startedAt.IsZero() {
				ct.onPacket(internal_type.ObservabilityMetricRecordPacket{
					ContextID: payload.ContextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record:    observability.NewMetricTTSLatencyMs(time.Since(startedAt), observability.Attributes{"provider": ct.Name()}),
				})
			}
			ct.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: payload.ContextID, AudioChunk: audio})
			continue
		}
		ct.stateMu.Unlock()
	}
}

func (ct *cartesiaTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ct.ctx.Err(); err != nil {
		return err
	}
	if input, ok := in.(internal_type.TextToSpeechInterruptPacket); ok {
		ct.stateMu.Lock()
		if input.ContextID == "" {
			ct.stateMu.Unlock()
			return nil
		}
		ct.retired[input.ContextID] = struct{}{}
		if input.ContextID != ct.contextId || ct.connection == nil {
			if input.ContextID == ct.contextId {
				ct.contextId = ""
			}
			ct.stateMu.Unlock()
			return nil
		}
		// Interruption cannot wait behind the write it needs to stop.
		if !ct.writeMu.TryLock() {
			connection := ct.connection
			ct.connection = nil
			ct.contextId = ""
			ct.stateMu.Unlock()
			_ = connection.Close()
			ct.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
				},
			})
			return nil
		}
		ct.stateMu.Unlock()
	} else {
		ct.writeMu.Lock()
	}
	defer ct.writeMu.Unlock()

	// Replacement text cancels the old context before starting the new one.
	for {
		ct.stateMu.Lock()
		if err := ctx.Err(); err != nil {
			ct.stateMu.Unlock()
			return err
		}
		if err := ct.ctx.Err(); err != nil {
			ct.stateMu.Unlock()
			return err
		}
		contextID := in.ContextId()
		var message interface{}
		switch input := in.(type) {
		case internal_type.TurnChangePacket:
			ct.stateMu.Unlock()
			return nil
		case internal_type.TextToSpeechInterruptPacket:
			if input.ContextID != ct.contextId || ct.connection == nil {
				ct.stateMu.Unlock()
				return nil
			}
			message = map[string]interface{}{"context_id": input.ContextID, "cancel": true}
		case internal_type.TextToSpeechTextPacket:
			if input.ContextID == "" || input.Text == "" {
				ct.stateMu.Unlock()
				return nil
			}
			if _, retired := ct.retired[input.ContextID]; retired || (input.ContextID == ct.contextId && ct.textClosed) {
				ct.stateMu.Unlock()
				return nil
			}
			if ct.contextId != "" && ct.contextId != input.ContextID {
				contextID = ct.contextId
				ct.retired[contextID] = struct{}{}
				if ct.connection != nil {
					message = map[string]interface{}{"context_id": contextID, "cancel": true}
					break
				}
			}
			contextID = input.ContextID
			if ct.contextId != input.ContextID {
				ct.contextId = input.ContextID
				ct.textClosed = false
				ct.stateMu.Unlock()
				err := ct.connect(ctx)
				ct.stateMu.Lock()
				if err != nil {
					_, retired := ct.retired[input.ContextID]
					ct.retired[input.ContextID] = struct{}{}
					if ct.contextId == input.ContextID {
						ct.contextId = ""
					}
					ct.stateMu.Unlock()
					if err := ctx.Err(); err != nil {
						return err
					}
					if err := ct.ctx.Err(); err != nil {
						return err
					}
					if !retired {
						ct.onPacket(internal_type.TextToSpeechErrorPacket{
							ContextID: input.ContextID, Error: fmt.Errorf("cartesia-tts: connect: %w", err),
							Type: internal_type.TTSNetworkTimeout,
						})
					}
					return nil
				}
				ct.ttsStartedAt = time.Now()
			}
			if ct.contextId != input.ContextID || ct.connection == nil {
				ct.stateMu.Unlock()
				return nil
			}
			message = ct.GetTextToSpeechInput(ct.normalizer.Normalize(input.Text), map[string]interface{}{
				"continue": true, "context_id": input.ContextID,
			})
		case internal_type.TextToSpeechDonePacket:
			if input.ContextID == "" || input.ContextID != ct.contextId || ct.connection == nil || ct.textClosed {
				ct.stateMu.Unlock()
				return nil
			}
			ct.textClosed = true
			message = ct.GetTextToSpeechInput("", map[string]interface{}{"continue": false, "context_id": input.ContextID})
		default:
			ct.stateMu.Unlock()
			return fmt.Errorf("cartesia-tts: unsupported input type %T", in)
		}
		connection := ct.connection
		ct.stateMu.Unlock()

		writeCancelled := make(chan struct{})
		stopWriteCancellation := context.AfterFunc(ctx, func() {
			ct.stateMu.Lock()
			if ct.connection == connection {
				ct.connection = nil
			}
			ct.retired[contextID] = struct{}{}
			if ct.contextId == contextID {
				ct.contextId = ""
			}
			ct.stateMu.Unlock()
			_ = connection.Close()
			close(writeCancelled)
		})
		err := connection.WriteJSON(message)
		// Finish cancellation before this connection can be reused.
		if !stopWriteCancellation() {
			<-writeCancelled
		}
		ct.stateMu.Lock()
		if ctx.Err() != nil || ct.ctx.Err() != nil {
			ct.stateMu.Unlock()
			return ctx.Err()
		}
		if ct.connection != connection {
			ct.stateMu.Unlock()
			if contextID != in.ContextId() {
				continue
			}
			return nil
		}
		if err != nil {
			ct.connection = nil
			ct.retired[contextID] = struct{}{}
			if ct.contextId == contextID {
				ct.contextId = ""
			}
			ct.stateMu.Unlock()
			_ = connection.Close()
			ct.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: contextID, Error: fmt.Errorf("cartesia-tts: send: %w", err),
				Type: internal_type.TTSNetworkTimeout,
			})
			if contextID != in.ContextId() {
				continue
			}
			return nil
		}
		// Keep the retired context identifiable until its cancel write finishes.
		if _, retired := ct.retired[contextID]; retired && ct.contextId == contextID {
			ct.contextId = ""
		}
		ct.stateMu.Unlock()
		if contextID != in.ContextId() {
			continue
		}
		switch in.(type) {
		case internal_type.TextToSpeechInterruptPacket:
			ct.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
				},
			})
		case internal_type.TextToSpeechTextPacket:
			ct.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
					Attributes: observability.Attributes{"type": "speaking", "text": message.(cartesia_internal.TextToSpeechInput).Transcript},
					OccurredAt: time.Now(),
				},
			})
		}
		return nil
	}
}

func (ct *cartesiaTTS) Close(ctx context.Context) error {
	ct.ctxCancel()
	ct.writeMu.Lock()
	ct.stateMu.Lock()
	if ct.closed {
		ct.stateMu.Unlock()
		ct.writeMu.Unlock()
		ct.readers.Wait()
		return nil
	}
	ct.closed = true
	ctxID := ct.contextId
	connectedAt := ct.ttsConnectedAt
	ct.ttsConnectedAt = time.Time{}

	if ct.connection != nil {
		conn := ct.connection
		ct.connection = nil // mark before Close so readLoop sees intentional
		_ = conn.Close()
	}
	ct.stateMu.Unlock()
	ct.writeMu.Unlock()
	ct.readers.Wait()

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
