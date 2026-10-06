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
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	resembleai_internal "github.com/rapidaai/api/assistant-api/internal/transformer/resembleai/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"golang.org/x/sync/semaphore"
)

type resembleaiTTS struct {
	*resembleaiOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	// writeLock permits cancellation while waiting for a writer; stateMu protects message ownership.
	writeLock        *semaphore.Weighted
	stateMu          sync.Mutex
	workers          sync.WaitGroup
	connection       *websocket.Conn
	contextId        string
	ttsConnectedAt   time.Time
	ttsStartedAt     time.Time
	textClosed       bool
	synthesisPending bool
	pendingDrains    int

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewResembleAITextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	resembleaiOpts, err := NewResembleAIOption(logger, credential, opts)
	if err != nil {
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(ctx)
	return &resembleaiTTS{
		ctx: ct, ctxCancel: ctxCancel,
		writeLock: semaphore.NewWeighted(1),
		logger:    logger, resembleaiOption: resembleaiOpts, onPacket: onPacket,
	}, nil
}

func (*resembleaiTTS) Name() string { return "resembleai-tts" }

func (rt *resembleaiTTS) Initialize() error {
	if err := rt.writeLock.Acquire(rt.ctx, 1); err != nil {
		return err
	}
	defer rt.writeLock.Release(1)
	return rt.connect(rt.ctx)
}

// connect requires writeLock ownership. A failed request is never replayed.
func (rt *resembleaiTTS) connect(ctx context.Context) error {
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
	start := time.Now()
	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	stopCancel := context.AfterFunc(rt.ctx, cancel)
	defer stopCancel()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+rt.GetKey())
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
	conn, resp, err := dialer.DialContext(dialCtx, RESEMBLEAI_WS_URL, header)
	if stopDial != nil {
		stopDial()
	}
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err := dialCtx.Err(); err != nil {
			return err
		}
		return fmt.Errorf("resembleai-tts: connect failed: %w", err)
	}
	if err := dialCtx.Err(); err != nil {
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
	rt.stateMu.Unlock()
	rt.workers.Go(func() { rt.readLoop(conn) })
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
				Level: observability.LevelInfo, Message: "resembleai-tts: initialization completed",
				Attributes: observability.Attributes{"component": observability.ComponentTTS.String(), "provider": rt.Name()},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (rt *resembleaiTTS) readLoop(conn *websocket.Conn) {
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
		var response resembleai_internal.ResembleAITextToSpeechResponse
		var audio []byte
		if err != nil {
			err = fmt.Errorf("resembleai-tts: read: %w", err)
		} else {
			errorType = internal_type.TTSUnknownError
			if err = json.Unmarshal(message, &response); err != nil {
				err = fmt.Errorf("resembleai-tts: response: %w", err)
			} else {
				switch response.Type {
				case "audio":
					if audio, err = base64.StdEncoding.DecodeString(response.AudioContent); err != nil {
						err = fmt.Errorf("resembleai-tts: audio encoding: %w", err)
					}
				case "error":
					errorType = internal_type.TTSInvalidInput
					err = fmt.Errorf("resembleai-tts: server error: %s", message)
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
			if len(audio) == 0 {
				rt.stateMu.Unlock()
				continue
			}
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
		case "audio_end":
			if rt.pendingDrains == 0 {
				rt.stateMu.Unlock()
				continue
			}
			rt.pendingDrains--
			if !rt.textClosed || rt.pendingDrains > 0 {
				rt.stateMu.Unlock()
				continue
			}
			rt.stateMu.Unlock()
			// The last audio_end can arrive before its text write returns.
			if err := rt.writeLock.Acquire(rt.ctx, 1); err != nil {
				return
			}
			rt.stateMu.Lock()
			if rt.connection != conn || !rt.synthesisPending || rt.ctx.Err() != nil {
				rt.stateMu.Unlock()
				rt.writeLock.Release(1)
				return
			}
			rt.connection = nil
			rt.synthesisPending = false
			rt.stateMu.Unlock()
			_ = conn.Close()
			rt.writeLock.Release(1)
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
			return
		default:
			rt.stateMu.Unlock()
		}
	}
}

func (rt *resembleaiTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rt.ctx.Err(); err != nil {
		return err
	}
	switch input := in.(type) {
	case internal_type.TurnChangePacket:
		return nil
	case internal_type.TextToSpeechTextPacket:
		if input.ContextID == "" || input.Text == "" {
			return nil
		}
		rt.stateMu.Lock()
		if input.ContextID == rt.contextId && (!rt.synthesisPending || rt.textClosed) {
			rt.stateMu.Unlock()
			return nil
		}
		var previousConnection *websocket.Conn
		if input.ContextID != rt.contextId {
			if rt.synthesisPending {
				previousConnection = rt.connection
				rt.connection = nil
			}
			rt.contextId = input.ContextID
			rt.textClosed = false
			rt.pendingDrains = 0
			rt.synthesisPending = true
			rt.ttsStartedAt = time.Now()
		}
		// Count admitted requests so Done also waits for text queued behind a writer.
		rt.pendingDrains++
		rt.stateMu.Unlock()
		if previousConnection != nil {
			_ = previousConnection.Close()
		}
		defer func() {
			if ctx.Err() == nil {
				return
			}
			rt.stateMu.Lock()
			if input.ContextID != rt.contextId {
				rt.stateMu.Unlock()
				return
			}
			connection := rt.connection
			rt.connection = nil
			rt.synthesisPending = false
			rt.stateMu.Unlock()
			if connection != nil {
				_ = connection.Close()
			}
		}()
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
		// ResembleAI audio is untagged. Closing the socket discards synthesis and unblocks any writer.
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

	writeCtx, cancelWrite := context.WithCancel(ctx)
	defer cancelWrite()
	stopSession := context.AfterFunc(rt.ctx, cancelWrite)
	defer stopSession()
	if err := rt.writeLock.Acquire(writeCtx, 1); err != nil {
		return err
	}
	var packets []internal_type.Packet
	defer func() {
		rt.writeLock.Release(1)
		if len(packets) > 0 {
			rt.onPacket(packets...)
		}
	}()
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
		if input.ContextID != rt.contextId || !rt.synthesisPending {
			rt.stateMu.Unlock()
			return nil
		}
		rt.stateMu.Unlock()
		if err := rt.connect(ctx); err != nil {
			rt.stateMu.Lock()
			emitError := input.ContextID == rt.contextId && rt.synthesisPending && ctx.Err() == nil && rt.ctx.Err() == nil
			if input.ContextID == rt.contextId {
				rt.synthesisPending = false
			}
			rt.stateMu.Unlock()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if rt.ctx.Err() != nil {
				return rt.ctx.Err()
			}
			if emitError {
				packets = append(packets, internal_type.TextToSpeechErrorPacket{ContextID: input.ContextID, Error: err, Type: internal_type.TTSNetworkTimeout})
			}
			return nil
		}
		rt.stateMu.Lock()
		if input.ContextID != rt.contextId || rt.connection == nil || !rt.synthesisPending {
			rt.stateMu.Unlock()
			return nil
		}
		message = map[string]interface{}{
			"voice_uuid":      rt.GetVoiceUUID(),
			"data":            input.Text,
			"output_format":   "wav",
			"sample_rate":     rt.GetSampleRate(),
			"precision":       "PCM_16",
			"no_audio_header": true,
		}
	case internal_type.TextToSpeechDonePacket:
		if input.ContextID != rt.contextId || !rt.synthesisPending || rt.textClosed {
			rt.stateMu.Unlock()
			return nil
		}
		rt.textClosed = true
		if rt.pendingDrains > 0 {
			rt.stateMu.Unlock()
			return nil
		}
		connection := rt.connection
		rt.connection = nil
		rt.synthesisPending = false
		rt.stateMu.Unlock()
		_ = connection.Close()
		packets = append(packets,
			internal_type.TextToSpeechEndPacket{ContextID: input.ContextID},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS, Event: observability.TTSCompleted,
					Attributes: observability.Attributes{"type": "completed"}, OccurredAt: time.Now(),
				},
			},
		)
		return nil
	default:
		rt.stateMu.Unlock()
		return fmt.Errorf("resembleai-tts: unsupported packet type %T", in)
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
		packets = append(packets, internal_type.TextToSpeechErrorPacket{
			ContextID: in.ContextId(), Error: fmt.Errorf("resembleai-tts: send: %w", err), Type: internal_type.TTSNetworkTimeout,
		})
		return nil
	}
	if input, ok := in.(internal_type.TextToSpeechTextPacket); ok {
		packets = append(packets, internal_type.ObservabilityEventRecordPacket{
			ContextID: input.ContextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.RecordEvent{
				Component: observability.ComponentTTS, Event: observability.TTSSpeaking,
				Attributes: observability.Attributes{"type": "speaking", "text": input.Text}, OccurredAt: time.Now(),
			},
		})
	}
	return nil
}

func (rt *resembleaiTTS) Close(ctx context.Context) error {
	rt.ctxCancel()
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
	_ = rt.writeLock.Acquire(context.WithoutCancel(ctx), 1)
	rt.writeLock.Release(1)
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
