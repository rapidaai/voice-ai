// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package google_internal

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

func newTestGoogleNormalizer(t *testing.T, opts utils.Option) *googleNormalizer {
	t.Helper()
	logger := testutil.NewTestLogger() // Use testutil logger for better test output integration
	normalizer := NewGoogleNormalizer(logger, opts)
	gn, ok := normalizer.(*googleNormalizer)
	require.True(t, ok, "expected *googleNormalizer type")
	return gn
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewGoogleNormalizer(t *testing.T) {
	tests := []struct {
		name         string
		opts         utils.Option
		expectedLang string
		hasConj      bool
	}{
		{
			name:         "default options",
			opts:         utils.Option{},
			expectedLang: "en-US",
			hasConj:      false,
		},
		{
			name: "with explicit language",
			opts: utils.Option{
				"speaker.language": "de-DE",
			},
			expectedLang: "de-DE",
			hasConj:      false,
		},
		{
			name: "with empty language",
			opts: utils.Option{
				"speaker.language": "",
			},
			expectedLang: "en-US",
			hasConj:      false,
		},
		{
			name: "with conjunction boundaries",
			opts: utils.Option{
				"speaker.conjunction.boundaries": "and<|||>but<|||>or",
				"speaker.conjunction.break":      uint64(300),
			},
			expectedLang: "en-US",
			hasConj:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gn := newTestGoogleNormalizer(t, tt.opts)
			assert.NotNil(t, gn.logger)
			if tt.hasConj {
				assert.NotNil(t, gn.conjunctionPattern)
			} else {
				assert.Nil(t, gn.conjunctionPattern)
			}
		})
	}
}

// =============================================================================
// Normalize Tests
// =============================================================================

func TestNormalize_EmptyString(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})
	result := gn.Normalize("")
	assert.Equal(t, "", result)
}

func TestNormalize_XMLEscaping(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "all three entities",
			input:    "a < b & c > d",
			expected: "a &lt; b &amp; c &gt; d",
		},
		{
			name:     "no escaping needed",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "quotes are NOT escaped (Google uses 3 entities)",
			input:    `She said "hello"`,
			expected: `She said "hello"`,
		},
		{
			name:     "apostrophe is NOT escaped (Google uses 3 entities)",
			input:    "it's fine",
			expected: "it's fine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gn.Normalize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalize_ConjunctionBreaks(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but",
		"speaker.conjunction.break":      uint64(250),
	}
	gn := newTestGoogleNormalizer(t, opts)

	result := gn.Normalize("cats and dogs but not fish")
	assert.Contains(t, result, `and<break time="250ms"/>`)
	assert.Contains(t, result, `but<break time="250ms"/>`)
}

func TestNormalize_NoConjunctionBreaksWhenNotConfigured(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})

	result := gn.Normalize("cats and dogs but not fish")
	assert.NotContains(t, result, "<break")
	assert.Equal(t, "cats and dogs but not fish", result)
}

func TestNormalize_PreNormalizedTextPassthrough(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})

	// Pre-normalized text (no markdown, clean whitespace) should pass through
	// with only XML escaping applied
	input := "Hello world. This is pre-normalized text."
	result := gn.Normalize(input)
	assert.Equal(t, input, result)
}

func TestNormalize_MarkdownIsNotStripped(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})

	// After centralization, markdown removal is done upstream.
	// The provider normalizer should NOT strip markdown itself.
	input := "**bold** text"
	result := gn.Normalize(input)
	// The asterisks should still be present (only XML escaping applied)
	assert.Contains(t, result, "**bold**")
}

// =============================================================================
// SSML Helper Tests
// =============================================================================

func TestWrapWithSSML(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})
	result := gn.WrapWithSSML("Hello world")
	assert.Equal(t, "<speak>Hello world</speak>", result)
}

func TestAddBreak(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})
	result := gn.AddBreak(500)
	assert.Equal(t, `<break time="500ms"/>`, result)
}

func TestAddProsody(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})

	result := gn.AddProsody("hello", "fast", "high", "loud")
	assert.Contains(t, result, `rate="fast"`)
	assert.Contains(t, result, `pitch="high"`)
	assert.Contains(t, result, `volume="loud"`)
	assert.Contains(t, result, "hello")

	// Empty attrs returns text unchanged
	result = gn.AddProsody("hello", "", "", "")
	assert.Equal(t, "hello", result)
}

func TestAddEmphasis(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})
	result := gn.AddEmphasis("important", "strong")
	assert.Equal(t, `<emphasis level="strong">important</emphasis>`, result)
}

func TestSayAs(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})

	result := gn.SayAs("123", "cardinal", "")
	assert.Equal(t, `<say-as interpret-as="cardinal">123</say-as>`, result)

	result = gn.SayAs("2024-01-15", "date", "ymd")
	assert.Equal(t, `<say-as interpret-as="date" format="ymd">2024-01-15</say-as>`, result)
}

func TestAddAudio(t *testing.T) {
	gn := newTestGoogleNormalizer(t, utils.Option{})

	result := gn.AddAudio("https://example.com/beep.wav", "beep sound")
	assert.Equal(t, `<audio src="https://example.com/beep.wav">beep sound</audio>`, result)

	result = gn.AddAudio("https://example.com/beep.wav", "")
	assert.Equal(t, `<audio src="https://example.com/beep.wav"/>`, result)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNormalize_SimpleText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewGoogleNormalizer(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}

func BenchmarkNormalize_WithConjunctions(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but<|||>or",
		"speaker.conjunction.break":      uint64(250),
	}
	normalizer := NewGoogleNormalizer(logger, opts)
	text := "I like cats and dogs but not fish or snakes and birds"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}

func BenchmarkNormalize_XMLEscaping(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewGoogleNormalizer(logger, utils.Option{})
	text := "Tom & Jerry said a < b > c & d < e"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}
