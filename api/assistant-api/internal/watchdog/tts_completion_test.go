// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package watchdog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func TestEstimateSpeechDuration_UsesWordCountAndWordsPerMinute(t *testing.T) {
	assert.Equal(t, 30*time.Second, EstimateSpeechDuration("hello world", 4))
	assert.Equal(t, 500*time.Millisecond, EstimateSpeechDuration("hello", 120))
	assert.Equal(t, time.Duration(0), EstimateSpeechDuration("  ...   ", 120))
	assert.Equal(t, 500*time.Millisecond, EstimateSpeechDuration("hello", 0))
}

func TestTTSCompletionWatchdog_StartFromTextUsesMinimumTimeoutAndGracePeriod(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithWordsPerMinute(120),
		WithMinimumTimeout(40*time.Millisecond),
		WithGracePeriod(10*time.Millisecond),
	)
	<-pushedPackets

	require.True(t, ttsCompletionWatchdog.StartFromText("ctx-greeting", ""))

	expiredLogPacket := <-pushedPackets
	observabilityLogPacket, ok := expiredLogPacket.(internal_type.ObservabilityLogRecordPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-greeting", observabilityLogPacket.ContextID)
	assert.Equal(t, "tts-completion-watchdog: deadline expired", observabilityLogPacket.Record.Message)
	assert.Equal(t, "40", observabilityLogPacket.Record.Attributes["estimated_audio_duration_ms"])

	errorPacket := <-pushedPackets
	textToSpeechErrorPacket, ok := errorPacket.(internal_type.TextToSpeechErrorPacket)
	require.True(t, ok)
	assert.Equal(t, "ctx-greeting", textToSpeechErrorPacket.ContextID)
	assert.Equal(t, internal_type.TTSNetworkTimeout, textToSpeechErrorPacket.Type)
	assert.Contains(t, textToSpeechErrorPacket.ErrMessage(), "deadline expired")
}

func TestTTSCompletionWatchdog_ExpiresWhenDeadlinePasses(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithMinimumTimeout(20*time.Millisecond),
		WithGracePeriod(10*time.Millisecond),
	)
	<-pushedPackets

	require.True(t, ttsCompletionWatchdog.Start("ctx-expire", 20*time.Millisecond))

	select {
	case packet := <-pushedPackets:
		observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-expire", observabilityLogPacket.ContextID)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("tts completion watchdog did not expire")
	}

	select {
	case packet := <-pushedPackets:
		textToSpeechErrorPacket, ok := packet.(internal_type.TextToSpeechErrorPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-expire", textToSpeechErrorPacket.ContextID)
		assert.Equal(t, internal_type.TTSNetworkTimeout, textToSpeechErrorPacket.Type)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("tts completion watchdog did not push text to speech error")
	}

	assert.False(t, ttsCompletionWatchdog.Complete("ctx-expire"))
}

func TestTTSCompletionWatchdog_CompleteCancelsExpiration(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithMinimumTimeout(30*time.Millisecond),
		WithGracePeriod(10*time.Millisecond),
	)
	<-pushedPackets

	require.True(t, ttsCompletionWatchdog.Start("ctx-complete", 30*time.Millisecond))
	require.True(t, ttsCompletionWatchdog.Complete("ctx-complete"))
	require.False(t, ttsCompletionWatchdog.Complete("ctx-complete"))

	select {
	case packet := <-pushedPackets:
		t.Fatalf("tts completion watchdog pushed packet after completion: %+v", packet)
	case <-time.After(90 * time.Millisecond):
	}
}

func TestTTSCompletionWatchdog_CompleteIgnoresDifferentContext(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithMinimumTimeout(20*time.Millisecond),
		WithGracePeriod(10*time.Millisecond),
	)
	<-pushedPackets

	require.True(t, ttsCompletionWatchdog.Start("ctx-active", 20*time.Millisecond))
	require.False(t, ttsCompletionWatchdog.Complete("ctx-other"))

	select {
	case packet := <-pushedPackets:
		observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-active", observabilityLogPacket.ContextID)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("tts completion watchdog did not expire after mismatched completion")
	}
}

