//go:build !cgo

// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

import (
	"testing"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/stretchr/testify/require"
)

func TestQuickQualityFallsBackWithoutCGO(t *testing.T) {
	resampler := New(WithQuickQuality())
	source := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()

	output, err := resampler.Resample(generateLinear16Data(160), source, target)
	require.NoError(t, err)
	require.NotEmpty(t, output)

	engine, exists := resampler.engines.Load(ratePair{sourceRate: 8000, targetRate: 16000})
	require.True(t, exists)
	require.Nil(t, engine.(*cachedEngine).soxrResampler)
}
