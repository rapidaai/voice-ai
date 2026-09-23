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
	azure "github.com/rapidaai/api/assistant-api/internal/transformer/azure"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureTTSLifecycle(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	tts, err := azure.NewAzureTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
	require.NoError(t, err)
	require.NotNil(t, tts)
	assert.Equal(t, "azure-tts", tts.Name())

	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	assertMetricValue(t, collector, observability.MetricTTSInitLatencyMs, 0)
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "azure-tts-lifecycle",
		Text:      "Hello world, this is an Azure test.",
	}))
	assert.Empty(t, collector.EndPackets(), "completed text must wait for Done before ending the message")
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "azure-tts-lifecycle",
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
	assert.Contains(t, eventTypes, "completed")
	t.Logf("tts_event_sequence=%v", eventTypes)

	assertTTSLatencyMetric(t, collector)
}

func TestAzureTTSStreamingDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := azure.NewAzureTextToSpeech(ctx, logger,
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
			ContextID: "azure-tts-streaming", Text: chunk}))
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "azure-tts-streaming"}))
	collector.WaitForTTSEnd(t, 30*time.Second)
	require.NotEmpty(t, collector.AudioPackets())
	t.Logf("audio_packets=%d", len(collector.AudioPackets()))

}

func TestAzureTTSInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	audioActive := make(chan struct{}, 1)
	interruptDone := make(chan struct{})
	onPacket := func(packets ...internal_type.Packet) error {
		if err := collector.OnPacket(packets...); err != nil {
			return err
		}
		for _, packet := range packets {
			if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
				audioActive <- struct{}{}
				select {
				case <-interruptDone:
				case <-ctx.Done():
				}
			}
		}
		return nil
	}
	tts, err := azure.NewAzureTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), onPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	// Text is synchronous. Hold its audio callback until the interrupt is issued.
	producerDone := make(chan struct{})
	var producerErr error
	go func() {
		defer close(producerDone)
		producerErr = tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: "azure-tts-interrupt",
			Text:      "This sentence should be interrupted before it finishes being spoken aloud."})
		if producerErr == nil {
			producerErr = tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "azure-tts-interrupt"})
		}
	}()
	defer func() {
		cancel()
		<-producerDone
	}()
	select {
	case <-audioActive:
	case <-producerDone:
		t.Fatalf("synthesis stopped before audio: %v", producerErr)
	case <-ctx.Done():
		t.Fatal("context cancelled before audio started")
	}
	err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{
		ContextID: "azure-tts-interrupt"})
	close(interruptDone)
	require.NoError(t, err, "interruption should not error")
	select {
	case <-producerDone:
		require.NoError(t, producerErr)
	case <-ctx.Done():
		t.Fatal("context cancelled before interrupted synthesis stopped")
	}
	assert.Empty(t, collector.EndPackets(), "interrupted text must not emit End even after Done")

	eventTypes := ttsEventTypes(collector.EventPackets())
	assert.Contains(t, eventTypes, "interrupted")
	t.Logf("event_sequence=%v", eventTypes)
}

func TestAzureTTSReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	cred := testutil.BuildCredential(pcfg.Credential)
	opts := testutil.BuildOptions(pcfg.Options)

	for attempt := 0; attempt < 2; attempt++ {
		collector := testutil.NewPacketCollector()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		tts, err := azure.NewAzureTextToSpeech(ctx, logger, cred, collector.OnPacket, opts)
		require.NoError(t, err, "attempt %d", attempt)
		require.NoError(t, tts.Initialize(), "attempt %d", attempt)

		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: fmt.Sprintf("azure-tts-reconnect-%d", attempt), Text: "Reconnect test."}))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
			ContextID: fmt.Sprintf("azure-tts-reconnect-%d", attempt)}))
		collector.WaitForTTSEnd(t, 20*time.Second)
		assert.NotEmpty(t, collector.AudioPackets(), "attempt %d: should produce audio", attempt)
		t.Logf("attempt=%d audio_packets=%d", attempt, len(collector.AudioPackets()))

		tts.Close(ctx)
		cancel()
	}
}

