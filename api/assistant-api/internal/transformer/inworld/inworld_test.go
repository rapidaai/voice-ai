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
// URL. The production code posts to INWORLD_STREAM_URL; tests route
// requests to httptest by installing an http.Transport whose RoundTrip
// rewrites the request's scheme+host to the mock server's. This leaves
// synth() and the production URL constant untouched.
func newTestInworldTTS(ctx context.Context, url string, collector *packetCollector) *inworldTTS {
	ctx2, cancel := context.WithCancel(ctx)
	return &inworldTTS{
		ctx:       ctx2,
		ctxCancel: cancel,
		onPacket:  collector.onPacket,
		logger:    newTestLogger(),
		client:    buildRoutedClient(hostOf(url)),
		turns:     make(map[string]*turnRunner),
		inworldOption: &inworldOption{
			key:     "test-key",
			logger:  newTestLogger(),
			mdlOpts: utils.Option{},
		},
	}
}

// rewritingRoundTripper sends every outbound request to `host` regardless
// of what URL the caller asked for. With a tuned idle-conn pool it is
// byte-for-byte equivalent to production's HTTP transport, so the
// keep-alive reuse test is meaningful.
type rewritingRoundTripper struct {
	host string // "127.0.0.1:NNNN"
	base http.RoundTripper
}

func (r *rewritingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	u := *req.URL
	u.Scheme = "http"
	u.Host = r.host
	out.URL = &u
	out.Host = u.Host
	return r.base.RoundTrip(out)
}

func buildRoutedClient(target string) *http.Client {
	return &http.Client{Transport: &rewritingRoundTripper{
		host: target,
		base: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 4,
			IdleConnTimeout:     90 * time.Second,
		},
	}}
}

// hostOf strips "http://" (or "https://") from an httptest URL, returning
// just the host:port portion so rewritingRoundTripper can rewrite into it.
func hostOf(url string) string {
	for _, prefix := range []string{"http://", "https://"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			return url[len(prefix):]
		}
	}
	return url
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
