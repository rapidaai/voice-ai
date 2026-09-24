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
	"net"
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

type elevenlabsTTSContext struct {
	startedAt  time.Time
	textClosed bool
}

type elevenlabsTTS struct {
	*elevenLabsOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	// writeMu serializes dialing and writes; stateMu protects connection and context ownership.
	stateMu        sync.Mutex
	writeMu        sync.Mutex
	connection     *websocket.Conn
	contexts       map[string]*elevenlabsTTSContext
	ttsConnectedAt time.Time
	closed         bool
	workers        sync.WaitGroup

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
		contexts:         make(map[string]*elevenlabsTTSContext),
		onPacket:         onPacket,
		logger:           logger,
		elevenLabsOption: eleOpts,
	}, nil
}

func (*elevenlabsTTS) Name() string {
	return "elevenlabs-tts"
}

func (t *elevenlabsTTS) Initialize() error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.connect(t.ctx)
}

// connect requires writeMu ownership and starts one reader and keepalive per socket.
func (t *elevenlabsTTS) connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := t.ctx.Err(); err != nil {
		return err
	}
	t.stateMu.Lock()
	if t.connection != nil {
		t.stateMu.Unlock()
		return nil
	}
	t.stateMu.Unlock()
	start := time.Now()
	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	stopCancel := context.AfterFunc(t.ctx, cancel)
	defer stopCancel()
	header := http.Header{}
	header.Set("xi-api-key", t.GetKey())
	dialer := *websocket.DefaultDialer
	dial := dialer.NetDialContext
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	if dialer.NetDialTLSContext != nil {
		dial = dialer.NetDialTLSContext
	}
	// Close the transport on cancellation while Gorilla waits for the HTTP upgrade.
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
	conn, resp, err := dialer.DialContext(dialCtx, t.GetTextToSpeechConnectionString(), header)
	if stopDial != nil {
		stopDial()
	}
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return fmt.Errorf("elevenlabs-tts: connect failed: %w", err)
	}
	if err := dialCtx.Err(); err != nil {
		_ = conn.Close()
		return err
	}
	t.stateMu.Lock()
	if t.ctx.Err() != nil {
		t.stateMu.Unlock()
		_ = conn.Close()
		return t.ctx.Err()
	}
	t.connection = conn
	if t.ttsConnectedAt.IsZero() {
		t.ttsConnectedAt = time.Now()
	}
	done := make(chan struct{})
	stopClose := context.AfterFunc(t.ctx, func() { _ = conn.Close() })
	t.workers.Add(2)
	t.stateMu.Unlock()
	go func() {
		defer t.workers.Done()
		defer close(done)
		defer stopClose()
		defer conn.Close()
		t.readLoop(conn)
	}()
	go func() {
		defer t.workers.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-t.ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				t.writeMu.Lock()
				t.stateMu.Lock()
				if t.connection != conn || t.closed || t.ctx.Err() != nil {
					t.stateMu.Unlock()
					t.writeMu.Unlock()
					return
				}
				var messages []map[string]interface{}
				for contextID, synthesis := range t.contexts {
					if synthesis == nil || synthesis.textClosed {
						continue
					}
					messages = append(messages, map[string]interface{}{"text": "", "context_id": contextID})
				}
				t.stateMu.Unlock()
				if len(messages) == 0 {
					messages = append(messages, map[string]interface{}{"text": ""})
				}
				for _, message := range messages {
					if err := conn.WriteJSON(message); err != nil {
						_ = conn.Close()
						t.writeMu.Unlock()
						return
					}
				}
				t.writeMu.Unlock()
			}
		}
	}()
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
				Message: "elevenlabs-tts: initialization completed",
				Attributes: observability.Attributes{
					"component": observability.ComponentTTS.String(),
					"provider":  t.Name(),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (t *elevenlabsTTS) readLoop(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		var response struct {
			elevenlabs_internal.ElevenlabTextToSpeechResponse
			Error json.RawMessage `json:"error"`
		}
		var audio []byte
		if err != nil {
			err = fmt.Errorf("elevenlabs-tts: receive failed: %w", err)
		} else if err = json.Unmarshal(message, &response); err != nil {
			err = fmt.Errorf("elevenlabs-tts: invalid response: %w", err)
		} else if response.ContextId == nil && len(response.Error) > 0 && string(response.Error) != "null" {
			err = fmt.Errorf("elevenlabs-tts: provider error: %s", response.Error)
		}
		t.stateMu.Lock()
		if t.connection != conn || t.ctx.Err() != nil {
			t.stateMu.Unlock()
			return
		}
		if err == nil {
			if response.ContextId == nil || t.contexts[*response.ContextId] == nil {
				t.stateMu.Unlock()
				continue
			}
			if response.Audio != "" && (len(response.Error) == 0 || string(response.Error) == "null") {
				if audio, err = base64.StdEncoding.DecodeString(response.Audio); err != nil {
					err = fmt.Errorf("elevenlabs-tts: invalid audio: %w", err)
				}
			}
		}
		var packets []internal_type.Packet
		if err != nil {
			t.connection = nil
			for contextID, synthesis := range t.contexts {
				if synthesis != nil {
					t.contexts[contextID] = nil
					packets = append(packets, internal_type.TextToSpeechErrorPacket{
						ContextID: contextID, Error: err, Type: internal_type.TTSNetworkTimeout,
					})
				}
			}
			t.stateMu.Unlock()
			// Release blocked writes before publishing to a potentially blocked consumer.
			_ = conn.Close()
			if len(packets) > 0 {
				t.onPacket(packets...)
			}
			return
		}
		contextID := *response.ContextId
		synthesis := t.contexts[contextID]
		if len(response.Error) > 0 && string(response.Error) != "null" {
			packets = append(packets, internal_type.TextToSpeechErrorPacket{
				ContextID: contextID, Type: internal_type.TTSUnknownError,
				Error: fmt.Errorf("elevenlabs-tts: provider error: %s", response.Error),
			})
			if !t.writeMu.TryLock() {
				t.connection = nil
				for activeContextID, activeSynthesis := range t.contexts {
					if activeSynthesis != nil {
						t.contexts[activeContextID] = nil
						if activeContextID != contextID {
							packets = append(packets, internal_type.TextToSpeechErrorPacket{
								ContextID: activeContextID, Type: internal_type.TTSNetworkTimeout,
								Error: fmt.Errorf("elevenlabs-tts: connection closed after context %s failed", contextID),
							})
						}
					}
				}
				t.stateMu.Unlock()
				_ = conn.Close()
				t.onPacket(packets...)
				return
			}
			// Keep the failed context interruptible until its close write has finished.
			synthesis.textClosed = true
			t.stateMu.Unlock()
			err := conn.WriteJSON(map[string]interface{}{"context_id": contextID, "close_context": true})
			t.stateMu.Lock()
			t.contexts[contextID] = nil
			if err != nil && t.connection == conn {
				t.connection = nil
				for activeContextID, activeSynthesis := range t.contexts {
					if activeSynthesis != nil {
						t.contexts[activeContextID] = nil
						packets = append(packets, internal_type.TextToSpeechErrorPacket{
							ContextID: activeContextID, Type: internal_type.TTSNetworkTimeout,
							Error: fmt.Errorf("elevenlabs-tts: close failed context: %w", err),
						})
					}
				}
			}
			t.stateMu.Unlock()
			if err != nil {
				_ = conn.Close()
			}
			t.writeMu.Unlock()
			t.onPacket(packets...)
			continue
		}
		if response.Audio != "" {
			if !synthesis.startedAt.IsZero() {
				packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record:    observability.NewMetricTTSLatencyMs(time.Since(synthesis.startedAt), observability.Attributes{"provider": t.Name()}),
				})
				synthesis.startedAt = time.Time{}
			}
			packets = append(packets, internal_type.TextToSpeechAudioPacket{ContextID: contextID, AudioChunk: audio})
		}
		if response.IsFinal != nil && *response.IsFinal && synthesis.textClosed {
			t.contexts[contextID] = nil
			packets = append(packets,
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
		}
		t.stateMu.Unlock()
		if len(packets) > 0 {
			t.onPacket(packets...)
		}
	}
}

