//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Cartesia integration tests focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.

package integration_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	cartesia "github.com/rapidaai/api/assistant-api/internal/transformer/cartesia"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCartesiaTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "cartesia-tts", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	assertMetricValue(t, collector, observability.MetricTTSInitLatencyMs, 0)

	// Done sends continue:false; Cartesia's done response completes the message.
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "cartesia-tts-lifecycle",
		Text:      "Hello world, this is a Cartesia test.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "cartesia-tts-lifecycle",
	}))

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

func TestCartesiaTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger,
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
			ContextID: "cartesia-tts-streaming", Text: chunk}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "cartesia-tts-streaming"}))

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

func TestCartesiaTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "cartesia-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud."}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "cartesia-tts-interrupt"}))

	collector.WaitForAudio(t, 15*time.Second)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "cartesia-tts-interrupt"}))
	time.Sleep(2 * time.Second)

	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")
	t.Logf("event_sequence=%v", eventTypes)
}

func TestCartesiaTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: fmt.Sprintf("cartesia-tts-reconnect-%d", attempt), Text: "Reconnect test."}))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
			ContextID: fmt.Sprintf("cartesia-tts-reconnect-%d", attempt)}))

		collector.WaitForTTSEnd(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		assert.NotEmpty(t, collector.EndPackets(), "attempt %d: should emit end packet", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

func TestCartesiaTTSFlow_DeltaInterruptDeltaDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger,
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

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-1"}))
	time.Sleep(500 * time.Millisecond)
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")

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
}

func TestCartesiaTTSFlow_DeltaDoneInterrupt(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-late", Text: "Short sentence."}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-late"}))
	collector.WaitForTTSEnd(t, 15*time.Second)
	assert.NotEmpty(t, collector.EndPackets())

	err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-late"})
	require.NoError(t, err, "late interrupt should not error")
	time.Sleep(1 * time.Second)
	assert.NotContains(t, ttsEventTypes(collector.EventPackets()), "interrupted", "completed synthesis ignores late interruption")

	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-after-late", Text: "I can still speak after a late interrupt."}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-after-late"}))
	collector.WaitForAudio(t, 15*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after late interrupt")
}

func TestCartesiaTTSFlow_MultipleInterrupts(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	for round := 1; round <= 2; round++ {
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: fmt.Sprintf("round-%d", round),
			Text:      fmt.Sprintf("Attempt %d at speaking.", round)}))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
			ContextID: fmt.Sprintf("round-%d", round)}))
		collector.WaitForAudio(t, 15*time.Second)
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
			ContextID: fmt.Sprintf("round-%d", round),
		}))
		time.Sleep(500 * time.Millisecond)
		collector.Clear()
	}

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "round-3", Text: "Third time is the charm."}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "round-3"}))
	collector.WaitForTTSEnd(t, 15*time.Second)

	assert.NotEmpty(t, collector.AudioPackets())
	assert.NotEmpty(t, collector.EndPackets())
	finalEvents := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, finalEvents, "speaking")
	assert.Contains(t, finalEvents, "completed")
}

func TestCartesiaTTSFlow_RapidDeltasDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := cartesia.NewCartesiaTextToSpeech(ctx, logger,
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
	assert.NotEmpty(t, collector.EndPackets())
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
	assertMetricValue(t, collector, "tts.latency_ms", 1)
}

func assertMetricValue(t *testing.T, collector *testutil.PacketCollector, metricName string, minValue int) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == metricName {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, minValue, "%s should be >= %d", metricName, minValue)
				t.Logf("%s=%d", metricName, ms)
				return
			}
		}
	}
	t.Errorf("should have %s metric", metricName)
}
