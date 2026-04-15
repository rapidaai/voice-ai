// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_inworld

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	inworld_internal "github.com/rapidaai/api/assistant-api/internal/transformer/inworld/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// inworldTTS implements internal_type.TextToSpeechTransformer against
// Inworld's bidirectional streaming TTS WebSocket. A single WebSocket
// connection is held open per TTS turn; the turn ends when Inworld emits
// contextClosed (or done:true) in response to our close_context frame.
type inworldTTS struct {
	*inworldOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	connection     *websocket.Conn
	contextId      string // current Rapida turn id — also used as Inworld context_id
	contextCreated bool   // whether we've sent a create frame for contextId
	ttsConnectedAt time.Time
	ttsStartedAt   time.Time
	ttsMetricSent  bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

// NewInworldTextToSpeech constructs the Inworld TTS transformer. The caller
// is expected to call Initialize() before sending text.
func NewInworldTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	iwOpts, err := NewInworldOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("inworld-tts: initializing inworld failed %+v", err)
		return nil, err
	}
	ctx2, contextCancel := context.WithCancel(ctx)
	return &inworldTTS{
		ctx:           ctx2,
		ctxCancel:     contextCancel,
		onPacket:      onPacket,
		logger:        logger,
		inworldOption: iwOpts,
	}, nil
}

// Name identifies this transformer in logs and events.
func (*inworldTTS) Name() string {
	return "inworld-text-to-speech"
}

// Initialize opens a fresh WebSocket connection to Inworld and starts the
// read goroutine. Called at session start and after each interruption so the
// connection is warm before the first text delta arrives. The create frame
// is not sent here — it is deferred until the first delta so it can be
// keyed off the real Rapida context id.
func (it *inworldTTS) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Basic %s", it.GetKey()))
	conn, resp, err := websocket.DefaultDialer.Dial(it.GetTextToSpeechConnectionString(), header)
	if err != nil {
		it.logger.Errorf("inworld-tts: dial failed %s with response %v", err, resp)
		return err
	}

	it.mu.Lock()
	it.connection = conn
	it.contextCreated = false
	if it.ttsConnectedAt.IsZero() {
		it.ttsConnectedAt = time.Now()
	}
	it.mu.Unlock()

	go it.readLoop(conn)
	_ = it.onPacket(internal_type.ConversationEventPacket{
		Name: "tts",
		Data: map[string]string{
			"type":     "initialized",
			"provider": it.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})
	return nil
}

// readLoop owns a single WebSocket connection for the duration of one TTS
// turn. It exits when the connection closes — intentionally (interrupt /
// flush complete) or unexpectedly (network drop).
func (it *inworldTTS) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-it.ctx.Done():
			return
		default:
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			it.mu.Lock()
			intentional := it.connection == nil // set to nil before conn.Close() on intentional paths
			if !intentional {
				it.connection = nil // unintentional drop: next delta will reconnect
				it.contextCreated = false
			}
			it.mu.Unlock()
			if !intentional {
				it.logger.Errorf("inworld-tts: connection lost: %v", err)
			}
			return
		}

		var payload inworld_internal.InworldTTSResponse
		if err := json.Unmarshal(msg, &payload); err != nil {
			it.logger.Errorf("inworld-tts: error parsing response: %v", err)
			continue
		}

		if payload.Error != nil {
			it.logger.Errorf("inworld-tts: server error: %s", payload.Error.Message)
			continue
		}

		// Terminal: either a plain done:true frame or result.contextClosed.
		if (payload.Done != nil && *payload.Done) ||
			(payload.Result != nil && payload.Result.ContextClosed != nil) {
			it.handleFlushComplete(conn)
			return
		}

		if payload.Result == nil || payload.Result.AudioChunk == nil {
			continue
		}
		audioContent := payload.Result.AudioChunk.AudioContent
		if audioContent == "" {
			continue
		}

		rawAudio, err := base64.StdEncoding.DecodeString(audioContent)
		if err != nil {
			it.logger.Errorf("inworld-tts: base64 decode failed: %v", err)
			continue
		}

		it.mu.Lock()
		ctxId := it.contextId
		startedAt := it.ttsStartedAt
		metricSent := it.ttsMetricSent
		if !metricSent && !startedAt.IsZero() {
			it.ttsMetricSent = true
		}
		it.mu.Unlock()

		if ctxId == "" {
			// Audio arrived before a delta set the context id — unusual but
			// skip emission since downstream has nothing to correlate to.
			continue
		}
		if !metricSent && !startedAt.IsZero() {
			_ = it.onPacket(internal_type.AssistantMessageMetricPacket{
				ContextID: ctxId,
				Metrics: []*protos.Metric{{
					Name:  "tts_latency_ms",
					Value: fmt.Sprintf("%d", time.Since(startedAt).Milliseconds()),
				}},
			})
		}
		_ = it.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: ctxId, AudioChunk: rawAudio})
	}
}

// handleFlushComplete is called when Inworld signals end-of-context. It
// emits TextToSpeechEndPacket — correctly ordered after the last audio chunk
// — and closes the per-turn connection.
func (it *inworldTTS) handleFlushComplete(conn *websocket.Conn) {
	it.mu.Lock()
	ctxId := it.contextId
	it.connection = nil // mark before Close so readLoop error handler sees intentional
	it.contextCreated = false
	it.mu.Unlock()

	_ = it.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: ctxId},
		internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{"type": "completed"},
			Time: time.Now(),
		},
	)
	_ = conn.Close()
}

