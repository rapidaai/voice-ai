//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Inworld integration tests — verify connection, event sequencing, and
// audio I/O against the real wss://api.inworld.ai TTS endpoint. Skipped
// automatically when integration_config.yaml is missing or the inworld
// provider is disabled.

package internal_transformer_inworld

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// ---------------------------------------------------------------------------
// Inworld TTS Integration Tests
// ---------------------------------------------------------------------------

// TestInworldTTSLifecycle verifies the full TTS flow:
// create → initialize (event) → transform delta+done → audio output → end packet → events in order.
func TestInworldTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "inworld")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewInworldTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "inworld-text-to-speech", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	events := collector.EventPackets()
	require.NotEmpty(t, events, "should emit initialized event")
	assert.Equal(t, "tts", events[0].Name)
	assert.Equal(t, "initialized", events[0].Data["type"])
	_, err = strconv.Atoi(events[0].Data["init_ms"])
	assert.NoError(t, err, "init_ms should be a valid integer")

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "iw-tts-lifecycle",
		Text:      "Hello world, this is an Inworld test.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "iw-tts-lifecycle",
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
	assert.Contains(t, eventTypes, "initialized")
	assert.Contains(t, eventTypes, "speaking")
	assert.Contains(t, eventTypes, "completed")
	t.Logf("tts_event_sequence=%v", eventTypes)

	assertTTSLatencyMetric(t, collector)
}

// TestInworldTTSStreamingDeltas verifies that multiple streaming delta chunks
// each trigger a speaking event and together produce audio output.
func TestInworldTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "inworld")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewInworldTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
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
			ContextID: "iw-tts-streaming",
			Text:      chunk,
		}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "iw-tts-streaming",
	}))

	collector.WaitForTTSEnd(t, 30*time.Second)

	require.NotEmpty(t, collector.AudioPackets())

	speakingCount := 0
	for _, ev := range collector.EventPackets() {
		if ev.Name == "tts" && ev.Data["type"] == "speaking" {
			speakingCount++
		}
	}
	assert.Equal(t, len(chunks), speakingCount,
		"should emit one speaking event per delta chunk")
	t.Logf("chunks=%d speaking_events=%d audio_packets=%d",
		len(chunks), speakingCount, len(collector.AudioPackets()))
}

// TestInworldTTSInterruption verifies the interruption flow:
// send delta+done → audio starts → interrupt → "interrupted" event → reconnect → second "initialized" event.
func TestInworldTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "inworld")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewInworldTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "iw-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "iw-tts-interrupt",
	}))

	collector.WaitForAudio(t, 15*time.Second)

	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "iw-tts-interrupt",
		Source:    internal_type.InterruptionSourceVad,
	}))

	time.Sleep(2 * time.Second)

	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")

	initCount := 0
	for _, typ := range eventTypes {
		if typ == "initialized" {
			initCount++
		}
	}
	assert.GreaterOrEqual(t, initCount, 2,
		"should see at least 2 initialized events (original + reconnect)")
	t.Logf("event_sequence=%v", eventTypes)
}

// TestInworldTTSReconnect verifies two sequential TTS sessions work cleanly.
func TestInworldTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "inworld")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := NewInworldTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: fmt.Sprintf("iw-tts-reconnect-%d", attempt),
			Text:      "Reconnect test.",
		}))
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
			ContextID: fmt.Sprintf("iw-tts-reconnect-%d", attempt),
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
// Helpers
// ---------------------------------------------------------------------------

func ttsEventTypes(events []internal_type.ConversationEventPacket) []string {
	var out []string
	for _, ev := range events {
		if ev.Name == "tts" {
			out = append(out, ev.Data["type"])
		}
	}
	return out
}

func assertTTSLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Metrics {
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
