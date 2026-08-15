//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// AssemblyAI integration tests — focused on verifying the flow (connection,
// initialization, event sequence, audio I/O) rather than transcript content.
// AssemblyAI is an STT-only provider; no TTS tests.

package internal_transformer_assemblyai

import (
	"context"
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
// AssemblyAI STT Integration Tests
// ---------------------------------------------------------------------------

// TestAssemblyaiSTTLifecycle verifies the full STT flow:
// create → initialize (metric/log) → feed audio → transcripts arrive →
// event sequence includes completed when final transcripts arrive.
func TestAssemblyaiSTTLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "assemblyai")
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
	assert.Equal(t, "assemblyai-stt", stt.Name())

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

// TestAssemblyaiSTTAudioAcceptance verifies that the STT transformer accepts audio
// chunks without returning errors.
func TestAssemblyaiSTTAudioAcceptance(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "assemblyai")
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
			ContextID: "aai-stt-accept", Audio: chunk})
		require.NoError(t, err, "chunk %d should be accepted", i)
	}
	t.Logf("chunks_accepted=%d", len(chunks))
}

// TestAssemblyaiSTTSilentAudio verifies that sending silent audio does not
// produce false transcripts.
func TestAssemblyaiSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "assemblyai")
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

// TestAssemblyaiSTTReconnect verifies two sequential STT sessions work cleanly.
func TestAssemblyaiSTTReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "assemblyai")
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
		assertSTTInitMetric(t, collector)

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

		t.Logf("attempt=%d transcripts=%d", attempt, len(collector.TranscriptPackets()))

		stt.Close(ctx)
		cancel()
		time.Sleep(500 * time.Millisecond)
	}
}

// TestAssemblyaiSTTCloseWhileStreaming verifies that closing the STT transformer
// while audio is actively being fed does not panic or return unexpected errors.
func TestAssemblyaiSTTCloseWhileStreaming(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "assemblyai")
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
				ContextID: "aai-stt-close-mid", Audio: chunk})
			time.Sleep(time.Duration(testutil.FrameDuration) * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	err = stt.Close(ctx)
	assert.NoError(t, err, "closing STT mid-stream should not error")
}

// TestAssemblyaiSTTTranscriptContent verifies that real speech audio produces
// a transcript containing the expected words.
func TestAssemblyaiSTTTranscriptContent(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "assemblyai")
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
				t.Logf("%s=%d", observability.MetricSTTInitLatencyMs, ms)
				return
			}
		}
	}
	t.Errorf("should have %s metric", observability.MetricSTTInitLatencyMs)
}
