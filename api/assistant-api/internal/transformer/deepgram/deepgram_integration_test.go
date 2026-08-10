//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Deepgram integration tests — focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.

package internal_transformer_deepgram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	deepgram_internal "github.com/rapidaai/api/assistant-api/internal/transformer/deepgram/internal"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Deepgram TTS Integration Tests
// ---------------------------------------------------------------------------

// TestDeepgramTTSLifecycle verifies the full TTS flow:
// create → initialize (metric/log) → transform delta+done → audio output → end packet.
func TestDeepgramTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewDeepgramTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, deepgram_internal.TextToSpeechTransformerName, tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	assertTTSInitMetric(t, collector)

	// Send text delta + done
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "dg-tts-lifecycle",
		Text:      "Hello world, this is a Deepgram test.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "dg-tts-lifecycle",
	}))

	// Wait for the pipeline to complete
	collector.WaitForTTSEnd(t, 20*time.Second)

	// Flow: audio output was produced
	audioPackets := collector.AudioPackets()
	require.NotEmpty(t, audioPackets, "should produce audio packets")
	totalBytes := 0
	for _, ap := range audioPackets {
		totalBytes += len(ap.AudioChunk)
	}
	assert.Greater(t, totalBytes, 0)
	t.Logf("audio_packets=%d total_bytes=%d", len(audioPackets), totalBytes)

	// Flow: end packet closes the context
	endPackets := collector.EndPackets()
	require.NotEmpty(t, endPackets)
	assert.Equal(t, "dg-tts-lifecycle", endPackets[0].ContextID)

	// Flow: event sequence includes speaking → completed
	allEvents := collector.EventPackets()
	eventTypes := ttsEventTypes(allEvents)
	assert.Contains(t, eventTypes, "speaking")
	assert.Contains(t, eventTypes, "completed")
	t.Logf("tts_event_sequence=%v", eventTypes)

	// Flow: latency metric emitted
	assertTTSLatencyMetric(t, collector)
}

// TestDeepgramTTSStreamingDeltas verifies that multiple streaming delta chunks
// each trigger a speaking event and together produce audio output.
func TestDeepgramTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewDeepgramTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	chunks := []string{
		"The quick brown fox ",
		"jumps over the lazy dog. ",
		"Pack my box with five dozen liquor jugs.",
	}
	for _, chunk := range chunks {
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "dg-tts-streaming",
			Text:      chunk,
		}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "dg-tts-streaming",
	}))

	collector.WaitForTTSEnd(t, 30*time.Second)

	// Flow: audio was produced
	require.NotEmpty(t, collector.AudioPackets())

	// Flow: one speaking event per delta chunk
	speakingCount := 0
	for _, ev := range collector.EventPackets() {
		if ev.Record.Component.String() == "tts" && ev.Record.Attributes["type"] == "speaking" {
			speakingCount++
		}
	}
	assert.Equal(t, len(chunks), speakingCount,
		"should emit one speaking event per delta chunk")
	t.Logf("chunks=%d speaking_events=%d audio_packets=%d",
		len(chunks), speakingCount, len(collector.AudioPackets()))
}

// TestDeepgramTTSInterruption verifies the interruption flow:
// send delta+done → audio starts → interrupt → "interrupted" event → reconnect.
func TestDeepgramTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewDeepgramTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	assertTTSInitMetric(t, collector)

	// Send delta + done to trigger audio generation (Deepgram needs Flush to produce audio)
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "dg-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "dg-tts-interrupt",
	}))

	// Wait for audio to start flowing
	collector.WaitForAudio(t, 15*time.Second)

	// Send interruption mid-stream
	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "dg-tts-interrupt",
		Source:    internal_type.InterruptionSourceVad,
	}))

	// Allow reconnect
	time.Sleep(2 * time.Second)

	// Flow: "interrupted" event was emitted
	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")

	// Flow: reconnect emits another TTS init metric.
	assertTTSInitMetricCountAtLeast(t, collector, 2)
	t.Logf("event_sequence=%v", eventTypes)
}

