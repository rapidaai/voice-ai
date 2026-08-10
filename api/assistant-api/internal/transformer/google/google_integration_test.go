//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Google Speech integration tests — focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.

package internal_transformer_google

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Google TTS Integration Tests
// ---------------------------------------------------------------------------

// TestGoogleTTSLifecycle verifies the full TTS flow:
// create → initialize (metric) → transform delta+done → audio output → end packet → events in order.
func TestGoogleTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewGoogleTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "google-tts", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	assertTTSInitMetric(t, collector)

	// Send text delta + done
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "google-tts-lifecycle",
		Text:      "Hello world, this is a Google Speech test.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "google-tts-lifecycle",
	}))

	// Wait for audio to arrive (Google TTS streams audio back as it synthesizes)
	collector.WaitForAudio(t, 20*time.Second)

	// Wait for TTS end packet (CloseSend on done triggers server EOF → end packet)
	collector.WaitForTTSEnd(t, 10*time.Second)

	// Flow: audio output was produced
	audioPackets := collector.AudioPackets()
	require.NotEmpty(t, audioPackets, "should produce audio packets")
	totalBytes := 0
	for _, ap := range audioPackets {
		totalBytes += len(ap.AudioChunk)
	}
	assert.Greater(t, totalBytes, 0)
	t.Logf("audio_packets=%d total_bytes=%d", len(audioPackets), totalBytes)

	// Flow: end packet was emitted
	endPackets := collector.EndPackets()
	require.NotEmpty(t, endPackets, "should emit TextToSpeechEndPacket")

	// Flow: event sequence includes speaking → completed
	allEvents := collector.EventPackets()
	eventTypes := ttsEventTypes(allEvents)
	assert.Contains(t, eventTypes, "speaking")
	assert.Contains(t, eventTypes, "completed")
	t.Logf("tts_event_sequence=%v", eventTypes)

	// Flow: latency metric emitted
	assertTTSLatencyMetric(t, collector)
}

// TestGoogleTTSStreamingDeltas verifies that multiple streaming delta chunks
// each trigger a speaking event and together produce audio output.
func TestGoogleTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewGoogleTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
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
			ContextID: "google-tts-streaming",
			Text:      chunk,
		}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "google-tts-streaming",
	}))

	// Wait for audio to arrive
	collector.WaitForAudio(t, 30*time.Second)

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

// TestGoogleTTSInterruption verifies the interruption flow:
// send delta+done → audio starts → interrupt → "interrupted" event → reconnect → second init metric.
func TestGoogleTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewGoogleTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Send delta + done to trigger audio generation
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "google-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "google-tts-interrupt",
	}))

	// Wait for audio to start flowing
	collector.WaitForAudio(t, 15*time.Second)

	// Send interruption mid-stream
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "google-tts-interrupt",
	}))

	// Allow reconnect
	time.Sleep(2 * time.Second)

	// Flow: "interrupted" event was emitted
	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")

	assertTTSInitMetricCountAtLeast(t, collector, 2)
	t.Logf("event_sequence=%v", eventTypes)
}

// TestGoogleTTSReconnect verifies two sequential TTS sessions work cleanly
// (create → use → close → create → use → close).
func TestGoogleTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := NewGoogleTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: fmt.Sprintf("google-tts-reconnect-%d", attempt),
			Text:      "Reconnect test.",
		}))
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
			ContextID: fmt.Sprintf("google-tts-reconnect-%d", attempt),
		}))

		collector.WaitForAudio(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

// ---------------------------------------------------------------------------
// Google TTS Flow Combination Tests
//
// Real-world voice flow: Initialize → [Interrupt | Delta | Done]*
// These tests exercise every realistic ordering to catch regressions.
// ---------------------------------------------------------------------------

// TestGoogleTTSFlow_DeltaInterruptDeltaDone verifies:
//
//	init → delta(ctx-1) → audio → interrupt → delta(ctx-2) → done → audio+end
//
// The most common real-world pattern: user interrupts mid-speech, new LLM
// response starts on a fresh stream.
func TestGoogleTTSFlow_DeltaInterruptDeltaDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Phase 1: first utterance
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-1",
		Text:      "The weather today is sunny with clear skies.",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	t.Logf("phase1: audio_packets=%d", len(collector.AudioPackets()))

	// Phase 2: user interrupts mid-speech
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-1",
	}))
	time.Sleep(500 * time.Millisecond) // allow reconnect

	eventsAfterInterrupt := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventsAfterInterrupt, "interrupted")
	t.Logf("after_interrupt: events=%v", eventsAfterInterrupt)

	// Phase 3: new LLM response on fresh stream (ctx-2)
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-2",
		Text:      "Actually, it will rain later this evening.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-2",
	}))

	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	// Verify second utterance produced audio + end
	assert.NotEmpty(t, collector.AudioPackets(), "second utterance should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet for ctx-2")
	phase3Events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, phase3Events, "speaking")
	assert.Contains(t, phase3Events, "completed")
	t.Logf("phase3: events=%v audio_packets=%d", phase3Events, len(collector.AudioPackets()))
}

