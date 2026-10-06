//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	google "github.com/rapidaai/api/assistant-api/internal/transformer/google"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := google.NewGoogleTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "google-tts", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	assert.Empty(t, collector.MetricPackets(), "Initialize does not open a synthesis stream")
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "google-tts-lifecycle",
		Text:      "Hello world, this is a Google Speech test.",
	}))
	assertTTSInitMetric(t, collector)
	assert.Empty(t, collector.EndPackets(), "text alone must not end the message")
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "google-tts-lifecycle",
	}))
	collector.WaitForAudio(t, 20*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
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

func TestGoogleTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := google.NewGoogleTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
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
			ContextID: "google-tts-streaming",
			Text:      chunk,
		}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "google-tts-streaming",
	}))
	collector.WaitForAudio(t, 30*time.Second)
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

func TestGoogleTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	audioActive := make(chan struct{}, 1)
	interruptDone := make(chan struct{})
	onPacket := func(packets ...internal_type.Packet) error {
		if err := collector.OnPacket(packets...); err != nil {
			return err
		}
		for _, packet := range packets {
			if audio, ok := packet.(internal_type.TextToSpeechAudioPacket); ok && audio.ContextID == "google-tts-interrupt" {
				audioActive <- struct{}{}
				select {
				case <-interruptDone:
				case <-ctx.Done():
				}
			}
		}
		return nil
	}
	tts, err := google.NewGoogleTextToSpeech(ctx, logger, cred, onPacket, opts)
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer func() {
		cancel()
		tts.Close(ctx)
	}()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "google-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "google-tts-interrupt",
	}))
	select {
	case <-audioActive:
	case <-ctx.Done():
		t.Fatal("context cancelled before audio started")
	}
	metricsBeforeInterrupt := collector.MetricPackets()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "google-tts-interrupt",
	}))
	close(interruptDone)
	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")
	assert.Empty(t, collector.EndPackets(), "an interrupted message must not complete")
	assert.Equal(t, metricsBeforeInterrupt, collector.MetricPackets(), "interrupt must not open another stream")

	// The next message's Text opens the replacement stream.
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "google-tts-recovered", Text: "Speech resumes after interruption.",
	}))
	assertTTSInitMetricCountAtLeast(t, collector, 2)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "google-tts-recovered",
	}))
	collector.WaitForTTSEnd(t, 10*time.Second)
	for _, end := range collector.EndPackets() {
		assert.Equal(t, "google-tts-recovered", end.ContextID)
	}
	t.Logf("event_sequence=%v", eventTypes)
}

func TestGoogleTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := google.NewGoogleTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: fmt.Sprintf("google-tts-reconnect-%d", attempt),
			Text:      "Reconnect test.",
		}))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
			ContextID: fmt.Sprintf("google-tts-reconnect-%d", attempt),
		}))

		collector.WaitForAudio(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

func TestGoogleTTSFlow_DeltaInterruptDeltaDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := google.NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-1",
		Text:      "The weather today is sunny with clear skies.",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	t.Logf("phase1: audio_packets=%d", len(collector.AudioPackets()))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-1",
	}))
	eventsAfterInterrupt := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventsAfterInterrupt, "interrupted")
	t.Logf("after_interrupt: events=%v", eventsAfterInterrupt)
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-2",
		Text:      "Actually, it will rain later this evening.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-2",
	}))

	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "second utterance should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet for ctx-2")
	phase3Events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, phase3Events, "speaking")
	assert.Contains(t, phase3Events, "completed")
	t.Logf("phase3: events=%v audio_packets=%d", phase3Events, len(collector.AudioPackets()))
}

func TestGoogleTTSFlow_DeltaDoneInterrupt(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := google.NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-late",
		Text:      "Short sentence.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-late",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	assert.NotEmpty(t, collector.EndPackets(), "should have completed before interrupt")
	t.Logf("before_interrupt: events=%v", ttsEventTypes(collector.EventPackets()))
	err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-late",
	})
	require.NoError(t, err, "late interrupt should not error")

	allEvents := ttsEventTypes(collector.EventPackets())
	assert.NotContains(t, allEvents, "interrupted", "a completed message ignores late interruption")
	t.Logf("after_late_interrupt: events=%v", allEvents)
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-after-late",
		Text:      "I can still speak after a late interrupt.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-after-late",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after late interrupt")
}

func TestGoogleTTSFlow_InterruptBeforeDelta(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := google.NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-early",
	})
	require.NoError(t, err, "early interrupt should not error")
	earlyEvents := ttsEventTypes(collector.EventPackets())
	assert.NotContains(t, earlyEvents, "interrupted",
		"interrupt on empty context should be a no-op")
	t.Logf("after_early_interrupt: events=%v", earlyEvents)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-after-early",
		Text:      "This should work fine after an early interrupt.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-after-early",
	}))

	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)

	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio")
	assert.NotEmpty(t, collector.EndPackets(), "should emit end packet")
	t.Logf("final: events=%v audio=%d", ttsEventTypes(collector.EventPackets()), len(collector.AudioPackets()))
}

func TestGoogleTTSFlow_MultipleInterrupts(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tts, err := google.NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "round-1", Text: "First attempt at speaking."}))
	collector.WaitForAudio(t, 15*time.Second)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "round-1"}))
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "round-2", Text: "Second attempt, interrupted again."}))
	collector.WaitForAudio(t, 15*time.Second)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "round-2"}))
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "round-3", Text: "Third time is the charm."}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
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

func TestGoogleTTSFlow_DeltaInterruptNoComplete(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := google.NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-no-done",
		Text:      "This sentence will never be completed because the user interrupts.",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "ctx-no-done",
	}))
	events := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, events, "interrupted")
	assert.Empty(t, collector.EndPackets(), "interruption without Done must not complete")
	t.Logf("events=%v", events)
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-recover",
		Text:      "Recovered after interrupted delta.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-recover",
	}))
	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "should produce audio after recovery")
}

func TestGoogleTTSFlow_RapidDeltasDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := google.NewGoogleTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	words := []string{"Hello", " there,", " how", " are", " you", " doing", " today?"}
	for _, w := range words {
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: "ctx-rapid",
			Text:      w,
		}))
	}
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "ctx-rapid",
	}))

	collector.WaitForAudio(t, 20*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
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

func ttsEventTypes(events []internal_type.ObservabilityEventRecordPacket) []string {
	var out []string
	for _, ev := range events {
		if ev.Record.Component.String() == "tts" {
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
