// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_linear

import (
	"encoding/binary"
	"fmt"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/zaf/g711"
)

// linearResampler converts formats, channel counts, and sample rates.
type linearResampler struct {
	logger commons.Logger
}

type Option func(*linearResampler)

// WithLogger sets the resampler logger.
func WithLogger(logger commons.Logger) Option {
	return func(resampler *linearResampler) {
		resampler.logger = logger
	}
}

// New creates a linear-interpolation audio resampler.
func New(options ...Option) internal_type.AudioResampler {
	return newLinearResampler(options...)
}

// NewConverter creates an audio sample converter.
func NewConverter(options ...Option) internal_type.AudioConverter {
	return newLinearResampler(options...)
}

func newLinearResampler(options ...Option) *linearResampler {
	resampler := &linearResampler{}
	for _, option := range options {
		if option != nil {
			option(resampler)
		}
	}
	return resampler
}

// Resample converts audio data from source format to target format
func (r *linearResampler) Resample(data []byte, source, target *protos.AudioConfig) ([]byte, error) {

	// Early return only if sample rate, channels, AND format all match
	if source.SampleRate == target.SampleRate &&
		source.Channels == target.Channels &&
		source.AudioFormat == target.AudioFormat {
		return data, nil
	}

	samples, err := r.decodeToFloat64(data, source)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	if source.SampleRate != target.SampleRate {
		samples = r.resampleFloat64(samples, source.SampleRate, target.SampleRate)
	}

	if source.Channels != target.Channels {
		samples = r.convertChannels(samples, source.Channels, target.Channels)
	}

	result, err := r.encodeFromFloat64(samples, target)
	if err != nil {
		return nil, fmt.Errorf("failed to encode audio: %w", err)
	}

	return result, nil
}

// ConvertToFloat32Samples converts byte audio data to float32 samples
func (r *linearResampler) ConvertToFloat32Samples(data []byte, config *protos.AudioConfig) ([]float32, error) {
	float64Samples, err := r.decodeToFloat64(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode to float64: %w", err)
	}

	float32Samples := make([]float32, len(float64Samples))
	for i, sample := range float64Samples {
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		float32Samples[i] = float32(sample)
	}

	return float32Samples, nil
}

// ConvertToByteSamples converts float32 samples to byte audio data
func (r *linearResampler) ConvertToByteSamples(samples []float32, config *protos.AudioConfig) ([]byte, error) {
	float64Samples := make([]float64, len(samples))
	for i, sample := range samples {
		float64Samples[i] = float64(sample)
	}
	return r.encodeFromFloat64(float64Samples, config)
}

// -------------------- Decode / Encode --------------------

// decodeToFloat64 converts audio bytes to unit-range float64 samples.
func (r *linearResampler) decodeToFloat64(data []byte, config *protos.AudioConfig) ([]float64, error) {
	switch config.GetAudioFormat() {
	case protos.AudioConfig_LINEAR16:
		return r.decodePCM16ToFloat64(data), nil
	case protos.AudioConfig_MuLaw8:
		return r.decodeMuLawToFloat64(data), nil
	default:
		return nil, fmt.Errorf("unsupported input format: %v", config.GetAudioFormat())
	}
}

// encodeFromFloat64 converts unit-range float64 samples to audio bytes.
func (r *linearResampler) encodeFromFloat64(samples []float64, config *protos.AudioConfig) ([]byte, error) {
	switch config.GetAudioFormat() {
	case protos.AudioConfig_LINEAR16:
		return r.encodeFloat64ToPCM16(samples), nil
	case protos.AudioConfig_MuLaw8:
		return r.encodeFloat64ToMuLaw(samples), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %v", config.GetAudioFormat())
	}
}

// -------------------- PCM16 --------------------

func (r *linearResampler) decodePCM16ToFloat64(data []byte) []float64 {
	samples := make([]float64, len(data)/2)
	for i := 0; i < len(samples); i++ {
		sample := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		samples[i] = float64(sample) / 32768.0
	}
	return samples
}

func (r *linearResampler) encodeFloat64ToPCM16(samples []float64) []byte {
	data := make([]byte, len(samples)*2)
	const maxInt16 = 32767.0

	for i, sample := range samples {
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		value := int16(sample * maxInt16)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(value))
	}
	return data
}

// -------------------- μ-law (G.711) --------------------

// mu-law (8-bit) → linear PCM16 → float64
func (r *linearResampler) decodeMuLawToFloat64(data []byte) []float64 {
	pcm := g711.DecodeUlaw(data)

	// g711.DecodeUlaw returns PCM16 bytes (2 bytes per sample)
	numSamples := len(pcm) / 2
	samples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sample := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		samples[i] = float64(sample) / 32768.0
	}
	return samples
}

// float64 → linear PCM16 → mu-law (8-bit)
func (r *linearResampler) encodeFloat64ToMuLaw(samples []float64) []byte {
	pcmBytes := make([]byte, len(samples)*2)

	for i, sample := range samples {
		// Clamp
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}

		pcm := int16(sample * 32767.0)
		binary.LittleEndian.PutUint16(
			pcmBytes[i*2:i*2+2],
			uint16(pcm),
		)
	}

	return g711.EncodeUlaw(pcmBytes)
}

// -------------------- Resampling & Channels --------------------

// resampleFloat64 performs linear interpolation resampling
func (r *linearResampler) resampleFloat64(samples []float64, sourceSR, targetSR uint32) []float64 {
	if sourceSR == targetSR {
		return samples
	}

	ratio := float64(sourceSR) / float64(targetSR)
	outputLength := int(float64(len(samples)) / ratio)
	resampled := make([]float64, outputLength)

	for i := 0; i < outputLength; i++ {
		sourceIndex := float64(i) * ratio
		index := int(sourceIndex)
		frac := sourceIndex - float64(index)

		if index >= len(samples)-1 {
			resampled[i] = samples[len(samples)-1]
		} else {
			resampled[i] = samples[index]*(1-frac) + samples[index+1]*frac
		}
	}

	return resampled
}

// convertChannels handles mono/stereo conversion
func (r *linearResampler) convertChannels(samples []float64, sourceChannels, targetChannels uint32) []float64 {
	if sourceChannels == targetChannels {
		return samples
	}

	if sourceChannels == 1 && targetChannels == 2 {
		stereo := make([]float64, len(samples)*2)
		for i, s := range samples {
			stereo[i*2] = s
			stereo[i*2+1] = s
		}
		return stereo
	}

	if sourceChannels == 2 && targetChannels == 1 {
		mono := make([]float64, len(samples)/2)
		for i := 0; i < len(mono); i++ {
			mono[i] = (samples[i*2] + samples[i*2+1]) / 2.0
		}
		return mono
	}

	return samples
}
