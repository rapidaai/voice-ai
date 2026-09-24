// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_sarvam

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
	sarvam_internal "github.com/rapidaai/api/assistant-api/internal/transformer/sarvam/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type sarvamTextToSpeech struct {
	*sarvamOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	// writeMu serializes dialing and writes; stateMu protects connection and message ownership.
	writeMu          sync.Mutex
	stateMu          sync.Mutex
	workers          sync.WaitGroup
	connection       *websocket.Conn
	contextId        string
	ttsConnectedAt   time.Time
	ttsStartedAt     time.Time
	textClosed       bool
	synthesisPending bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewSarvamTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	sarvamOpts, err := NewSarvamOption(logger, credential, opts)
	if err != nil {
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(ctx)
	return &sarvamTextToSpeech{
		ctx: ct, ctxCancel: ctxCancel,
		logger: logger, sarvamOption: sarvamOpts, onPacket: onPacket,
	}, nil
}

func (*sarvamTextToSpeech) Name() string { return "sarvam-tts" }

func (rt *sarvamTextToSpeech) Initialize() error {
	rt.writeMu.Lock()
	defer rt.writeMu.Unlock()
	return rt.connect(rt.ctx)
}

// connect requires writeMu ownership. A failed request is never replayed.
func (rt *sarvamTextToSpeech) connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rt.ctx.Err(); err != nil {
		return err
	}
	rt.stateMu.Lock()
	if rt.connection != nil {
		rt.stateMu.Unlock()
		return nil
	}
	rt.stateMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	stop := context.AfterFunc(rt.ctx, cancel)
	defer stop()
	start := time.Now()
	header := http.Header{}
	header.Set("Api-Subscription-Key", rt.GetKey())
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, rt.textToSpeechUrl(), header)
	if err != nil {
		return fmt.Errorf("sarvam-tts: connect: %w", err)
	}
	canceled := make(chan struct{})
	stopWrite := context.AfterFunc(ctx, func() { _ = conn.Close(); close(canceled) })
	err = conn.WriteJSON(rt.configureTextToSpeech())
	if !stopWrite() {
		<-canceled
		if err == nil {
			err = ctx.Err()
		}
	}
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("sarvam-tts: config: %w", err)
	}
	if err := ctx.Err(); err != nil {
		_ = conn.Close()
		return err
	}
	rt.stateMu.Lock()
	if err := rt.ctx.Err(); err != nil {
		rt.stateMu.Unlock()
		_ = conn.Close()
		return err
	}
	rt.connection = conn
	if rt.ttsConnectedAt.IsZero() {
		rt.ttsConnectedAt = time.Now()
	}
	rt.workers.Add(2)
	rt.stateMu.Unlock()
	done := make(chan struct{})
	go func() {
		defer rt.workers.Done()
		defer close(done)
		rt.readLoop(conn)
	}()
	go func() {
		defer rt.workers.Done()
		rt.keepalive(conn, done)
	}()
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
				Level: observability.LevelInfo, Message: "sarvam-tts: initialization completed",
				Attributes: observability.Attributes{"component": observability.ComponentTTS.String(), "provider": rt.Name()},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (rt *sarvamTextToSpeech) keepalive(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-rt.ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			rt.writeMu.Lock()
			rt.stateMu.Lock()
			if rt.connection != conn || rt.ctx.Err() != nil {
				rt.stateMu.Unlock()
				rt.writeMu.Unlock()
				return
			}
			rt.stateMu.Unlock()
			if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
				// Closing wakes the reader, which owns error reporting and failed-socket cleanup.
				rt.logger.Errorf("sarvam-tts: ping: %v", err)
				_ = conn.Close()
				rt.writeMu.Unlock()
				return
			}
			rt.writeMu.Unlock()
		}
	}
}