// TestDeepgramTTSReconnect verifies two sequential TTS sessions work cleanly
// (create → use → close → create → use → close).
func TestDeepgramTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := NewDeepgramTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: fmt.Sprintf("dg-tts-reconnect-%d", attempt),
			Text:      "Reconnect test.",
		}))
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
			ContextID: fmt.Sprintf("dg-tts-reconnect-%d", attempt),
		}))

		collector.WaitForTTSEnd(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		assert.NotEmpty(t, collector.EndPackets(), "attempt %d: should emit end packet", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

// ---------------------------------------------------------------------------
// Deepgram TTS Flow Combination Tests
// ---------------------------------------------------------------------------

// TestDeepgramTTSFlow_DeltaInterruptDeltaDone verifies:
//
//	init → delta(ctx-1) → done → audio → interrupt → delta(ctx-2) → done → audio+end
//
// The most common real-world pattern: user interrupts mid-speech, new LLM
// response starts on a fresh stream.
func TestDeepgramTTSFlow_DeltaInterruptDeltaDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewDeepgramTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Phase 1: first utterance
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-1", Text: "The weather today is sunny with clear skies."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-1"}))
	collector.WaitForAudio(t, 15*time.Second)
	t.Logf("phase1: audio_packets=%d", len(collector.AudioPackets()))

	// Phase 2: user interrupts mid-speech
	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-1", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(500 * time.Millisecond)

	eventsAfterInterrupt := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventsAfterInterrupt, "interrupted")
	t.Logf("after_interrupt: events=%v", eventsAfterInterrupt)

	// Phase 3: new LLM response on fresh stream (ctx-2)
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-2", Text: "Actually, it will rain later this evening."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-2"}))

	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	assert.NotEmpty(t, collector.AudioPackets(), "second utterance should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet for ctx-2")
	phase3Events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, phase3Events, "speaking")
	assert.Contains(t, phase3Events, "completed")
	t.Logf("phase3: events=%v audio_packets=%d", phase3Events, len(collector.AudioPackets()))
}

// TestDeepgramTTSFlow_DeltaDoneInterrupt verifies:
//
//	init → delta → done → audio+end → interrupt (late interrupt after completion)
//
// Edge case: interruption arrives after TTS has already finished. The interrupt
// should still succeed (reinitialize) without errors.
func TestDeepgramTTSFlow_DeltaDoneInterrupt(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewDeepgramTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Normal flow: delta → done → completion
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-late", Text: "Short sentence."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-late"}))
	collector.WaitForTTSEnd(t, 15*time.Second)

	assert.NotEmpty(t, collector.EndPackets(), "should have completed before interrupt")
	t.Logf("before_interrupt: events=%v", ttsEventTypes(collector.EventPackets()))

	// Late interrupt after TTS already finished
	err = tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-late", Source: internal_type.InterruptionSourceVad})
	require.NoError(t, err, "late interrupt should not error")
	time.Sleep(1 * time.Second)

	allEvents := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, allEvents, "interrupted")
	t.Logf("after_late_interrupt: events=%v", allEvents)

	// Verify new stream is usable
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-after-late", Text: "I can still speak after a late interrupt."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-after-late"}))
	collector.WaitForAudio(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after late interrupt")
}

// TestDeepgramTTSFlow_MultipleInterrupts verifies:
//
//	init → delta(1) → interrupt → delta(2) → interrupt → delta(3) → done → audio+end
//
// Simulates a chatty user who keeps interrupting the assistant.
func TestDeepgramTTSFlow_MultipleInterrupts(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tts, err := NewDeepgramTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Round 1: delta → done → audio → interrupt
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "round-1", Text: "First attempt at speaking."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "round-1"}))
	collector.WaitForAudio(t, 15*time.Second)
	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "round-1", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(500 * time.Millisecond)

	// Round 2: delta → done → audio → interrupt
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "round-2", Text: "Second attempt, interrupted again."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "round-2"}))
	collector.WaitForAudio(t, 15*time.Second)
	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "round-2", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(500 * time.Millisecond)

	// Round 3: delta → done → end (finally completes)
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "round-3", Text: "Third time is the charm."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "round-3"}))
	collector.WaitForTTSEnd(t, 15*time.Second)

	assert.NotEmpty(t, collector.AudioPackets(), "final round should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "final round should emit end packet")
	finalEvents := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, finalEvents, "speaking")
	assert.Contains(t, finalEvents, "completed")
	t.Logf("round3: events=%v audio=%d", finalEvents, len(collector.AudioPackets()))
}

