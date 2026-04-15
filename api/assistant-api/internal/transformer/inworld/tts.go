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

// turnState holds all per-turn Inworld state. Each Inworld context is
// bound to its own WebSocket connection so that late frames from a prior
// turn cannot bleed into the current one — every readLoop goroutine only
// observes/mutates the turnState for the context it was started with.
type turnState struct {
	conn           *websocket.Conn
	contextCreated bool
	ttsStartedAt   time.Time
	ttsMetricSent  bool
}

// inworldTTS implements internal_type.TextToSpeechTransformer against
// Inworld's bidirectional streaming TTS WebSocket. A dedicated WebSocket
// connection is opened per TTS turn; the turn ends when Inworld emits
// contextClosed (or done:true) in response to our close_context frame, or
// when a server error terminates the turn.
type inworldTTS struct {
	*inworldOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu sync.Mutex
	// turns is keyed by the Inworld context_id (same as the Rapida turn
	// ContextId we pass through). One entry per live turn.
	turns map[string]*turnState
	// pendingConn is a connection dialed by Initialize that has not yet
	// been bound to a turn. Adopted by the next LLMResponseDeltaPacket.
	pendingConn *websocket.Conn
	// activeContext is the most-recent Rapida turn id seen by Transform.
	// Tracked for diagnostics only — per-turn writes are always routed via
	// the turnState looked up by the packet's own ContextId.
	activeContext string
	// ttsConnectedAt is the first-ever connect time; drives the duration
	// metric emitted at Close.
	ttsConnectedAt time.Time

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
		turns:         make(map[string]*turnState),
	}, nil
}

// Name identifies this transformer in logs and events.
func (*inworldTTS) Name() string {
	return "inworld-text-to-speech"
}

// Initialize opens a fresh WebSocket connection to Inworld, storing it as a
// pending connection that will be adopted by the next turn on its first
// delta. Called at session start and after each interruption so a warm
// socket is available before the first text delta arrives. The read loop
// is not started here — it is bound to a turn in Transform so every loop
// has an authoritative contextID in scope.
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
	// If Initialize is called again before the pendingConn was adopted (e.g.
	// repeated interrupt), the previous pendingConn must be closed so it
	// does not leak.
	if it.pendingConn != nil {
		_ = it.pendingConn.Close()
	}
	it.pendingConn = conn
	if it.ttsConnectedAt.IsZero() {
		it.ttsConnectedAt = time.Now()
	}
	it.mu.Unlock()

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
// turn. The boundCtxID captures the Inworld context_id this conn was
// create'd against — all packet emission uses it (or the server-echoed
// payload.Result.ContextID when present) rather than any shared state, so
// late frames cannot bleed into a different turn.
func (it *inworldTTS) readLoop(conn *websocket.Conn, boundCtxID string) {
	for {
		select {
		case <-it.ctx.Done():
			return
		default:
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			it.mu.Lock()
			ts, ok := it.turns[boundCtxID]
			intentional := !ok || ts.conn != conn
			if !intentional {
				delete(it.turns, boundCtxID) // unintentional drop: clear turn state
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
			it.handleTurnError(conn, boundCtxID, payload.Error.Message)
			return
		}

		// Terminal: either a plain done:true frame or result.contextClosed.
		if (payload.Done != nil && *payload.Done) ||
			(payload.Result != nil && payload.Result.ContextClosed != nil) {
			it.handleFlushComplete(conn, boundCtxID)
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

		// Prefer the server-echoed context id when present; fall back to the
		// id we bound to the conn at turn start. This is defense-in-depth:
		// the conn is already keyed by boundCtxID so they should match.
		emitCtxID := boundCtxID
		if payload.Result.ContextID != "" {
			emitCtxID = payload.Result.ContextID
		}

		it.mu.Lock()
		ts, ok := it.turns[boundCtxID]
		if !ok || ts.conn != conn {
			// This conn has already been detached from its turn (e.g.
			// interruption, error, flushComplete). Drop stale audio.
			it.mu.Unlock()
			continue
		}
		startedAt := ts.ttsStartedAt
		metricSent := ts.ttsMetricSent
		if !metricSent && !startedAt.IsZero() {
			ts.ttsMetricSent = true
		}
		it.mu.Unlock()

		if !metricSent && !startedAt.IsZero() {
			_ = it.onPacket(internal_type.AssistantMessageMetricPacket{
				ContextID: emitCtxID,
				Metrics: []*protos.Metric{{
					Name:  "tts_latency_ms",
					Value: fmt.Sprintf("%d", time.Since(startedAt).Milliseconds()),
				}},
			})
		}
		_ = it.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: emitCtxID, AudioChunk: rawAudio})
	}
}

