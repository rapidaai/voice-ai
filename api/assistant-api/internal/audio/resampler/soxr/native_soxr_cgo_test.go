//go:build cgo

// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

import (
	"encoding/binary"
	"testing"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/stretchr/testify/require"
)

func TestQuickQualityUsesNativeSOXR(t *testing.T) {
	resampler := New(WithQuickQuality())
	source := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()

	output, err := resampler.Resample(generateLinear16Data(160), source, target)
	require.NoError(t, err)
	if len(output) == 0 {
		output, err = resampler.Resample(generateLinear16Data(160), source, target)
		require.NoError(t, err)
	}
	require.NotEmpty(t, output)
	require.Zero(t, len(output)%pcm16BytesPerSample)
	require.LessOrEqual(t, len(output), 640)

	engine, exists := resampler.engines.Load(ratePair{sourceRate: 8000, targetRate: 16000})
	require.True(t, exists)
	require.NotNil(t, engine.(*cachedEngine).soxrResampler)
}

func TestHighQualityUsesLiveKitStyleNativeSOXR(t *testing.T) {
	resampler := New(WithHighQuality())
	source := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()
	var output []byte
	writer, err := resampler.NewWriter(source, target, func(chunk []byte) error {
		output = append(output, chunk...)
		return nil
	})
	require.NoError(t, err)

	for range 4 {
		require.NoError(t, writer.Write(generateLinear16Data(160)))
	}

	require.NotEmpty(t, output)
	require.Zero(t, len(output)%640)
	engine, exists := resampler.engines.Load(ratePair{sourceRate: 8000, targetRate: 16000})
	require.True(t, exists)
	require.NotNil(t, engine.(*cachedEngine).soxrResampler)

	require.NoError(t, writer.Flush())
	_, exists = resampler.engines.Load(ratePair{sourceRate: 8000, targetRate: 16000})
	require.False(t, exists)
}

func TestNativeSOXRDoesNotPadFirstFrameWithZeroSamples(t *testing.T) {
	resampler, err := newNativePCM16Resampler(16000, 8000, nativeSOXRQualityQuick)
	require.NoError(t, err)
	t.Cleanup(resampler.Close)

	input := make([]byte, 441*pcm16BytesPerSample)
	for sampleIndex := range 441 {
		binary.LittleEndian.PutUint16(input[sampleIndex*pcm16BytesPerSample:], uint16(12000))
	}

	output, err := resampler.Resample(input)
	require.NoError(t, err)
	if len(output) == 0 {
		output, err = resampler.Resample(input)
		require.NoError(t, err)
	}
	require.Len(t, output, 220*pcm16BytesPerSample)
	require.NotZero(t, binary.LittleEndian.Uint16(output[:pcm16BytesPerSample]))
	require.NotZero(t, binary.LittleEndian.Uint16(output[len(output)-pcm16BytesPerSample:]))
}
