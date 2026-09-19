// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package deepgram_internal

import (
	"testing"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Setup Helpers
// =============================================================================

func newTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err, "failed to create test logger")
	return logger
}

func newTestNormalizer(t *testing.T, opts utils.Option) *deepgramNormalizer {
	t.Helper()
	logger := testutil.NewTestLogger()
	normalizer := NewDeepgramNormalizer(logger, opts)
	dn, ok := normalizer.(*deepgramNormalizer)
	require.True(t, ok, "expected *deepgramNormalizer type")
	return dn
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewDeepgramNormalizer(t *testing.T) {
	tests := []struct {
		name         string
		opts         utils.Option
		expectedLang string
	}{
		{
			name:         "default options - no language",
			opts:         utils.Option{},
			expectedLang: "en",
		},
		{
			name: "with explicit language",
			opts: utils.Option{
				"speaker.language": "es",
			},
			expectedLang: "es",
		},
		{
			name: "with empty language string",
			opts: utils.Option{
				"speaker.language": "",
			},
			expectedLang: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.NewTestLogger()
			normalizer := NewDeepgramNormalizer(logger, tt.opts)

			require.NotNil(t, normalizer, "normalizer should not be nil")

			dn, ok := normalizer.(*deepgramNormalizer)
			require.True(t, ok, "should return *deepgramNormalizer")

			assert.Equal(t, tt.expectedLang, dn.language)
			assert.NotNil(t, dn.logger)
		})
	}
}

// =============================================================================
// Normalize Method Tests
// =============================================================================

func TestNormalize_EmptyString(t *testing.T) {
	normalizer := newTestNormalizer(t, utils.Option{})
	result := normalizer.Normalize("")
	assert.Equal(t, "", result)
}

func TestNormalize_Passthrough(t *testing.T) {
	normalizer := newTestNormalizer(t, utils.Option{})

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple sentence",
			input: "Hello world",
		},
		{
			name:  "multiple sentences",
			input: "Hello world. How are you today?",
		},
		{
			name:  "sentence with numbers",
			input: "I have 5 apples and 3 oranges.",
		},
		{
			name:  "sentence with special characters",
			input: "Contact us at support@example.com!",
		},
		{
			name:  "text with XML-like content",
			input: "Use the <tag> element",
		},
		{
			name:  "text with ampersand",
			input: "Tom & Jerry show",
		},
		{
			name:  "markdown text (not stripped - upstream responsibility)",
			input: "# Header\n**bold** text",
		},
		{
			name:  "unicode characters",
			input: "Hello 世界 Привет مرحبا",
		},
		{
			name:  "emojis",
			input: "Hello 👋 World 🌍",
		},
		{
			name:  "whitespace preserved (upstream responsibility)",
			input: "Hello    world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			assert.Equal(t, tt.input, result, "Normalize should return text unchanged")
		})
	}
}

func TestNormalize_NoSSMLOutput(t *testing.T) {
	normalizer := newTestNormalizer(t, utils.Option{})

	// Deepgram doesn't support SSML, so output should never contain SSML tags
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "text with XML-like content",
			input: "Use the <tag> element",
		},
		{
			name:  "text with ampersand",
			input: "Tom & Jerry show",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			// Should NOT contain SSML tags like <speak>, <break>, etc.
			assert.NotContains(t, result, "<speak>")
			assert.NotContains(t, result, "</speak>")
			assert.NotContains(t, result, "<break")
			// Should NOT XML-escape entities (passthrough)
			assert.NotContains(t, result, "&amp;")
			assert.NotContains(t, result, "&lt;")
			assert.NotContains(t, result, "&gt;")
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNormalize_SimpleText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewDeepgramNormalizer(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}

func BenchmarkNormalize_LongText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewDeepgramNormalizer(logger, utils.Option{})

	// Generate a longer text
	text := ""
	for i := 0; i < 100; i++ {
		text += "This is sentence number " + string(rune(i)) + ". "
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}
