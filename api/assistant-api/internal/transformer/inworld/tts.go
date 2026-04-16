// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_inworld

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	inworld_internal "github.com/rapidaai/api/assistant-api/internal/transformer/inworld/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// sentenceItem carries one unit of work on the turn's sentence channel:
// either a delta's text (done=false), or a terminal Done marker
// (done=true, text empty). Using an in-band sentinel instead of
// close(sentences) lets Transform push Done without racing concurrent
// delta sends — no goroutine ever closes the channel, so "send on
// closed channel" and "close of closed channel" panics are structurally
// impossible.
type sentenceItem struct {
	text string
	done bool
}

// turnRunner owns all state for one Rapida turn. Exactly one goroutine
// drains `sentences`, so audio for a given turn is always emitted in the
// order the aggregator delivered the sentences — and across turns we
// simply have one runner per context id. runCtx cancels in-flight synth
// requests (interrupts) while ctxID is the turn's Rapida context id that
// every emitted packet carries.
type turnRunner struct {
	ctxID     string
	sentences chan sentenceItem

	// runCtx is a child of inworldTTS.ctx. Canceling it aborts the
	// in-flight synth() and causes the runner goroutine to exit its loop.
	runCtx    context.Context
	runCancel context.CancelFunc

	// startedAt timestamps the first Delta for the turn so synth() can emit
	// a one-shot tts_latency_ms metric with the true TTFB. Only touched by
	// the runner goroutine after construction.
	startedAt  time.Time
	metricSent bool
}

// inworldTTS implements internal_type.TextToSpeechTransformer against
// Inworld's HTTP streaming endpoint (api.inworld.ai/tts/v1/voice:stream).
// There is one HTTP POST per sentence — Rapida's aggregator already splits
// LLM output at sentence boundaries before it reaches Transform, so the
// bidirectional WebSocket's character-streaming capability was never being
// exploited. HTTP streaming buys us the same latency (with keep-alive on a
// shared *http.Transport) while eliminating the concurrency machinery a
// per-turn WebSocket required.
type inworldTTS struct {
	*inworldOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	client *http.Client
	// streamURL is the target for every synth request. Initialized to
	// INWORLD_STREAM_URL by the production constructor and overridable
	// by in-package tests (the test file builds inworldTTS directly and
	// points this at an httptest.Server URL — no URL-rewriting transport
	// hack needed).
	streamURL string

	mu            sync.Mutex
	turns         map[string]*turnRunner
	activeContext string // most recent Transform ctx id — diagnostics only

	// ttsConnectedAt is set the first time we route a sentence through the
	// shared client. It drives the duration metric emitted at Close.
	ttsConnectedAt time.Time

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

// NewInworldTextToSpeech constructs the Inworld TTS transformer.
func NewInworldTextToSpeech(ctx context.Context, logger commons.Logger, credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option) (internal_type.TextToSpeechTransformer, error) {
	iwOpts, err := NewInworldOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("inworld-tts: initializing inworld failed %+v", err)
		return nil, err
	}
	ctx2, cancel := context.WithCancel(ctx)
	return &inworldTTS{
		ctx:           ctx2,
		ctxCancel:     cancel,
		onPacket:      onPacket,
		logger:        logger,
		inworldOption: iwOpts,
		client:        newInworldHTTPClient(),
		streamURL:     INWORLD_STREAM_URL,
		turns:         make(map[string]*turnRunner),
	}, nil
}

// Name identifies this transformer in logs and events.
func (*inworldTTS) Name() string {
	return "inworld-text-to-speech"
}