func (rt *sarvamTextToSpeech) readLoop(conn *websocket.Conn) {
	canceled := make(chan struct{})
	stop := context.AfterFunc(rt.ctx, func() { _ = conn.Close(); close(canceled) })
	defer func() {
		if !stop() {
			<-canceled
		}
	}()
	defer conn.Close()
	for {
		_, message, err := conn.ReadMessage()
		errorType := internal_type.TTSNetworkTimeout
		var response sarvam_internal.SarvamTextToSpeechResponse
		var audio []byte
		var event struct {
			EventType string `json:"event_type"`
		}
		if err != nil {
			err = fmt.Errorf("sarvam-tts: read: %w", err)
		} else {
			errorType = internal_type.TTSUnknownError
			if err = json.Unmarshal(message, &response); err != nil {
				err = fmt.Errorf("sarvam-tts: response: %w", err)
			} else {
				switch response.Type {
				case "audio":
					payload, decodeErr := response.Audio()
					if decodeErr != nil {
						err = fmt.Errorf("sarvam-tts: audio: %w", decodeErr)
					} else if audio, err = base64.StdEncoding.DecodeString(payload.Audio); err != nil {
						err = fmt.Errorf("sarvam-tts: audio encoding: %w", err)
					}
				case "event":
					if err = json.Unmarshal(response.Data, &event); err != nil {
						err = fmt.Errorf("sarvam-tts: event: %w", err)
					}
				case "error":
					err = fmt.Errorf("sarvam-tts: unknown provider error")
					if payload, decodeErr := response.AsError(); decodeErr == nil {
						err = fmt.Errorf("sarvam-tts: %s", payload.Message)
					}
				}
			}
		}
		rt.stateMu.Lock()
		if rt.connection != conn {
			rt.stateMu.Unlock()
			return
		}
		contextID := rt.contextId
		if err != nil {
			rt.connection = nil
			emitError := rt.synthesisPending && rt.ctx.Err() == nil
			rt.synthesisPending = false
			rt.stateMu.Unlock()
			// Unblock the writer before publishing an error to a potentially blocked consumer.
			_ = conn.Close()
			if emitError {
				rt.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: contextID, Error: err, Type: errorType})
			}
			return
		}
		if !rt.synthesisPending || rt.ctx.Err() != nil {
			rt.stateMu.Unlock()
			continue
		}
		switch response.Type {
		case "audio":
			packets := []internal_type.Packet{}
			if !rt.ttsStartedAt.IsZero() {
				packets = append(packets, internal_type.ObservabilityMetricRecordPacket{
					ContextID: contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.NewMetricTTSLatencyMs(time.Since(rt.ttsStartedAt), observability.Attributes{"provider": rt.Name()}),
				})
				rt.ttsStartedAt = time.Time{}
			}
			packets = append(packets, internal_type.TextToSpeechAudioPacket{ContextID: contextID, AudioChunk: audio})
			rt.stateMu.Unlock()
			rt.onPacket(packets...)
		case "event":
			if event.EventType != "final" || !rt.textClosed {
				rt.stateMu.Unlock()
				continue
			}
			rt.synthesisPending = false
			rt.stateMu.Unlock()
			rt.onPacket(
				internal_type.TextToSpeechEndPacket{ContextID: contextID},
				internal_type.ObservabilityEventRecordPacket{
					ContextID: contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentTTS, Event: observability.TTSCompleted,
						Attributes: observability.Attributes{"type": "completed"}, OccurredAt: time.Now(),
					},
				},
			)
		default:
			rt.stateMu.Unlock()
		}
	}
}

