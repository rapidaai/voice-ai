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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	inworld_internal "github.com/rapidaai/api/assistant-api/internal/transformer/inworld/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	l, _ := commons.NewApplicationLogger()
	return l
}

func newVaultCredential(m map[string]interface{}) *protos.VaultCredential {
	val, _ := structpb.NewStruct(m)
	return &protos.VaultCredential{Value: val}
}

// --- Constructor Tests ---

func TestNewInworldOption_ValidCredentials(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "test-api-key"})
	opt, err := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, opt)
	assert.Equal(t, "test-api-key", opt.GetKey())
}

func TestNewInworldOption_MissingKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"other": "value"})
	opt, err := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "illegal vault config")
}

func TestNewInworldOption_EmptyVault(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{})
	opt, err := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
}

func TestNewInworldOption_EmptyKey(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": ""})
	opt, err := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "empty vault key")
}

// --- Encoding Tests ---

func TestInworldGetEncoding(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, "pcm_16000", opt.GetEncoding())
}

// --- Voice / Model Defaults ---

func TestInworldVoiceAndModelDefaults(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewInworldOption(newTestLogger(), cred, utils.Option{})
	assert.Equal(t, INWORLD_DEFAULT_VOICE_ID, opt.GetVoiceID())
	assert.Equal(t, INWORLD_DEFAULT_MODEL_ID, opt.GetModelID())
}

func TestInworldVoiceAndModelOverride(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opts := utils.Option{
		"speak.voice.id": "Hades",
		"speak.model":    "inworld-tts-1.5-mini",
	}
	opt, _ := NewInworldOption(newTestLogger(), cred, opts)
	assert.Equal(t, "Hades", opt.GetVoiceID())
	assert.Equal(t, "inworld-tts-1.5-mini", opt.GetModelID())
}

// --- GetTextToSpeechConnectionString ---

func TestInworldGetTextToSpeechConnectionString(t *testing.T) {
	cred := newVaultCredential(map[string]interface{}{"key": "k"})
	opt, _ := NewInworldOption(newTestLogger(), cred, utils.Option{})
	connStr := opt.GetTextToSpeechConnectionString()
	// Auth and configuration live in headers and frames — URL is static.
	assert.Equal(t, "wss://api.inworld.ai/tts/v1/voice:streamBidirectional", connStr)
}

// --- Name ---

func TestInworldTTSName(t *testing.T) {
	// Name is fixed — we test it on a zero-valued instance to avoid dialing.
	tts := &inworldTTS{}
	assert.Equal(t, "inworld-text-to-speech", tts.Name())
}

// --- Shared test harness for the streaming-layer tests ---

// packetCollector is a tiny sink that captures the packets Transform and
// readLoop emit so tests can inspect them.
type packetCollector struct {
	mu      sync.Mutex
	audio   []internal_type.TextToSpeechAudioPacket
	ends    []internal_type.TextToSpeechEndPacket
	events  []internal_type.ConversationEventPacket
	metrics []internal_type.AssistantMessageMetricPacket
}

func (c *packetCollector) onPacket(pkts ...internal_type.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range pkts {
		switch v := p.(type) {
		case internal_type.TextToSpeechAudioPacket:
			c.audio = append(c.audio, v)
		case internal_type.TextToSpeechEndPacket:
			c.ends = append(c.ends, v)
		case internal_type.ConversationEventPacket:
			c.events = append(c.events, v)
		case internal_type.AssistantMessageMetricPacket:
			c.metrics = append(c.metrics, v)
		}
	}
	return nil
}

func (c *packetCollector) audioForCtx(ctx string) []internal_type.TextToSpeechAudioPacket {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []internal_type.TextToSpeechAudioPacket
	for _, a := range c.audio {
		if a.ContextID == ctx {
			out = append(out, a)
		}
	}
	return out
}

func (c *packetCollector) endsForCtx(ctx string) []internal_type.TextToSpeechEndPacket {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []internal_type.TextToSpeechEndPacket
	for _, e := range c.ends {
		if e.ContextID == ctx {
			out = append(out, e)
		}
	}
	return out
}

// newMockInworldServer spins up an in-process WebSocket endpoint that hands
// each new connection off to the supplied handler. The handler receives the
// server-side *websocket.Conn and is responsible for reading client frames
// and writing mock responses.
func newMockInworldServer(t *testing.T, handler func(*websocket.Conn)) (*httptest.Server, string) {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer c.Close()
		handler(c)
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}

// dialMock dials the mock server and returns the client-side *websocket.Conn
// — i.e. the end the transformer would hold after a successful Initialize.
func dialMock(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	return c
}