// TestGoogleTTSFlow_DeltaDoneInterrupt verifies:
//
//	init → delta → done → audio+end → interrupt (late interrupt after completion)
//
// Edge case: interruption arrives after TTS has already finished. The interrupt
// should still succeed (reinitialize) without errors.
func TestGoogleTTSFlow_DeltaDoneInterrupt(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Normal flow: delta → done → completion
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-late",
		Text:      "Short sentence.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-late",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	assert.NotEmpty(t, collector.EndPackets(), "should have completed before interrupt")
	t.Logf("before_interrupt: events=%v", ttsEventTypes(collector.EventPackets()))

	// Late interrupt after TTS already finished
	err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-late",
	})
	require.NoError(t, err, "late interrupt should not error")

	time.Sleep(1 * time.Second)

	// Verify interrupted + reinitialized
	allEvents := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, allEvents, "interrupted")
	t.Logf("after_late_interrupt: events=%v", allEvents)

	// Verify new stream is usable: send another utterance
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-after-late",
		Text:      "I can still speak after a late interrupt.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-after-late",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after late interrupt")
}

// TestGoogleTTSFlow_InterruptBeforeDelta verifies:
//
//	init → interrupt (no prior text on this context) → delta → done → audio+end
//
// Edge case: interrupt arrives before any text has been sent on the stream.
// Since no audio has been generated yet, the interrupt should be a no-op
// and the subsequent delta+done should work normally.
func TestGoogleTTSFlow_InterruptBeforeDelta(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Interrupt before any text — contextId is "" so this should be a no-op
	err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-early",
	})
	require.NoError(t, err, "early interrupt should not error")

	// Interruption on empty context should NOT emit interrupted event
	earlyEvents := ttsEventTypes(collector.EventPackets())
	assert.NotContains(t, earlyEvents, "interrupted",
		"interrupt on empty context should be a no-op")
	t.Logf("after_early_interrupt: events=%v", earlyEvents)

	// Normal flow should still work
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-after-early",
		Text:      "This should work fine after an early interrupt.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-after-early",
	}))

	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet")
	t.Logf("final: events=%v audio=%d", ttsEventTypes(collector.EventPackets()), len(collector.AudioPackets()))
}

// TestGoogleTTSFlow_MultipleInterrupts verifies:
//
//	init → delta(1) → interrupt → delta(2) → interrupt → delta(3) → done → audio+end
//
// Simulates a chatty user who keeps interrupting the assistant.
func TestGoogleTTSFlow_MultipleInterrupts(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tts, err := NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Round 1: delta → audio → interrupt
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "round-1", Text: "First attempt at speaking."}))
	collector.WaitForAudio(t, 15*time.Second)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "round-1"}))
	time.Sleep(500 * time.Millisecond)

	// Round 2: delta → audio → interrupt
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "round-2", Text: "Second attempt, interrupted again."}))
	collector.WaitForAudio(t, 15*time.Second)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "round-2"}))
	time.Sleep(500 * time.Millisecond)

	// Round 3: delta → done → end (finally completes)
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "round-3", Text: "Third time is the charm."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "round-3"}))
	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	assert.NotEmpty(t, collector.AudioPackets(), "final round should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "final round should emit end packet")
	finalEvents := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, finalEvents, "speaking")
	assert.Contains(t, finalEvents, "completed")
	t.Logf("round3: events=%v audio=%d", finalEvents, len(collector.AudioPackets()))
}

