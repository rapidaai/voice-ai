// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_audio

import "github.com/rapidaai/protos"

func NewMulaw8khzMonoAudioConfig() *protos.AudioConfig {
	return &protos.AudioConfig{
		SampleRate:  8000,
		AudioFormat: protos.AudioConfig_MuLaw8,
		Channels:    1,
	}
}

func NewLinear24khzMonoAudioConfig() *protos.AudioConfig {
	return &protos.AudioConfig{
		SampleRate:  24000,
		AudioFormat: protos.AudioConfig_LINEAR16,
		Channels:    1,
	}
}

func NewLinear16khzMonoAudioConfig() *protos.AudioConfig {
	return &protos.AudioConfig{
		SampleRate:  16000,
		AudioFormat: protos.AudioConfig_LINEAR16,
		Channels:    1,
	}
}

func NewLinear8khzMonoAudioConfig() *protos.AudioConfig {
	return &protos.AudioConfig{
		SampleRate:  8000,
		AudioFormat: protos.AudioConfig_LINEAR16,
		Channels:    1,
	}
}
