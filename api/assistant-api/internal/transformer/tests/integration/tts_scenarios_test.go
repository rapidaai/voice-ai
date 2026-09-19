// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ttsTestFactory keeps provider setup outside the shared behavioral scenarios.
type ttsTestFactory func(context.Context, func(...internal_type.Packet) error) (internal_type.TextToSpeechTransformer, error)

// testTTSBasic verifies text and Done produce audio followed by completion.
func testTTSBasic(t *testing.T, newTransformer ttsTestFactory) {
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tts, err := newTransformer(ctx, collector.OnPacket)
	require.NoError(t, err, "factory should succeed")
	require.NotNil(t, tts)

	t.Cleanup(func() { cancel(); assert.NoError(t, tts.Close(ctx)) })
	err = tts.Initialize()
	require.NoError(t, err, "Initialize should succeed")

	err = tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "test-tts-basic",
		Text:      "Hello world, this is a test.",
	})
	require.NoError(t, err, "Transform delta should succeed")

	err = tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "test-tts-basic",
	})
	require.NoError(t, err, "Transform done should succeed")

	collector.WaitForTTSEnd(t, 20*time.Second)

	audioPackets := collector.AudioPackets()
	assert.NotEmpty(t, audioPackets, "should receive at least one audio packet")

	totalBytes := 0
	for _, ap := range audioPackets {
		assert.NotEmpty(t, ap.AudioChunk, "audio chunk should not be empty")
		totalBytes += len(ap.AudioChunk)
	}
	assert.Greater(t, totalBytes, 0, "total audio bytes should be > 0")

	endPackets := collector.EndPackets()
	assert.NotEmpty(t, endPackets, "should receive TTS end packet")
}

// testTTSMetricsAndEvents verifies latency and completion belong to each synthesized message.
func testTTSMetricsAndEvents(t *testing.T, newTransformer ttsTestFactory) {
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tts, err := newTransformer(ctx, collector.OnPacket)
	require.NoError(t, err)
	t.Cleanup(func() {
		cancel()
		if tts != nil {
			assert.NoError(t, tts.Close(ctx))
		}
	})
	providerName := tts.Name()
	require.NoError(t, tts.Initialize())
	messageIDs := []string{"test-tts-metrics-first", "test-tts-metrics-second"}
	for _, contextID := range messageIDs {
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: contextID, Text: "Testing metrics and events.",
		}))
		require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		collector.WaitFor(t, 20*time.Second, "message completion", func() bool {
			for _, packet := range collector.EndPackets() {
				if packet.ContextID == contextID {
					return true
				}
			}
			return false
		})
	}
	cancel()
	require.NoError(t, tts.Close(ctx))
	tts = nil

	latencyMetrics := make(map[string]int)
	completionEvents := make(map[string]int)
	endPackets := make(map[string]int)
	audioPackets := make(map[string]int)
	for _, packet := range collector.GetPackets() {
		switch packet := packet.(type) {
		case internal_type.ObservabilityMetricRecordPacket:
			for _, metric := range packet.Record.Metrics {
				if metric.GetName() != observability.MetricTTSLatencyMs {
					continue
				}
				assert.Contains(t, messageIDs, packet.ContextID)
				assert.Equal(t, internal_type.ObservabilityRecordScopeAssistantMessage, packet.Scope)
				assert.Equal(t, observability.AttributeValue(providerName), packet.Record.Attributes["provider"])
				latency, err := strconv.ParseInt(metric.GetValue(), 10, 64)
				assert.NoError(t, err, "latency must be an integer number of milliseconds")
				assert.GreaterOrEqual(t, latency, int64(0))
				latencyMetrics[packet.ContextID]++
			}
		case internal_type.ObservabilityEventRecordPacket:
			if packet.Record.Event == observability.TTSCompleted {
				assert.Contains(t, messageIDs, packet.ContextID)
				assert.Equal(t, internal_type.ObservabilityRecordScopeAssistantMessage, packet.Scope)
				assert.Equal(t, observability.ComponentTTS, packet.Record.Component)
				completionEvents[packet.ContextID]++
			}
		case internal_type.TextToSpeechAudioPacket:
			assert.Contains(t, messageIDs, packet.ContextID)
			assert.NotEmpty(t, packet.AudioChunk)
			audioPackets[packet.ContextID]++
		case internal_type.TextToSpeechEndPacket:
			assert.Contains(t, messageIDs, packet.ContextID)
			endPackets[packet.ContextID]++
		case internal_type.TextToSpeechErrorPacket:
			t.Errorf("unexpected TTS failure for %s: %v", packet.ContextID, packet.Error)
		}
	}
	for _, contextID := range messageIDs {
		assert.Equal(t, 1, latencyMetrics[contextID], "latency metrics for %s", contextID)
		assert.Equal(t, 1, completionEvents[contextID], "completion events for %s", contextID)
		assert.Equal(t, 1, endPackets[contextID], "end packets for %s", contextID)
		assert.Positive(t, audioPackets[contextID], "audio packets for %s", contextID)
	}
}

