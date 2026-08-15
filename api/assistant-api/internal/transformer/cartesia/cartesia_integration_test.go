//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Cartesia integration tests — focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.

package internal_transformer_cartesia

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Cartesia TTS Integration Tests
// ---------------------------------------------------------------------------

func TestCartesiaTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := NewCartesiaTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "cartesia-tts", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	assertMetricValue(t, collector, "tts_init_ms", 0)

	// Send delta + done (done sends continue:false, flush:true → Cartesia responds with done:true)
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "cartesia-tts-lifecycle",
		Text:      "Hello world, this is a Cartesia test.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
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

	tts, err := NewCartesiaTextToSpeech(ctx, logger,
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
			ContextID: "cartesia-tts-streaming", Text: chunk}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
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

	tts, err := NewCartesiaTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "cartesia-tts-interrupt",
		Text:      "This sentence should be interrupted before it finishes being spoken aloud."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "cartesia-tts-interrupt"}))

	collector.WaitForAudio(t, 15*time.Second)

	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "cartesia-tts-interrupt", Source: internal_type.InterruptionSourceVad}))
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

		tts, err := NewCartesiaTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
			ContextID: fmt.Sprintf("cartesia-tts-reconnect-%d", attempt), Text: "Reconnect test."}))
		require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
			ContextID: fmt.Sprintf("cartesia-tts-reconnect-%d", attempt)}))

		collector.WaitForTTSEnd(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		assert.NotEmpty(t, collector.EndPackets(), "attempt %d: should emit end packet", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

// ---------------------------------------------------------------------------
// Cartesia TTS Flow Combination Tests
// ---------------------------------------------------------------------------

func TestCartesiaTTSFlow_DeltaInterruptDeltaDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewCartesiaTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), collector.OnPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-1", Text: "The weather today is sunny with clear skies."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
		ContextID: "ctx-1"}))
	collector.WaitForAudio(t, 15*time.Second)

	require.NoError(t, tts.Transform(ctx, internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-1", Source: internal_type.InterruptionSourceVad}))
	time.Sleep(500 * time.Millisecond)
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")

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

func TestCartesiaTTSFlow_DeltaDoneInterrupt(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewCartesiaTextToSpeech(ctx, logger,
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

	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDeltaPacket{
		ContextID: "ctx-after-late", Text: "I can still speak after a late interrupt."}))
	require.NoError(t, tts.Transform(ctx, internal_type.LLMResponseDonePacket{
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

	tts, err := NewCartesiaTextToSpeech(ctx, logger,
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

func TestCartesiaTTSFlow_RapidDeltasDone(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := NewCartesiaTextToSpeech(ctx, logger,
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
}

// ---------------------------------------------------------------------------
// Cartesia STT Integration Tests
// ---------------------------------------------------------------------------

func TestCartesiaSTTLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(
		testutil.BuildCredential(pcfg.Credential)), WithOnPacket(
		collector.OnPacket), WithOptions(

		testutil.BuildOptions(pcfg.Options)))

	require.NoError(t, err)
	require.NotNil(t, stt)
	assert.Equal(t, "cartesia-stt", stt.Name())

	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	assertMetricValue(t, collector, observability.MetricSTTInitLatencyMs, 0)

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

	collector.WaitForAnyTranscript(t, 10*time.Second)

	transcripts := collector.TranscriptPackets()
	finals := collector.FinalTranscripts()
	t.Logf("transcripts=%d finals=%d", len(transcripts), len(finals))

	for _, tr := range transcripts {
		assert.NotEmpty(t, tr.Script, "transcript script should not be empty")
	}

	if len(finals) > 0 {
		eventTypes := sttEventTypes(collector.EventPackets())
		assert.Contains(t, eventTypes, "completed")

		interruptions := collector.InterruptionDetectedPackets()
		assert.NotEmpty(t, interruptions, "should emit interruption packets with transcripts")

		assertSTTLatencyMetric(t, collector)
	}
}

func TestCartesiaSTTAudioAcceptance(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(
		testutil.BuildCredential(pcfg.Credential)), WithOnPacket(
		collector.OnPacket), WithOptions(

		testutil.BuildOptions(pcfg.Options)))

	require.NoError(t, err)
	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	chunks := testutil.ChunkAudio(testutil.SineTonePCM(440, 1.0), testutil.FrameSize)
	for i, chunk := range chunks {
		err := stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
			ContextID: "cartesia-stt-accept", Audio: chunk})
		require.NoError(t, err, "chunk %d should be accepted", i)
	}
	t.Logf("chunks_accepted=%d", len(chunks))
}

func TestCartesiaSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(
		testutil.BuildCredential(pcfg.Credential)), WithOnPacket(
		collector.OnPacket), WithOptions(

		testutil.BuildOptions(pcfg.Options)))

	require.NoError(t, err)
	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	silence := testutil.SilentPCM(2.0)
	go testutil.FeedAudio(ctx, t, stt, silence)
	time.Sleep(4 * time.Second)

	finals := collector.FinalTranscripts()
	t.Logf("final_transcripts_from_silence=%d", len(finals))
	for _, f := range finals {
		assert.Empty(t, f.Script,
			"silence should not produce non-empty final transcripts, got: %q", f.Script)
	}
}

func TestCartesiaSTTReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "cartesia")
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

		feedDone := make(chan struct{})
		go func() {
			testutil.FeedAudio(ctx, t, stt, speech)
			close(feedDone)
		}()

		select {
		case <-feedDone:
		case <-ctx.Done():
			t.Fatalf("attempt %d: context cancelled", attempt)
		}

		assertMetricValue(t, collector, observability.MetricSTTInitLatencyMs, 0)
		t.Logf("attempt=%d transcripts=%d", attempt, len(collector.TranscriptPackets()))

		stt.Close(ctx)
		cancel()
		time.Sleep(500 * time.Millisecond)
	}
}

func TestCartesiaSTTCloseWhileStreaming(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(
		testutil.BuildCredential(pcfg.Credential)), WithOnPacket(
		collector.OnPacket), WithOptions(

		testutil.BuildOptions(pcfg.Options)))

	require.NoError(t, err)
	require.NoError(t, stt.Initialize())

	go func() {
		chunks := testutil.ChunkAudio(testutil.SineTonePCM(440, 3.0), testutil.FrameSize)
		for _, chunk := range chunks {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
				ContextID: "cartesia-stt-close-mid", Audio: chunk})
			time.Sleep(time.Duration(testutil.FrameDuration) * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	err = stt.Close(ctx)
	assert.NoError(t, err, "closing STT mid-stream should not error")

	events := collector.EventPackets()
	require.NotEmpty(t, events)
	assert.Contains(t, sttEventTypes(events), "closed")
}

func TestCartesiaSTTTranscriptContent(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "cartesia")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(
		testutil.BuildCredential(pcfg.Credential)), WithOnPacket(
		collector.OnPacket), WithOptions(

		testutil.BuildOptions(pcfg.Options)))

	require.NoError(t, err)
	require.NoError(t, stt.Initialize())
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
	require.NotEmpty(t, finals)

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

func assertTTSLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	assertMetricValue(t, collector, "tts.latency_ms", 1)
}

func assertSTTLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	assertMetricValue(t, collector, "stt.latency_ms", 0)
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