func TestTTSCompletionWatchdog_ExtendDelaysExpirationAndTracksAudioDuration(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithMinimumTimeout(40*time.Millisecond),
		WithGracePeriod(10*time.Millisecond),
	)
	<-pushedPackets

	require.True(t, ttsCompletionWatchdog.Start("ctx-audio", 40*time.Millisecond))
	time.Sleep(15 * time.Millisecond)

	require.True(t, ttsCompletionWatchdog.Extend("ctx-audio", 80*time.Millisecond))

	select {
	case packet := <-pushedPackets:
		t.Fatalf("tts completion watchdog pushed packet before extended deadline: %+v", packet)
	case <-time.After(60 * time.Millisecond):
	}

	select {
	case packet := <-pushedPackets:
		observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-audio", observabilityLogPacket.ContextID)
		assert.Equal(t, "80", observabilityLogPacket.Record.Attributes["observed_audio_duration_ms"])
	case <-time.After(250 * time.Millisecond):
		t.Fatal("tts completion watchdog did not expire after extended deadline")
	}
}

func TestTTSCompletionWatchdog_ExtendUsesObservedAudioInsteadOfTextEstimate(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithMinimumTimeout(20*time.Millisecond),
		WithGracePeriod(20*time.Millisecond),
	)
	<-pushedPackets

	startedAt := time.Now()
	require.True(t, ttsCompletionWatchdog.Start("ctx-estimate", 220*time.Millisecond))
	time.Sleep(20 * time.Millisecond)

	require.True(t, ttsCompletionWatchdog.Extend("ctx-estimate", 120*time.Millisecond))

	select {
	case packet := <-pushedPackets:
		observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-estimate", observabilityLogPacket.ContextID)
		assert.Equal(t, "120", observabilityLogPacket.Record.Attributes["observed_audio_duration_ms"])
	case <-time.After(time.Until(startedAt.Add(300 * time.Millisecond))):
		t.Fatal("tts completion watchdog kept text-estimated deadline after observed audio")
	}
}

func TestTTSCompletionWatchdog_ExtendIgnoresInactiveAndDifferentContext(t *testing.T) {
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithMinimumTimeout(30*time.Millisecond),
		WithGracePeriod(10*time.Millisecond),
	)

	require.False(t, ttsCompletionWatchdog.Extend("ctx-inactive", time.Second))

	require.True(t, ttsCompletionWatchdog.Start("ctx-active", 30*time.Millisecond))
	require.False(t, ttsCompletionWatchdog.Extend("ctx-other", time.Second))
	require.True(t, ttsCompletionWatchdog.Complete("ctx-active"))
}

func TestTTSCompletionWatchdog_StartReplacesPreviousContext(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	ttsCompletionWatchdog := NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithMinimumTimeout(25*time.Millisecond),
		WithGracePeriod(10*time.Millisecond),
	)
	<-pushedPackets

	require.True(t, ttsCompletionWatchdog.Start("ctx-old", 25*time.Millisecond))
	require.True(t, ttsCompletionWatchdog.Start("ctx-new", 120*time.Millisecond))
	defer ttsCompletionWatchdog.Cancel()

	select {
	case packet := <-pushedPackets:
		t.Fatalf("previous context pushed packet after replacement: %+v", packet)
	case <-time.After(70 * time.Millisecond):
	}

	require.True(t, ttsCompletionWatchdog.Complete("ctx-new"))
}

func TestTTSCompletionWatchdog_ConstructorPushesInitializationInfo(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 1)

	NewTTSCompletionWatchdog(
		WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				pushedPackets <- packet
			}
			return nil
		}),
		WithWordsPerMinute(150),
		WithMinimumTimeout(40*time.Millisecond),
		WithGracePeriod(20*time.Millisecond),
	)

	packet := <-pushedPackets
	observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
	require.True(t, ok)
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, observabilityLogPacket.Scope)
	assert.Equal(t, observability.LevelInfo, observabilityLogPacket.Record.Level)
	assert.Equal(t, "tts-completion-watchdog: initialization completed", observabilityLogPacket.Record.Message)
	assert.Equal(t, "tts_completion", observabilityLogPacket.Record.Attributes["watchdog"])
	assert.Equal(t, "150", observabilityLogPacket.Record.Attributes["words_per_minute"])
	assert.Equal(t, "40", observabilityLogPacket.Record.Attributes["minimum_timeout_ms"])
	assert.Equal(t, "20", observabilityLogPacket.Record.Attributes["grace_period_ms"])
}
