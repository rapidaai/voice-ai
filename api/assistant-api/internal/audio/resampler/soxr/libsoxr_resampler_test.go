// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

import (
	"encoding/binary"
	"math"
	"sync"
	"testing"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	resampling "github.com/tphakala/go-audio-resampler"
	"github.com/zaf/g711"
)

func newTestLogger(t testing.TB) commons.Logger {
	logger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
		commons.Name("resampler-test"),
		commons.Level("error"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = logger.Sync() })
	return logger
}

func newTestResampler(t testing.TB) *Resampler {
	return New(WithLogger(newTestLogger(t)), WithHighQuality())
}

func TestNewAudioResampler(t *testing.T) {
	resampler := New(WithLogger(newTestLogger(t)))
	assert.NotNil(t, resampler)
	assert.Equal(t, resampling.QualityHigh, resampler.quality)
}

func TestNewAudioResamplerAppliesQuickQuality(t *testing.T) {
	resampler := New(WithLogger(newTestLogger(t)), WithQuickQuality())
	assert.Equal(t, resampling.QualityQuick, resampler.quality)
}

func TestNewChunkAppliesOptions(t *testing.T) {
	resampler := NewChunk(WithLogger(newTestLogger(t)), WithQuickQuality())
	chunk, ok := resampler.(*chunkResampler)
	require.True(t, ok)
	assert.NotNil(t, chunk.logger)
	assert.Equal(t, resampling.QualityQuick, chunk.quality)
}

func TestRealtimeAudioResamplerEmitsSpeechWithoutSyntheticPadding(tester *testing.T) {
	resampler := New(WithLogger(newTestLogger(tester)), WithQuickQuality())
	source := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()

	for frameIndex := 0; frameIndex < 4; frameIndex++ {
		input := generateLinear16Data(160)
		output, err := resampler.Resample(input, source, target)
		require.NoError(tester, err)
		if frameIndex == 0 && len(output) == 0 {
			continue
		}
		require.NotEmpty(tester, output)
		require.Zero(tester, len(output)%pcm16BytesPerSample)
		require.LessOrEqual(tester, len(output), 640)
	}
}

func TestRealtimeAudioResamplerInstancesKeepIndependentState(tester *testing.T) {
	source := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()
	firstInput := generateLinear16Data(160)
	secondInput := generateLinear16Data(160)

	firstResampler := New(WithLogger(newTestLogger(tester)), WithQuickQuality())
	secondResampler := New(WithLogger(newTestLogger(tester)), WithQuickQuality())
	referenceResampler := New(WithLogger(newTestLogger(tester)), WithQuickQuality())

	_, err := firstResampler.Resample(firstInput, source, target)
	require.NoError(tester, err)
	actualOutput, err := secondResampler.Resample(secondInput, source, target)
	require.NoError(tester, err)
	expectedOutput, err := referenceResampler.Resample(secondInput, source, target)
	require.NoError(tester, err)
	require.NotEqual(tester, make([]byte, len(expectedOutput)), expectedOutput)
	require.Equal(tester, expectedOutput, actualOutput)
}

func TestRealtimeAudioResamplerRejectsUseAfterClose(t *testing.T) {
	resampler := New(WithQuickQuality())
	source := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()

	_, err := resampler.Resample(generateLinear16Data(160), source, target)
	require.NoError(t, err)
	resampler.Close()
	resampler.Close()

	_, err = resampler.Resample(generateLinear16Data(160), source, target)
	require.ErrorIs(t, err, ErrResamplerClosed)
}

func TestAudioResamplerRequiresAudioConfigs(t *testing.T) {
	_, err := New().Resample([]byte{0, 0}, nil, nil)
	require.ErrorIs(t, err, ErrAudioConfigRequired)
}

