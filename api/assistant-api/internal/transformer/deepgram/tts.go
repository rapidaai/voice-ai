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
	"strconv"
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
	stateMu        sync.Mutex
	writeMu        sync.Mutex
	readers        sync.WaitGroup

	ttsStartedAt time.Time
	clearDone    chan struct{}

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

// Initialize opens the session connection, reusing it while it remains healthy.
func (t *deepgramTTS) Initialize() error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	return t.connect(t.ctx)
}

// connect requires both locks; the reader cannot replace a socket during setup.
func (t *deepgramTTS) connect(ctx context.Context) error {
	if err := t.ctx.Err(); err != nil {
		return err
	}
	if t.connection != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	stop := context.AfterFunc(t.ctx, cancel)
	defer stop()
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("token %s", t.GetKey()))
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, t.GetTextToSpeechConnectionString(), header)
	if err != nil {
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
		return fmt.Errorf("deepgram-tts: connect: %w", err)
	}
	if err := ctx.Err(); err != nil {
		_ = conn.Close()
		return err
	}

	t.connection = conn
	if t.ttsConnectedAt.IsZero() {
		t.ttsConnectedAt = time.Now()
	}
	t.readers.Go(func() { t.readLoop(conn) })
	t.onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricTTSInitLatencyMs,
					Value:       strconv.FormatInt(time.Since(start).Milliseconds(), 10),
					Description: "TTS initialization latency in milliseconds",
				}},
				Attributes: observability.Attributes{"provider": t.Name()},
			},
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
	return deepgram_internal.TextToSpeechTransformerName
}

// The reader accepts output only from its connection and current synthesis boundary.
func (t *deepgramTTS) readLoop(conn *websocket.Conn) {
	stop := context.AfterFunc(t.ctx, func() { _ = conn.Close() })
	defer stop()
	defer conn.Close()
	for {
		msgType, data, err := conn.ReadMessage()
		var envelope deepgram_internal.DeepgramTextToSpeechResponse
		if err == nil && msgType == websocket.TextMessage {
			if err := json.Unmarshal(data, &envelope); err != nil {
				continue
			}
			if envelope.Type == "Error" {
				err = fmt.Errorf("%s: %s", envelope.Code, envelope.Message)
			}
		}
		t.stateMu.Lock()
		if t.connection != conn {
			t.stateMu.Unlock()
			return
		}
		if err != nil {
			t.connection = nil
			contextID := t.contextId
			if t.clearDone != nil {
				close(t.clearDone)
				t.clearDone = nil
			}
			t.stateMu.Unlock()
			_ = conn.Close()
			if contextID != "" && t.ctx.Err() == nil {
				t.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: contextID, Error: fmt.Errorf("deepgram-tts: receive: %w", err),
					Type: internal_type.TTSNetworkTimeout,
				})
			}
			return
		}

		if msgType == websocket.BinaryMessage {
			if t.contextId == "" {
				t.stateMu.Unlock()
				continue
			}
			contextID, startedAt := t.contextId, t.ttsStartedAt
			t.ttsStartedAt = time.Time{}
			t.stateMu.Unlock()
			if !startedAt.IsZero() {
				t.onPacket(internal_type.ObservabilityMetricRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record:    observability.NewMetricTTSLatencyMs(time.Since(startedAt), observability.Attributes{"provider": t.Name()}),
				})
			}
			t.onPacket(internal_type.TextToSpeechAudioPacket{
				ContextID:  contextID,
				AudioChunk: data,
			})
			continue
		}

		switch envelope.Type {
		case "Metadata":
		case "Flushed":
			if t.contextId != "" {
				contextID := t.contextId
				t.contextId = ""
				t.stateMu.Unlock()
				t.onPacket(
					internal_type.TextToSpeechEndPacket{ContextID: contextID},
					internal_type.ObservabilityEventRecordPacket{
						ContextID: contextID,
						Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
						Record: observability.RecordEvent{
							Component: observability.ComponentTTS, Event: observability.TTSCompleted,
							Attributes: observability.Attributes{"type": "completed"}, OccurredAt: time.Now(),
						},
					},
				)
				continue
			}
		case "Cleared":
			if t.clearDone != nil {
				close(t.clearDone)
				t.clearDone = nil
			}
		case "Warning":
			t.logger.Warnf("deepgram-tts warning code=%s message=%s", envelope.Code, envelope.Message)
		default:
			t.logger.Debugf("deepgram-tts: unhandled message type: %s", envelope.Type)
		}
		t.stateMu.Unlock()
	}
}

