//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Smallest AI integration tests — focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.

package internal_transformer_smallest

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
// Smallest TTS Integration Tests
// ---------------------------------------------------------------------------

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

	tts, err := NewSmallestTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
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

	tts, err := NewSmallestTextToSpeech(ctx, logger,
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
// send delta+done → audio starts → interrupt → "interrupted" event → reconnect → second init metric.
func TestSmallestTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := NewSmallestTextToSpeech(ctx, logger,
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

	assertTTSInitMetricCountAtLeast(t, collector, 2)
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

		tts, err := NewSmallestTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
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

// ---------------------------------------------------------------------------
// Smallest TTS Flow Combination Tests
// ---------------------------------------------------------------------------

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

	tts, err := NewSmallestTextToSpeech(ctx, logger,
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

	tts, err := NewSmallestTextToSpeech(ctx, logger,
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

// TestSmallestTTSVoiceModelMismatch verifies that pairing a voice with the
// wrong model pool (e.g. a Pro-only voice on lightning_v3.1) surfaces a
// TextToSpeechErrorPacket instead of hanging forever. Smallest sends exactly
// one status:"error" frame for a bad pairing and then holds the connection
// open without ever sending "complete" — readLoop must treat "error" as
// terminal for the turn.
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

	tts, err := NewSmallestTextToSpeech(ctx, logger, testutil.BuildCredential(pcfg.Credential), collector.OnPacket, opts)
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

// ---------------------------------------------------------------------------
// Smallest STT Integration Tests
// ---------------------------------------------------------------------------

// TestSmallestSTTLifecycle verifies the full STT flow:
// create → initialize (event) → feed audio (no errors) → transcripts arrive →
// init metric is emitted.
func TestSmallestSTTLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "smallest")
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
	assert.Equal(t, "smallest-stt", stt.Name())

	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	assertSTTInitMetric(t, collector)

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
		t.Logf("stt_event_sequence=%v", eventTypes)

		interruptions := collector.InterruptionDetectedPackets()
		assert.NotEmpty(t, interruptions, "should emit interruption packets with transcripts")

		assertSTTLatencyMetric(t, collector)
	}
}

// TestSmallestSTTAudioAcceptance verifies that the STT transformer accepts
// audio chunks without returning errors.
func TestSmallestSTTAudioAcceptance(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "smallest")
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
			ContextID: "smallest-stt-accept", Audio: chunk})
		require.NoError(t, err, "chunk %d should be accepted", i)
	}
	t.Logf("chunks_accepted=%d", len(chunks))
}

// TestSmallestSTTSilentAudio verifies that sending silent audio does not
// produce false transcripts.
func TestSmallestSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "smallest")
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

// TestSmallestSTTReconnect verifies two sequential STT sessions work cleanly.
func TestSmallestSTTReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "smallest")
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
			t.Fatalf("attempt %d: context cancelled before audio feeding completed", attempt)
		}

		assertSTTInitMetric(t, collector)
		t.Logf("attempt=%d transcripts=%d", attempt, len(collector.TranscriptPackets()))

		stt.Close(ctx)
		cancel()
		time.Sleep(500 * time.Millisecond)
	}
}

// TestSmallestSTTCloseWhileStreaming verifies that closing the STT
// transformer while audio is actively being fed does not panic or return
// unexpected errors.
func TestSmallestSTTCloseWhileStreaming(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "smallest")
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
				ContextID: "smallest-stt-close-mid", Audio: chunk})
			time.Sleep(time.Duration(testutil.FrameDuration) * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	err = stt.Close(ctx)
	assert.NoError(t, err, "closing STT mid-stream should not error")

	assertSTTInitMetric(t, collector)
}

// TestSmallestSTTTranscriptContent verifies that real speech audio produces
// a transcript containing the expected words.
// TestSmallestSTTFeatureFlags verifies that word_timestamps, sentence_timestamps,
// diarize, redact_pii, redact_pci, and format all flow correctly through the
// real NewSmallestSpeechToText -> Transform() -> packet pipeline (not just the
// connection-string builder in isolation) and that the enriched response data
// (words, utterances) surfaces as event attributes.
func TestSmallestSTTFeatureFlags(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "smallest")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := testutil.BuildOptions(map[string]interface{}{
		"listen.word_timestamps":     true,
		"listen.sentence_timestamps": true,
		"listen.diarize":             true,
		"listen.redact_pii":          true,
		"listen.redact_pci":          true,
		"listen.smart_format":        true,
	})

	stt, err := NewSpeechToText(WithContext(ctx), WithLogger(logger), WithCredential(testutil.BuildCredential(pcfg.Credential)), WithOnPacket(collector.OnPacket), WithOptions(opts))
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
	require.NotEmpty(t, finals, "should still produce a final transcript with all feature flags on")
	combined := ""
	for _, f := range finals {
		combined += " " + f.Script
	}
	lower := strings.ToLower(combined)
	assert.True(t,
		strings.Contains(lower, "hello") || strings.Contains(lower, "world"),
		"expected transcript to contain 'hello' or 'world' even with feature flags on, got: %q", combined)

	var completedAttrs map[string]string
	for _, ev := range collector.EventPackets() {
		if ev.Record.Component.String() == "stt" && ev.Record.Attributes["type"] == "completed" {
			completedAttrs = ev.Record.Attributes
			break
		}
	}
	require.NotNil(t, completedAttrs, "should emit a completed STT event")

	wordCount, err := strconv.Atoi(completedAttrs["word_timestamp_count"])
	require.NoError(t, err, "word_timestamp_count should be a valid integer, got %q", completedAttrs["word_timestamp_count"])
	assert.Greater(t, wordCount, 0, "word_timestamps=true should populate per-word timing data")

	utteranceCount, err := strconv.Atoi(completedAttrs["utterance_count"])
	require.NoError(t, err, "utterance_count should be a valid integer, got %q", completedAttrs["utterance_count"])
	assert.Greater(t, utteranceCount, 0, "sentence_timestamps=true should populate utterance data")

	t.Logf("transcript=%q word_timestamp_count=%d utterance_count=%d", strings.TrimSpace(combined), wordCount, utteranceCount)
}

func TestSmallestSTTTranscriptContent(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "smallest")
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

func assertTTSInitMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	assertTTSInitMetricCountAtLeast(t, collector, 1)
}

func assertTTSInitMetricCountAtLeast(t *testing.T, collector *testutil.PacketCollector, minimumCount int) {
	t.Helper()
	count := 0
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "tts_init_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "tts_init_ms should be non-negative")
				count++
			}
		}
	}
	assert.GreaterOrEqual(t, count, minimumCount, "should have enough tts_init_ms metrics")
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

func assertSTTInitMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "stt_init_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "stt_init_ms should be non-negative")
				return
			}
		}
	}
	t.Error("should have stt_init_ms metric")
}