// Transform routes a single LLM/control packet into the Inworld protocol.
// Supported packet types:
//   - InterruptionDetectedPacket: tear down the connection and reconnect.
//   - LLMResponseDeltaPacket: create the context on first delta, then
//     forward the text with flush_context to keep audio flowing.
//   - LLMResponseDonePacket: send close_context to drain & end the turn.
func (it *inworldTTS) Transform(ctx context.Context, in internal_type.LLMPacket) error {
	it.mu.Lock()
	if in.ContextId() != it.contextId {
		it.contextId = in.ContextId()
		it.contextCreated = false
		it.ttsStartedAt = time.Time{}
		it.ttsMetricSent = false
	}
	connection := it.connection
	it.mu.Unlock()

	switch input := in.(type) {
	case internal_type.InterruptionDetectedPacket:
		it.mu.Lock()
		it.contextId = ""
		it.contextCreated = false
		it.ttsStartedAt = time.Time{}
		it.ttsMetricSent = false
		conn := it.connection
		it.connection = nil
		it.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		_ = it.onPacket(internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{"type": "interrupted"},
			Time: time.Now(),
		})
		if err := it.Initialize(); err != nil {
			it.logger.Errorf("inworld-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.LLMResponseDeltaPacket:
		// Fallback reconnect: handles Initialize() failure or an unintentional drop.
		if connection == nil {
			if err := it.Initialize(); err != nil {
				return fmt.Errorf("inworld-tts: failed to connect: %w", err)
			}
			it.mu.Lock()
			connection = it.connection
			if it.ttsStartedAt.IsZero() {
				it.ttsStartedAt = time.Now()
			}
			it.mu.Unlock()
		} else {
			it.mu.Lock()
			if it.ttsStartedAt.IsZero() {
				it.ttsStartedAt = time.Now()
			}
			it.mu.Unlock()
		}

		it.mu.Lock()
		ctxId := it.contextId
		needCreate := !it.contextCreated
		it.mu.Unlock()

		// Inworld requires a "create" frame to open each context before any
		// send_text. Send one lazily on the first delta for this context id.
		if needCreate {
			createFrame := inworld_internal.CreateRequest{
				ContextID: ctxId,
				Create: inworld_internal.CreateBody{
					VoiceID: it.GetVoiceID(),
					ModelID: it.GetModelID(),
					AudioConfig: inworld_internal.AudioConfig{
						AudioEncoding:   INWORLD_AUDIO_ENCODING,
						SampleRateHertz: INWORLD_SAMPLE_RATE,
					},
				},
			}
			if err := connection.WriteJSON(createFrame); err != nil {
				it.logger.Errorf("inworld-tts: create frame write failed: %v", err)
				return err
			}
			it.mu.Lock()
			it.contextCreated = true
			it.mu.Unlock()
		}

		// flush_context on every send_text keeps the server synthesizing as
		// text arrives rather than buffering for end-of-input.
		sendFrame := inworld_internal.SendTextRequest{
			ContextID: ctxId,
			SendText: inworld_internal.SendTextBody{
				Text:         input.Text,
				FlushContext: map[string]interface{}{},
			},
		}
		if err := connection.WriteJSON(sendFrame); err != nil {
			it.logger.Errorf("inworld-tts: send_text write failed: %v", err)
			return err
		}
		_ = it.onPacket(internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{"type": "speaking", "text": input.Text},
			Time: time.Now(),
		})

	case internal_type.LLMResponseDonePacket:
		// Interrupted before done arrived — nothing to flush.
		if connection == nil {
			return nil
		}
		it.mu.Lock()
		ctxId := it.contextId
		created := it.contextCreated
		it.mu.Unlock()
		// If we never opened a context (empty response), there is nothing to close.
		if !created || ctxId == "" {
			return nil
		}
		closeFrame := inworld_internal.CloseContextRequest{
			ContextID:    ctxId,
			CloseContext: map[string]interface{}{},
		}
		if err := connection.WriteJSON(closeFrame); err != nil {
			it.logger.Errorf("inworld-tts: close_context write failed: %v", err)
			return err
		}
		// TextToSpeechEndPacket is emitted by handleFlushComplete once
		// result.contextClosed or done:true arrives.

	default:
		return fmt.Errorf("inworld-tts: unsupported input type %T", in)
	}
	return nil
}

// Close tears down the connection and emits a final duration metric. Safe to
// call even if Initialize was never invoked.
func (it *inworldTTS) Close(ctx context.Context) error {
	it.ctxCancel()
	it.mu.Lock()
	ctxID := it.contextId
	connectedAt := it.ttsConnectedAt
	it.ttsConnectedAt = time.Time{}

	if it.connection != nil {
		conn := it.connection
		it.connection = nil // mark before Close so readLoop sees intentional
		_ = conn.Close()
	}
	it.mu.Unlock()

	if !connectedAt.IsZero() {
		_ = it.onPacket(
			internal_type.ConversationEventPacket{
				ContextID: ctxID,
				Name:      "tts",
				Data: map[string]string{
					"type":     "closed",
					"provider": it.Name(),
				},
				Time: time.Now(),
			},
			internal_type.ConversationMetricPacket{
				ContextID: 0,
				Metrics: []*protos.Metric{{
					Name:        type_enums.CONVERSATION_TTS_DURATION.String(),
					Value:       fmt.Sprintf("%d", time.Since(connectedAt).Nanoseconds()),
					Description: "Total TTS connection duration in nanoseconds",
				}},
			},
		)
	}
	return nil
}
