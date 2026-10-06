// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package rime_internal

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

func newTestRimeProcessor(t *testing.T, opts utils.Option) *rimeProcessor {
	t.Helper()
	logger := testutil.NewTestLogger()
	processor := NewRimeProcessor(logger, opts)
	rn, ok := processor.(*rimeProcessor)
	require.True(t, ok, "expected *rimeProcessor type")
	return rn
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewRimeProcessor(t *testing.T) {
	tests := []struct {
		name         string
		opts         utils.Option
		expectedLang string
		hasConj      bool
	}{
		{
			name:         "default options",
			opts:         utils.Option{},
			expectedLang: "eng",
			hasConj:      false,
		},
		{
			name: "with explicit language",
			opts: utils.Option{
				"speaker.language": "spa",
			},
			expectedLang: "spa",
			hasConj:      false,
		},
		{
			name: "with empty language",
			opts: utils.Option{
				"speaker.language": "",
			},
			expectedLang: "eng",
			hasConj:      false,
		},
		{
			name: "with conjunction boundaries",
			opts: utils.Option{
				"speaker.conjunction.boundaries": "and<|||>but<|||>or",
				"speaker.conjunction.break":      uint64(400),
			},
			expectedLang: "eng",
			hasConj:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rn := newTestRimeProcessor(t, tt.opts)
			assert.Equal(t, tt.expectedLang, rn.language)
			assert.NotNil(t, rn.logger)
			if tt.hasConj {
				assert.NotNil(t, rn.conjunctionPattern)
			} else {
				assert.Nil(t, rn.conjunctionPattern)
			}
		})
	}
}

// =============================================================================
// Process Tests
// =============================================================================

func TestProcess_EmptyString(t *testing.T) {
	rn := newTestRimeProcessor(t, utils.Option{})
	result := rn.Process("")
	assert.Equal(t, "", result)
}

func TestProcess_PlainTextPassthrough(t *testing.T) {
	rn := newTestRimeProcessor(t, utils.Option{})

	// Rime has no XML escaping, so plain text passes through unchanged
	input := "Hello world. This is preprocessed text."
	result := rn.Process(input)
	assert.Equal(t, input, result)
}

func TestProcess_NoXMLEscaping(t *testing.T) {
	rn := newTestRimeProcessor(t, utils.Option{})

	// Rime does NOT support SSML, so no XML escaping should be applied
	input := "Tom & Jerry said a < b > c"
	result := rn.Process(input)
	assert.Equal(t, input, result, "Rime processor should not escape XML entities")
}

func TestProcess_ConjunctionBreaks_CustomMsSyntax(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but",
		"speaker.conjunction.break":      uint64(500),
	}
	rn := newTestRimeProcessor(t, opts)

	result := rn.Process("cats and dogs but not fish")
	// Rime uses custom <ms> syntax, NOT standard SSML
	assert.Contains(t, result, "and <500> ")
	assert.Contains(t, result, "but <500> ")
	// Should NOT contain standard SSML break tags
	assert.NotContains(t, result, `<break time=`)
}

func TestProcess_ConjunctionBreaks_DifferentDurations(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and",
		"speaker.conjunction.break":      uint64(250),
	}
	rn := newTestRimeProcessor(t, opts)

	result := rn.Process("cats and dogs")
	assert.Contains(t, result, "and <250> ")
}

func TestProcess_NoConjunctionBreaksWhenNotConfigured(t *testing.T) {
	rn := newTestRimeProcessor(t, utils.Option{})

	result := rn.Process("cats and dogs but not fish")
	assert.Equal(t, "cats and dogs but not fish", result)
	assert.NotContains(t, result, "<")
}

func TestProcess_MarkdownIsNotStripped(t *testing.T) {
	rn := newTestRimeProcessor(t, utils.Option{})

	input := "**bold** text"
	result := rn.Process(input)
	assert.Contains(t, result, "**bold**")
}

func TestProcess_WhitespacePreserved(t *testing.T) {
	rn := newTestRimeProcessor(t, utils.Option{})

	// After centralization, whitespace cleanup is done upstream.
	// The provider processor should not collapse whitespace.
	input := "Hello    world"
	result := rn.Process(input)
	assert.Equal(t, input, result)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkProcess_SimpleText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewRimeProcessor(logger, utils.Option{})
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
	processor := NewRimeProcessor(logger, opts)
	text := "I like cats and dogs but not fish or snakes and birds"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}