// Transform streams text into Deepgram
func (t *deepgramTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := t.ctx.Err(); err != nil {
		return err
	}
	if input, ok := in.(internal_type.TextToSpeechInterruptPacket); ok {
		t.stateMu.Lock()
		if input.ContextID != t.contextId || t.contextId == "" || t.connection == nil {
			t.stateMu.Unlock()
			return nil
		}
		// Interruption cannot wait behind the write it needs to cancel.
		if !t.writeMu.TryLock() {
			connection := t.connection
			t.connection = nil
			if t.clearDone != nil {
				close(t.clearDone)
				t.clearDone = nil
			}
			t.stateMu.Unlock()
			if connection != nil {
				_ = connection.Close()
			}
			t.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
				},
			})
			return nil
		}
		t.stateMu.Unlock()
	} else {
		t.writeMu.Lock()
	}
	defer t.writeMu.Unlock()

	// A replacement clears the old synthesis before sending its text.
	for {
		t.stateMu.Lock()
		if err := ctx.Err(); err != nil {
			t.stateMu.Unlock()
			return err
		}
		if err := t.ctx.Err(); err != nil {
			t.stateMu.Unlock()
			return err
		}
		var command map[string]string
		switch input := in.(type) {
		case internal_type.TurnChangePacket:
			t.stateMu.Unlock()
			return nil
		case internal_type.TextToSpeechInterruptPacket:
			if input.ContextID != t.contextId || t.contextId == "" || t.connection == nil {
				t.stateMu.Unlock()
				return nil
			}
			t.contextId = ""
			t.clearDone = make(chan struct{})
			command = map[string]string{"type": "Clear"}
		case internal_type.TextToSpeechTextPacket:
			if input.ContextID == "" {
				t.stateMu.Unlock()
				return fmt.Errorf("deepgram-tts: text requires a context ID")
			}
			// A failed socket cannot resume the synthesis that was using it.
			if t.connection == nil && input.ContextID == t.contextId {
				t.stateMu.Unlock()
				return nil
			}
			if t.contextId != "" && t.contextId != input.ContextID {
				t.contextId = ""
				if t.connection != nil {
					t.clearDone = make(chan struct{})
					command = map[string]string{"type": "Clear"}
					break
				}
			}
			if clearDone := t.clearDone; clearDone != nil {
				t.stateMu.Unlock()
				// Untagged audio must reach Clear before a new message starts.
				timer := time.NewTimer(2 * time.Second)
				defer timer.Stop()
				select {
				case <-clearDone:
				case <-ctx.Done():
					return ctx.Err()
				case <-t.ctx.Done():
					return t.ctx.Err()
				case <-timer.C:
					t.stateMu.Lock()
					if t.clearDone != nil {
						_ = t.connection.Close()
						t.connection = nil
						close(t.clearDone)
						t.clearDone = nil
					}
					t.stateMu.Unlock()
				}
				continue
			}
			if err := t.connect(ctx); err != nil {
				t.contextId = input.ContextID
				t.stateMu.Unlock()
				if err := t.ctx.Err(); err != nil {
					return err
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				t.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: input.ContextID, Error: err, Type: internal_type.TTSNetworkTimeout})
				return nil
			}
			if input.ContextID != t.contextId {
				t.contextId = input.ContextID
				t.ttsStartedAt = time.Now()
			}
			command = map[string]string{"type": "Speak", "text": t.normalizer.Normalize(input.Text)}
		case internal_type.TextToSpeechDonePacket:
			if input.ContextID != t.contextId || t.contextId == "" || t.connection == nil {
				t.stateMu.Unlock()
				return nil
			}
			command = map[string]string{"type": "Flush"}
		default:
			t.stateMu.Unlock()
			return fmt.Errorf("deepgram-tts: unsupported input type %T", in)
		}
		connection := t.connection
		t.stateMu.Unlock()

		writeCancelled := make(chan struct{})
		stopWriteCancellation := context.AfterFunc(ctx, func() {
			t.stateMu.Lock()
			if t.connection == connection {
				t.connection = nil
				if t.clearDone != nil {
					close(t.clearDone)
					t.clearDone = nil
				}
			}
			t.stateMu.Unlock()
			_ = connection.Close()
			close(writeCancelled)
		})
		err := connection.WriteJSON(command)
		// A late cancellation must finish before this socket can be reused.
		if !stopWriteCancellation() {
			<-writeCancelled
		}
		t.stateMu.Lock()
		if ctx.Err() != nil || t.ctx.Err() != nil {
			t.stateMu.Unlock()
			return ctx.Err()
		}
		if t.connection != connection && command["type"] != "Clear" {
			t.stateMu.Unlock()
			return nil
		}
		if err != nil {
			_ = connection.Close()
			t.connection = nil
			if t.clearDone != nil {
				close(t.clearDone)
				t.clearDone = nil
			}
		}
		t.stateMu.Unlock()
		if command["type"] == "Clear" && in.PacketName() == internal_type.PacketNameTextToSpeechText {
			continue
		}
		if err != nil {
			err = fmt.Errorf("deepgram-tts: %s: %w", command["type"], err)
			if in.PacketName() == internal_type.PacketNameTextToSpeechInterrupt {
				return err
			}
			t.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: in.ContextId(), Error: err, Type: internal_type.TTSNetworkTimeout})
			return nil
		}
		switch in.(type) {
		case internal_type.TextToSpeechInterruptPacket:
			t.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: in.ContextId(), Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
				},
			})
		case internal_type.TextToSpeechTextPacket:
			t.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: in.ContextId(), Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
					Attributes: observability.Attributes{"type": "speaking", "text": command["text"]}, OccurredAt: time.Now(),
				},
			})
		}
		return nil
	}
}

// Close gracefully closes the Deepgram connection
func (t *deepgramTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.writeMu.Lock()
	t.stateMu.Lock()
	connectedAt := t.ttsConnectedAt
	t.ttsConnectedAt = time.Time{}

	conn := t.connection
	t.connection = nil
	if t.clearDone != nil {
		close(t.clearDone)
		t.clearDone = nil
	}
	t.stateMu.Unlock()
	if conn != nil {
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
		_ = conn.WriteJSON(map[string]string{"type": "Close"})
		_ = conn.Close()
	}
	t.writeMu.Unlock()
	t.readers.Wait()
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
