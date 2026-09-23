// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sarvam_internal

import (
	"testing"

	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSarvamNormalizer(t *testing.T) {
	tests := []struct {
		name         string
		opts         utils.Option
		expectedLang string
	}{
		{
			name:         "default language (Hindi)",
			opts:         utils.Option{},
			expectedLang: "hi-IN",
		},
		{
			name:         "explicit language",
			opts:         utils.Option{"speaker.language": "ta-IN"},
			expectedLang: "ta-IN",
		},
		{
			name:         "empty language defaults to hi-IN",
			opts:         utils.Option{"speaker.language": ""},
			expectedLang: "hi-IN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := commons.NewApplicationLogger()
			require.NoError(t, err)
			normalizer := NewSarvamNormalizer(logger, tt.opts)
			require.NotNil(t, normalizer)
			sn, ok := normalizer.(*sarvamNormalizer)
			require.True(t, ok)
			assert.Equal(t, tt.expectedLang, sn.language)
		})
	}
}

func TestNormalize_Passthrough(t *testing.T) {
	logger := testutil.NewTestLogger()
	normalizer := NewSarvamNormalizer(logger, utils.Option{})

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple text", "Hello world"},
		{"hindi text", "नमस्ते दुनिया"},
		{"markdown preserved", "# Header **bold**"},
		{"xml chars preserved", "Tom & Jerry a < b > c"},
		{"whitespace preserved", "Hello    world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			assert.Equal(t, tt.input, result)
		})
	}
}

func TestNormalize_NoSSML(t *testing.T) {
	logger := testutil.NewTestLogger()
	normalizer := NewSarvamNormalizer(logger, utils.Option{})

	result := normalizer.Normalize("Tom & Jerry")
	assert.NotContains(t, result, "&amp;")
	assert.NotContains(t, result, "<break")
}

func BenchmarkNormalize(b *testing.B) {
	logger := testutil.NewTestLogger()
	normalizer := NewSarvamNormalizer(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}
