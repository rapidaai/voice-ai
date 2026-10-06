// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package elevenlabs_internal

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

func newTestElevenLabsProcessor(t *testing.T, opts utils.Option) *elevenlabsProcessor {
	t.Helper()
	logger := testutil.NewTestLogger()
	processor := NewElevenLabsProcessor(logger, opts)
	en, ok := processor.(*elevenlabsProcessor)
	require.True(t, ok, "expected *elevenlabsProcessor type")
	return en
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewElevenLabsProcessor(t *testing.T) {
	tests := []struct {
		name         string
		opts         utils.Option
		expectedLang string
		hasConj      bool
	}{
		{
			name:         "default options",
			opts:         utils.Option{},
			expectedLang: "en",
			hasConj:      false,
		},
		{
			name: "with explicit language",
			opts: utils.Option{
				"speaker.language": "es",
			},
			expectedLang: "es",
			hasConj:      false,
		},
		{
			name: "with empty language",
			opts: utils.Option{
				"speaker.language": "",
			},
			expectedLang: "en",
			hasConj:      false,
		},
		{
			name: "with conjunction boundaries",
			opts: utils.Option{
				"speaker.conjunction.boundaries": "and<|||>but<|||>or",
				"speaker.conjunction.break":      uint64(500),
			},
			expectedLang: "en",
			hasConj:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			en := newTestElevenLabsProcessor(t, tt.opts)
			assert.Equal(t, tt.expectedLang, en.language)
			assert.NotNil(t, en.logger)
			if tt.hasConj {
				assert.NotNil(t, en.conjunctionPattern)
			} else {
				assert.Nil(t, en.conjunctionPattern)
			}
		})
	}
}

// =============================================================================
// Process Tests
// =============================================================================

func TestProcess_EmptyString(t *testing.T) {
	en := newTestElevenLabsProcessor(t, utils.Option{})
	result := en.Process("")
	assert.Equal(t, "", result)
}

func TestProcess_XMLEscaping(t *testing.T) {
	en := newTestElevenLabsProcessor(t, utils.Option{})

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
			name:     "quotes are NOT escaped (ElevenLabs uses 3 entities)",
			input:    `She said "hello"`,
			expected: `She said "hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := en.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcess_ConjunctionBreaks_SecondsFormat(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but",
		"speaker.conjunction.break":      uint64(500),
	}
	en := newTestElevenLabsProcessor(t, opts)

	result := en.Process("cats and dogs but not fish")
	// ElevenLabs uses seconds format, not milliseconds
	assert.Contains(t, result, `and<break time="0.50s"/>`)
	assert.Contains(t, result, `but<break time="0.50s"/>`)
}

func TestProcess_ConjunctionBreaks_SubSecond(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and",
		"speaker.conjunction.break":      uint64(250),
	}
	en := newTestElevenLabsProcessor(t, opts)

	result := en.Process("cats and dogs")
	assert.Contains(t, result, `<break time="0.25s"/>`)
}

func TestProcess_NoConjunctionBreaksWhenNotConfigured(t *testing.T) {
	en := newTestElevenLabsProcessor(t, utils.Option{})

	result := en.Process("cats and dogs but not fish")
	assert.NotContains(t, result, "<break")
}

func TestProcess_MarkdownIsNotStripped(t *testing.T) {
	en := newTestElevenLabsProcessor(t, utils.Option{})

	input := "**bold** text"
	result := en.Process(input)
	assert.Contains(t, result, "**bold**")
}

// =============================================================================
// SSML Helper Tests
// =============================================================================

func TestAddBreak_SecondsFormat(t *testing.T) {
	en := newTestElevenLabsProcessor(t, utils.Option{})

	tests := []struct {
		name       string
		durationMs int
		expected   string
	}{
		{
			name:       "half second",
			durationMs: 500,
			expected:   `<break time="0.50s"/>`,
		},
		{
			name:       "one second",
			durationMs: 1000,
			expected:   `<break time="1.00s"/>`,
		},
		{
			name:       "quarter second",
			durationMs: 250,
			expected:   `<break time="0.25s"/>`,
		},
		{
			name:       "zero",
			durationMs: 0,
			expected:   `<break time="0.00s"/>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := en.AddBreak(tt.durationMs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddPhoneme(t *testing.T) {
	en := newTestElevenLabsProcessor(t, utils.Option{})
	result := en.AddPhoneme("tomato", "t@meItoU")
	assert.Equal(t, `<phoneme alphabet="ipa" ph="t@meItoU">tomato</phoneme>`, result)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkProcess_SimpleText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewElevenLabsProcessor(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}

func BenchmarkProcess_WithConjunctions(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but<|||>or",
		"speaker.conjunction.break":      uint64(250),
	}
	processor := NewElevenLabsProcessor(logger, opts)
	text := "I like cats and dogs but not fish or snakes and birds"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}

func BenchmarkProcess_XMLEscaping(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewElevenLabsProcessor(logger, utils.Option{})
	text := "Tom & Jerry said a < b > c & d < e"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}
