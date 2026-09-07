// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_linear

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

func newTestResampler(t testing.TB) *linearResampler {
	r := New(WithLogger(newTestLogger(t)))
	res, ok := r.(*linearResampler)
	require.True(t, ok)
	return res
}

func TestNewAppliesOptions(t *testing.T) {
	logger := newTestLogger(t)
	resampler, ok := New(WithLogger(logger)).(*linearResampler)
	require.True(t, ok)
	assert.Equal(t, logger, resampler.logger)
}

func TestNewConverterCreatesLinearConverter(t *testing.T) {
	converter := NewConverter(WithLogger(newTestLogger(t)))
	_, ok := converter.(*linearResampler)
	assert.True(t, ok)
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

// TestConvertToFloat32Samples tests conversion to float32
func TestConvertToFloat32Samples(t *testing.T) {
	resampler := newTestResampler(t)
	config := internal_audio.NewLinear16khzMonoAudioConfig()

	tests := []struct {
		name      string
		input     []int16
		checkFunc func(t *testing.T, samples []float32)
	}{
		{
			name:  "zero samples",
			input: []int16{0, 0, 0},
			checkFunc: func(t *testing.T, samples []float32) {
				assert.Len(t, samples, 3)
				for _, s := range samples {
					assert.Equal(t, float32(0), s)
				}
			},
		},
		{
			name:  "positive values",
			input: []int16{100, 200, 300},
			checkFunc: func(t *testing.T, samples []float32) {
				assert.Len(t, samples, 3)
				for i, s := range samples {
					assert.True(t, s > 0, "sample %d should be positive", i)
					assert.True(t, s <= 1.0, "sample %d should be within the unit range", i)
				}
			},
		},
		{
			name:  "negative values",
			input: []int16{-100, -200, -300},
			checkFunc: func(t *testing.T, samples []float32) {
				assert.Len(t, samples, 3)
				for i, s := range samples {
					assert.True(t, s < 0, "sample %d should be negative", i)
					assert.True(t, s >= -1.0, "sample %d should be within the unit range", i)
				}
			},
		},
		{
			name:  "max values",
			input: []int16{32767, -32768},
			checkFunc: func(t *testing.T, samples []float32) {
				assert.Len(t, samples, 2)
				assert.True(t, samples[0] <= 1.0)
				assert.True(t, samples[1] >= -1.0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := int16SliceToBytes(tt.input)
			samples, err := resampler.ConvertToFloat32Samples(data, config)
			require.NoError(t, err)
			tt.checkFunc(t, samples)
		})
	}
}

// TestConvertToByteSamples tests float32 to byte conversion for both formats
func TestConvertToByteSamples(t *testing.T) {
	resampler := newTestResampler(t)

	tests := []struct {
		name   string
		input  []float32
		config *protos.AudioConfig
	}{
		{"Linear16 mono", []float32{0.0, 0.5, -0.5, 1.0}, internal_audio.NewLinear16khzMonoAudioConfig()},
		{"Mulaw8 mono", []float32{0.0, 0.5, -0.5}, internal_audio.NewMulaw8khzMonoAudioConfig()},
		{"Empty samples", []float32{}, internal_audio.NewLinear16khzMonoAudioConfig()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := resampler.ConvertToByteSamples(tt.input, tt.config)
			require.NoError(t, err)
			if len(tt.input) == 0 {
				assert.Empty(t, data)
			} else {
				assert.NotEmpty(t, data)
			}
		})
	}
}

// TestGetAudioInfo tests audio information extraction
func TestGetAudioInfo(t *testing.T) {
	tests := []struct {
		name            string
		dataSize        int
		config          *protos.AudioConfig
		expectedSamples int
	}{
		{"16-bit 1sec@16kHz", 32000, internal_audio.NewLinear16khzMonoAudioConfig(), 16000},
		{"8-bit 1sec@8kHz", 8000, internal_audio.NewMulaw8khzMonoAudioConfig(), 8000},
		{"16-bit 2sec@16kHz", 64000, internal_audio.NewLinear16khzMonoAudioConfig(), 32000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.dataSize)
			info := internal_audio.GetAudioInfo(data, tt.config)
			assert.Equal(t, tt.config.SampleRate, info.SampleRate)
			assert.Equal(t, tt.expectedSamples, info.SamplesPerChannel)
			assert.Equal(t, tt.config.Channels, info.Channels)
			assert.Greater(t, info.DurationMs, 0.0)
		})
	}
}

// TestAudioInfoString tests the String() method
func TestAudioInfoString(t *testing.T) {
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	data := generateLinear16Data(16000)

	info := internal_audio.GetAudioInfo(data, config)
	infoStr := info.String()
	assert.NotEmpty(t, infoStr)
	assert.Contains(t, infoStr, "Linear16")
	assert.Contains(t, infoStr, "16000")
	assert.Contains(t, infoStr, "Mono")
}