// TestGoogleTTSFlow_DeltaInterruptNoComplete verifies:
//
//	init → delta → audio → interrupt (user abandons without done)
//
// The interrupt should cleanly tear down the old stream and reinitialize.
// No TextToSpeechEndPacket from the normal path, but the old stream's
// recvLoop should exit cleanly.
func TestGoogleTTSFlow_DeltaInterruptNoComplete(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Send delta (no done) → wait for audio
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-no-done",
		Text:      "This sentence will never be completed because the user interrupts.",
	}))
	collector.WaitForAudio(t, 15*time.Second)

	// Interrupt without ever sending done
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-no-done",
	}))
	time.Sleep(1 * time.Second)

	events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, events, "interrupted")
	t.Logf("events=%v", events)

	// Verify: can still use the stream after interrupt-without-done
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-recover",
		Text:      "Recovered after interrupted delta.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-recover",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after recovery")
}

// TestGoogleTTSFlow_RapidDeltasDone verifies:
//
//	init → delta × N (rapid fire) → done → audio+end
//
// Tests that many small deltas sent in quick succession are all processed.
func TestGoogleTTSFlow_RapidDeltasDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Send many small deltas rapidly (simulating token-by-token LLM streaming)
	words := []string{"Hello", " there,", " how", " are", " you", " doing", " today?"}
	for _, w := range words {
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: "ctx-rapid",
			Text:      w,
		}))
	}
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-rapid",
	}))

	collector.WaitForAudio(t, 20*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	// Verify all speaking events emitted
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
// Google STT Integration Tests
// ---------------------------------------------------------------------------

// TestGoogleSTTLifecycle verifies the full STT flow:
// create → initialize (metric) → feed audio (no errors) → transcripts arrive.
// If transcripts arrive, verify they carry
// the expected metadata fields.
func TestGoogleSTTLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(cred), WithOnPacket(collector.OnPacket), WithOptions(opts))
	require.NoError(t, err)
	require.NotNil(t, stt)
	assert.Equal(t, "google-stt", stt.Name())

	// Flow: Initialize succeeds and emits init metric
	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	assertSTTInitMetric(t, collector)

	// Flow: Feed audio without errors
	feedDone := make(chan struct{})
	go func() {
		testutil.FeedAudio(ctx, t, stt, speech)
		close(feedDone)
	}()

	// Wait for feeding to complete
	select {
	case <-feedDone:
	case <-ctx.Done():
		t.Fatal("context cancelled before audio feeding completed")
	}

	// Wait for at least one transcript instead of fixed sleep
	collector.WaitForAnyTranscript(t, 10*time.Second)

	// Log what we received
	transcripts := collector.TranscriptPackets()
	interims := collector.InterimTranscripts()
	finals := collector.FinalTranscripts()
	t.Logf("transcripts=%d (interims=%d finals=%d)", len(transcripts), len(interims), len(finals))

	// If transcripts arrived, verify their shape
	for _, tr := range transcripts {
		assert.NotEmpty(t, tr.Script, "transcript script should not be empty")
	}

	// Only final transcripts carry confidence from Google Speech;
	// interim results have confidence = 0 which is expected.
	for _, tr := range finals {
		assert.NotEmpty(t, tr.Script, "final transcript should not be empty")
	}

	// Verify transcript content — hello_world.pcm should produce something
	// containing "hello" or "world" (case-insensitive).
	if len(finals) > 0 {
		combined := ""
		for _, f := range finals {
			combined += " " + f.Script
		}
		lower := strings.ToLower(combined)
		assert.True(t,
			strings.Contains(lower, "hello") || strings.Contains(lower, "world"),
			"expected transcript to contain 'hello' or 'world', got: %q", combined)
	}

	// If final transcripts arrived, verify events + metrics
	if len(finals) > 0 {
		eventTypes := sttEventTypes(collector.EventPackets())
		assert.Contains(t, eventTypes, "completed")
		t.Logf("stt_event_sequence=%v", eventTypes)

		// Verify interruption packets accompany transcripts
		interruptions := collector.InterruptionDetectedPackets()
		assert.NotEmpty(t, interruptions, "should emit interruption packets with transcripts")

		// Verify latency metric
		assertSTTLatencyMetric(t, collector)
	}
}

