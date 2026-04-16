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
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	inworld_internal "github.com/rapidaai/api/assistant-api/internal/transformer/inworld/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
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

func TestNewInworldOption_NilCredential(t *testing.T) {
	opt, err := NewInworldOption(newTestLogger(), nil, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "nil vault credential")
}

func TestNewInworldOption_NilVaultValue(t *testing.T) {
	opt, err := NewInworldOption(newTestLogger(), &protos.VaultCredential{}, utils.Option{})
	assert.Error(t, err)
	assert.Nil(t, opt)
	assert.Contains(t, err.Error(), "nil vault value")
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
	assert.Equal(t, "https://api.inworld.ai/tts/v1/voice:stream", connStr)
}

// --- Name ---

func TestInworldTTSName(t *testing.T) {
	// Name is fixed — we test it on a zero-valued instance to avoid dialing.
	tts := &inworldTTS{}
	assert.Equal(t, "inworld-text-to-speech", tts.Name())
}

// --- Streaming-layer test harness ---

// packetCollector captures the packets Transform and the runner emit so
// tests can inspect them.
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

func (c *packetCollector) hasErrorEvent(ctx string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e.ContextID == ctx && e.Data["type"] == "error" {
			return true
		}
	}
	return false
}

// writeAudioLine serializes one NDJSON response line.
func writeAudioLine(t *testing.T, w http.ResponseWriter, audio []byte) {
	t.Helper()
	chunk := inworld_internal.StreamChunk{
		Result: &inworld_internal.StreamResult{
			AudioContent: base64.StdEncoding.EncodeToString(audio),
		},
	}
	enc := json.NewEncoder(w)
	require.NoError(t, enc.Encode(chunk))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// newTestInworldTTS builds an inworldTTS pointed at the given httptest
// URL. Tests set streamURL directly — no URL rewriting required because
// the production synth() reads the target URL off the struct rather
// than a hard-coded constant. A plain http.Transport with a small idle
// pool stands in for newInworldHTTPClient so the keep-alive reuse test
// still exercises real connection pooling.
func newTestInworldTTS(ctx context.Context, url string, collector *packetCollector) *inworldTTS {
	ctx2, cancel := context.WithCancel(ctx)
	return &inworldTTS{
		ctx:       ctx2,
		ctxCancel: cancel,
		onPacket:  collector.onPacket,
		logger:    newTestLogger(),
		client: &http.Client{Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 4,
			IdleConnTimeout:     90 * time.Second,
		}},
		streamURL: url,
		turns:     make(map[string]*turnRunner),
		inworldOption: &inworldOption{
			key:     "test-key",
			logger:  newTestLogger(),
			mdlOpts: utils.Option{},
		},
	}
}

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

// --- Streams audio in order ---