// newTestInworldTTS builds an inworldTTS by hand, bypassing Initialize so
// tests can inject the connection directly via pendingConn.
func newTestInworldTTS(ctx context.Context, collector *packetCollector) *inworldTTS {
	ctx2, cancel := context.WithCancel(ctx)
	return &inworldTTS{
		ctx:       ctx2,
		ctxCancel: cancel,
		onPacket:  collector.onPacket,
		logger:    newTestLogger(),
		turns:     make(map[string]*turnState),
		inworldOption: &inworldOption{
			key:     "test-key",
			logger:  newTestLogger(),
			mdlOpts: utils.Option{},
		},
	}
}

// writeAudioFrame serializes a minimal audio-chunk response for the mock
// server to push to the client.
func writeAudioFrame(t *testing.T, c *websocket.Conn, ctxID string, audio []byte) {
	t.Helper()
	resp := inworld_internal.InworldTTSResponse{
		Result: &inworld_internal.Result{
			ContextID: ctxID,
			AudioChunk: &inworld_internal.AudioChunk{
				AudioContent: base64.StdEncoding.EncodeToString(audio),
			},
		},
	}
	require.NoError(t, c.WriteJSON(resp))
}

// waitFor polls fn until it returns true or the timeout elapses. Replaces
// arbitrary sleeps that would make tests flaky on slow CI runners.
func waitFor(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fn()
}

// --- Fix 2 regression: turn state must not bleed across turns ---

// TestInworldTTSTurnsDoNotBleed drives two interleaved turns through
// Transform with distinct ContextIDs and asserts each turn's audio is
// always emitted under its own ContextID — even when a late frame for
// turn A arrives after turn B has already started. The previous
// (shared-state) impl would have misattributed that late frame to ctx-B
// because readLoop read the mutable `it.contextId` at emission time;
// with per-turn state and a per-loop captured ctx id this is no longer
// possible.
func TestInworldTTSTurnsDoNotBleed(t *testing.T) {
	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), collector)
	defer iw.Close(context.Background())

	// Each turn's mock handler reads the create frame, echoes the ctx id
	// back on a channel, then emits audio chunks on demand until the test
	// releases it. This lets us choreograph the bleed scenario: turn A
	// stays alive while turn B opens, then fires a late frame *after*
	// turn B has already started streaming.
	type serverTurn struct {
		ctxReady chan string
		pulse    chan struct{} // each send triggers one audio frame
		stopped  chan struct{}
	}
	newServerTurn := func() *serverTurn {
		return &serverTurn{
			ctxReady: make(chan string, 1),
			pulse:    make(chan struct{}, 4),
			stopped:  make(chan struct{}),
		}
	}
	turn1 := newServerTurn()
	turn2 := newServerTurn()
	turns := []*serverTurn{turn1, turn2}
	var turnIdx int
	var turnMu sync.Mutex

	_, url := newMockInworldServer(t, func(c *websocket.Conn) {
		turnMu.Lock()
		idx := turnIdx
		turnIdx++
		turnMu.Unlock()
		if idx >= len(turns) {
			return
		}
		tdata := turns[idx]
		defer close(tdata.stopped)
		_, raw, err := c.ReadMessage()
		if err != nil {
			return
		}
		var create inworld_internal.CreateRequest
		if err := json.Unmarshal(raw, &create); err != nil {
			return
		}
		tdata.ctxReady <- create.ContextID
		// Drain subsequent client frames (send_text, close_context) in the
		// background so the server-side ReadMessage doesn't block audio
		// emission.
		go func() {
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					return
				}
			}
		}()
		for range tdata.pulse {
			writeAudioFrame(t, c, create.ContextID, []byte("chunk-"+create.ContextID))
		}
	})

	// --- Open turn A and drive one audio frame through ---
	iw.pendingConn = dialMock(t, url)
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-A",
		Text:      "hello A",
	}))
	require.Equal(t, "ctx-A", <-turn1.ctxReady)
	turn1.pulse <- struct{}{} // server sends audio frame #1 for ctx-A
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.audioForCtx("ctx-A")) >= 1
	}), "turn A should receive first audio frame tagged ctx-A")

	// --- Open turn B on a fresh conn while turn A is still alive ---
	iw.pendingConn = dialMock(t, url)
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-B",
		Text:      "hello B",
	}))
	require.Equal(t, "ctx-B", <-turn2.ctxReady)
	turn2.pulse <- struct{}{} // server sends audio frame for ctx-B
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.audioForCtx("ctx-B")) >= 1
	}), "turn B should receive audio tagged ctx-B while turn A is still open")

	// --- Late frame on turn A's still-open conn (the regression case) ---
	// Under the old shared-state impl, readLoop would have read the
	// transformer's mutable `contextId` at emission time — which by now
	// points at ctx-B — and misattributed this audio chunk. Per-turn
	// state makes that impossible.
	turn1.pulse <- struct{}{}
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.audioForCtx("ctx-A")) >= 2
	}), "late frame on turn A's conn must be tagged ctx-A, not ctx-B")

	close(turn1.pulse)
	close(turn2.pulse)

	// Every audio packet ever emitted must match the ctx id of the conn
	// that produced it — double-check the invariant across the full set.
	collector.mu.Lock()
	defer collector.mu.Unlock()
	assert.NotEmpty(t, collector.audio)
	for _, a := range collector.audio {
		assert.Contains(t, []string{"ctx-A", "ctx-B"}, a.ContextID)
	}
}

