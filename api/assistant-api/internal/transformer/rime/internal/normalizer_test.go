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

func newTestRimeNormalizer(t *testing.T, opts utils.Option) *rimeNormalizer {
	t.Helper()
	logger := testutil.NewTestLogger()
	normalizer := NewRimeNormalizer(logger, opts)
	rn, ok := normalizer.(*rimeNormalizer)
	require.True(t, ok, "expected *rimeNormalizer type")
	return rn
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewRimeNormalizer(t *testing.T) {
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
			rn := newTestRimeNormalizer(t, tt.opts)
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
// Normalize Tests
// =============================================================================

func TestNormalize_EmptyString(t *testing.T) {
	rn := newTestRimeNormalizer(t, utils.Option{})
	result := rn.Normalize("")
	assert.Equal(t, "", result)
}

func TestNormalize_PlainTextPassthrough(t *testing.T) {
	rn := newTestRimeNormalizer(t, utils.Option{})

	// Rime has no XML escaping, so plain text passes through unchanged
	input := "Hello world. This is pre-normalized text."
	result := rn.Normalize(input)
	assert.Equal(t, input, result)
}

func TestNormalize_NoXMLEscaping(t *testing.T) {
	rn := newTestRimeNormalizer(t, utils.Option{})

	// Rime does NOT support SSML, so no XML escaping should be applied
	input := "Tom & Jerry said a < b > c"
	result := rn.Normalize(input)
	assert.Equal(t, input, result, "Rime normalizer should not escape XML entities")
}

func TestNormalize_ConjunctionBreaks_CustomMsSyntax(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but",
		"speaker.conjunction.break":      uint64(500),
	}
	rn := newTestRimeNormalizer(t, opts)

	result := rn.Normalize("cats and dogs but not fish")
	// Rime uses custom <ms> syntax, NOT standard SSML
	assert.Contains(t, result, "and <500> ")
	assert.Contains(t, result, "but <500> ")
	// Should NOT contain standard SSML break tags
	assert.NotContains(t, result, `<break time=`)
}

func TestNormalize_ConjunctionBreaks_DifferentDurations(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and",
		"speaker.conjunction.break":      uint64(250),
	}
	rn := newTestRimeNormalizer(t, opts)

	result := rn.Normalize("cats and dogs")
	assert.Contains(t, result, "and <250> ")
}

func TestNormalize_NoConjunctionBreaksWhenNotConfigured(t *testing.T) {
	rn := newTestRimeNormalizer(t, utils.Option{})

	result := rn.Normalize("cats and dogs but not fish")
	assert.Equal(t, "cats and dogs but not fish", result)
	assert.NotContains(t, result, "<")
}

func TestNormalize_MarkdownIsNotStripped(t *testing.T) {
	rn := newTestRimeNormalizer(t, utils.Option{})

	input := "**bold** text"
	result := rn.Normalize(input)
	assert.Contains(t, result, "**bold**")
}

func TestNormalize_WhitespacePreserved(t *testing.T) {
	rn := newTestRimeNormalizer(t, utils.Option{})

	// After centralization, whitespace cleanup is done upstream.
	// The provider normalizer should not collapse whitespace.
	input := "Hello    world"
	result := rn.Normalize(input)
	assert.Equal(t, input, result)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNormalize_SimpleText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewRimeNormalizer(logger, utils.Option{})
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
	normalizer := NewRimeNormalizer(logger, opts)
	text := "I like cats and dogs but not fish or snakes and birds"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}