// TestInworldTTSStreamsAudioInOrder fires two Delta packets for the same
// turn and asserts audio arrives in submission order. The mock server
// returns distinct audio bytes per sentence so the ordering assertion is
// unambiguous.
func TestInworldTTSStreamsAudioInOrder(t *testing.T) {
	type reqBody struct {
		Text string `json:"text"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req reqBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		// Two chunks per sentence so we can see intra-sentence order too.
		writeAudioLine(t, w, []byte(req.Text+"#1"))
		writeAudioLine(t, w, []byte(req.Text+"#2"))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-order",
		Text:      "hello.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-order",
		Text:      "world.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-order",
	}))

	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-order")) > 0
	}), "end packet should be emitted after Done drains")

	audio := collector.audioForCtx("ctx-order")
	require.Len(t, audio, 4, "two sentences × two chunks each")
	assert.Equal(t, "hello.#1", string(audio[0].AudioChunk))
	assert.Equal(t, "hello.#2", string(audio[1].AudioChunk))
	assert.Equal(t, "world.#1", string(audio[2].AudioChunk))
	assert.Equal(t, "world.#2", string(audio[3].AudioChunk))
}

// --- End after Done ---

// TestInworldTTSEmitsEndAfterDone verifies that Done triggers a single
// TextToSpeechEndPacket after all queued sentences have finished streaming.
func TestInworldTTSEmitsEndAfterDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		writeAudioLine(t, w, []byte("a"))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-end",
		Text:      "s1.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-end",
		Text:      "s2.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-end",
	}))

	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-end")) == 1
	}), "should emit exactly one end packet")

	// `completed` event should follow the end packet.
	collector.mu.Lock()
	defer collector.mu.Unlock()
	sawCompleted := false
	for _, ev := range collector.events {
		if ev.ContextID == "ctx-end" && ev.Data["type"] == "completed" {
			sawCompleted = true
		}
	}
	assert.True(t, sawCompleted, "should emit tts=completed event")
}

// --- Interrupt cancels in-flight synth ---

// TestInworldTTSInterruptCancelsInFlight fires a Delta, waits for the
// mock server to start emitting audio, sends an Interrupt, and asserts
// the runner exits without continuing to emit audio from the
// still-open response body.
func TestInworldTTSInterruptCancelsInFlight(t *testing.T) {
	firstChunk := make(chan struct{})
	releaseSecond := make(chan struct{})
	secondSent := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		writeAudioLine(t, w, []byte("chunk-1"))
		select {
		case firstChunk <- struct{}{}:
		default:
		}
		select {
		case <-releaseSecond:
			writeAudioLine(t, w, []byte("chunk-2"))
			close(secondSent)
		case <-r.Context().Done():
			return
		}
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-int",
		Text:      "hello world.",
	}))

	select {
	case <-firstChunk:
	case <-time.After(2 * time.Second):
		t.Fatal("never got first chunk")
	}
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.audioForCtx("ctx-int")) >= 1
	}), "should see the first audio packet before interrupting")

	require.NoError(t, iw.Transform(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-int",
		Source:    internal_type.InterruptionSourceVad,
	}))

	// Runner must have exited and removed the turn from the map.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		iw.mu.Lock()
		defer iw.mu.Unlock()
		_, exists := iw.turns["ctx-int"]
		return !exists
	}), "turn state should be cleared after interrupt")

	// Let the server proceed — if the runner were still alive, a second
	// audio packet would arrive here. With the interrupt, the request's
	// context was canceled and the scanner read returned early.
	close(releaseSecond)
	select {
	case <-secondSent:
	case <-time.After(1 * time.Second):
		// Server may never have flushed the second line because the
		// client closed early; either outcome is fine.
	}

	// Audio count should be 1 — the chunk we saw before the interrupt.
	assert.Equal(t, 1, len(collector.audioForCtx("ctx-int")),
		"no audio should be emitted after interrupt")

	// Interrupt event should have fired.
	collector.mu.Lock()
	defer collector.mu.Unlock()
	interrupted := false
	for _, ev := range collector.events {
		if ev.ContextID == "ctx-int" && ev.Data["type"] == "interrupted" {
			interrupted = true
		}
	}
	assert.True(t, interrupted, "should emit tts=interrupted event")
}

// --- Idle runner responds to interrupt ---

// TestInworldTTSInterruptWakesIdleRunner reproduces the runner-leak
// regression: after a sentence finishes synthesizing, the runner
// goroutine is parked on `<-tr.sentences` waiting for the next Delta.
// Prior to the select-on-runCtx.Done() fix, calling runCancel() would
// silently mark the context done but leave the goroutine blocked on
// the channel receive forever. This test drives that idle state, fires
// an interrupt, and asserts the runner exits (turn removed from map)
// within a bounded window.
func TestInworldTTSInterruptWakesIdleRunner(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		writeAudioLine(t, w, []byte("ok"))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	// Delta + drain: the runner now loops back to receive from an empty
	// channel and parks — synth is NOT in flight.
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-idle",
		Text:      "hello.",
	}))
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.audioForCtx("ctx-idle")) >= 1
	}), "runner should finish its first synth and go idle")

	// Fire interrupt while the runner is parked on <-tr.sentences.
	require.NoError(t, iw.Transform(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-idle",
		Source:    internal_type.InterruptionSourceVad,
	}))

	// The runner MUST exit — without the select fix, this goroutine
	// would stay parked forever and this assertion would time out.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		iw.mu.Lock()
		defer iw.mu.Unlock()
		_, exists := iw.turns["ctx-idle"]
		return !exists
	}), "idle runner must be reaped after interrupt")
}

// --- Interrupt is scoped to its ContextID ---

// TestInworldTTSInterruptScopedToContext asserts that interrupting turn
// A does not tear down turn B. Prior to scoping the cancellation by
// input.ContextID, any interrupt would cancel every live runner and
// drop audio/end packets for bystander turns.
func TestInworldTTSInterruptScopedToContext(t *testing.T) {
	// Block turn B's synth on a signal so we can keep it alive while we
	// interrupt turn A. The server only replies with audio once the
	// test releases the block.
	releaseB := make(chan struct{})
	type reqBody struct {
		Text string `json:"text"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		// Decode the full JSON body so we can branch on which turn this
		// request belongs to — a single r.Body.Read is permitted to
		// short-read under the io.Reader contract.
		var req reqBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request body: %v", err)
			return
		}
		if req.Text == "B-text." {
			select {
			case <-releaseB:
			case <-r.Context().Done():
				return
			}
		}
		writeAudioLine(t, w, []byte("ok"))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	// Turn A — let it finish so its runner is idle.
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-A",
		Text:      "A-text.",
	}))
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.audioForCtx("ctx-A")) >= 1
	}), "turn A should receive its audio")

	// Turn B — pending (server holds the request until releaseB closes).
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-B",
		Text:      "B-text.",
	}))
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		iw.mu.Lock()
		defer iw.mu.Unlock()
		_, existsB := iw.turns["ctx-B"]
		return existsB
	}), "turn B runner should be registered before interrupt")

	// Interrupt targeting turn A ONLY.
	require.NoError(t, iw.Transform(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-A",
		Source:    internal_type.InterruptionSourceVad,
	}))

	// Turn A should be reaped; turn B must still be live.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		iw.mu.Lock()
		defer iw.mu.Unlock()
		_, aExists := iw.turns["ctx-A"]
		_, bExists := iw.turns["ctx-B"]
		return !aExists && bExists
	}), "interrupt on ctx-A must not cancel ctx-B")

	// Let turn B's synth finish, then Done → end packet for B.
	close(releaseB)
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-B",
	}))
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-B")) > 0
	}), "turn B must produce its terminal end packet — the interrupt on A should not have affected it")

	// Turn A: no end packet (interrupt path swallows it) but the
	// interrupted event must have fired.
	assert.Empty(t, collector.endsForCtx("ctx-A"),
		"interrupted turn must not emit a terminal end packet")
}

