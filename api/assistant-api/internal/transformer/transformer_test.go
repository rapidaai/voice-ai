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
			name:     "Custom TTS",
			input:    CUSTOM_TTS,
			expected: "custom-tts",
		},
		{
			name:     "Custom STT",
			input:    CUSTOM_STT,
			expected: "custom-stt",
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
		{
			name:     "Smallest",
			input:    SMALLEST,
			expected: "smallest",
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
			name:            "Custom TTS",
			transformerType: CUSTOM_TTS,
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

func TestNewSpeechToText(t *testing.T) {
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
			name:            "Custom STT",
			transformerType: CUSTOM_STT,
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
			transformer, err := NewSpeechToText(
				WithContext(ctx),
				WithLogger(mockLogger),
				WithProvider(tt.transformerType.String()),
				WithCredential(credential),
				WithOnPacket(func(pkt ...internal_type.Packet) error { return nil }),
				WithOptions(utils.Option{}),
			)

			if tt.transformerType == AudioTransformer("invalid") {
				assert.Error(t, err)
				assert.Nil(t, transformer)
				assert.Equal(t, "stt: provider \"invalid\" is not implemented", err.Error())
			} else if tt.shouldError {
				assert.Error(t, err)
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

			_, sttErr := NewSpeechToText(
				WithContext(ctx),
				WithLogger(mockLogger),
				WithProvider(tt.sttType.String()),
				WithCredential(credential),
				WithOnPacket(func(pkt ...internal_type.Packet) error { return nil }),
				WithOptions(utils.Option{}),
			)
			if tt.wantSTTErr {
				assert.Error(t, sttErr)
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
			transformer, err := NewSpeechToText(
				WithContext(ctx),
				WithLogger(mockLogger),
				WithProvider(AudioTransformer(invalidType).String()),
				WithCredential(credential),
				WithOnPacket(func(pkt ...internal_type.Packet) error { return nil }),
				WithOptions(utils.Option{}),
			)
			assert.Error(t, err)
			assert.Nil(t, transformer)
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
		CUSTOM_TTS,
		GOOGLE_SPEECH_SERVICE,
		REVAI,
		SARVAM,
		ELEVENLABS,
		SMALLEST,
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
		CUSTOM_STT,
		SMALLEST,
	}

	for _, tt := range transformerTypes {
		t.Run(tt.String(), func(t *testing.T) {
			_, _ = NewSpeechToText(
				WithContext(ctx),
				WithLogger(mockLogger),
				WithProvider(tt.String()),
				WithCredential(credential),
				WithOnPacket(func(pkt ...internal_type.Packet) error { return nil }),
				WithOptions(utils.Option{}),
			)
		})
	}
}

// TestSmallestFactorySelection asserts that, given valid credentials, the
// SMALLEST branch of both factories actually constructs a transformer
// instead of merely not panicking (the shared *CallFactory loops above use
// an empty credential, so they pass whether SMALLEST succeeds or falls
// through to an error).
func TestSmallestFactorySelection(t *testing.T) {
	mockLogger, _ := commons.NewApplicationLogger()
	ctx := context.Background()
	value, err := structpb.NewStruct(map[string]interface{}{"key": "test-api-key"})
	assert.NoError(t, err)
	credential := &protos.VaultCredential{Value: value}
	noopCallback := func(pkt ...internal_type.Packet) error { return nil }

	tts, err := GetTextToSpeechTransformer(ctx, mockLogger, SMALLEST.String(), credential, noopCallback, utils.Option{})
	assert.NoError(t, err)
	assert.NotNil(t, tts)

	stt, err := NewSpeechToText(
		WithContext(ctx),
		WithLogger(mockLogger),
		WithProvider(SMALLEST.String()),
		WithCredential(credential),
		WithOnPacket(noopCallback),
		WithOptions(utils.Option{}),
	)
	assert.NoError(t, err)
	assert.NotNil(t, stt)
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

func BenchmarkNewSpeechToText(b *testing.B) {
	mockLogger, _ := commons.NewApplicationLogger()

	ctx := context.Background()
	credential := &protos.VaultCredential{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewSpeechToText(
			WithContext(ctx),
			WithLogger(mockLogger),
			WithProvider(DEEPGRAM.String()),
			WithCredential(credential),
			WithOnPacket(func(pkt ...internal_type.Packet) error { return nil }),
			WithOptions(utils.Option{}),
		)
	}
}
