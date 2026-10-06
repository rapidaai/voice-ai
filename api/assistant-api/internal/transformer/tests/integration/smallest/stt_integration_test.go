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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	smallest "github.com/rapidaai/api/assistant-api/internal/transformer/smallest"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	stt, err := smallest.NewSpeechToText(smallest.WithContext(ctx), smallest.WithLogger(logger), smallest.WithCredential(cred), smallest.WithOnPacket(collector.OnPacket), smallest.WithOptions(opts))
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

	stt, err := smallest.NewSpeechToText(smallest.WithContext(ctx), smallest.WithLogger(logger), smallest.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), smallest.WithOnPacket(
		collector.OnPacket), smallest.WithOptions(

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

	stt, err := smallest.NewSpeechToText(smallest.WithContext(ctx), smallest.WithLogger(logger), smallest.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), smallest.WithOnPacket(
		collector.OnPacket), smallest.WithOptions(

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

		stt, err := smallest.NewSpeechToText(smallest.WithContext(ctx), smallest.WithLogger(logger), smallest.WithCredential(cred), smallest.WithOnPacket(collector.OnPacket), smallest.WithOptions(opts))
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

	stt, err := smallest.NewSpeechToText(smallest.WithContext(ctx), smallest.WithLogger(logger), smallest.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), smallest.WithOnPacket(
		collector.OnPacket), smallest.WithOptions(

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

	stt, err := smallest.NewSpeechToText(smallest.WithContext(ctx), smallest.WithLogger(logger), smallest.WithCredential(testutil.BuildCredential(pcfg.Credential)), smallest.WithOnPacket(collector.OnPacket), smallest.WithOptions(opts))
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

	stt, err := smallest.NewSpeechToText(smallest.WithContext(ctx), smallest.WithLogger(logger), smallest.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), smallest.WithOnPacket(
		collector.OnPacket), smallest.WithOptions(

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

func sttEventTypes(events []internal_type.ObservabilityEventRecordPacket) []string {
	var out []string
	for _, ev := range events {
		if ev.Record.Component.String() == "stt" {
			out = append(out, ev.Record.Attributes["type"])
		}
	}
	return out
}

func assertSTTLatencyMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == "stt.latency_ms" {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "stt.latency_ms should be non-negative")
				t.Logf("stt.latency_ms=%d", ms)
				return
			}
		}
	}
	t.Error("should have stt.latency_ms metric")
}

func assertSTTInitMetric(t *testing.T, collector *testutil.PacketCollector) {
	t.Helper()
	for _, m := range collector.MetricPackets() {
		for _, metric := range m.Record.Metrics {
			if metric.Name == observability.MetricSTTInitLatencyMs {
				ms, err := strconv.Atoi(metric.Value)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ms, 0, "%s should be non-negative", observability.MetricSTTInitLatencyMs)
				return
			}
		}
	}
	t.Errorf("should have %s metric", observability.MetricSTTInitLatencyMs)
}
