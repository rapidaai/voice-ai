// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package transformer_testutil

import (
	"os"
	"path/filepath"
	"testing"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
)

func TestIntegrationConfigExample(t *testing.T) {
	path := filepath.Join(testdataDir(), "integration_config.yaml.example")
	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("read shipped example: %v", err)
	}
	t.Setenv("TRANSFORMER_TEST_CONFIG", path)
	cfg := LoadConfig(t)

	for name, providers := range map[string]map[string]ProviderConfig{"tts": cfg.TTS, "stt": cfg.STT} {
		t.Run(name, func(t *testing.T) {
			if len(providers) == 0 {
				t.Fatal("example must contain providers")
			}
			for provider, config := range providers {
				t.Run(provider, func(t *testing.T) {
					if config.Enabled {
						t.Error("shipped provider must be disabled")
					}
					if _, ok := config.Options["speak.voice"]; ok {
						t.Error("use speak.voice.id instead of speak.voice")
					}
				})
			}
			for _, tc := range []struct {
				name      string
				required  []string
				forbidden []string
			}{
				{name: "aws", required: []string{"access_key_id", "secret_access_key", "region"}, forbidden: []string{"key", "secret"}},
				{name: "azure-speech-service", required: []string{"subscription_key", "endpoint"}, forbidden: []string{"key", "region"}},
				{name: "nvidia", required: []string{"key", "function_id"}},
				{name: "google-speech-service", required: []string{"project_id"}, forbidden: []string{"key"}},
			} {
				t.Run(tc.name+"_credentials", func(t *testing.T) {
					credential := providers[tc.name].Credential
					for _, key := range tc.required {
						if credential[key] == "" {
							t.Errorf("missing credential placeholder %q", key)
						}
					}
					for _, key := range tc.forbidden {
						if _, ok := credential[key]; ok {
							t.Errorf("unexpected credential key %q", key)
						}
					}
				})
			}
			t.Run("google_credentials", func(t *testing.T) {
				value, ok := providers["google-speech-service"].Credential["service_account_key"]
				if !ok || value != "" {
					t.Error("service_account_key must be an empty string placeholder for JSON or ADC")
				}
			})
		})
	}

	t.Run("voice_ids", func(t *testing.T) {
		for _, provider := range []string{
			"deepgram", "elevenlabs", "cartesia", "google-speech-service", "azure-speech-service",
			"rime", "resembleai", "neuphonic", "minimax", "nvidia", "groq", "aws", "smallest",
		} {
			t.Run(provider, func(t *testing.T) {
				voice, err := BuildOptions(cfg.TTS[provider].Options).GetString(internal_options.SpeakOptionVoiceID)
				if err != nil || voice == "" || voice == "default" {
					t.Errorf("expected voice ID or explicit placeholder, got %q (%v)", voice, err)
				}
			})
		}
	})
	t.Run("aws_engine", func(t *testing.T) {
		options := cfg.TTS["aws"].Options
		if options[internal_options.SpeakOptionModel] != "neural" {
			t.Error("AWS engine must be configured via speak.model")
		}
		if _, ok := options["speak.engine"]; ok {
			t.Error("unsupported speak.engine key")
		}
	})
	t.Run("deepgram_voice", func(t *testing.T) {
		options := cfg.TTS["deepgram"].Options
		if options[internal_options.SpeakOptionVoiceID] != "aura-asteria-en" {
			t.Error("Deepgram TTS model must be configured via speak.voice.id")
		}
		if _, ok := options[internal_options.SpeakOptionModel]; ok {
			t.Error("Deepgram TTS does not read speak.model")
		}
	})
	t.Run("minimax_credentials", func(t *testing.T) {
		if cfg.TTS["minimax"].Credential["group_id"] == "" {
			t.Error("MiniMax requires a group_id placeholder")
		}
	})
	t.Run("revai_stt_only", func(t *testing.T) {
		if _, ok := cfg.TTS["revai"]; ok {
			t.Error("RevAI TTS is unsupported")
		}
		if cfg.STT["revai"].Credential["key"] == "" {
			t.Error("RevAI STT must retain its API key placeholder")
		}
	})
}