func (rt *sarvamTextToSpeech) Transform(ctx context.Context, in internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rt.ctx.Err(); err != nil {
		return err
	}
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		return nil
	case internal_type.TextToSpeechInterruptPacket:
		rt.stateMu.Lock()
		if input.ContextID != rt.contextId || !rt.synthesisPending {
			rt.stateMu.Unlock()
			return nil
		}
		rt.synthesisPending = false
		connection := rt.connection
		rt.connection = nil
		rt.stateMu.Unlock()
		// Sarvam audio is untagged. Closing the socket discards synthesis and unblocks any writer.
		if connection != nil {
			_ = connection.Close()
		}
		rt.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"}, OccurredAt: time.Now(),
			},
		})
		return nil
	}

	rt.writeMu.Lock()
	defer rt.writeMu.Unlock()
	rt.stateMu.Lock()
	if err := ctx.Err(); err != nil {
		rt.stateMu.Unlock()
		return err
	}
	if err := rt.ctx.Err(); err != nil {
		rt.stateMu.Unlock()
		return err
	}
	var message any
	switch input := in.(type) {
	case internal_type.TextToSpeechTextPacket:
		if input.ContextID == "" || (input.ContextID == rt.contextId && (!rt.synthesisPending || rt.textClosed)) {
			rt.stateMu.Unlock()
			return nil
		}
		if input.ContextID != rt.contextId {
			if rt.synthesisPending && rt.connection != nil {
				connection := rt.connection
				rt.connection = nil
				rt.stateMu.Unlock()
				_ = connection.Close()
				rt.stateMu.Lock()
			}
			rt.contextId = input.ContextID
			rt.textClosed = false
			rt.synthesisPending = true
			rt.ttsStartedAt = time.Now()
		}
		rt.stateMu.Unlock()
		if err := rt.connect(ctx); err != nil {
			rt.stateMu.Lock()
			emitError := rt.synthesisPending && ctx.Err() == nil && rt.ctx.Err() == nil
			rt.synthesisPending = false
			rt.stateMu.Unlock()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if rt.ctx.Err() != nil {
				return rt.ctx.Err()
			}
			if emitError {
				rt.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: input.ContextID, Error: err, Type: internal_type.TTSNetworkTimeout})
			}
			return nil
		}
		rt.stateMu.Lock()
		if rt.connection == nil || !rt.synthesisPending {
			rt.stateMu.Unlock()
			return nil
		}
		message = map[string]any{"type": "text", "data": map[string]string{"text": input.Text}}
	case internal_type.TextToSpeechDonePacket:
		if input.ContextID != rt.contextId || rt.connection == nil || !rt.synthesisPending || rt.textClosed {
			rt.stateMu.Unlock()
			return nil
		}
		rt.textClosed = true
		message = map[string]string{"type": "flush"}
	default:
		rt.stateMu.Unlock()
		return fmt.Errorf("sarvam-tts: unsupported packet type %T", in)
	}
	connection := rt.connection
	rt.stateMu.Unlock()

	writeCancelled := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		rt.stateMu.Lock()
		if rt.connection == connection {
			rt.connection = nil
			rt.synthesisPending = false
		}
		rt.stateMu.Unlock()
		_ = connection.Close()
		close(writeCancelled)
	})
	err := connection.WriteJSON(message)
	if !stop() {
		<-writeCancelled
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if rt.ctx.Err() != nil {
		return rt.ctx.Err()
	}
	if err != nil {
		rt.stateMu.Lock()
		if rt.connection != connection {
			rt.stateMu.Unlock()
			return nil
		}
		rt.connection = nil
		rt.synthesisPending = false
		rt.stateMu.Unlock()
		_ = connection.Close()
		rt.onPacket(internal_type.TextToSpeechErrorPacket{
			ContextID: in.ContextId(), Error: fmt.Errorf("sarvam-tts: send: %w", err), Type: internal_type.TTSNetworkTimeout,
		})
		return nil
	}
	if input, ok := in.(internal_type.TextToSpeechTextPacket); ok {
		rt.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
				Attributes: observability.Attributes{"type": "speaking", "text": input.Text}, OccurredAt: time.Now(),
			},
		})
	}
	return nil
}

func (rt *sarvamTextToSpeech) Close(ctx context.Context) error {
	rt.ctxCancel()
	rt.writeMu.Lock()
	rt.stateMu.Lock()
	connectedAt := rt.ttsConnectedAt
	rt.ttsConnectedAt = time.Time{}
	conn := rt.connection
	rt.connection = nil
	rt.synthesisPending = false
	rt.stateMu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	rt.writeMu.Unlock()
	rt.workers.Wait()
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
	rt.onPacket(internal_type.ObservabilityEventRecordPacket{
		Scope: internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS, Event: observability.TTSClosed,
			Attributes: observability.Attributes{"type": "closed", "provider": rt.Name()}, OccurredAt: time.Now(),
		},
	})
	return nil
}