// handleFlushComplete is called when Inworld signals end-of-context. It
// clears the turn's state (if still held), emits TextToSpeechEndPacket —
// correctly ordered after the last audio chunk — and closes this turn's
// connection. Other turns are untouched.
func (it *inworldTTS) handleFlushComplete(conn *websocket.Conn, boundCtxID string) {
	it.mu.Lock()
	if ts, ok := it.turns[boundCtxID]; ok && ts.conn == conn {
		delete(it.turns, boundCtxID)
	}
	it.mu.Unlock()

	_ = it.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: boundCtxID},
		internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{"type": "completed"},
			Time: time.Now(),
		},
	)
	_ = conn.Close()
}

// handleTurnError is the terminal handler for server-emitted error frames.
// It tears down the turn just like a successful flush (so callers waiting
// on TextToSpeechEndPacket stop waiting) and additionally surfaces the
// error as a ConversationEventPacket for observability.
func (it *inworldTTS) handleTurnError(conn *websocket.Conn, boundCtxID, message string) {
	it.mu.Lock()
	if ts, ok := it.turns[boundCtxID]; ok && ts.conn == conn {
		delete(it.turns, boundCtxID)
	}
	it.mu.Unlock()

	_ = it.onPacket(
		internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{"type": "error", "message": message},
			Time: time.Now(),
		},
		internal_type.TextToSpeechEndPacket{ContextID: boundCtxID},
	)
	_ = conn.Close()
}

// acquireConn returns a connection ready to host a new turn. It prefers the
// pending conn stashed by Initialize; if there isn't one (Initialize
// hasn't run yet, or the pending conn was consumed by a prior turn), it
// dials a fresh one.
func (it *inworldTTS) acquireConn() (*websocket.Conn, error) {
	it.mu.Lock()
	if it.pendingConn != nil {
		conn := it.pendingConn
		it.pendingConn = nil
		it.mu.Unlock()
		return conn, nil
	}
	it.mu.Unlock()

	if err := it.Initialize(); err != nil {
		return nil, err
	}
	it.mu.Lock()
	conn := it.pendingConn
	it.pendingConn = nil
	it.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("inworld-tts: pending connection missing after Initialize")
	}
	return conn, nil
}

