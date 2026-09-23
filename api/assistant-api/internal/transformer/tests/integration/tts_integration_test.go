//go:build integration

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_test

import (
	"context"
	"testing"

	transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// TestTTSIntegration runs the shared contract against explicitly enabled providers.
func TestTTSIntegration(t *testing.T) {
	config := testutil.LoadConfig(t)
	for _, provider := range []string{
		"deepgram", "elevenlabs", "cartesia", "google-speech-service", "azure-speech-service",
		"sarvamai", "rime", "resembleai", "neuphonic", "minimax", "nvidia", "groq",
		"speechmatics", "aws", "smallest", "custom-tts",
	} {
		t.Run(provider, func(t *testing.T) {
			settings := config.TTSProvider(t, provider)
			for _, scenario := range []struct {
				name string
				run  func(*testing.T, ttsTestFactory)
			}{
				{name: "basic", run: testTTSBasic},
				{name: "metrics", run: testTTSMetricsAndEvents},
				{name: "chunks", run: testTTSMultiChunk},
				{name: "interruption", run: testTTSInterruption},
				{name: "new-session", run: testTTSNewSession},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					scenario.run(t, func(ctx context.Context, onPacket func(...internal_type.Packet) error) (internal_type.TextToSpeechTransformer, error) {
						return transformer.GetTextToSpeechTransformer(ctx, testutil.NewTestLogger(), provider,
							testutil.BuildCredential(settings.Credential), onPacket, testutil.BuildOptions(settings.Options))
					})
				})
			}
		})
	}
}