// --- Fix 5 wiring: create frame carries auto_mode and send_text has no flush_context ---

// TestInworldTTSCreateFrameEnablesAutoMode verifies the wire-level shape of
// the frames Transform sends on the first delta: the create frame sets
// auto_mode:true and the send_text frame omits flush_context entirely.
func TestInworldTTSCreateFrameEnablesAutoMode(t *testing.T) {
	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), collector)
	defer iw.Close(context.Background())

	createRaw := make(chan []byte, 1)
	sendRaw := make(chan []byte, 1)

	_, url := newMockInworldServer(t, func(c *websocket.Conn) {
		_, raw, err := c.ReadMessage()
		if err != nil {
			return
		}
		createRaw <- raw
		_, raw, err = c.ReadMessage()
		if err != nil {
			return
		}
		sendRaw <- raw
		// Keep the connection open so the client-side doesn't see a
		// premature close while the test inspects the captured frames.
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})

	iw.pendingConn = dialMock(t, url)
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-auto",
		Text:      "hi",
	}))

	// Inspect create frame — auto_mode must be present and true.
	select {
	case raw := <-createRaw:
		var envelope struct {
			ContextID string `json:"context_id"`
			Create    struct {
				VoiceID     string `json:"voice_id"`
				ModelID     string `json:"model_id"`
				AudioConfig struct {
					AudioEncoding   string `json:"audio_encoding"`
					SampleRateHertz int    `json:"sample_rate_hertz"`
				} `json:"audio_config"`
				AutoMode bool `json:"auto_mode"`
			} `json:"create"`
		}
		require.NoError(t, json.Unmarshal(raw, &envelope))
		assert.Equal(t, "ctx-auto", envelope.ContextID)
		assert.True(t, envelope.Create.AutoMode, "create frame should enable auto_mode")
	case <-time.After(2 * time.Second):
		t.Fatal("create frame never arrived at mock server")
	}

	// Inspect send_text frame — flush_context must NOT be present. We match
	// on the raw JSON because an empty map vs a missing key is the whole
	// distinction this test protects against.
	select {
	case raw := <-sendRaw:
		assert.NotContains(t, string(raw), "flush_context",
			"send_text must not carry flush_context under auto_mode")
		// Sanity: text field still flows through.
		assert.Contains(t, string(raw), `"text":"hi"`)
	case <-time.After(2 * time.Second):
		t.Fatal("send_text frame never arrived at mock server")
	}
}

// --- Fix 3: server error frames must terminate the turn ---

// TestInworldTTSServerErrorTerminatesTurn asserts that when Inworld pushes
// an error response frame, readLoop emits a TextToSpeechEndPacket (so any
// caller waiting on end-of-turn can unblock), tears down the turn state,
// and closes the underlying connection.
func TestInworldTTSServerErrorTerminatesTurn(t *testing.T) {
	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), collector)
	defer iw.Close(context.Background())

	connReady := make(chan struct{})
	_, url := newMockInworldServer(t, func(c *websocket.Conn) {
		// Server-side: just push an error frame immediately.
		close(connReady)
		errFrame := inworld_internal.InworldTTSResponse{
			Error: &inworld_internal.ErrorBody{
				Message: "boom",
				Code:    42,
			},
		}
		_ = c.WriteJSON(errFrame)
		// Hold the conn open until the client closes it.
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})

	clientConn := dialMock(t, url)
	<-connReady

	// Seed a turn that owns this conn as if Transform had already bound it.
	iw.mu.Lock()
	iw.turns["ctx-err"] = &turnState{conn: clientConn, contextCreated: true, ttsStartedAt: time.Now()}
	iw.mu.Unlock()
	go iw.readLoop(clientConn, "ctx-err")

	// The end packet is what unblocks callers — must arrive even though the
	// server never sent done/contextClosed.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-err")) > 0
	}), "server error should emit TextToSpeechEndPacket for the turn")

	// Turn state must be cleared so a subsequent turn with the same ctx-id
	// would dial a fresh conn.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		iw.mu.Lock()
		defer iw.mu.Unlock()
		_, exists := iw.turns["ctx-err"]
		return !exists
	}), "server error should delete turn state")
}