// TestDeepgramTTSFlow_DeltaInterruptNoComplete verifies:
//
//	init → delta → done → audio → interrupt (user abandons without waiting for end)
//
// The interrupt should cleanly tear down the old stream and reinitialize.
func TestDeepgramTTSFlow_DeltaInterruptNoComplete(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewDeepgramTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Send delta + done → wait for audio → interrupt before end packet
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-no-complete",
		Text:      "This sentence will be interrupted before the end packet arrives.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-no-complete"}))
	collector.WaitForAudio(t, 15*time.Second)

	// Interrupt before end packet arrives
	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-no-complete", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(1 * time.Second)

	events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, events, "interrupted")
	t.Logf("events=%v", events)

	// Verify: can still use the stream after interrupt
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-recover", Text: "Recovered after interrupted stream."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-recover"}))
	collector.WaitForTTSEnd(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after recovery")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet after recovery")
}

// TestDeepgramTTSFlow_RapidDeltasDone verifies:
//
//	init → delta × N (rapid fire) → done → audio+end
//
// Tests that many small deltas sent in quick succession are all processed.
func TestDeepgramTTSFlow_RapidDeltasDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewDeepgramTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	words := []string{"Hello", " there,", " how", " are", " you", " doing", " today?"}
	for _, w := range words {
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "ctx-rapid", Text: w}))
	}
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-rapid"}))

	collector.WaitForTTSEnd(t, 20*time.Second)

	speakingCount := 0
	for _, ev := range collector.EventPackets() {
		if ev.Record.Component.String() == "tts" && ev.Record.Attributes["type"] == "speaking" {
			speakingCount++
		}
	}
	assert.Equal(t, len(words), speakingCount, "one speaking event per word delta")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet")
	t.Logf("words=%d speaking=%d audio=%d", len(words), speakingCount, len(collector.AudioPackets()))
}

// ---------------------------------------------------------------------------
// Deepgram STT Integration Tests
// ---------------------------------------------------------------------------

// TestDeepgramSTTLifecycle verifies the full STT flow:
// create (connect + metric/log) → feed audio (no errors) → transcripts arrive.
// If transcripts arrive, verify they carry the expected metadata fields.
func TestDeepgramSTTLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(
		WithContext(ctx),
		WithLogger(logger),
		WithCredential(cred),
		WithOnPacket(collector.OnPacket),
		WithOptions(opts),
	)
	require.NoError(t, err)
	require.NotNil(t, stt)
	assert.Equal(t, deepgram_internal.SpeechToTextTransformerName, stt.Name())

	defer stt.Close(ctx)
	assertSTTInitMetric(t, collector)

	// Flow: Feed audio without errors
	feedDone := make(chan struct{})
	go func() {
		testutil.FeedAudio(ctx, t, stt, speech)
		close(feedDone)
	}()

	select {
	case <-feedDone:
	case <-ctx.Done():
		t.Fatal("context cancelled before audio feeding completed")
	}

	// Wait for transcripts
	collector.WaitForAnyTranscript(t, 10*time.Second)

	transcripts := collector.TranscriptPackets()
	interims := collector.InterimTranscripts()
	finals := collector.FinalTranscripts()
	t.Logf("transcripts=%d (interims=%d finals=%d)", len(transcripts), len(interims), len(finals))

	// If transcripts arrived, verify their shape
	for _, tr := range transcripts {
		assert.NotEmpty(t, tr.Script, "transcript script should not be empty")
		assert.Greater(t, tr.Confidence, 0.0, "confidence should be > 0")
	}

	// If final transcripts arrived, verify events + metrics
	if len(finals) > 0 {
		eventTypes := sttEventTypes(collector.EventPackets())
		assert.Contains(t, eventTypes, "completed")
		t.Logf("stt_event_sequence=%v", eventTypes)

		interruptions := collector.InterruptionDetectedPackets()
		assert.NotEmpty(t, interruptions, "should emit interruption packets with transcripts")

		assertSTTLatencyMetric(t, collector)
	}
}