// --- Server error surfaces ---

// TestInworldTTSServerErrorSurfaces asserts that an NDJSON error line
// from the server terminates the synth attempt, surfaces an error event,
// and still emits a TextToSpeechEndPacket for the turn once Done arrives
// so callers waiting on end-of-turn don't block.
func TestInworldTTSServerErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		// Error line — no audio.
		chunk := inworld_internal.StreamChunk{
			Error: &inworld_internal.ErrorBody{Message: "voice not found"},
		}
		require.NoError(t, json.NewEncoder(w).Encode(chunk))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-err",
		Text:      "boom.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-err",
	}))

	// End packet should still fire — callers unblock.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-err")) > 0
	}), "server error must still produce an end packet")

	// Error event should carry the message.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return collector.hasErrorEvent("ctx-err")
	}), "server error should surface as tts=error event")

	collector.mu.Lock()
	defer collector.mu.Unlock()
	for _, ev := range collector.events {
		if ev.ContextID == "ctx-err" && ev.Data["type"] == "error" {
			assert.Contains(t, ev.Data["message"], "voice not found")
		}
	}
}

// --- Keep-alive reuses TCP conn ---

// TestInworldTTSKeepAliveReusesConn fires three sentences in succession
// and asserts the underlying transport accepts only one TCP connection.
// Regression: without keep-alive, each sentence dials fresh — and the
// whole point of rewriting away from WebSocket was to keep first-byte
// latency competitive via connection reuse.
func TestInworldTTSKeepAliveReusesConn(t *testing.T) {
	connCount := int64(0)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	counting := &countingListener{Listener: listener, count: &connCount}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)
			writeAudioLine(t, w, []byte("ok"))
		}),
	}
	go srv.Serve(counting)
	defer srv.Close()

	url := "http://" + listener.Addr().String()
	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), url, collector)
	defer iw.Close(context.Background())

	for i, text := range []string{"one.", "two.", "three."} {
		require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
			ContextID: "ctx-ka",
			Text:      text,
		}), "sentence %d", i)
	}
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-ka",
	}))

	require.True(t, waitFor(t, 3*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-ka")) > 0
	}), "all three sentences should stream through and finish")

	assert.Equal(t, int64(1), atomic.LoadInt64(&connCount),
		"keep-alive should reuse a single TCP conn across three sentences")
}