// TestFormatConversions tests conversions between different audio formats
func TestFormatConversions(t *testing.T) {
	resampler := newTestResampler(t)
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

// TestComplexResample tests combining sample rate + format + channel changes
func TestComplexResample(t *testing.T) {
	resampler := newTestResampler(t)
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
	resampler := newTestResampler(t)
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

// TestConversionPreservesEnergy tests that conversions maintain signal energy
func TestConversionPreservesEnergy(t *testing.T) {
	resampler := newTestResampler(t)
	input := make([]int16, 1000)
	for i := range input {
		input[i] = int16(math.Sin(float64(i)*2*math.Pi/100) * 30000)
	}
	data := int16SliceToBytes(input)
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	samples, err := resampler.ConvertToFloat32Samples(data, config)
	require.NoError(t, err)
	byteData, err := resampler.ConvertToByteSamples(samples, config)
	require.NoError(t, err)
	assert.Equal(t, len(data), len(byteData))
	energyIn := calculateEnergy(int16SliceToFloat64(input))
	energyOut := calculateEnergy(int16SliceToFloat64(bytesToInt16Slice(byteData)))
	assert.InDelta(t, energyIn, energyOut, energyIn*0.05)
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	resampler := newTestResampler(t)
	tests := []struct {
		name     string
		testFunc func(*testing.T, *linearResampler)
	}{
		{"single sample", func(t *testing.T, r *linearResampler) {
			config := internal_audio.NewLinear16khzMonoAudioConfig()
			data := []byte{0x00, 0x01}
			samples, err := r.ConvertToFloat32Samples(data, config)
			require.NoError(t, err)
			assert.Len(t, samples, 1)
		}},
		{"very large data", func(t *testing.T, r *linearResampler) {
			config := internal_audio.NewLinear16khzMonoAudioConfig()
			data := generateLinear16Data(1000000)
			samples, err := r.ConvertToFloat32Samples(data, config)
			require.NoError(t, err)
			assert.Equal(t, 1000000, len(samples))
		}},
		{"alternating channels", func(t *testing.T, r *linearResampler) {
			source := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 2}
			target := &protos.AudioConfig{SampleRate: 16000, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
			data := generateLinear16Data(2000)
			result, err := r.Resample(data, source, target)
			require.NoError(t, err)
			assert.Equal(t, len(data)/2, len(result))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.testFunc(t, resampler) })
	}
}

// TestMuLawRoundTrip tests MuLaw encode/decode round trip
func TestMuLawRoundTrip(t *testing.T) {
	resampler := newTestResampler(t)
	tests := []struct {
		name   string
		values []int16
	}{
		{"zeros", []int16{0, 0, 0}},
		{"small values", []int16{100, 200, -100, -200}},
		{"medium values", []int16{1000, 2000, -1000, -2000, 5000, -5000}},
		{"large values", []int16{10000, 20000, -10000, -20000}},
		{"max values", []int16{32767, -32768, 32700, -32700}},
		{"mixed range", []int16{0, 100, -100, 1000, -1000, 10000, -10000, 32767, -32768}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			float64Samples := make([]float64, len(tt.values))
			for i, v := range tt.values {
				float64Samples[i] = float64(v) / 32768.0
			}
			muLawData, err := resampler.encodeFromFloat64(float64Samples, internal_audio.NewMulaw8khzMonoAudioConfig())
			require.NoError(t, err)

			// μ-law is 1 byte per sample
			assert.Equal(t, len(tt.values), len(muLawData), "mulaw data should be 1 byte per sample")

			decoded, err := resampler.decodeToFloat64(muLawData, internal_audio.NewMulaw8khzMonoAudioConfig())
			require.NoError(t, err)
			assert.Equal(t, len(float64Samples), len(decoded), "decoded sample count should match")

			// Check sign preservation and approximate value preservation
			for i := range float64Samples {
				if float64Samples[i] > 0 {
					assert.True(t, decoded[i] > 0, "positive sample %d should remain positive", i)
				} else if float64Samples[i] < 0 {
					assert.True(t, decoded[i] < 0, "negative sample %d should remain negative", i)
				} else {
					// Zero should be approximately zero (μ-law has quantization)
					assert.InDelta(t, 0.0, decoded[i], 0.01, "zero sample should remain near zero")
				}

				// μ-law is lossy, but should be within reasonable delta for non-zero values
				if float64Samples[i] != 0 {
					// Allow up to 10% error for μ-law quantization
					assert.InDelta(t, float64Samples[i], decoded[i], 0.1,
						"sample %d: original=%f, decoded=%f should be within 10%% tolerance",
						i, float64Samples[i], decoded[i])
				}
			}
		})
	}
}

// TestPCM16ToMuLaw8Conversion tests direct conversion between LINEAR16 and MuLaw8
func TestPCM16ToMuLaw8Conversion(t *testing.T) {
	resampler := newTestResampler(t)

	tests := []struct {
		name     string
		samples  []int16
		checkLen func(t *testing.T, inputLen, outputLen int)
	}{
		{
			name:    "simple conversion",
			samples: []int16{0, 100, -100, 1000, -1000, 10000, -10000},
			checkLen: func(t *testing.T, inputLen, outputLen int) {
				// PCM16 is 2 bytes per sample, MuLaw8 is 1 byte per sample
				assert.Equal(t, inputLen/2, outputLen, "mulaw output should be half the PCM16 input bytes")
			},
		},
		{
			name: "sine wave",
			samples: func() []int16 {
				const numSamples = 1000
				samples := make([]int16, numSamples)
				for i := 0; i < numSamples; i++ {
					samples[i] = int16(32767.0 * math.Sin(2*math.Pi*float64(i)/100))
				}
				return samples
			}(),
			checkLen: func(t *testing.T, inputLen, outputLen int) {
				assert.Equal(t, inputLen/2, outputLen)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert int16 to PCM16 bytes
			pcm16Data := int16SliceToBytes(tt.samples)

			// Convert LINEAR16 to MuLaw8 - use same sample rate to avoid resampling
			mulaw8Config := &protos.AudioConfig{
				SampleRate:  16000,
				AudioFormat: protos.AudioConfig_MuLaw8,
				Channels:    1,
			}
			linear16Config := &protos.AudioConfig{
				SampleRate:  16000,
				AudioFormat: protos.AudioConfig_LINEAR16,
				Channels:    1,
			}

			mulaw8Data, err := resampler.Resample(pcm16Data, linear16Config, mulaw8Config)
			require.NoError(t, err)
			tt.checkLen(t, len(pcm16Data), len(mulaw8Data))

			// Convert back to LINEAR16
			pcm16Restored, err := resampler.Resample(mulaw8Data, mulaw8Config, linear16Config)
			require.NoError(t, err)
			assert.Equal(t, len(pcm16Data), len(pcm16Restored), "restored data should have same length")

			// Convert back to int16 for comparison
			restoredSamples := bytesToInt16Slice(pcm16Restored)
			assert.Equal(t, len(tt.samples), len(restoredSamples))

			// Check that values are approximately preserved (μ-law is lossy)
			for i := range tt.samples {
				// Allow up to 10% error for μ-law quantization
				tolerance := float64(max(abs(tt.samples[i]), 100)) * 0.1
				assert.InDelta(t, tt.samples[i], restoredSamples[i], tolerance,
					"sample %d: original=%d, restored=%d", i, tt.samples[i], restoredSamples[i])
			}
		})
	}
}

// TestMuLaw8ByteSize tests that MuLaw8 produces correct byte sizes
func TestMuLaw8ByteSize(t *testing.T) {
	resampler := newTestResampler(t)

	tests := []struct {
		name          string
		numSamples    int
		expectedBytes int
	}{
		{"1 sample", 1, 1},
		{"10 samples", 10, 10},
		{"100 samples", 100, 100},
		{"1000 samples", 1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test samples
			samples := make([]float32, tt.numSamples)
			for i := range samples {
				samples[i] = float32(math.Sin(float64(i) * 0.1))
			}

			// Convert to MuLaw8
			mulaw8Config := internal_audio.NewMulaw8khzMonoAudioConfig()
			mulaw8Data, err := resampler.ConvertToByteSamples(samples, mulaw8Config)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBytes, len(mulaw8Data), "mulaw8 should be 1 byte per sample")

			// Verify we can decode back
			decodedSamples, err := resampler.ConvertToFloat32Samples(mulaw8Data, mulaw8Config)
			require.NoError(t, err)
			assert.Equal(t, tt.numSamples, len(decodedSamples))
		})
	}
}