func (t *elevenlabsTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := t.ctx.Err(); err != nil {
		return err
	}
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		return nil
	case internal_type.TextToSpeechInterruptPacket:
		t.stateMu.Lock()
		if err := t.ctx.Err(); err != nil {
			t.stateMu.Unlock()
			return err
		}
		synthesis := t.contexts[input.ContextID]
		t.contexts[input.ContextID] = nil
		if synthesis == nil || t.connection == nil {
			t.stateMu.Unlock()
			return nil
		}
		if !t.writeMu.TryLock() {
			connection := t.connection
			t.connection = nil
			for contextID := range t.contexts {
				t.contexts[contextID] = nil
			}
			t.stateMu.Unlock()
			_ = connection.Close()
			t.onPacket(internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
					Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
				},
			})
			return nil
		}
		t.stateMu.Unlock()
	default:
		t.writeMu.Lock()
	}
	var packets []internal_type.Packet
	defer func() {
		t.writeMu.Unlock()
		if len(packets) > 0 {
			t.onPacket(packets...)
		}
	}()

	t.stateMu.Lock()
	if err := ctx.Err(); err != nil {
		t.stateMu.Unlock()
		return err
	}
	if err := t.ctx.Err(); err != nil {
		t.stateMu.Unlock()
		return err
	}
	var messages []map[string]interface{}
	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		messages = append(messages, map[string]interface{}{"context_id": input.ContextID, "close_context": true})
	case internal_type.TextToSpeechTextPacket:
		if input.Text == "" {
			t.stateMu.Unlock()
			return nil
		}
		if input.ContextID == "" {
			t.stateMu.Unlock()
			return fmt.Errorf("elevenlabs-tts: text requires a context ID")
		}
		synthesis, exists := t.contexts[input.ContextID]
		if exists && (synthesis == nil || synthesis.textClosed) {
			t.stateMu.Unlock()
			return nil
		}
		if !exists {
			for contextID, synthesis := range t.contexts {
				if synthesis != nil {
					t.contexts[contextID] = nil
					messages = append(messages, map[string]interface{}{"context_id": contextID, "close_context": true})
				}
			}
			t.contexts[input.ContextID] = &elevenlabsTTSContext{startedAt: time.Now()}
			messages = append(messages, map[string]interface{}{
				"text": " ", "context_id": input.ContextID,
				"voice_settings": map[string]interface{}{"stability": 0.5, "similarity_boost": 0.75},
			})
		}
		messages = append(messages, map[string]interface{}{"text": input.Text, "context_id": input.ContextID})
	case internal_type.TextToSpeechDonePacket:
		synthesis := t.contexts[input.ContextID]
		if synthesis == nil || synthesis.textClosed || t.connection == nil {
			t.stateMu.Unlock()
			return nil
		}
		synthesis.textClosed = true
		messages = append(messages,
			map[string]interface{}{"context_id": input.ContextID, "flush": true},
			map[string]interface{}{"context_id": input.ContextID, "close_context": true},
		)
	default:
		t.stateMu.Unlock()
		return fmt.Errorf("elevenlabs-tts: unsupported input type %T", in)
	}
	t.stateMu.Unlock()

	for _, message := range messages {
		t.stateMu.Lock()
		if err := ctx.Err(); err != nil {
			connection := t.connection
			t.connection = nil
			for contextID := range t.contexts {
				t.contexts[contextID] = nil
			}
			t.contexts[in.ContextId()] = nil
			t.stateMu.Unlock()
			if connection != nil {
				_ = connection.Close()
			}
			return err
		}
		if err := t.ctx.Err(); err != nil {
			t.stateMu.Unlock()
			return err
		}
		contextID := message["context_id"].(string)
		if _, text := in.(internal_type.TextToSpeechTextPacket); text && contextID == in.ContextId() {
			if synthesis, exists := t.contexts[contextID]; exists && synthesis == nil {
				t.stateMu.Unlock()
				return nil
			}
			t.stateMu.Unlock()
			if err := t.connect(ctx); err != nil {
				t.stateMu.Lock()
				synthesis, exists := t.contexts[contextID]
				t.contexts[contextID] = nil
				t.stateMu.Unlock()
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if t.ctx.Err() != nil {
					return t.ctx.Err()
				}
				if !exists || synthesis != nil {
					packets = append(packets, internal_type.TextToSpeechErrorPacket{
						ContextID: contextID, Error: err, Type: internal_type.TTSNetworkTimeout,
					})
				}
				return nil
			}
			t.stateMu.Lock()
			if synthesis, exists := t.contexts[contextID]; exists && synthesis == nil {
				t.stateMu.Unlock()
				return nil
			}
			if message["voice_settings"] != nil {
				t.contexts[contextID].startedAt = time.Now()
			}
		}
		connection := t.connection
		t.stateMu.Unlock()
		if connection == nil {
			if contextID != in.ContextId() {
				continue
			}
			return nil
		}

		writeCancelled := make(chan struct{})
		stop := context.AfterFunc(ctx, func() {
			t.stateMu.Lock()
			if t.connection == connection {
				t.connection = nil
				for contextID := range t.contexts {
					t.contexts[contextID] = nil
				}
			}
			t.contexts[in.ContextId()] = nil
			t.stateMu.Unlock()
			_ = connection.Close()
			close(writeCancelled)
		})
		err := connection.WriteJSON(message)
		if !stop() {
			<-writeCancelled
		}
		t.stateMu.Lock()
		if ctx.Err() != nil || t.ctx.Err() != nil {
			if t.connection == connection {
				t.connection = nil
				for contextID := range t.contexts {
					t.contexts[contextID] = nil
				}
			}
			t.contexts[in.ContextId()] = nil
			t.stateMu.Unlock()
			_ = connection.Close()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return t.ctx.Err()
		}
		if t.connection != connection {
			t.stateMu.Unlock()
			if contextID != in.ContextId() {
				continue
			}
			return nil
		}
		if err != nil {
			t.connection = nil
			for activeContextID, synthesis := range t.contexts {
				if synthesis != nil {
					t.contexts[activeContextID] = nil
					packets = append(packets, internal_type.TextToSpeechErrorPacket{
						ContextID: activeContextID, Error: fmt.Errorf("elevenlabs-tts: send failed: %w", err), Type: internal_type.TTSNetworkTimeout,
					})
				}
			}
			t.stateMu.Unlock()
			_ = connection.Close()
			if contextID != in.ContextId() {
				continue
			}
			return nil
		}
		t.stateMu.Unlock()
	}
	switch input := in.(type) {
	case internal_type.TextToSpeechTextPacket:
		packets = append(packets, internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
				Attributes: observability.Attributes{"type": "speaking", "text": input.Text}, OccurredAt: time.Now(),
			},
		})
	case internal_type.TextToSpeechInterruptPacket:
		packets = append(packets, internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
			},
		})
	}
	return nil
}

func (t *elevenlabsTTS) Close(ctx context.Context) error {
	t.ctxCancel()
	t.writeMu.Lock()
	t.stateMu.Lock()
	if t.closed {
		t.stateMu.Unlock()
		t.writeMu.Unlock()
		t.workers.Wait()
		return nil
	}
	t.closed = true
	connection := t.connection
	t.connection = nil
	t.contexts = nil
	connectedAt := t.ttsConnectedAt
	t.ttsConnectedAt = time.Time{}
	t.stateMu.Unlock()
	if connection != nil {
		_ = connection.Close()
	}
	t.writeMu.Unlock()
	t.workers.Wait()
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
	t.onPacket(internal_type.ObservabilityEventRecordPacket{
		Scope: internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS, Event: observability.TTSClosed,
			Attributes: observability.Attributes{"type": "closed", "provider": t.Name()}, OccurredAt: time.Now(),
		},
	})
	return nil
}