// Initialize emits the initialized event and, if this is the first call,
// stamps the connection timestamp used by the Close duration metric. No
// network work happens here — the HTTP client lazily dials on the first
// synth request. We keep Initialize around so the transformer interface
// behaves identically to the WebSocket version (and so the integration
// test that asserts an "initialized" event still fires).
func (it *inworldTTS) Initialize() error {
	start := time.Now()
	it.mu.Lock()
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

// Transform routes one LLM/control packet into the Inworld protocol.
//   - LLMResponseDeltaPacket: enqueue a sentence onto the per-turn runner,
//     creating the runner (and starting its goroutine) on first delta.
//   - LLMResponseDonePacket: close the turn's sentence channel; the runner
//     drains its queue, emits TextToSpeechEndPacket, then exits.
//   - InterruptionDetectedPacket: cancel every live runner, drop their
//     state, emit a tts=interrupted event.
func (it *inworldTTS) Transform(ctx context.Context, in internal_type.LLMPacket) error {
	switch input := in.(type) {
	case internal_type.InterruptionDetectedPacket:
		// Scope the cancellation to the turn identified by the packet.
		// Rapida's pipeline only has one active TTS turn at a time in
		// practice, but if two contexts are ever synthesizing in parallel
		// we must not drop audio/end packets on the bystander turn.
		//
		// If the inbound ContextID is unset (defensive — the dispatcher
		// should always set it), fall through to a whole-session
		// teardown so an interrupt is never silently ignored.
		it.mu.Lock()
		var cancels []context.CancelFunc
		if input.ContextID != "" {
			if tr, ok := it.turns[input.ContextID]; ok {
				cancels = append(cancels, tr.runCancel)
				delete(it.turns, input.ContextID)
			}
			if it.activeContext == input.ContextID {
				it.activeContext = ""
			}
		} else {
			cancels = make([]context.CancelFunc, 0, len(it.turns))
			for id, tr := range it.turns {
				cancels = append(cancels, tr.runCancel)
				delete(it.turns, id)
			}
			it.activeContext = ""
		}
		it.mu.Unlock()
		for _, c := range cancels {
			c()
		}
		_ = it.onPacket(internal_type.ConversationEventPacket{
			ContextID: input.ContextID,
			Name:      "tts",
			Data:      map[string]string{"type": "interrupted"},
			Time:      time.Now(),
		})
		// Emit a fresh initialized event so the debugger UI sees a clean
		// boundary between the interrupted turn and whatever follows. The
		// HTTP client stays live — there is nothing to re-dial.
		_ = it.onPacket(internal_type.ConversationEventPacket{
			Name: "tts",
			Data: map[string]string{
				"type":     "initialized",
				"provider": it.Name(),
				"init_ms":  "0",
			},
			Time: time.Now(),
		})
		return nil

	case internal_type.LLMResponseDeltaPacket:
		tr := it.getOrCreateRunner(input.ContextID)
		select {
		case tr.sentences <- sentenceItem{text: input.Text}:
			// Only emit the speaking event on a successful enqueue so we
			// never publish "speaking: X" for text the runner will never
			// actually synthesize (the runCtx.Done case below drops the
			// delta silently).
			_ = it.onPacket(internal_type.ConversationEventPacket{
				ContextID: input.ContextID,
				Name:      "tts",
				Data:      map[string]string{"type": "speaking", "text": input.Text},
				Time:      time.Now(),
			})
		case <-tr.runCtx.Done():
			// Turn was just canceled (interrupt) — drop the delta. No
			// speaking event: the text will never reach TTS.
		}
		return nil

	case internal_type.LLMResponseDonePacket:
		it.mu.Lock()
		tr, ok := it.turns[input.ContextID]
		it.mu.Unlock()
		if !ok {
			// Interrupted before Done arrived, or Done fired for a turn
			// that never produced a delta (empty response) — nothing to do.
			return nil
		}
		// Send an in-band done marker rather than close(tr.sentences).
		// Closing the channel would race catastrophically with concurrent
		// delta sends (send on closed channel) and with a duplicate Done
		// (close of closed channel); the sentinel item side-steps both.
		// If the runner has already exited (interrupt beat Done), the
		// runCtx.Done branch fires and we drop the signal harmlessly.
		select {
		case tr.sentences <- sentenceItem{done: true}:
		case <-tr.runCtx.Done():
		}
		return nil

	default:
		return fmt.Errorf("inworld-tts: unsupported input type %T", in)
	}
}

// getOrCreateRunner returns the turnRunner for ctxID, starting one (and
// its goroutine) on first call. Called from Transform on every Delta; the
// double-checked lookup keeps the fast path lock-free after the turn is
// installed.
func (it *inworldTTS) getOrCreateRunner(ctxID string) *turnRunner {
	it.mu.Lock()
	if tr, ok := it.turns[ctxID]; ok {
		it.activeContext = ctxID
		it.mu.Unlock()
		return tr
	}
	// cancel is stashed in tr.runCancel and released by run()'s defer on
	// every exit path (Done, interrupt, or parent-ctx cancel). gosec's
	// intra-function analysis can't see that hop so silence G118 here.
	runCtx, cancel := context.WithCancel(it.ctx) //nolint:gosec // cancel is invoked via tr.runCancel
	tr := &turnRunner{
		ctxID:     ctxID,
		sentences: make(chan sentenceItem, 8),
		runCtx:    runCtx,
		runCancel: cancel,
		startedAt: time.Now(),
	}
	it.turns[ctxID] = tr
	it.activeContext = ctxID
	it.mu.Unlock()
	go it.run(tr)
	return tr
}

// run is the per-turn goroutine. It consumes sentences one at a time,
// issues a synth() request per sentence, and emits the terminal packets
// on the Done path (sentence channel closed) — or exits silently on the
// interrupt path (runCtx canceled). The select below is load-bearing:
// a plain `for text := range tr.sentences` would park forever if the
// runner is idle between synths when an interrupt fires, because
// runCtx cancellation doesn't wake a channel receive on its own.
func (it *inworldTTS) run(tr *turnRunner) {
	// Always release the run-context on exit. For the Done path the context
	// would otherwise only be freed when the parent inworldTTS.ctx cancels;
	// deferring here keeps the lifetime tight regardless of exit path.
	defer tr.runCancel()
	defer func() {
		// Remove ourselves from the live-turn map on exit, unless another
		// turn with the same id has already replaced us (possible after an
		// interrupt + fresh turn race). Comparing by pointer identity keeps
		// the cleanup safe without extra bookkeeping. Also clear
		// activeContext if it still points at us — mirrors the Interrupt
		// path so diagnostics don't report a stale ctx id after Done.
		it.mu.Lock()
		if cur, ok := it.turns[tr.ctxID]; ok && cur == tr {
			delete(it.turns, tr.ctxID)
		}
		if it.activeContext == tr.ctxID {
			it.activeContext = ""
		}
		it.mu.Unlock()
	}()

	for {
		select {
		case <-tr.runCtx.Done():
			// Interrupt or transformer Close. Transform already emitted
			// the tts=interrupted event on the interrupt path; no
			// terminal end packet on a cancel — the caller initiated the
			// teardown and is not waiting on completion.
			return
		case item := <-tr.sentences:
			if item.done {
				// Done sentinel: Transform's Done path signaled end of
				// turn. Guard against the Done-then-Interrupt race: if
				// both fired before the select woke, skip the terminal
				// packets because the interrupt semantically supersedes.
				if tr.runCtx.Err() != nil {
					return
				}
				_ = it.onPacket(
					internal_type.TextToSpeechEndPacket{ContextID: tr.ctxID},
					internal_type.ConversationEventPacket{
						ContextID: tr.ctxID,
						Name:      "tts",
						Data:      map[string]string{"type": "completed"},
						Time:      time.Now(),
					},
				)
				return
			}
			if err := it.synth(tr.runCtx, tr, item.text); err != nil {
				if tr.runCtx.Err() != nil {
					// Canceled mid-synth by an interrupt — exit quietly.
					return
				}
				it.logger.Errorf("inworld-tts: synth failed for ctx=%s: %v", tr.ctxID, err)
				// Surface the error to the observability channel but
				// keep looping — subsequent sentences may succeed and,
				// critically, callers waiting on TextToSpeechEndPacket
				// still need one when Done eventually arrives.
				_ = it.onPacket(internal_type.ConversationEventPacket{
					ContextID: tr.ctxID,
					Name:      "tts",
					Data:      map[string]string{"type": "error", "message": err.Error()},
					Time:      time.Now(),
				})
			}
		}
	}
}

// synth POSTs one sentence to voice:stream, decodes the NDJSON response,
// and emits one TextToSpeechAudioPacket per audio chunk. The first chunk
// also triggers a one-shot tts_latency_ms metric carrying the sentence's
// TTFB. tr.metricSent is only ever written by this runner goroutine
// (single-writer — no lock needed); tr.startedAt is set in
// getOrCreateRunner *before* `go it.run(tr)`, so the write is
// happens-before the runner's first read.
func (it *inworldTTS) synth(ctx context.Context, tr *turnRunner, text string) error {
	body, err := json.Marshal(inworld_internal.StreamRequest{
		Text:    text,
		VoiceID: it.GetVoiceID(),
		ModelID: it.GetModelID(),
		AudioConfig: inworld_internal.AudioConfig{
			AudioEncoding:   INWORLD_AUDIO_ENCODING,
			SampleRateHertz: INWORLD_SAMPLE_RATE,
		},
	})
	if err != nil {
		return fmt.Errorf("inworld-tts: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		it.streamURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("inworld-tts: build request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+it.GetKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("X-User-Agent", INWORLD_USER_AGENT)

	resp, err := it.client.Do(req)
	if err != nil {
		return fmt.Errorf("inworld-tts: stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Cap the error-body read so a misbehaving server can't force us
		// to buffer an unbounded response. 4 KiB is enough for any JSON
		// error envelope Inworld actually returns.
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		// Drain whatever the LimitReader left so http.Transport can
		// return the underlying conn to the idle pool (HTTP/2 streams
		// don't release until the body is fully consumed).
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("inworld-tts: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return it.streamChunks(ctx, tr, resp.Body)
}

// streamChunks decodes the NDJSON body and emits audio packets. It is
// split out of synth so synth stays within the lint's cognitive-complexity
// budget and so unit tests can reason about the decode path in isolation.
func (it *inworldTTS) streamChunks(ctx context.Context, tr *turnRunner, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	// Default token size is 64KB — raise the cap because a single NDJSON
	// line can carry several seconds of base64 audio.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		audio, err := it.decodeChunk(line)
		if err != nil {
			return err
		}
		if audio == nil {
			continue
		}
		if first {
			first = false
			it.maybeEmitFirstChunkMetric(tr)
		}
		_ = it.onPacket(internal_type.TextToSpeechAudioPacket{
			ContextID:  tr.ctxID,
			AudioChunk: audio,
		})
	}
	if err := scanner.Err(); err != nil {
		// Treat context cancellation quietly — caller distinguishes
		// intentional interrupt by checking ctx.Err().
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("inworld-tts: stream read: %w", err)
	}
	return nil
}

// decodeChunk parses one NDJSON line. Returns (audio, nil) for an audio
// chunk, (nil, nil) for a chunk that has nothing to emit (parse error,
// empty result, ignorable envelope), and (nil, err) for a fatal server
// error frame that must terminate the whole synth attempt.
func (it *inworldTTS) decodeChunk(line []byte) ([]byte, error) {
	var chunk inworld_internal.StreamChunk
	if err := json.Unmarshal(line, &chunk); err != nil {
		it.logger.Errorf("inworld-tts: parse chunk: %v", err)
		return nil, nil
	}
	if chunk.Error != nil {
		return nil, fmt.Errorf("inworld-tts: server error: %s", chunk.Error.Message)
	}
	if chunk.Result == nil || chunk.Result.AudioContent == "" {
		return nil, nil
	}
	audio, err := base64.StdEncoding.DecodeString(chunk.Result.AudioContent)
	if err != nil {
		it.logger.Errorf("inworld-tts: base64 decode: %v", err)
		return nil, nil
	}
	// Inworld wraps each LINEAR16 payload in a minimal RIFF/WAVE
	// container — concatenating those verbatim embeds a WAV header every
	// few milliseconds and produces audible clicks. pcmFromStreamChunk
	// extracts the `data` subchunk when a RIFF wrapper is present and
	// passes bare PCM through untouched.
	return pcmFromStreamChunk(audio), nil
}

// maybeEmitFirstChunkMetric emits the one-shot tts_latency_ms metric for
// a turn. No-op after the first call per turn. Safe to call without
// locking because the runner goroutine is the only writer to these fields.
func (it *inworldTTS) maybeEmitFirstChunkMetric(tr *turnRunner) {
	if tr.metricSent || tr.startedAt.IsZero() {
		return
	}
	tr.metricSent = true
	_ = it.onPacket(internal_type.AssistantMessageMetricPacket{
		ContextID: tr.ctxID,
		Metrics: []*protos.Metric{{
			Name:  "tts_latency_ms",
			Value: fmt.Sprintf("%d", time.Since(tr.startedAt).Milliseconds()),
		}},
	})
}

// Close cancels every live runner, waits briefly for them to drain, and
// emits the final duration metric. Safe to call even if Initialize was
// never invoked.
func (it *inworldTTS) Close(ctx context.Context) error {
	it.ctxCancel()

	it.mu.Lock()
	activeCtxID := it.activeContext
	connectedAt := it.ttsConnectedAt
	it.ttsConnectedAt = time.Time{}
	cancels := make([]context.CancelFunc, 0, len(it.turns))
	for id, tr := range it.turns {
		cancels = append(cancels, tr.runCancel)
		delete(it.turns, id)
	}
	it.mu.Unlock()
	for _, c := range cancels {
		c()
	}

	// Release the HTTP transport's idle pool — nothing on this transformer
	// will run another request.
	if tr, ok := it.client.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
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