// TestDeepgramSTTAudioAcceptance verifies that the STT transformer accepts audio
// chunks without returning errors — the core flow for real-time streaming.
func TestDeepgramSTTAudioAcceptance(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(
		WithContext(ctx),
		WithLogger(logger),
		WithCredential(cred),
		WithOnPacket(collector.OnPacket),
		WithOptions(opts),
	)
	require.NoError(t, err)
	defer stt.Close(ctx)

	// Flow: each Transform call accepts the audio chunk without error
	chunks := testutil.ChunkAudio(testutil.SineTonePCM(440, 1.0), testutil.FrameSize)
	for i, chunk := range chunks {
		err := stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
			ContextID: "dg-stt-accept",
			Audio:     chunk,
		})
		require.NoError(t, err, "chunk %d should be accepted", i)
	}
	t.Logf("chunks_accepted=%d", len(chunks))
}

// TestDeepgramSTTSilentAudio verifies that sending silent audio does not
// produce false transcripts.
func TestDeepgramSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(
		WithContext(ctx),
		WithLogger(logger),
		WithCredential(cred),
		WithOnPacket(collector.OnPacket),
		WithOptions(opts),
	)
	require.NoError(t, err)
	defer stt.Close(ctx)

	silence := testutil.SilentPCM(2.0)
	go testutil.FeedAudio(ctx, t, stt, silence)

	time.Sleep(4 * time.Second)

	finals := collector.FinalTranscripts()
	t.Logf("final_transcripts_from_silence=%d", len(finals))
	for _, f := range finals {
		assert.Empty(t, f.Script,
			"silence should not produce non-empty final transcripts, got: %q (confidence=%.4f)", f.Script, f.Confidence)
	}
}

// TestDeepgramSTTReconnect verifies two sequential STT sessions work cleanly
// (create → use → close → create → use → close).
func TestDeepgramSTTReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

		stt, err := NewSpeechToText(
			WithContext(ctx),
			WithLogger(logger),
			WithCredential(cred),
			WithOnPacket(collector.OnPacket),
			WithOptions(opts),
		)
		require.NoError(t, err, "attempt %d", attempt)
		assertSTTInitMetric(t, collector)

		feedDone := make(chan struct{})
		go func() {
			testutil.FeedAudio(ctx, t, stt, speech)
			close(feedDone)
		}()

		select {
		case <-feedDone:
		case <-ctx.Done():
			t.Fatalf("attempt %d: context cancelled before audio feeding completed", attempt)
		}

		t.Logf("attempt=%d transcripts=%d", attempt, len(collector.TranscriptPackets()))

		stt.Close(ctx)
		cancel()

		time.Sleep(500 * time.Millisecond)
	}
}

