//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Rime integration tests — focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.
// Rime is a TTS-only provider; no STT tests.

package internal_transformer_rime

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Rime TTS Integration Tests
// ---------------------------------------------------------------------------

// TestRimeTTSLifecycle verifies the full TTS flow:
// create → initialize (metric/log) → transform delta+done → audio output → end packet → events in order.
func TestRimeTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewRimeTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "rime-tts", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	assertTTSInitMetric(t, collector)

	// Send text delta + done (done sends {"operation":"eos"} → Rime responds with "done")
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "rime-tts-lifecycle",
		Text:      "Hello world, this is a Rime test.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "rime-tts-lifecycle",
	}))

	// Wait for pipeline to complete (Rime "done" message → end packet)
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

// TestRimeTTSStreamingDeltas verifies that multiple streaming delta chunks
// each trigger a speaking event and together produce audio output.
func TestRimeTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewRimeTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
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
			ContextID: "rime-tts-streaming", Text: chunk}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "rime-tts-streaming"}))

	collector.WaitForTTSEnd(t, 30*time.Second)
	require.NotEmpty(t, collector.AudioPackets())

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

// TestRimeTTSInterruption verifies the interruption flow:
// send delta+done → audio starts → interrupt → "interrupted" event → reconnect → second init metric.
func TestRimeTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewRimeTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "rime-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "rime-tts-interrupt"}))

	collector.WaitForAudio(t, 15*time.Second)

	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "rime-tts-interrupt", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(2 * time.Second)

	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")

	assertTTSInitMetricCountAtLeast(t, collector, 2)
	t.Logf("event_sequence=%v", eventTypes)
}

// TestRimeTTSReconnect verifies two sequential TTS sessions work cleanly.
func TestRimeTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := NewRimeTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: fmt.Sprintf("rime-tts-reconnect-%d", attempt), Text: "Reconnect test."}))
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
			ContextID: fmt.Sprintf("rime-tts-reconnect-%d", attempt)}))

		collector.WaitForTTSEnd(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		assert.NotEmpty(t, collector.EndPackets(), "attempt %d: should emit end packet", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

// ---------------------------------------------------------------------------
// Rime TTS Flow Combination Tests
// ---------------------------------------------------------------------------

// TestRimeTTSFlow_DeltaInterruptDeltaDone verifies:
//
//	init → delta(ctx-1) → done → audio → interrupt → delta(ctx-2) → done → audio+end
func TestRimeTTSFlow_DeltaInterruptDeltaDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewRimeTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Phase 1
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-1", Text: "The weather today is sunny with clear skies."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-1"}))
	collector.WaitForAudio(t, 15*time.Second)

	// Phase 2: interrupt
	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-1", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(500 * time.Millisecond)
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")

	// Phase 3: fresh stream
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-2", Text: "Actually, it will rain later this evening."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-2"}))

	collector.WaitForTTSEnd(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "second utterance should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet for ctx-2")
	phase3Events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, phase3Events, "speaking")
	assert.Contains(t, phase3Events, "completed")
}

// TestRimeTTSFlow_DeltaDoneInterrupt verifies:
//
//	init → delta → done → audio+end → interrupt (late interrupt after completion)
func TestRimeTTSFlow_DeltaDoneInterrupt(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewRimeTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-late", Text: "Short sentence."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-late"}))
	collector.WaitForTTSEnd(t, 15*time.Second)
	assert.NotEmpty(t, collector.EndPackets())

	err = tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-late", Source: internal_type.InterruptionSourceVad})
	require.NoError(t, err, "late interrupt should not error")
	time.Sleep(1 * time.Second)
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")

	// Verify new stream is usable
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-after-late", Text: "I can still speak after a late interrupt."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-after-late"}))
	collector.WaitForAudio(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after late interrupt")
}

// TestRimeTTSFlow_MultipleInterrupts verifies:
//
//	init → delta(1) → done → audio → interrupt → delta(2) → done → audio → interrupt → delta(3) → done → audio+end
func TestRimeTTSFlow_MultipleInterrupts(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tts, err := NewRimeTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	for round := 1; round <= 2; round++ {
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: fmt.Sprintf("round-%d", round),
			Text:      fmt.Sprintf("Attempt %d at speaking.", round)}))
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
			ContextID: fmt.Sprintf("round-%d", round)}))
		collector.WaitForAudio(t, 15*time.Second)
		require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
			ContextID: fmt.Sprintf("round-%d", round),
			Source:    internal_type.InterruptionSourceVad}))
		time.Sleep(500 * time.Millisecond)
		collector.Clear()
	}

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "round-3", Text: "Third time is the charm."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "round-3"}))
	collector.WaitForTTSEnd(t, 15*time.Second)

	assert.NotEmpty(t, collector.AudioPackets())
	assert.NotEmpty(t, collector.EndPackets())
	finalEvents := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, finalEvents, "speaking")
	assert.Contains(t, finalEvents, "completed")
}

// TestRimeTTSFlow_DeltaInterruptNoComplete verifies:
//
//	init → delta → done → audio → interrupt (before end packet) → recovery
func TestRimeTTSFlow_DeltaInterruptNoComplete(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewRimeTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-no-complete",
		Text:      "This sentence will be interrupted before the end packet arrives."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-no-complete"}))
	collector.WaitForAudio(t, 15*time.Second)

	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-no-complete", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(1 * time.Second)
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")

	// Verify recovery
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-recover", Text: "Recovered after interrupted stream."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-recover"}))
	collector.WaitForTTSEnd(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets())
	assert.NotEmpty(t, collector.EndPackets())
}

// TestRimeTTSFlow_RapidDeltasDone verifies:
//
//	init → delta × N (rapid fire) → done → audio+end
func TestRimeTTSFlow_RapidDeltasDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "rime")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewRimeTextToSpeech(ctx, logger,
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
	assert.NotEmpty(t, collector.EndPackets())
	t.Logf("words=%d speaking=%d audio=%d", len(words), speakingCount, len(collector.AudioPackets()))
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

func assertTTSLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "tts.latency_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.Greater(t, ms, 0, "tts.latency_ms should be positive")
				t.Logf("tts.latency_ms=%d", ms)
				return
			}
		}
	}
	t.Error("should have tts.latency_ms metric")
}

func assertTTSInitMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	assertTTSInitMetricCountAtLeast(t, collector, 1)
}

func assertTTSInitMetricCountAtLeast(t *testing.T, collector *testutil.PacketCollector, minimumCount int) {
	t.Helper()
	count := 0
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "tts_init_latency_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "tts_init_latency_ms should be non-negative")
				count++
			}
		}
	}
	assert.GreaterOrEqual(t, count, minimumCount, "should have enough tts_init_latency_ms metrics")
}
