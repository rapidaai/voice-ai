// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package azure_internal

import (
	"testing"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Setup Helpers
// =============================================================================

func newTestAzureProcessor(t *testing.T, opts utils.Option) *azureProcessor {
	t.Helper()
	logger := testutil.NewTestLogger()
	processor := NewAzureProcessor(logger, opts)
	an, ok := processor.(*azureProcessor)
	require.True(t, ok, "expected *azureProcessor type")
	return an
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewAzureProcessor(t *testing.T) {
	tests := []struct {
		name          string
		opts          utils.Option
		expectedLang  string
		expectedVoice string
		hasConj       bool
	}{
		{
			name:          "default options",
			opts:          utils.Option{},
			expectedLang:  "en-US",
			expectedVoice: "",
			hasConj:       false,
		},
		{
			name: "with voice and language",
			opts: utils.Option{
				"speaker.voice.name": "en-US-JennyNeural",
				"speaker.language":   "en-US",
			},
			expectedLang:  "en-US",
			expectedVoice: "en-US-JennyNeural",
			hasConj:       false,
		},
		{
			name: "with empty language defaults",
			opts: utils.Option{
				"speaker.language": "",
			},
			expectedLang:  "en-US",
			expectedVoice: "",
			hasConj:       false,
		},
		{
			name: "with conjunction boundaries",
			opts: utils.Option{
				"speaker.conjunction.boundaries": "and<|||>but<|||>or",
				"speaker.conjunction.break":      uint64(300),
			},
			expectedLang:  "en-US",
			expectedVoice: "",
			hasConj:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			an := newTestAzureProcessor(t, tt.opts)
			assert.Equal(t, tt.expectedLang, an.language)
			assert.Equal(t, tt.expectedVoice, an.voiceName)
			assert.NotNil(t, an.logger)
			if tt.hasConj {
				assert.NotNil(t, an.conjunctionPattern)
			} else {
				assert.Nil(t, an.conjunctionPattern)
			}
		})
	}
}

// =============================================================================
// Process Tests
// =============================================================================

func TestProcess_EmptyString(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})
	result := an.Process("")
	assert.Equal(t, "", result)
}

func TestProcess_XMLEscaping(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})

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
			name:     "double quote",
			input:    `She said "hello"`,
			expected: `She said &quot;hello&quot;`,
		},
		{
			name:     "apostrophe",
			input:    "it's fine",
			expected: "it&apos;s fine",
		},
		{
			name:     "all five entities",
			input:    `"a" < b & c > 'd'`,
			expected: `&quot;a&quot; &lt; b &amp; c &gt; &apos;d&apos;`,
		},
		{
			name:     "no escaping needed",
			input:    "Hello world",
			expected: "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := an.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcess_ConjunctionBreaks(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but",
		"speaker.conjunction.break":      uint64(250),
	}
	an := newTestAzureProcessor(t, opts)

	result := an.Process("cats and dogs but not fish")
	assert.Contains(t, result, `and<break time="250ms"/>`)
	assert.Contains(t, result, `but<break time="250ms"/>`)
}

func TestProcess_NoConjunctionBreaksWhenNotConfigured(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})

	result := an.Process("cats and dogs but not fish")
	assert.NotContains(t, result, "<break")
}

func TestProcess_MarkdownIsNotStripped(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})

	// After centralization, markdown removal is done upstream.
	input := "**bold** text"
	result := an.Process(input)
	assert.Contains(t, result, "**bold**")
}

// =============================================================================
// SSML Helper Tests
// =============================================================================

func TestWrapWithSSML(t *testing.T) {
	opts := utils.Option{
		"speaker.voice.name": "en-US-JennyNeural",
		"speaker.language":   "en-US",
	}
	an := newTestAzureProcessor(t, opts)
	result := an.WrapWithSSML("Hello world")
	assert.Contains(t, result, `xml:lang="en-US"`)
	assert.Contains(t, result, `name="en-US-JennyNeural"`)
	assert.Contains(t, result, "Hello world")
	assert.Contains(t, result, "xmlns:mstts")
}

func TestAddBreak(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})
	result := an.AddBreak(500)
	assert.Equal(t, `<break time="500ms"/>`, result)
}

func TestAddProsody(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})

	result := an.AddProsody("hello", "fast", "high", "loud")
	assert.Contains(t, result, `rate="fast"`)
	assert.Contains(t, result, `pitch="high"`)
	assert.Contains(t, result, `volume="loud"`)

	result = an.AddProsody("hello", "", "", "")
	assert.Equal(t, "hello", result)
}

func TestAddEmphasis(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})
	result := an.AddEmphasis("important", "strong")
	assert.Equal(t, `<emphasis level="strong">important</emphasis>`, result)
}

func TestAddExpressAs(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})
	result := an.AddExpressAs("I am so happy!", "cheerful")
	assert.Equal(t, `<mstts:express-as style="cheerful">I am so happy!</mstts:express-as>`, result)
}

func TestSayAs(t *testing.T) {
	an := newTestAzureProcessor(t, utils.Option{})

	result := an.SayAs("123", "cardinal", "")
	assert.Equal(t, `<say-as interpret-as="cardinal">123</say-as>`, result)

	result = an.SayAs("2024-01-15", "date", "ymd")
	assert.Equal(t, `<say-as interpret-as="date" format="ymd">2024-01-15</say-as>`, result)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkProcess_SimpleText(b *testing.B) {
	logger := testutil.NewTestLogger()
	processor := NewAzureProcessor(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}

func BenchmarkProcess_WithConjunctions(b *testing.B) {
	logger := testutil.NewTestLogger()
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but<|||>or",
		"speaker.conjunction.break":      uint64(250),
	}
	processor := NewAzureProcessor(logger, opts)
	text := "I like cats and dogs but not fish or snakes and birds"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}

func BenchmarkProcess_XMLEscaping(b *testing.B) {
	logger := testutil.NewTestLogger()
	processor := NewAzureProcessor(logger, utils.Option{})
	text := `Tom & Jerry said "hello" it's a < b > c`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}
