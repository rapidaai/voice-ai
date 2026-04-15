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

// turnRunner owns all state for one Rapida turn. Exactly one goroutine
// drains `sentences`, so audio for a given turn is always emitted in the
// order the aggregator delivered the sentences — and across turns we
// simply have one runner per context id. runCtx cancels in-flight synth
// requests (interrupts) while ctxID is the turn's Rapida context id that
// every emitted packet carries.
type turnRunner struct {
	ctxID     string
	sentences chan string

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
		it.mu.Lock()
		cancels := make([]context.CancelFunc, 0, len(it.turns))
		for id, tr := range it.turns {
			cancels = append(cancels, tr.runCancel)
			delete(it.turns, id)
		}
		it.activeContext = ""
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
		case tr.sentences <- input.Text:
		case <-tr.runCtx.Done():
			// Turn was just canceled (interrupt) — drop the delta.
		}
		_ = it.onPacket(internal_type.ConversationEventPacket{
			ContextID: input.ContextID,
			Name:      "tts",
			Data:      map[string]string{"type": "speaking", "text": input.Text},
			Time:      time.Now(),
		})
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
		// Closing `sentences` signals the runner to drain + emit the
		// terminal TextToSpeechEndPacket once the in-flight synth returns.
		// The runner owns its own lifecycle and removes itself from
		// it.turns when it exits, so there is nothing else to clean up here.
		close(tr.sentences)
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
		sentences: make(chan string, 8),
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
// after the channel closes (Done path) or when the run context is
// canceled (interrupt path).
func (it *inworldTTS) run(tr *turnRunner) {
	// Always release the run-context on exit. For the Done path the context
	// would otherwise only be freed when the parent inworldTTS.ctx cancels;
	// deferring here keeps the lifetime tight regardless of exit path.
	defer tr.runCancel()
	defer func() {
		// Remove ourselves from the live-turn map on exit, unless another
		// turn with the same id has already replaced us (possible after an
		// interrupt + fresh turn race). Comparing by pointer identity keeps
		// the cleanup safe without extra bookkeeping.
		it.mu.Lock()
		if cur, ok := it.turns[tr.ctxID]; ok && cur == tr {
			delete(it.turns, tr.ctxID)
		}
		it.mu.Unlock()
	}()

	interrupted := false
	for text := range tr.sentences {
		if err := it.synth(tr.runCtx, tr, text); err != nil {
			if tr.runCtx.Err() != nil {
				// Canceled by an interrupt — drain any remaining queued
				// sentences and exit without a terminal end packet.
				interrupted = true
				break
			}
			it.logger.Errorf("inworld-tts: synth failed for ctx=%s: %v", tr.ctxID, err)
			// Surface the error to the observability channel but keep
			// draining — subsequent sentences may succeed and, critically,
			// callers waiting on TextToSpeechEndPacket still need one.
			_ = it.onPacket(internal_type.ConversationEventPacket{
				ContextID: tr.ctxID,
				Name:      "tts",
				Data:      map[string]string{"type": "error", "message": err.Error()},
				Time:      time.Now(),
			})
		}
	}

	if interrupted {
		// Interrupt path already emitted its own tts=interrupted event on
		// the Transform side; nothing more to send here.
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
}

// synth POSTs one sentence to voice:stream, decodes the NDJSON response,
// and emits one TextToSpeechAudioPacket per audio chunk. The first chunk
// also triggers a one-shot tts_latency_ms metric carrying the sentence's
// TTFB. synth is called sequentially from the runner goroutine, so there
// is no need to guard tr.metricSent/startedAt.
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
		it.GetTextToSpeechConnectionString(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("inworld-tts: build request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+it.GetKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")

	resp, err := it.client.Do(req)
	if err != nil {
		return fmt.Errorf("inworld-tts: stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
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
	return audio, nil
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