// Helper functions for new tests
func abs(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}

// TestAudioInfoConsistency tests that AudioInfo values are consistent
func TestAudioInfoConsistency(t *testing.T) {
	config := internal_audio.NewLinear16khzMonoAudioConfig()
	secondOfAudio := make([]byte, 16000*2)
	info := internal_audio.GetAudioInfo(secondOfAudio, config)
	assert.Equal(t, uint32(16000), info.SampleRate)
	assert.Equal(t, int16(16000), int16(info.SamplesPerChannel))
	assert.Equal(t, uint32(1), info.Channels)
	assert.Equal(t, 2, info.BytesPerSample)
	assert.Equal(t, 32000, info.TotalBytes)
	assert.InDelta(t, 1000.0, info.DurationMs, 0.01)
}

// TestDifferentSampleRates tests conversion across supported sample rates
func TestDifferentSampleRates(t *testing.T) {
	resampler := newTestResampler(t)
	sampleRates := []uint32{8000, 16000, 24000}
	for _, sr := range sampleRates {
		config := &protos.AudioConfig{SampleRate: sr, AudioFormat: protos.AudioConfig_LINEAR16, Channels: 1}
		data := generateLinear16Data(int(sr))
		samples, err := resampler.ConvertToFloat32Samples(data, config)
		require.NoError(t, err)
		assert.Equal(t, int(sr), len(samples))
	}
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
