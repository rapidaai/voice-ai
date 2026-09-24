//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	google "github.com/rapidaai/api/assistant-api/internal/transformer/google"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	stt, err := google.NewSpeechToText(google.WithContext(ctx), google.WithLogger(logger), google.WithCredential(cred), google.WithOnPacket(collector.OnPacket), google.WithOptions(opts))
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

func TestGoogleSTTAudioAcceptance(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := google.NewSpeechToText(google.WithContext(ctx), google.WithLogger(logger), google.WithCredential(cred), google.WithOnPacket(collector.OnPacket), google.WithOptions(opts))
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

func TestGoogleSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := google.NewSpeechToText(google.WithContext(ctx), google.WithLogger(logger), google.WithCredential(cred), google.WithOnPacket(collector.OnPacket), google.WithOptions(opts))
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

		stt, err := google.NewSpeechToText(google.WithContext(ctx), google.WithLogger(logger), google.WithCredential(cred), google.WithOnPacket(collector.OnPacket), google.WithOptions(opts))
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

func TestGoogleSTTCloseWhileStreaming(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "google-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := google.NewSpeechToText(google.WithContext(ctx), google.WithLogger(logger), google.WithCredential(cred), google.WithOnPacket(collector.OnPacket), google.WithOptions(opts))
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

	stt, err := google.NewSpeechToText(google.WithContext(ctx), google.WithLogger(logger), google.WithCredential(cred), google.WithOnPacket(collector.OnPacket), google.WithOptions(opts))
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

func sttEventTypes(events []internal_type.ObservabilityEventRecordPacket) []string {
	var out []string
	for _, ev := range events {
		if ev.Record.Component.String() == "stt" {
			out = append(out, ev.Record.Attributes["type"])
		}
	}
	return out
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