// countingListener wraps a net.Listener and atomically counts Accept calls.
type countingListener struct {
	net.Listener
	count *int64
}

func (l *countingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		atomic.AddInt64(l.count, 1)
	}
	return c, err
}

// --- HTTP non-200 status ---

// TestInworldTTSHTTPErrorStatus asserts that a non-200 HTTP response
// surfaces as a tts=error event, still emits a terminal end packet on
// Done (so callers unblock), and does not cause the runner to exit —
// subsequent sentences should still be attempted.
func TestInworldTTSHTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-401",
		Text:      "hi.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-401",
	}))

	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-401")) > 0
	}), "HTTP error must still produce a terminal end packet")

	collector.mu.Lock()
	defer collector.mu.Unlock()
	sawErr := false
	for _, ev := range collector.events {
		if ev.ContextID == "ctx-401" && ev.Data["type"] == "error" {
			sawErr = true
			assert.Contains(t, ev.Data["message"], "401",
				"error event should mention the status code")
		}
	}
	assert.True(t, sawErr, "HTTP error should surface as tts=error event")
}

// --- Malformed NDJSON chunk is silently skipped ---

// TestInworldTTSMalformedChunkSkipped asserts that a bad JSON line mid-
// stream is logged and dropped, but subsequent valid chunks still flow
// through. This locks in the silent-drop branch in decodeChunk — the
// alternative (failing the whole synth on one bad line) would be
// worse.
func TestInworldTTSMalformedChunkSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		writeAudioLine(t, w, []byte("good-1"))
		// Bad line: not valid JSON.
		_, _ = w.Write([]byte("{not json at all\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Another bad line: valid JSON but audioContent isn't base64.
		_, _ = w.Write([]byte(`{"result":{"audioContent":"not!!!base64!!!"}}` + "\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		writeAudioLine(t, w, []byte("good-2"))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-bad",
		Text:      "hi.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-bad",
	}))

	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-bad")) > 0
	}), "malformed chunks should not prevent the turn from completing")

	audio := collector.audioForCtx("ctx-bad")
	require.Len(t, audio, 2, "both valid chunks should be emitted, bad lines skipped")
	assert.Equal(t, "good-1", string(audio[0].AudioChunk))
	assert.Equal(t, "good-2", string(audio[1].AudioChunk))
}

// --- Unsupported packet type returns an error ---

// TestInworldTTSUnsupportedPacketErrors covers the `default:` branch of
// Transform — passing a packet type the transformer doesn't handle
// should return an error without panicking or leaking goroutines.
func TestInworldTTSUnsupportedPacketErrors(t *testing.T) {
	collector := &packetCollector{}
	// No httptest server needed — nothing should hit the network.
	iw := newTestInworldTTS(context.Background(), "http://unused.invalid", collector)
	defer iw.Close(context.Background())

	// LLMToolCallPacket is an LLMPacket but not one Transform handles.
	err := iw.Transform(context.Background(), internal_type.LLMToolCallPacket{
		ContextID: "ctx-unsupported",
		Name:      "some_tool",
	})
	require.Error(t, err, "unsupported packet type must return an error")
	assert.Contains(t, err.Error(), "unsupported input type")
}

