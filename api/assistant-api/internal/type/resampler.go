// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"fmt"

	"github.com/rapidaai/protos"
)

type AudioInfo struct {
	SampleRate        uint32
	Format            protos.AudioConfig_AudioFormat
	Channels          uint32
	SamplesPerChannel int
	BytesPerSample    int
	TotalBytes        int
	DurationMs        float64
}

type AudioConverter interface {
	ConvertToFloat32Samples(data []byte, config *protos.AudioConfig) ([]float32, error)
	ConvertToByteSamples(samples []float32, config *protos.AudioConfig) ([]byte, error)
}

type AudioResampler interface {
	Resample(data []byte, source, target *protos.AudioConfig) ([]byte, error)
	Close()
}

// AudioResampleSink consumes writer output before Write or Flush returns.
// Implementations must copy data retained after the callback returns.
type AudioResampleSink func([]byte) error

// AudioStreamResampler owns stream state for one source to target conversion.
type AudioStreamResampler interface {
	Write(data []byte) error
	Flush() error
	Close()
}

// String returns a formatted string representation of AudioInfo
func (info AudioInfo) String() string {
	formatName := "Unknown"
	switch info.Format {
	case protos.AudioConfig_LINEAR16:
		formatName = "Linear16"
	case protos.AudioConfig_MuLaw8:
		formatName = "μ-law 8-bit"
	}

	channelName := "Mono"
	if info.Channels == 2 {
		channelName = "Stereo"
	} else if info.Channels > 2 {
		channelName = fmt.Sprintf("%d channels", info.Channels)
	}

	return fmt.Sprintf("Audio: %s, %d Hz, %s, %.2f ms (%d samples, %d bytes)",
		formatName, info.SampleRate, channelName, info.DurationMs,
		info.SamplesPerChannel, info.TotalBytes)
}
