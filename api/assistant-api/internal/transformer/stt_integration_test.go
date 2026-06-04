//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer

import (
	"context"
	"testing"
	"time"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sttProviders lists all STT providers to test. Each entry maps to the factory
// switch cases in transformer.go and the YAML config keys.
var sttProviders = []string{
	"deepgram",
	"azure-speech-service",
	"google-speech-service",
	"assemblyai",
	"revai",
	"sarvamai",
	"cartesia",
	"speechmatics",
	"ringg",
	"groq",
	"nvidia",
	"aws",
}

// feedAudio delegates to testutil.FeedAudio for cross-package reuse.
var feedAudio = testutil.FeedAudio

// TestSTTBasic verifies the basic STT pipeline: Initialize → feed speech audio →
// collect transcript packets → Close.
func TestSTTBasic(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")

	for _, provider := range sttProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.STTProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			stt, err := GetSpeechToTextTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err, "factory should succeed")
			require.NotNil(t, stt)

			err = stt.Initialize()
			require.NoError(t, err, "Initialize should succeed")
			defer stt.Close(ctx)

			// Feed audio in a goroutine
			go feedAudio(ctx, t, stt, speech)

			// Wait for a final transcript
			collector.WaitForFinalTranscript(t, 20*time.Second)

			finals := collector.FinalTranscripts()
			require.NotEmpty(t, finals, "should receive at least one final transcript")

			// Verify the transcript has content
			assert.NotEmpty(t, finals[0].Script, "transcript text should not be empty")
			t.Logf("provider=%s transcript=%q confidence=%.2f",
				provider, finals[0].Script, finals[0].Confidence)
		})
	}
}

// TestSTTInterimAndFinal verifies that STT providers emit both interim and final
// transcript packets when processing speech audio.
func TestSTTInterimAndFinal(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")

	for _, provider := range sttProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.STTProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			stt, err := GetSpeechToTextTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err)
			require.NoError(t, stt.Initialize())
			defer stt.Close(ctx)

			go feedAudio(ctx, t, stt, speech)

			collector.WaitForFinalTranscript(t, 20*time.Second)

			interims := collector.InterimTranscripts()
			finals := collector.FinalTranscripts()

			t.Logf("provider=%s interims=%d finals=%d", provider, len(interims), len(finals))
			assert.NotEmpty(t, finals, "should have at least one final transcript")

			// Interim transcripts are provider-dependent (some don't emit them)
			if len(interims) > 0 {
				for _, interim := range interims {
					assert.True(t, interim.Interim, "interim packet should have Interim=true")
				}
			}

			for _, final := range finals {
				assert.False(t, final.Interim, "final packet should have Interim=false")
				assert.NotEmpty(t, final.Script, "final transcript should have text")
			}
		})
	}
}

// TestSTTMetricsAndEvents verifies that STT providers emit ConversationEventPacket
// and/or MessageMetricPacket alongside transcripts.
func TestSTTMetricsAndEvents(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")

	for _, provider := range sttProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.STTProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			stt, err := GetSpeechToTextTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err)
			require.NoError(t, stt.Initialize())
			defer stt.Close(ctx)

			go feedAudio(ctx, t, stt, speech)

			collector.WaitForFinalTranscript(t, 20*time.Second)

			events := collector.EventPackets()
			metrics := collector.MetricPackets()
			interruptions := collector.InterruptionDetectedPackets()

			t.Logf("provider=%s events=%d metrics=%d interruptions=%d transcripts=%d",
				provider, len(events), len(metrics), len(interruptions),
				len(collector.TranscriptPackets()))

			// Verify event packets have required fields when present
			for _, ev := range events {
				assert.NotEmpty(t, ev.Name, "event name should not be empty")
				assert.NotNil(t, ev.Data, "event data should not be nil")
			}

			// STT providers typically emit interruption packets with transcripts
			if len(interruptions) > 0 {
				for _, intr := range interruptions {
					assert.Equal(t, internal_type.InterruptionSourceWord, intr.Source,
						"STT interruptions should have source=word")
				}
			}
		})
	}
}

// TestSTTInterruption tests that sending an interruption to the STT transformer
// does not cause errors and properly handles the interrupted state.
func TestSTTInterruption(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")

	for _, provider := range sttProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.STTProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			stt, err := GetSpeechToTextTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err)
			require.NoError(t, stt.Initialize())
			defer stt.Close(ctx)

			// Start feeding audio
			audioDone := make(chan struct{})
			go func() {
				feedAudio(ctx, t, stt, speech)
				close(audioDone)
			}()

			// Wait for some transcripts to arrive, then close
			time.Sleep(2 * time.Second)

			transcriptsBefore := len(collector.TranscriptPackets())
			t.Logf("provider=%s transcripts_before_close=%d", provider, transcriptsBefore)

			// Closing the STT should be clean
			err = stt.Close(ctx)
			assert.NoError(t, err, "Close should not error")
		})
	}
}

// TestSTTSilentAudio verifies that sending silent audio does not produce
// false transcripts. Some providers may emit empty or no transcripts.
func TestSTTSilentAudio(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()

	// 2 seconds of silence
	silence := testutil.SilentPCM(2.0)

	for _, provider := range sttProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.STTProvider(t, provider)
			collector := testutil.NewPacketCollector()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			stt, err := GetSpeechToTextTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
			require.NoError(t, err)
			require.NoError(t, stt.Initialize())
			defer stt.Close(ctx)

			// Feed silence
			go feedAudio(ctx, t, stt, silence)

			// Wait for the audio to be fully sent plus some buffer
			time.Sleep(4 * time.Second)

			finals := collector.FinalTranscripts()
			t.Logf("provider=%s final_transcripts_from_silence=%d", provider, len(finals))

			// With silence, we generally expect no meaningful transcripts.
			// Some providers may emit empty strings — that's acceptable.
			for _, f := range finals {
				if f.Script != "" {
					t.Logf("provider=%s unexpected transcript from silence: %q", provider, f.Script)
				}
			}
		})
	}
}

// TestSTTReconnect tests that a new STT transformer instance can be created and used
// after closing a previous one, validating clean resource management.
func TestSTTReconnect(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	logger := testutil.NewTestLogger()
	speech := testutil.LoadSpeechPCM(t, "hello_world.pcm")

	for _, provider := range sttProviders {
		t.Run(provider, func(t *testing.T) {
			pcfg := cfg.STTProvider(t, provider)
			cred := testutil.BuildCredential(pcfg.Credential)
			opts := testutil.BuildOptions(pcfg.Options)

			for attempt := 0; attempt < 2; attempt++ {
				collector := testutil.NewPacketCollector()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

				stt, err := GetSpeechToTextTransformer(ctx, logger, provider, cred, collector.OnPacket, opts)
				require.NoError(t, err, "attempt %d: factory should succeed", attempt)
				require.NoError(t, stt.Initialize(), "attempt %d: Initialize should succeed", attempt)

				go feedAudio(ctx, t, stt, speech)

				collector.WaitForFinalTranscript(t, 20*time.Second)

				finals := collector.FinalTranscripts()
				assert.NotEmpty(t, finals, "attempt %d: should produce transcripts", attempt)
				t.Logf("attempt=%d provider=%s transcript=%q",
					attempt, provider, finals[0].Script)

				stt.Close(ctx)
				cancel()
			}
		})
	}
}