// --- Done without any prior Delta is a no-op ---

// TestInworldTTSDoneWithoutDelta asserts that a Done packet for a ctx
// id with no live runner does nothing — no runner is spawned, no
// terminal end packet fires. Mirrors the "empty LLM response" path.
func TestInworldTTSDoneWithoutDelta(t *testing.T) {
	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), "http://unused.invalid", collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-empty",
	}))

	// Give any stray goroutine a moment to fire a bogus end packet.
	time.Sleep(50 * time.Millisecond)

	assert.Empty(t, collector.endsForCtx("ctx-empty"),
		"Done without a matching Delta must not produce an end packet")
	iw.mu.Lock()
	_, exists := iw.turns["ctx-empty"]
	iw.mu.Unlock()
	assert.False(t, exists, "Done without a matching Delta must not spawn a runner")
}

// --- Duplicate Done is harmless ---

// TestInworldTTSDuplicateDone asserts that firing Done twice for the
// same turn doesn't panic (prior to the sentinel-item refactor the
// second call would have hit close-of-closed-channel) and produces
// exactly one terminal end packet.
func TestInworldTTSDuplicateDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		writeAudioLine(t, w, []byte("ok"))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-dup",
		Text:      "hi.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-dup",
	}))
	// Wait for the runner to drain before firing the second Done, so
	// the second one hits the "no live runner" early-return branch.
	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-dup")) == 1
	}), "first Done should produce exactly one end packet")

	// Second Done — must not panic.
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-dup",
	}))

	// Still exactly one end packet (the second Done was a no-op).
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, len(collector.endsForCtx("ctx-dup")),
		"duplicate Done must not emit a second terminal end packet")
}

// --- WAV-wrapped chunks are stripped before emission ---

// TestInworldTTSStripsPerChunkWAVHeader exercises the whole decode
// path with a response that wraps each LINEAR16 payload in a RIFF/WAVE
// container — the shape Inworld actually sends. The audio packet
// handed to the consumer must contain ONLY the data subchunk bytes;
// concatenating two WAV-wrapped chunks verbatim would duplicate the
// header and produce clicks downstream.
func TestInworldTTSStripsPerChunkWAVHeader(t *testing.T) {
	pcm1 := []byte{0x01, 0x00, 0x02, 0x00, 0x03, 0x00}
	pcm2 := []byte{0x10, 0x00, 0x20, 0x00}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		// Each chunk is a complete mini-WAV, as Inworld returns them.
		writeAudioLine(t, w, buildMinimalWAV(t, pcm1))
		writeAudioLine(t, w, buildMinimalWAV(t, pcm2))
	}))
	defer srv.Close()

	collector := &packetCollector{}
	iw := newTestInworldTTS(context.Background(), srv.URL, collector)
	defer iw.Close(context.Background())

	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-wav",
		Text:      "hi.",
	}))
	require.NoError(t, iw.Transform(context.Background(), internal_type.LLMResponseDonePacket{
		ContextID: "ctx-wav",
	}))

	require.True(t, waitFor(t, 2*time.Second, func() bool {
		return len(collector.endsForCtx("ctx-wav")) > 0
	}), "WAV-stripped chunks should still complete the turn")

	audio := collector.audioForCtx("ctx-wav")
	require.Len(t, audio, 2, "both chunks should flow through")
	assert.Equal(t, pcm1, audio[0].AudioChunk,
		"first audio packet must contain only the data subchunk bytes")
	assert.Equal(t, pcm2, audio[1].AudioChunk,
		"second audio packet must contain only the data subchunk bytes")
}
