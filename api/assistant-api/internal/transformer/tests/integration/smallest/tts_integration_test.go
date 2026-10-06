//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Smallest AI integration tests focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.

package integration_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	smallest "github.com/rapidaai/api/assistant-api/internal/transformer/smallest"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSmallestTTSLifecycle verifies the full TTS flow:
// create → initialize (metric/log) → transform delta+done → audio output → end packet → events in order.
func TestSmallestTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := smallest.NewSmallestTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "smallest-tts", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	assertTTSInitMetric(t, collector)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "smallest-tts-lifecycle",
		Text:      "Hello world, this is a Smallest AI test.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "smallest-tts-lifecycle",
	}))

	// Smallest can wait up to complete_backoff_ms (default ~4s) after the
	// last chunk before sending status:"complete".
	collector.WaitForTTSEnd(t, 20*time.Second)

	audioPackets := collector.AudioPackets()
	require.NotEmpty(t, audioPackets, "should produce audio packets")
	totalBytes := 0
	for _, ap := range audioPackets {
		totalBytes += len(ap.AudioChunk)
	}
	assert.Greater(t, totalBytes, 0)
	t.Logf("audio_packets=%d total_bytes=%d", len(audioPackets), totalBytes)

	endPackets := collector.EndPackets()
	require.NotEmpty(t, endPackets, "should emit TextToSpeechEndPacket")

	allEvents := collector.EventPackets()
	eventTypes := ttsEventTypes(allEvents)
	assert.Contains(t, eventTypes, "speaking")
	assert.Contains(t, eventTypes, "completed")
	t.Logf("tts_event_sequence=%v", eventTypes)

	assertTTSLatencyMetric(t, collector)
}

// TestSmallestTTSStreamingDeltas verifies that multiple streaming delta chunks
// each trigger a speaking event and together produce audio output.
func TestSmallestTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := smallest.NewSmallestTextToSpeech(ctx, logger,
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
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: "smallest-tts-streaming",
			Text:      chunk,
		}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "smallest-tts-streaming",
	}))

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

// TestSmallestTTSInterruption verifies the interruption flow:
// Text and done start audio; interruption stops synthesis without another initialization.
func TestSmallestTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := smallest.NewSmallestTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "smallest-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "smallest-tts-interrupt",
	}))

	collector.WaitForAudio(t, 15*time.Second)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "smallest-tts-interrupt",
	}))

	time.Sleep(2 * time.Second)

	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")

	assertTTSInitMetricCount(t, collector, 1)
	t.Logf("event_sequence=%v", eventTypes)
}

// TestSmallestTTSReconnect verifies two sequential TTS sessions work cleanly.
func TestSmallestTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := smallest.NewSmallestTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: fmt.Sprintf("smallest-tts-reconnect-%d", attempt),
			Text:      "Reconnect test.",
		}))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
			ContextID: fmt.Sprintf("smallest-tts-reconnect-%d", attempt),
		}))

		collector.WaitForTTSEnd(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		assert.NotEmpty(t, collector.EndPackets(), "attempt %d: should emit end packet", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

// TestSmallestTTSFlow_DeltaInterruptDeltaDone verifies:
//
//	init → delta(ctx-1) → done → audio → interrupt → delta(ctx-2) → done → audio+end
func TestSmallestTTSFlow_DeltaInterruptDeltaDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := smallest.NewSmallestTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-1", Text: "The weather today is sunny with clear skies."}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-1"}))
	collector.WaitForAudio(t, 15*time.Second)
	t.Logf("phase1: audio_packets=%d", len(collector.AudioPackets()))

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-1"}))
	time.Sleep(500 * time.Millisecond)

	eventsAfterInterrupt := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventsAfterInterrupt, "interrupted")

	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-2", Text: "Actually, it will rain later this evening."}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-2"}))

	collector.WaitForTTSEnd(t, 15*time.Second)

	assert.NotEmpty(t, collector.AudioPackets(), "second utterance should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet for ctx-2")
	phase3Events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, phase3Events, "speaking")
	assert.Contains(t, phase3Events, "completed")
	t.Logf("phase3: events=%v audio_packets=%d", phase3Events, len(collector.AudioPackets()))
}

// TestSmallestTTSFlow_RapidDeltasDone verifies:
//
//	init → delta × N (rapid fire) → done → audio+end
func TestSmallestTTSFlow_RapidDeltasDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := smallest.NewSmallestTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	words := []string{"Hello", " there,", " how", " are", " you", " doing", " today?"}
	for _, w := range words {
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: "ctx-rapid", Text: w}))
	}
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
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

// TestSmallestTTSVoiceModelMismatch verifies that an incompatible voice and model
// produce a terminal TextToSpeechErrorPacket instead of waiting for completion.
func TestSmallestTTSVoiceModelMismatch(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// meher is a lightning_v3.1_pro-only voice; pairing it with lightning_v3.1
	// is a real user-reachable mismatch since speak.voice.id accepts free text.
	opts := testutil.BuildOptions(map[string]interface{}{
		"speak.voice.id": "meher",
		"speak.model":    "lightning_v3.1",
	})

	tts, err := smallest.NewSmallestTextToSpeech(ctx, logger, testutil.BuildCredential(pcfg.Credential), collector.OnPacket, opts)
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "smallest-tts-mismatch",
		Text:      "This pairing should be rejected by the server.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "smallest-tts-mismatch",
	}))

	var errPkt *internal_type.TextToSpeechErrorPacket
	collector.WaitFor(t, 10*time.Second, "TextToSpeechErrorPacket", func() bool {
		for _, p := range collector.GetPackets() {
			if e, ok := p.(internal_type.TextToSpeechErrorPacket); ok {
				errPkt = &e
				return true
			}
		}
		return false
	})

	require.NotNil(t, errPkt, "should surface a TextToSpeechErrorPacket instead of hanging")
	assert.Contains(t, errPkt.Error.Error(), "Invalid Voice ID")
	assert.Empty(t, collector.AudioPackets(), "mismatched pairing should not produce audio")
	assert.Empty(t, collector.EndPackets(), "mismatched pairing has no normal completion")
	t.Logf("mismatch error surfaced: %v", errPkt.Error)
}

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
	assertTTSInitMetricCount(t, collector, 1)
}

func assertTTSInitMetricCount(t *testing.T, collector *testutil.PacketCollector, expectedCount int) {
	t.Helper()
	count := 0
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == observability.MetricTTSInitLatencyMs {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "%s should be non-negative", observability.MetricTTSInitLatencyMs)
				count++
			}
		}
	}
	assert.Equal(t, expectedCount, count, "unexpected count of %s metrics", observability.MetricTTSInitLatencyMs)
}
