// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_neuphonic

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
	neuphonic_internal "github.com/rapidaai/api/assistant-api/internal/transformer/neuphonic/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"golang.org/x/sync/semaphore"
)

const neuphonicCompletionTimeout = 3 * time.Second

type neuphonicTTS struct {
	*neuphonicOption
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
	hasAudio         bool
	textClosed       bool
	synthesisPending bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewNeuPhonicTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	neuphonicOpts, err := NewNeuPhonicOption(logger, credential, opts)
	if err != nil {
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(ctx)
	return &neuphonicTTS{
		ctx: ct, ctxCancel: ctxCancel,
		writeLock: semaphore.NewWeighted(1),
		logger:    logger, neuphonicOption: neuphonicOpts, onPacket: onPacket,
	}, nil
}

func (*neuphonicTTS) Name() string { return "neuphonic-tts" }

func (rt *neuphonicTTS) Initialize() error {
	if err := rt.writeLock.Acquire(rt.ctx, 1); err != nil {
		return err
	}
	defer rt.writeLock.Release(1)
	return rt.connect(rt.ctx)
}

// connect requires writeLock ownership. A failed request is never replayed.
func (rt *neuphonicTTS) connect(ctx context.Context) error {
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
	header.Set("x-api-key", rt.GetKey())
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
	conn, resp, err := dialer.DialContext(dialCtx, rt.GetTextToSpeechConnectionString(), header)
	if stopDial != nil {
		stopDial()
	}
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return fmt.Errorf("neuphonic-tts: connect failed: %w", err)
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
				Level: observability.LevelInfo, Message: "neuphonic-tts: initialization completed",
				Attributes: observability.Attributes{"component": observability.ComponentTTS.String(), "provider": rt.Name()},
				OccurredAt: time.Now(),
			},
		},
	)
	return nil
}

func (rt *neuphonicTTS) keepalive(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-rt.ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := rt.writeLock.Acquire(rt.ctx, 1); err != nil {
				return
			}
			rt.stateMu.Lock()
			if rt.connection != conn || rt.ctx.Err() != nil {
				rt.stateMu.Unlock()
				rt.writeLock.Release(1)
				return
			}
			rt.stateMu.Unlock()
			if err := conn.WriteJSON(map[string]string{"text": ""}); err != nil {
				// Closing wakes the reader, which owns error reporting and failed-socket cleanup.
				rt.logger.Errorf("neuphonic-tts: ping: %v", err)
				_ = conn.Close()
				rt.writeLock.Release(1)
				return
			}
			rt.writeLock.Release(1)
		}
	}
}

// readLoop serializes audio delivery and inferred completion for an untagged socket.
func (rt *neuphonicTTS) readLoop(conn *websocket.Conn) {
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
		readError := err
		errorType := internal_type.TTSNetworkTimeout
		var response neuphonic_internal.NeuPhonicTextToSpeechResponse
		var audio []byte
		if err != nil {
			err = fmt.Errorf("neuphonic-tts: read: %w", err)
		} else {
			errorType = internal_type.TTSUnknownError
			if err = json.Unmarshal(message, &response); err != nil {
				err = fmt.Errorf("neuphonic-tts: response: %w", err)
			} else if audio, err = base64.StdEncoding.DecodeString(response.Data.Audio); err != nil {
				err = fmt.Errorf("neuphonic-tts: audio encoding: %w", err)
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
			emitTerminal := rt.synthesisPending && rt.ctx.Err() == nil
			timeout, ok := readError.(net.Error)
			completed := ok && timeout.Timeout() && rt.textClosed && rt.hasAudio
			rt.synthesisPending = false
			rt.stateMu.Unlock()
			// Close before publishing: this socket must never deliver audio to another message.
			_ = conn.Close()
			if !emitTerminal {
				return
			}
			if completed {
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
			} else {
				rt.onPacket(internal_type.TextToSpeechErrorPacket{ContextID: contextID, Error: err, Type: errorType})
			}
			return
		}
		if !rt.synthesisPending || rt.ctx.Err() != nil || len(audio) == 0 {
			rt.stateMu.Unlock()
			continue
		}
		rt.hasAudio = true
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
		rt.stateMu.Lock()
		// Refresh after delivery so a blocked consumer cannot expire queued audio.
		if rt.connection == conn && rt.synthesisPending && rt.textClosed {
			if err := conn.UnderlyingConn().SetReadDeadline(time.Now().Add(neuphonicCompletionTimeout)); err != nil {
				_ = conn.Close()
			}
		}
		rt.stateMu.Unlock()
	}
}

func (rt *neuphonicTTS) Transform(ctx context.Context, in internal_type.Packet) error {
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
		// Neuphonic audio is untagged. Closing the socket discards synthesis and unblocks any writer.
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
	defer rt.writeLock.Release(1)
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
		if input.ContextID == "" || input.Text == "" || (input.ContextID == rt.contextId && (!rt.synthesisPending || rt.textClosed)) {
			rt.stateMu.Unlock()
			return nil
		}
		if input.ContextID != rt.contextId {
			var previousConnection *websocket.Conn
			if rt.synthesisPending {
				previousConnection = rt.connection
				rt.connection = nil
			}
			rt.contextId = input.ContextID
			rt.hasAudio = false
			rt.textClosed = false
			rt.synthesisPending = true
			rt.ttsStartedAt = time.Now()
			rt.stateMu.Unlock()
			if previousConnection != nil {
				_ = previousConnection.Close()
			}
		} else {
			rt.stateMu.Unlock()
		}
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
		message = map[string]string{"text": input.Text + " <STOP>"}
	case internal_type.TextToSpeechDonePacket:
		if input.ContextID != rt.contextId || rt.connection == nil || !rt.synthesisPending || rt.textClosed {
			rt.stateMu.Unlock()
			return nil
		}
		message = map[string]string{"text": "<STOP>"}
	default:
		rt.stateMu.Unlock()
		return fmt.Errorf("neuphonic-tts: unsupported packet type %T", in)
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
	if _, done := in.(internal_type.TextToSpeechDonePacket); done && err == nil {
		rt.stateMu.Lock()
		if rt.connection == connection && rt.synthesisPending {
			// Neuphonic has no terminal event. Infer completion only after text closes and audio goes idle.
			rt.textClosed = true
			err = connection.UnderlyingConn().SetReadDeadline(time.Now().Add(neuphonicCompletionTimeout))
		}
		rt.stateMu.Unlock()
	}
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
			ContextID: in.ContextId(), Error: fmt.Errorf("neuphonic-tts: send: %w", err), Type: internal_type.TTSNetworkTimeout,
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

func (rt *neuphonicTTS) Close(ctx context.Context) error {
	rt.ctxCancel()
	_ = rt.writeLock.Acquire(context.Background(), 1)
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
