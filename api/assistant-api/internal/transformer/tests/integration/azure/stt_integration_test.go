//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	azure "github.com/rapidaai/api/assistant-api/internal/transformer/azure"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureSTTLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(logger), azure.WithCredential(cred), azure.WithOnPacket(collector.OnPacket), azure.WithOptions(opts))
	require.NoError(t, err)
	require.NotNil(t, stt)
	assert.Equal(t, "azure-stt", stt.Name())

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
		t.Logf("stt_event_sequence=%v", eventTypes)

		interruptions := collector.InterruptionDetectedPackets()
		assert.NotEmpty(t, interruptions, "should emit interruption packets with transcripts")

		assertSTTLatencyMetric(t, collector)
	}
}

func TestAzureSTTAudioAcceptance(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(logger), azure.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), azure.WithOnPacket(
		collector.OnPacket), azure.WithOptions(

		testutil.BuildOptions(pcfg.Options)))

	require.NoError(t, err)
	require.NoError(t, stt.Initialize())
	defer stt.Close(ctx)

	chunks := testutil.ChunkAudio(testutil.SineTonePCM(440, 1.0), testutil.FrameSize)
	for i, chunk := range chunks {
		err := stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
			ContextID: "azure-stt-accept", Audio: chunk})
		require.NoError(t, err, "chunk %d should be accepted", i)
	}
	t.Logf("chunks_accepted=%d", len(chunks))
}

func TestAzureSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(logger), azure.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), azure.WithOnPacket(
		collector.OnPacket), azure.WithOptions(

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

func TestAzureSTTReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

		stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(logger), azure.WithCredential(cred), azure.WithOnPacket(collector.OnPacket), azure.WithOptions(opts))
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

func TestAzureSTTCloseWhileStreaming(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(logger), azure.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), azure.WithOnPacket(
		collector.OnPacket), azure.WithOptions(

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
				ContextID: "azure-stt-close-mid", Audio: chunk})
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

func TestAzureSTTTranscriptContent(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.STTProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stt, err := azure.NewSpeechToText(azure.WithContext(ctx), azure.WithLogger(logger), azure.WithCredential(
		testutil.BuildCredential(pcfg.Credential)), azure.WithOnPacket(
		collector.OnPacket), azure.WithOptions(

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
	assertMetricValue(t, collector, "stt.latency_ms", 0)
}