// TestDeepgramSTTCloseWhileStreaming verifies that closing the STT transformer
// while audio is actively being fed does not panic or return unexpected errors.
func TestDeepgramSTTCloseWhileStreaming(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stt, err := NewSpeechToText(
		WithContext(ctx),
		WithLogger(logger),
		WithCredential(testutil.BuildCredential(pcfg.Credential)),
		WithOnPacket(collector.OnPacket),
		WithOptions(testutil.BuildOptions(pcfg.Options)),
	)
	require.NoError(t, err)
	assertSTTInitMetric(t, collector)

	go func() {
		chunks := testutil.ChunkAudio(testutil.SineTonePCM(440, 3.0), testutil.FrameSize)
		for _, chunk := range chunks {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
				ContextID: "dg-stt-close-mid", Audio: chunk})
			time.Sleep(time.Duration(testutil.FrameDuration) * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	err = stt.Close(ctx)
	assert.NoError(t, err, "closing STT mid-stream should not error")
}

// TestDeepgramSTTTranscriptContent verifies that real speech audio produces
// a transcript containing the expected words.
func TestDeepgramSTTTranscriptContent(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "deepgram")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stt, err := NewSpeechToText(
		WithContext(ctx),
		WithLogger(logger),
		WithCredential(testutil.BuildCredential(pcfg.Credential)),
		WithOnPacket(collector.OnPacket),
		WithOptions(testutil.BuildOptions(pcfg.Options)),
	)
	require.NoError(t, err)
	defer stt.Close(ctx)

	feedDone := make(chan struct{})
	go func() {
		testutil.FeedAudio(ctx, t, stt, speech)
		close(feedDone)
	}()

	select {
	case <-feedDone:
	case <-ctx.Done():
		t.Fatal("context cancelled before audio feeding completed")
	}

	collector.WaitForFinalTranscript(t, 10*time.Second)

	finals := collector.FinalTranscripts()
	require.NotEmpty(t, finals, "should produce at least one final transcript")

	combined := ""
	for _, f := range finals {
		combined += " " + f.Script
	}
	lower := strings.ToLower(combined)
	assert.True(t,
		strings.Contains(lower, "hello") || strings.Contains(lower, "world"),
		"expected transcript to contain 'hello' or 'world', got: %q", combined)
	t.Logf("transcript=%q", strings.TrimSpace(combined))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func ttsEventTypes(events []internal_type.ObservabilityEventRecordPacket) []string {
	var out []string
	for _, ev := range events {
		if ev.Record.Component.String() == "tts" {
			out = append(out, ev.Record.Attributes["type"])
		}
	}
	return out
}

func sttEventTypes(events []internal_type.ObservabilityEventRecordPacket) []string {
	var out []string
	for _, ev := range events {
		if ev.Record.Component.String() == "stt" {
			out = append(out, ev.Record.Attributes["type"])
		}
	}
	return out
}

func assertTTSInitMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == observability.MetricTTSInitLatencyMs {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "%s should be non-negative", observability.MetricTTSInitLatencyMs)
				t.Logf("%s=%d", observability.MetricTTSInitLatencyMs, ms)
				return
			}
		}
	}
	t.Errorf("should have %s metric", observability.MetricTTSInitLatencyMs)
}

func assertTTSInitMetricCountAtLeast(t *testing.T, collector *testutil.PacketCollector, minimumCount int) {
	t.Helper()
	count := 0
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == observability.MetricTTSInitLatencyMs {
				count++
			}
		}
	}
	assert.GreaterOrEqual(t, count, minimumCount, "should have enough %s metrics", observability.MetricTTSInitLatencyMs)
}

func assertTTSLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == observability.MetricTTSLatencyMs {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.Greater(t, ms, 0, "%s should be positive", observability.MetricTTSLatencyMs)
				t.Logf("%s=%d", observability.MetricTTSLatencyMs, ms)
				return
			}
		}
	}
	t.Errorf("should have %s metric", observability.MetricTTSLatencyMs)
}

func assertSTTInitMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == observability.MetricSTTInitLatencyMs {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "%s should be non-negative", observability.MetricSTTInitLatencyMs)
				t.Logf("%s=%d", observability.MetricSTTInitLatencyMs, ms)
				return
			}
		}
	}
	t.Errorf("should have %s metric", observability.MetricSTTInitLatencyMs)
}

func assertSTTLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == observability.MetricSTTLatencyMs {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "%s should be non-negative", observability.MetricSTTLatencyMs)
				t.Logf("%s=%d", observability.MetricSTTLatencyMs, ms)
				return
			}
		}
	}
	t.Errorf("should have %s metric", observability.MetricSTTLatencyMs)
}