// TestGoogleSTTAudioAcceptance verifies that the STT transformer accepts audio
// chunks without returning errors — the core flow for real-time streaming.
func TestGoogleSTTAudioAcceptance(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(cred), WithOnPacket(collector.OnPacket), WithOptions(opts))
	require.NoError(t, err)
	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	// Flow: each Transform call accepts the audio chunk without error
	chunks := testutil.ChunkAudio(testutil.SineTonePCM(440, 1.0), testutil.FrameSize)
	for i, chunk := range chunks {
		err := stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
			ContextID: "google-stt-accept",
			Audio:     chunk,
		})
		require.NoError(t, err, "chunk %d should be accepted", i)
	}
	t.Logf("chunks_accepted=%d", len(chunks))
}

// TestGoogleSTTSilentAudio verifies that sending silent audio does not
// produce false transcripts.
func TestGoogleSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(cred), WithOnPacket(collector.OnPacket), WithOptions(opts))
	require.NoError(t, err)
	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	// Feed 2 seconds of silence
	silence := testutil.SilentPCM(2.0)
	go testutil.FeedAudio(ctx, t, stt, silence)

	// Wait for audio + processing buffer
	time.Sleep(4 * time.Second)

	finals := collector.FinalTranscripts()
	t.Logf("final_transcripts_from_silence=%d", len(finals))
	for _, f := range finals {
		assert.Empty(t, f.Script,
			"silence should not produce non-empty final transcripts, got: %q (confidence=%.4f)", f.Script, f.Confidence)
	}
}

// TestGoogleSTTReconnect verifies two sequential STT sessions work cleanly
// (create → use → close → create → use → close).
func TestGoogleSTTReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

		stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(cred), WithOnPacket(collector.OnPacket), WithOptions(opts))
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, stt.Initialize(), "attempt %d", attempt)

		// Flow: feed audio and verify no errors
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

		assertSTTInitMetric(t, collector)
		t.Logf("attempt=%d transcripts=%d", attempt, len(collector.TranscriptPackets()))

		stt.Close(ctx)
		cancel()

		// Brief pause between sessions
		time.Sleep(500 * time.Millisecond)
	}
}

// TestGoogleSTTCloseWhileStreaming verifies that closing the STT transformer
// while audio is actively being fed does not panic or return unexpected errors.
func TestGoogleSTTCloseWhileStreaming(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(cred), WithOnPacket(collector.OnPacket), WithOptions(opts))
	require.NoError(t, err)
	require.NoError(t, stt.Initialize())

	// Start feeding audio in the background
	go func() {
		chunks := testutil.ChunkAudio(testutil.SineTonePCM(440, 3.0), testutil.FrameSize)
		for _, chunk := range chunks {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
				ContextID: "google-stt-close-mid",
				Audio:     chunk,
			})
			time.Sleep(time.Duration(testutil.FrameDuration) * time.Millisecond)
		}
	}()

	// Let some audio flow, then close mid-stream
	time.Sleep(500 * time.Millisecond)
	err = stt.Close(ctx)
	assert.NoError(t, err, "closing STT mid-stream should not error")

	assertSTTInitMetric(t, collector)
}

// TestGoogleSTTTranscriptContent verifies that real speech audio produces
// a transcript containing the expected words.
func TestGoogleSTTTranscriptContent(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(cred), WithOnPacket(collector.OnPacket), WithOptions(opts))
	require.NoError(t, err)
	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	// Feed real speech audio
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

	// Wait for a final transcript
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
			if metric.Name == "tts_init_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "tts_init_ms should be non-negative")
				t.Logf("tts_init_ms=%d", ms)
				return
			}
		}
	}
	t.Error("should have tts_init_ms metric")
}

func assertTTSInitMetricCountAtLeast(t *testing.T, collector *testutil.PacketCollector, minimumCount int) {
	t.Helper()
	count := 0
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "tts_init_ms" {
				count++
			}
		}
	}
	assert.GreaterOrEqual(t, count, minimumCount, "should have enough tts_init_ms metrics")
}

func assertTTSLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "tts_latency_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.Greater(t, ms, 0, "tts_latency_ms should be positive")
				t.Logf("tts_latency_ms=%d", ms)
				return
			}
		}
	}
	t.Error("should have tts_latency_ms metric")
}

func assertSTTInitMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "stt_init_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "stt_init_ms should be non-negative")
				t.Logf("stt_init_ms=%d", ms)
				return
			}
		}
	}
	t.Error("should have stt_init_ms metric")
}

func assertSTTLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "stt_latency_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "stt_latency_ms should be non-negative")
				t.Logf("stt_latency_ms=%d", ms)
				return
			}
		}
	}
	t.Error("should have stt_latency_ms metric")
}