func TestAzureTTSFlow_DeltaInterruptDelta(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	audioActive := make(chan struct{}, 1)
	interruptDone := make(chan struct{})
	onPacket := func(packets ...internal_type.Packet) error {
		if err := collector.OnPacket(packets...); err != nil {
			return err
		}
		for _, packet := range packets {
			if audio, ok := packet.(internal_type.TextToSpeechAudioPacket); ok && audio.ContextID == "ctx-1" {
				audioActive <- struct{}{}
				select {
				case <-interruptDone:
				case <-ctx.Done():
				}
			}
		}
		return nil
	}
	tts, err := azure.NewAzureTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), onPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)
	producerDone := make(chan struct{})
	var producerErr error
	go func() {
		defer close(producerDone)
		producerErr = tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: "ctx-1", Text: "The weather today is sunny with clear skies."})
		if producerErr == nil {
			producerErr = tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "ctx-1"})
		}
	}()
	defer func() {
		cancel()
		<-producerDone
	}()
	select {
	case <-audioActive:
	case <-producerDone:
		t.Fatalf("synthesis stopped before audio: %v", producerErr)
	case <-ctx.Done():
		t.Fatal("context cancelled before audio started")
	}
	t.Logf("phase1: audio_packets=%d", len(collector.AudioPackets()))
	err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "ctx-1"})
	close(interruptDone)
	require.NoError(t, err)
	select {
	case <-producerDone:
		require.NoError(t, producerErr)
	case <-ctx.Done():
		t.Fatal("context cancelled before interrupted synthesis stopped")
	}
	assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")
	assert.Empty(t, collector.EndPackets(), "interrupted message must not complete")
	collector.Clear()
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "ctx-2", Text: "Actually, it will rain later this evening."}))
	assert.Empty(t, collector.EndPackets(), "completed text must wait for Done")
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "ctx-2"}))

	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "second utterance should produce audio")
	t.Logf("phase3: audio_packets=%d", len(collector.AudioPackets()))
}

func TestAzureTTSFlow_RapidDeltas(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := azure.NewAzureTextToSpeech(ctx, logger,
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

	collector.WaitForAudio(t, 20*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
	assert.NotEmpty(t, collector.AudioPackets())
	t.Logf("words=%d audio_packets=%d", len(words), len(collector.AudioPackets()))
}

func TestAzureTTSFlow_MultipleInterrupts(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.TTSProvider(t, "azure-speech-service")
	logger := testutil.NewTestLogger()
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	audioActive := make(chan struct{}, 1)
	interruptDone := make(chan struct{})
	onPacket := func(packets ...internal_type.Packet) error {
		if err := collector.OnPacket(packets...); err != nil {
			return err
		}
		for _, packet := range packets {
			if audio, ok := packet.(internal_type.TextToSpeechAudioPacket); ok && audio.ContextID != "round-3" {
				audioActive <- struct{}{}
				select {
				case <-interruptDone:
				case <-ctx.Done():
				}
			}
		}
		return nil
	}
	tts, err := azure.NewAzureTextToSpeech(ctx, logger,
		testutil.BuildCredential(pcfg.Credential), onPacket,
		testutil.BuildOptions(pcfg.Options))
	require.NoError(t, err)
	require.NoError(t, tts.Initialize())
	defer tts.Close(ctx)

	for round := 1; round <= 2; round++ {
		contextID := fmt.Sprintf("round-%d", round)
		producerDone := make(chan struct{})
		var producerErr error
		go func() {
			defer close(producerDone)
			producerErr = tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
				ContextID: contextID, Text: fmt.Sprintf("Attempt %d at speaking.", round)})
			if producerErr == nil {
				producerErr = tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: contextID})
			}
		}()
		defer func() {
			cancel()
			<-producerDone
		}()
		select {
		case <-audioActive:
		case <-producerDone:
			t.Fatalf("round %d: synthesis stopped before audio: %v", round, producerErr)
		case <-ctx.Done():
			t.Fatal("context cancelled before audio started")
		}
		err = tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: contextID})
		close(interruptDone)
		require.NoError(t, err)
		select {
		case <-producerDone:
			require.NoError(t, producerErr)
		case <-ctx.Done():
			t.Fatal("context cancelled before interrupted synthesis stopped")
		}
		assert.Contains(t, ttsEventTypes(collector.EventPackets()), "interrupted")
		assert.Empty(t, collector.EndPackets(), "interrupted round must not complete")
		collector.Clear()
		interruptDone = make(chan struct{})
	}
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "round-3", Text: "Third time is the charm."}))
	assert.Empty(t, collector.EndPackets(), "completed text must wait for Done")
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "round-3"}))
	collector.WaitForAudio(t, 15*time.Second)
	collector.WaitForTTSEnd(t, 10*time.Second)
	assert.NotEmpty(t, collector.AudioPackets(), "final round should produce audio")
	t.Logf("round3: audio_packets=%d events=%v",
		len(collector.AudioPackets()), ttsEventTypes(collector.EventPackets()))
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