// testTTSMultiChunk tests sending multiple delta packets followed by a done packet,
// simulating a streaming LLM response.
func testTTSMultiChunk(t *testing.T, newTransformer ttsTestFactory) {
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tts, err := newTransformer(ctx, collector.OnPacket)
	require.NoError(t, err)
	t.Cleanup(func() { cancel(); assert.NoError(t, tts.Close(ctx)) })
	require.NoError(t, tts.Initialize())

	// Simulate streaming LLM deltas
	chunks := []string{
		"The quick brown fox ",
		"jumps over ",
		"the lazy dog. ",
		"This is a longer sentence to ensure ",
		"the provider can handle multi-chunk input.",
	}
	for _, chunk := range chunks {
		err = tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: "test-tts-multichunk",
			Text:      chunk,
		})
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond) // simulate streaming delay
	}

	err = tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
		ContextID: "test-tts-multichunk",
	})
	require.NoError(t, err)

	collector.WaitForTTSEnd(t, 30*time.Second)

	audioPackets := collector.AudioPackets()
	assert.NotEmpty(t, audioPackets, "should receive audio from multi-chunk input")

	totalBytes := 0
	for _, ap := range audioPackets {
		totalBytes += len(ap.AudioChunk)
	}
	t.Logf("provider=%s chunks_sent=%d audio_packets=%d total_bytes=%d",
		tts.Name(), len(chunks), len(audioPackets), totalBytes)
}

// testTTSInterruption interrupts active output and verifies the next message can finish.
func testTTSInterruption(t *testing.T, newTransformer ttsTestFactory) {
	collector := testutil.NewPacketCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	audioStarted, releaseAudio := make(chan struct{}), make(chan struct{})
	var firstAudio sync.Once
	tts, err := newTransformer(ctx, func(packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if err := collector.OnPacket(packet); err != nil {
				return err
			}
			if audio, ok := packet.(internal_type.TextToSpeechAudioPacket); ok && audio.ContextID == "test-tts-interrupt" {
				// Hold active synthesis so interruption cannot race an already completed message.
				firstAudio.Do(func() {
					close(audioStarted)
					select {
					case <-releaseAudio:
					case <-ctx.Done():
					}
				})
			}
		}
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		cancel()
		if tts != nil {
			assert.NoError(t, tts.Close(ctx))
		}
	})
	require.NoError(t, tts.Initialize())

	producerDone := make(chan struct{})
	var producerError error
	// Text may synchronously deliver audio; Done starts synthesis for buffered providers.
	go func() {
		defer close(producerDone)
		producerError = tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
			ContextID: "test-tts-interrupt",
			Text:      "This sentence should be interrupted while audio is still being generated for the current message.",
		})
		if producerError == nil {
			producerError = tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "test-tts-interrupt"})
		}
	}()
	t.Cleanup(func() { cancel(); <-producerDone })
	select {
	case <-audioStarted:
	case <-producerDone:
		require.NoError(t, producerError)
	case <-ctx.Done():
		t.Fatalf("waiting for TTS input or audio: %v", ctx.Err())
	}
	select {
	case <-audioStarted:
	case <-ctx.Done():
		t.Fatalf("waiting for active TTS audio: %v", ctx.Err())
	}
	require.Empty(t, collector.EndPackets(), "synthesis must still be active")
	audioCount := len(collector.AudioPackets())
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: "test-tts-interrupt"}))
	close(releaseAudio)
	<-producerDone
	if producerError != nil {
		require.ErrorIs(t, producerError, context.Canceled)
	}

	assert.Never(t, func() bool {
		return len(collector.AudioPackets()) != audioCount || len(collector.EndPackets()) != 0
	}, 500*time.Millisecond, 10*time.Millisecond, "interrupted synthesis must not emit more audio or completion")

	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
		ContextID: "test-tts-after-interrupt", Text: "This is the next message after interruption.",
	}))
	require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "test-tts-after-interrupt"}))
	collector.WaitFor(t, 20*time.Second, "next-message completion", func() bool {
		for _, packet := range collector.EndPackets() {
			if packet.ContextID == "test-tts-after-interrupt" {
				return true
			}
		}
		return false
	})
	cancel()
	require.NoError(t, tts.Close(ctx))
	tts = nil
	for _, packet := range collector.AudioPackets()[audioCount:] {
		assert.Equal(t, "test-tts-after-interrupt", packet.ContextID)
	}
	assert.Greater(t, len(collector.AudioPackets()), audioCount, "next message must produce audio")
	for _, packet := range collector.EndPackets() {
		assert.Equal(t, "test-tts-after-interrupt", packet.ContextID)
	}
	for _, packet := range collector.GetPackets() {
		if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			t.Errorf("unexpected TTS failure for %s: %v", failure.ContextID, failure.Error)
		}
	}
}

// testTTSNewSession verifies synthesis after closing the previous transformer instance.
func testTTSNewSession(t *testing.T, newTransformer ttsTestFactory) {
	for attempt := 0; attempt < 2; attempt++ {
		t.Run(strconv.Itoa(attempt), func(t *testing.T) {
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			tts, err := newTransformer(ctx, collector.OnPacket)
			require.NoError(t, err, "factory should succeed")
			t.Cleanup(func() { cancel(); assert.NoError(t, tts.Close(ctx)) })
			require.NoError(t, tts.Initialize(), "Initialize should succeed")

			require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{
				ContextID: "test-tts-new-session", Text: "New session test.",
			}))
			require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{
				ContextID: "test-tts-new-session",
			}))

			collector.WaitForTTSEnd(t, 20*time.Second)
			assert.NotEmpty(t, collector.AudioPackets(), "should produce audio")
		})
	}
}
