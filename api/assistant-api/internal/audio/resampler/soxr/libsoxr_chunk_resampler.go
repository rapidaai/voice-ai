// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

import (
	"fmt"
	"math"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	resampling "github.com/tphakala/go-audio-resampler"
)

// chunkResampler preserves each independent buffer's duration.
type chunkResampler struct {
	logger  commons.Logger
	quality resampling.QualityPreset
}

// NewChunk creates a stateless resampler for independent audio buffers.
func NewChunk(optionFunctions ...Option) internal_type.AudioResampler {
	configuration := options{quality: defaultQuality}
	for _, option := range optionFunctions {
		if option != nil {
			option(&configuration)
		}
	}
	return &chunkResampler{logger: configuration.logger, quality: configuration.quality}
}

func (r *chunkResampler) Resample(data []byte, source, target *protos.AudioConfig) ([]byte, error) {
	if source == nil || target == nil {
		return nil, ErrAudioConfigRequired
	}
	if len(data) == 0 {
		return []byte{}, nil
	}
	if source.SampleRate == target.SampleRate &&
		source.Channels == target.Channels &&
		source.AudioFormat == target.AudioFormat {
		return data, nil
	}

	expectedBytes := expectedOutputBytes(data, source, target)

	ops := &Resampler{}
	pcm := data
	if source.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = ops.convertToLinear16(data, source)
		if err != nil {
			return nil, err
		}
	}

	if source.SampleRate != target.SampleRate {
		out, err := resampling.ResampleMono(
			pcm16ToFloat64(pcm),
			float64(source.SampleRate),
			float64(target.SampleRate),
			r.quality,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrResamplingFailed, err)
		}
		pcm = float64ToPCM16(out)
	}

	if source.Channels != target.Channels {
		var err error
		pcm, err = ops.convertChannels(pcm, source.Channels, target.Channels)
		if err != nil {
			return nil, err
		}
	}

	if target.AudioFormat != protos.AudioConfig_LINEAR16 {
		var err error
		pcm, err = ops.convertFromLinear16(pcm, target)
		if err != nil {
			return nil, err
		}
	}

	return fitLength(pcm, expectedBytes), nil
}

func expectedOutputBytes(data []byte, source, target *protos.AudioConfig) int {
	sourceFrameSize := internal_audio.FrameSize(source)
	targetFrameSize := internal_audio.FrameSize(target)
	if sourceFrameSize == 0 || targetFrameSize == 0 || source.GetSampleRate() == 0 {
		return len(data)
	}
	sourceFrames := len(data) / sourceFrameSize
	targetFrames := int(math.Round(float64(sourceFrames) * float64(target.GetSampleRate()) / float64(source.GetSampleRate())))
	return targetFrames * targetFrameSize
}

func fitLength(data []byte, expected int) []byte {
	if expected <= 0 || len(data) == expected {
		return data
	}
	if len(data) > expected {
		return data[:expected]
	}
	padded := make([]byte, expected)
	copy(padded, data)
	return padded
}