func TestRealtimeMuLawResamplingMatchesDecodedPCM(t *testing.T) {
	muLawSource := internal_audio.NewMulaw8khzMonoAudioConfig()
	linearSource := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()
	encoded := g711.EncodeUlaw(generateLinear16Data(160))

	encodedResampler := New(WithQuickQuality())
	linearResampler := New(WithQuickQuality())
	got, err := encodedResampler.Resample(encoded, muLawSource, target)
	require.NoError(t, err)
	want, err := linearResampler.Resample(g711.DecodeUlaw(encoded), linearSource, target)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestWriterMatchesResampleAndFlushesTail(t *testing.T) {
	source := internal_audio.NewLinear8khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()
	input := generateLinear16Data(160)
	reference := New(WithHighQuality())
	writerResampler := New(WithHighQuality())
	var expected []byte
	var actual []byte
	writer, err := writerResampler.NewWriter(source, target, func(output []byte) error {
		actual = append(actual, output...)
		return nil
	})
	require.NoError(t, err)

	for range 4 {
		output, err := reference.Resample(input, source, target)
		require.NoError(t, err)
		expected = append(expected, output...)
		require.NoError(t, writer.Write(input))
	}
	require.Equal(t, expected, actual)
	require.NoError(t, writer.Flush())
	require.GreaterOrEqual(t, len(actual), len(expected))
	require.Equal(t, expected, actual[:len(expected)])
}

func TestNewWriterRequiresSink(t *testing.T) {
	_, err := New().NewWriter(
		internal_audio.NewLinear8khzMonoAudioConfig(),
		internal_audio.NewLinear16khzMonoAudioConfig(),
		nil,
	)
	require.ErrorIs(t, err, ErrResampleSinkRequired)
}

// TestResampleNoConversion tests when source and target are identical
func TestResampleNoConversion(t *testing.T) {
	resampler := newTestResampler(t)
	data := []byte{0x00, 0x01, 0x02, 0x03}

	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear16khzMonoAudioConfig()

	result, err := resampler.Resample(data, source, target)
	require.NoError(t, err)
	assert.Equal(t, data, result, "no conversion should return same data")
}

// TestResampleEmptyData tests resampling empty audio data
func TestResampleEmptyData(t *testing.T) {
	resampler := newTestResampler(t)
	data := []byte{}

	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()

	result, err := resampler.Resample(data, source, target)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestResampleSampleRateConversion exercises various sample-rate conversions
func TestResampleSampleRateConversion(t *testing.T) {
	resampler := newTestResampler(t)

	tests := []struct {
		name     string
		sourceSR uint32
		targetSR uint32
	}{
		{"8kHz to 16kHz", 8000, 16000},
		{"16kHz to 8kHz", 16000, 8000},
		{"16kHz to 24kHz", 16000, 24000},
		{"24kHz to 16kHz", 24000, 16000},
		{"8kHz to 24kHz", 8000, 24000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &protos.AudioConfig{SampleRate: tt.sourceSR, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
			target := &protos.AudioConfig{SampleRate: tt.targetSR, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
			data := generateLinear16Data(2000)
			result, err := resampler.Resample(data, source, target)
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

// TestFormatConversions tests conversions between different audio formats
func TestFormatConversions(t *testing.T) {
	tests := []struct {
		name       string
		sourceFunc func() *protos.AudioConfig
		targetFunc func() *protos.AudioConfig
	}{
		{"Linear16 to MuLaw8", internal_audio.NewLinear16khzMonoAudioConfig, internal_audio.NewMulaw8khzMonoAudioConfig},
		{"MuLaw8 to Linear16", internal_audio.NewMulaw8khzMonoAudioConfig, internal_audio.NewLinear16khzMonoAudioConfig},
		{"Linear16 to Linear16", internal_audio.NewLinear16khzMonoAudioConfig, internal_audio.NewLinear16khzMonoAudioConfig},
		{"MuLaw8 to MuLaw8", internal_audio.NewMulaw8khzMonoAudioConfig, internal_audio.NewMulaw8khzMonoAudioConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resampler := New(WithLogger(newTestLogger(t)), WithQuickQuality())
			source := tt.sourceFunc()
			target := tt.targetFunc()
			var data []byte
			if source.AudioFormat == protos.AudioConfig_LINEAR16 {
				data = generateLinear16Data(1000)
			} else {
				data = generateMuLawData(1000)
			}
			result, err := resampler.Resample(data, source, target)
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}

// TestChannelConversion tests mono/stereo conversions
func TestChannelConversion(t *testing.T) {
	resampler := newTestResampler(t)
	tests := []struct {
		name           string
		sourceChannels uint32
		targetChannels uint32
		inputSamples   int
		expectedGrowth float64
	}{
		{"Mono to Stereo", 1, 2, 1000, 2.0},
		{"Stereo to Mono", 2, 1, 2000, 0.5},
		{"Mono to Mono", 1, 1, 1000, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: tt.sourceChannels}
			target := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: tt.targetChannels}
			data := generateLinear16Data(tt.inputSamples)
			result, err := resampler.Resample(data, source, target)
			require.NoError(t, err)
			expectedSize := int(float64(len(data)) * tt.expectedGrowth)
			assert.Equal(t, expectedSize, len(result), "unexpected result size")
		})
	}
}

func TestChannelConversionRejectsInvalidFrames(t *testing.T) {
	resampler := newTestResampler(t)
	tests := []struct {
		name           string
		data           []byte
		sourceChannels uint32
		targetChannels uint32
		expectedError  error
	}{
		{name: "incomplete mono sample", data: []byte{1}, sourceChannels: 1, targetChannels: 2, expectedError: ErrInvalidPCM16ChannelFrame},
		{name: "incomplete stereo frame", data: make([]byte, 6), sourceChannels: 2, targetChannels: 1, expectedError: ErrInvalidPCM16ChannelFrame},
		{name: "unsupported channel count", data: make([]byte, 12), sourceChannels: 3, targetChannels: 1, expectedError: ErrUnsupportedChannelConversion},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: test.sourceChannels}
			target := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: test.targetChannels}

			_, err := resampler.Resample(test.data, source, target)
			require.ErrorIs(t, err, test.expectedError)
		})
	}
}

// TestComplexResample tests combining sample rate + format + channel changes
func TestComplexResample(t *testing.T) {
	resampler := New(WithLogger(newTestLogger(t)), WithQuickQuality())
	source := &protos.AudioConfig{SampleRate: 8000, AudioFormat: protos.AudioConfig_MuLaw8, Channels: 1}
	target := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 2}
	data := generateMuLawData(8000)
	result, err := resampler.Resample(data, source, target)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Greater(t, len(result), len(data))
}

// TestConcurrentResampling tests thread-safe resampling
func TestConcurrentResampling(t *testing.T) {
	const numGoroutines = 50
	const dataSize = 10000
	var wg sync.WaitGroup
	var errorCount int
	var mu sync.Mutex
	resampler := New(WithLogger(newTestLogger(t)), WithQuickQuality())
	source := internal_audio.NewLinear16khzMonoAudioConfig()
	target := internal_audio.NewLinear24khzMonoAudioConfig()
	data := generateLinear16Data(dataSize)
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			result, err := resampler.Resample(data, source, target)
			if err != nil || len(result) == 0 {
				mu.Lock()
				errorCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Zero(t, errorCount)
}

// =====================
// Helper functions
// =====================

func generateLinear16Data(samples int) []byte {
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		sample := int16(math.Sin(float64(i)*2*math.Pi/1000) * 30000)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return data
}

func generateMuLawData(samples int) []byte {
	data := make([]byte, samples)
	for i := 0; i < samples; i++ {
		data[i] = byte((i * 7) % 256)
	}
	return data
}

func int16SliceToBytes(samples []int16) []byte {
	data := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(sample))
	}
	return data
}

func bytesToInt16Slice(data []byte) []int16 {
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	return samples
}

func int16SliceToFloat64(samples []int16) []float64 {
	result := make([]float64, len(samples))
	for i, s := range samples {
		result[i] = float64(s) / 32768.0
	}
	return result
}

func calculateEnergy(samples []float64) float64 {
	var sum float64
	for _, s := range samples {
		sum += s * s
	}
	return math.Sqrt(sum / float64(len(samples)))
}
