// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer

import (
	"context"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestAudioTransformerString tests the String method
func TestAudioTransformerString(t *testing.T) {
	tests := []struct {
		name     string
		input    AudioTransformer
		expected string
	}{
		{
			name:     "Deepgram",
			input:    DEEPGRAM,
			expected: "deepgram",
		},
		{
			name:     "Google Speech Service",
			input:    GOOGLE_SPEECH_SERVICE,
			expected: "google-speech-service",
		},
		{
			name:     "Azure Speech Service",
			input:    AZURE_SPEECH_SERVICE,
			expected: "azure-speech-service",
		},
		{
			name:     "Cartesia",
			input:    CARTESIA,
			expected: "cartesia",
		},
		{
			name:     "RevAI",
			input:    REVAI,
			expected: "revai",
		},
		{
			name:     "Sarvam",
			input:    SARVAM,
			expected: "sarvamai",
		},
		{
			name:     "ElevenLabs",
			input:    ELEVENLABS,
			expected: "elevenlabs",
		},
		{
			name:     "AssemblyAI",
			input:    ASSEMBLYAI,
			expected: "assemblyai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetTextToSpeechTransformer tests text-to-speech transformer creation
func TestGetTextToSpeechTransformer(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()
	ctx := context.Background()
	credential := &protos.VaultCredential{}

	tests := []struct {
		name            string
		transformerType AudioTransformer
		shouldError     bool
	}{
		{
			name:            "Deepgram TTS",
			transformerType: DEEPGRAM,
			shouldError:     true, // Will fail due to missing credentials, but factory works
		},
		{
			name:            "Invalid TTS",
			transformerType: AudioTransformer("invalid"),
			shouldError:     true, // Should fail with factory error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer, err := GetTextToSpeechTransformer(ctx, mockLogger, tt.transformerType.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})

			if tt.transformerType == AudioTransformer("invalid") {
				// Invalid transformer type should return factory error
				assert.Error(t, err)
				assert.Nil(t, transformer)
				assert.Equal(t, "illegal text to speech idenitfier", err.Error())
			} else if tt.shouldError {
				// Valid transformer type but credential issues
				assert.Error(t, err) // Expected to fail due to credentials, but not factory error
				assert.Nil(t, transformer)
			}
		})
	}
}

// TestGetTextToSpeechTransformer_InworldRouting asserts the INWORLD switch
// branch is reachable via the factory: a non-empty "key" credential
// resolves through to a concrete *inworldTTS, and an unknown provider
// falls through to the same "illegal text to speech" error every other
// unknown provider hits. Protects the dispatcher wiring against silent
// regressions (it is trivially easy to delete a case and not notice, since
// the constructor tests bypass the factory).
func TestGetTextToSpeechTransformer_InworldRouting(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()
	ctx := context.Background()

	t.Run("inworld success path", func(t *testing.T) {
		val, err := structpb.NewStruct(map[string]interface{}{"key": "test-key"})
		assert.NoError(t, err)
		credential := &protos.VaultCredential{Value: val}

		transformer, err := GetTextToSpeechTransformer(ctx, mockLogger, INWORLD.String(), credential,
			func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
		assert.NoError(t, err, "inworld factory should not error on a valid key")
		assert.NotNil(t, transformer)
		assert.Equal(t, "inworld-text-to-speech", transformer.Name(),
			"factory should return the inworld TTS transformer, not another provider")
	})

	t.Run("inworld rejects empty key", func(t *testing.T) {
		val, err := structpb.NewStruct(map[string]interface{}{"key": ""})
		assert.NoError(t, err)
		credential := &protos.VaultCredential{Value: val}

		transformer, err := GetTextToSpeechTransformer(ctx, mockLogger, INWORLD.String(), credential,
			func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
		assert.Error(t, err)
		assert.Nil(t, transformer)
	})

	t.Run("unknown provider falls back to factory error", func(t *testing.T) {
		credential := &protos.VaultCredential{}
		transformer, err := GetTextToSpeechTransformer(ctx, mockLogger, "nonexistent-provider", credential,
			func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
		assert.Error(t, err)
		assert.Nil(t, transformer)
		assert.Equal(t, "illegal text to speech idenitfier", err.Error())
	})
}

// TestGetSpeechToTextTransformer tests speech-to-text transformer creation
func TestGetSpeechToTextTransformer(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()
	ctx := context.Background()
	credential := &protos.VaultCredential{}
	tests := []struct {
		name            string
		transformerType AudioTransformer
		shouldError     bool
	}{
		{
			name:            "Deepgram STT",
			transformerType: DEEPGRAM,
			shouldError:     true, // Will fail due to missing credentials, but factory works
		},
		{
			name:            "Invalid STT",
			transformerType: AudioTransformer("invalid"),
			shouldError:     true, // Should fail with factory error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer, err := GetSpeechToTextTransformer(ctx, mockLogger, tt.transformerType.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})

			if tt.transformerType == AudioTransformer("invalid") {
				// Invalid transformer type should return factory error
				assert.Error(t, err)
				assert.Nil(t, transformer)
				assert.Equal(t, "illegal speech to text idenitfier", err.Error())
			} else if tt.shouldError {
				// Valid transformer type but credential issues
				assert.Error(t, err) // Expected to fail due to credentials, but not factory error
				assert.Nil(t, transformer)
			}
		})
	}
}

// TestInvalidAudioTransformerTypesCombinations tests all types of invalid inputs
func TestInvalidAudioTransformerTypesCombinations(t *testing.T) {
	ctx := context.Background()
	mockLogger, _ := commons.NewApplicationLogger()
	credential := &protos.VaultCredential{}

	tests := []struct {
		name       string
		ttsType    AudioTransformer
		sttType    AudioTransformer
		wantTTSErr bool
		wantSTTErr bool
	}{
		{
			name:       "Empty string transformer",
			ttsType:    AudioTransformer(""),
			sttType:    AudioTransformer(""),
			wantTTSErr: true,
			wantSTTErr: true,
		},
		{
			name:       "Unknown transformer",
			ttsType:    AudioTransformer("unknown-provider"),
			sttType:    AudioTransformer("unknown-provider"),
			wantTTSErr: true,
			wantSTTErr: true,
		},
		{
			name:       "Case sensitive test",
			ttsType:    AudioTransformer("DEEPGRAM"),
			sttType:    AudioTransformer("DEEPGRAM"),
			wantTTSErr: true,
			wantSTTErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ttsErr := GetTextToSpeechTransformer(ctx, mockLogger, tt.ttsType.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
			if tt.wantTTSErr {
				assert.Error(t, ttsErr)
				assert.Equal(t, "illegal text to speech idenitfier", ttsErr.Error())
			} else {
				assert.NoError(t, ttsErr)
			}

			_, sttErr := GetSpeechToTextTransformer(ctx, mockLogger, tt.sttType.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
			if tt.wantSTTErr {
				assert.Error(t, sttErr)
				assert.Equal(t, "illegal speech to text idenitfier", sttErr.Error())
			} else {
				assert.NoError(t, sttErr)
			}
		})
	}
}

// TestInvalidAudioTransformerTypesTTS tests various invalid transformer types for TTS
func TestInvalidAudioTransformerTypesTTS(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()

	ctx := context.Background()
	credential := &protos.VaultCredential{}

	invalidTypes := []string{
		"",
		"invalid",
		"DEEPGRAM",
		"deepgram-extra",
		"unknown-service",
	}

	for _, invalidType := range invalidTypes {
		t.Run("Invalid_"+invalidType, func(t *testing.T) {
			transformer, err := GetTextToSpeechTransformer(ctx, mockLogger, AudioTransformer(invalidType).String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
			assert.Error(t, err)
			assert.Nil(t, transformer)
			assert.Equal(t, "illegal text to speech idenitfier", err.Error())
		})
	}
}

// TestInvalidAudioTransformerTypesSTT tests various invalid transformer types for STT
func TestInvalidAudioTransformerTypesSTT(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()

	ctx := context.Background()
	credential := &protos.VaultCredential{}

	invalidTypes := []string{
		"",
		"invalid",
		"DEEPGRAM",
		"deepgram-extra",
		"unknown-service",
	}

	for _, invalidType := range invalidTypes {
		t.Run("Invalid_"+invalidType, func(t *testing.T) {
			transformer, err := GetSpeechToTextTransformer(ctx, mockLogger, AudioTransformer(invalidType).String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
			assert.Error(t, err)
			assert.Nil(t, transformer)
			assert.Equal(t, "illegal speech to text idenitfier", err.Error())
		})
	}
}

// TestAllTextToSpeechTransformersAreDifferent validates factory doesn't panic for all types
func TestAllTextToSpeechTransformersCallFactory(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()

	ctx := context.Background()
	credential := &protos.VaultCredential{}

	transformerTypes := []AudioTransformer{
		DEEPGRAM,
		AZURE_SPEECH_SERVICE,
		CARTESIA,
		GOOGLE_SPEECH_SERVICE,
		REVAI,
		SARVAM,
		ELEVENLABS,
	}

	for _, tt := range transformerTypes {
		t.Run(tt.String(), func(t *testing.T) {
			// Just ensure factory can be called without panic
			_, _ = GetTextToSpeechTransformer(ctx, mockLogger, tt.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
		})
	}
}

// TestAllSpeechToTextTransformersCallFactory validates factory doesn't panic for all types
func TestAllSpeechToTextTransformersCallFactory(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()

	ctx := context.Background()
	credential := &protos.VaultCredential{}

	transformerTypes := []AudioTransformer{
		DEEPGRAM,
		AZURE_SPEECH_SERVICE,
		GOOGLE_SPEECH_SERVICE,
		ASSEMBLYAI,
		REVAI,
		SARVAM,
		CARTESIA,
	}

	for _, tt := range transformerTypes {
		t.Run(tt.String(), func(t *testing.T) {
			// Just ensure factory can be called without panic
			_, _ = GetSpeechToTextTransformer(ctx, mockLogger, tt.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
		})
	}
}

// BenchmarkGetTextToSpeechTransformer benchmarks TTS factory performance
func BenchmarkGetTextToSpeechTransformer(b *testing.B) {
	mockLogger, _ := commons.NewApplicationLogger()

	ctx := context.Background()
	credential := &protos.VaultCredential{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetTextToSpeechTransformer(ctx, mockLogger, DEEPGRAM.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
	}
}

// BenchmarkGetSpeechToTextTransformer benchmarks STT factory performance
func BenchmarkGetSpeechToTextTransformer(b *testing.B) {
	mockLogger, _ := commons.NewApplicationLogger()

	ctx := context.Background()
	credential := &protos.VaultCredential{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetSpeechToTextTransformer(ctx, mockLogger, DEEPGRAM.String(), credential, func(pkt ...internal_type.Packet) error { return nil }, utils.Option{})
	}
}