// Transform routes a single LLM/control packet into the Inworld protocol.
// Supported packet types:
//   - InterruptionDetectedPacket: tear down every open turn and reconnect.
//   - LLMResponseDeltaPacket: on the first delta of a turn, adopt a fresh
//     connection and send the create frame; subsequent deltas just send_text.
//   - LLMResponseDonePacket: send close_context to drain & end the turn.
func (it *inworldTTS) Transform(ctx context.Context, in internal_type.LLMPacket) error {
	switch input := in.(type) {
	case internal_type.InterruptionDetectedPacket:
		it.mu.Lock()
		conns := make([]*websocket.Conn, 0, len(it.turns)+1)
		for id, ts := range it.turns {
			conns = append(conns, ts.conn)
			delete(it.turns, id)
		}
		if it.pendingConn != nil {
			conns = append(conns, it.pendingConn)
			it.pendingConn = nil
		}
		it.activeContext = ""
		it.mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
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
		ctxID := in.ContextId()

		it.mu.Lock()
		ts, exists := it.turns[ctxID]
		it.mu.Unlock()

		if !exists {
			conn, err := it.acquireConn()
			if err != nil {
				return fmt.Errorf("inworld-tts: failed to connect: %w", err)
			}
			it.mu.Lock()
			// Re-check: another Transform call with the same ctxID may have
			// raced ahead while we were dialing.
			if existing, ok := it.turns[ctxID]; ok {
				it.mu.Unlock()
				_ = conn.Close()
				ts = existing
			} else {
				ts = &turnState{conn: conn, ttsStartedAt: time.Now()}
				it.turns[ctxID] = ts
				it.activeContext = ctxID
				it.mu.Unlock()
				go it.readLoop(conn, ctxID)
			}
		} else {
			it.mu.Lock()
			if ts.ttsStartedAt.IsZero() {
				ts.ttsStartedAt = time.Now()
			}
			it.activeContext = ctxID
			it.mu.Unlock()
		}

		it.mu.Lock()
		needCreate := !ts.contextCreated
		conn := ts.conn
		it.mu.Unlock()

		// Inworld requires a "create" frame to open each context before any
		// send_text. auto_mode lets the server control flushing for low
		// latency while maintaining quality, so we do not also set
		// flush_context on every send_text.
		if needCreate {
			createFrame := inworld_internal.CreateRequest{
				ContextID: ctxID,
				Create: inworld_internal.CreateBody{
					VoiceID: it.GetVoiceID(),
					ModelID: it.GetModelID(),
					AudioConfig: inworld_internal.AudioConfig{
						AudioEncoding:   INWORLD_AUDIO_ENCODING,
						SampleRateHertz: INWORLD_SAMPLE_RATE,
					},
					AutoMode: true,
				},
			}
			if err := conn.WriteJSON(createFrame); err != nil {
				it.logger.Errorf("inworld-tts: create frame write failed: %v", err)
				return err
			}
			it.mu.Lock()
			ts.contextCreated = true
			it.mu.Unlock()
		}

		// Under auto_mode the server decides when to flush buffered text, so
		// we omit flush_context entirely on each send_text.
		sendFrame := inworld_internal.SendTextRequest{
			ContextID: ctxID,
			SendText: inworld_internal.SendTextBody{
				Text: input.Text,
			},
		}
		if err := conn.WriteJSON(sendFrame); err != nil {
			it.logger.Errorf("inworld-tts: send_text write failed: %v", err)
			return err
		}
		_ = it.onPacket(internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{"type": "speaking", "text": input.Text},
			Time: time.Now(),
		})

	case internal_type.LLMResponseDonePacket:
		ctxID := in.ContextId()
		it.mu.Lock()
		ts, ok := it.turns[ctxID]
		if !ok || !ts.contextCreated {
			it.mu.Unlock()
			// Either interrupted before done arrived, or we never opened a
			// context (empty response) — nothing to close.
			return nil
		}
		conn := ts.conn
		it.mu.Unlock()

		closeFrame := inworld_internal.CloseContextRequest{
			ContextID:    ctxID,
			CloseContext: map[string]interface{}{},
		}
		if err := conn.WriteJSON(closeFrame); err != nil {
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

// Close tears down every open connection and emits a final duration metric.
// Safe to call even if Initialize was never invoked.
func (it *inworldTTS) Close(ctx context.Context) error {
	it.ctxCancel()
	it.mu.Lock()
	activeCtxID := it.activeContext
	connectedAt := it.ttsConnectedAt
	it.ttsConnectedAt = time.Time{}

	conns := make([]*websocket.Conn, 0, len(it.turns)+1)
	for id, ts := range it.turns {
		conns = append(conns, ts.conn)
		delete(it.turns, id)
	}
	if it.pendingConn != nil {
		conns = append(conns, it.pendingConn)
		it.pendingConn = nil
	}
	it.mu.Unlock()

	for _, c := range conns {
		_ = c.Close()
	}

	if !connectedAt.IsZero() {
		_ = it.onPacket(
			internal_type.ConversationEventPacket{
				ContextID: activeCtxID,
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
