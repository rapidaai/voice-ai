// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_exotel

import (
	"fmt"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	internal_channel_input "github.com/rapidaai/api/assistant-api/internal/channel/input"
	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

// AudioProcessor owns Exotel linear16 8kHz media conversion and buffering.
type AudioProcessor struct {
	logger             commons.Logger
	resampler          internal_type.AudioResampler
	inputWriter        internal_type.AudioStreamResampler
	outputWriter       internal_type.AudioStreamResampler
	providerInputAudio []byte
	exotelConfig       *protos.AudioConfig
	downstreamConfig   *protos.AudioConfig
	inputBuffer        internal_channel_input.InputBuffer
	outputBuffer       internal_telephony_output.FrameBuffer
	bridgeOutputBuffer internal_telephony_output.FrameBuffer
	silenceFrame       []byte
	ambientMixer       internal_ambient.Mixer
}

// NewAudioProcessor creates a new Exotel audio processor
func NewAudioProcessor(logger commons.Logger) (*AudioProcessor, error) {
	inputResampler := resampler_soxr.New(
		resampler_soxr.WithLogger(logger),
		resampler_soxr.WithHighQuality(),
	)
	outputResampler := resampler_soxr.New(
		resampler_soxr.WithLogger(logger),
		resampler_soxr.WithHighQuality(),
	)
	audioProcessor := &AudioProcessor{
		logger:             logger,
		resampler:          inputResampler,
		exotelConfig:       internal_audio.NewLinear8khzMonoAudioConfig(),
		downstreamConfig:   internal_audio.NewLinear16khzMonoAudioConfig(),
		inputBuffer:        internal_channel_input.NewBytesInputBuffer(InputBufferThreshold * 2),
		outputBuffer:       internal_telephony_output.NewBytesFrameBuffer(OutputChunkSize * 8),
		bridgeOutputBuffer: internal_telephony_output.NewBytesFrameBuffer(BridgeOutputFrameSize * 8),
	}
	inputWriter, err := inputResampler.NewWriter(audioProcessor.exotelConfig, audioProcessor.downstreamConfig, func(output []byte) error {
		audioProcessor.providerInputAudio = append(audioProcessor.providerInputAudio, output...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	audioProcessor.inputWriter = inputWriter
	outputWriter, err := outputResampler.NewWriter(audioProcessor.downstreamConfig, audioProcessor.exotelConfig, func(output []byte) error {
		audioProcessor.outputBuffer.Write(output)
		return nil
	})
	if err != nil {
		return nil, err
	}
	audioProcessor.outputWriter = outputWriter
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame()
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Resampler:         resampler_soxr.NewChunk(resampler_soxr.WithLogger(logger), resampler_soxr.WithHighQuality()),
		TargetAudioConfig: audioProcessor.exotelConfig,
		FrameBytes:        OutputChunkSize,
	})
	if err == nil {
		audioProcessor.ambientMixer = ambientMixer
	}

	return audioProcessor, nil
}

func (audioProcessor *AudioProcessor) ConfigureAmbient(ambientConfig internal_ambient.Config) error {
	if audioProcessor.ambientMixer == nil {
		return nil
	}
	return audioProcessor.ambientMixer.Configure(ambientConfig)
}

func (audioProcessor *AudioProcessor) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	inputFrame := internal_telephony_media.InputAudioFrame{
		ReceivedAt: frame.ReceivedAt,
	}
	if len(frame.Audio) == 0 {
		return inputFrame, nil
	}

	var converted []byte
	if audioProcessor.inputWriter != nil {
		audioProcessor.providerInputAudio = audioProcessor.providerInputAudio[:0]
		if err := audioProcessor.inputWriter.Write(frame.Audio); err != nil {
			return inputFrame, fmt.Errorf("%w: %w", ErrProviderAudioConversionFailed, err)
		}
		converted = append(converted, audioProcessor.providerInputAudio...)
		audioProcessor.providerInputAudio = audioProcessor.providerInputAudio[:0]
	} else {
		var err error
		converted, err = audioProcessor.resampler.Resample(frame.Audio, audioProcessor.exotelConfig, audioProcessor.downstreamConfig)
		if err != nil {
			return inputFrame, fmt.Errorf("%w: %w", ErrProviderAudioConversionFailed, err)
		}
	}

	inputFrame.BridgeAudio = converted
	if len(converted) > 0 {
		audioProcessor.inputBuffer.Write(converted)
		if pipelineAudio, ok := audioProcessor.inputBuffer.DrainIfReady(InputBufferThreshold); ok {
			inputFrame.PipelineAudio = pipelineAudio
		}
	}
	return inputFrame, nil
}

func (audioProcessor *AudioProcessor) ProcessAssistantAudio(audio []byte, completed bool) error {
	if len(audio) > 0 {
		if audioProcessor.outputWriter != nil {
			if err := audioProcessor.outputWriter.Write(audio); err != nil {
				return fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
			}
		} else {
			converted, err := audioProcessor.convertOutputAudio(audio)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
			}
			audioProcessor.outputBuffer.Write(converted)
		}
		audioProcessor.bridgeOutputBuffer.Write(audio)
	}
	if completed {
		if audioProcessor.outputWriter != nil {
			if err := audioProcessor.outputWriter.Flush(); err != nil {
				return fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
			}
		}
		audioProcessor.outputBuffer.Complete(OutputChunkSize, 0x00)
		audioProcessor.bridgeOutputBuffer.Complete(BridgeOutputFrameSize, 0x00)
	}
	return nil
}

func (audioProcessor *AudioProcessor) convertOutputAudio(audio []byte) ([]byte, error) {
	return audioProcessor.resampler.Resample(audio, audioProcessor.downstreamConfig, audioProcessor.exotelConfig)
}

func (audioProcessor *AudioProcessor) createSilenceFrame() []byte {
	return make([]byte, OutputChunkSize)
}

func (audioProcessor *AudioProcessor) OutputFrameDuration() time.Duration {
	return ChunkDuration
}

func (audioProcessor *AudioProcessor) applyAmbient(frame []byte) []byte {
	if audioProcessor.ambientMixer == nil {
		return frame
	}
	mixed, err := audioProcessor.ambientMixer.Mix(frame)
	if err != nil {
		return frame
	}
	return mixed
}

func (audioProcessor *AudioProcessor) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	providerAudio, ok := audioProcessor.outputBuffer.Next(OutputChunkSize)
	if !ok {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	bridgeAudio, _ := audioProcessor.bridgeOutputBuffer.Next(BridgeOutputFrameSize)
	return internal_telephony_media.AssistantOutputFrame{
		ProviderAudio: audioProcessor.applyAmbient(providerAudio),
		BridgeAudio:   bridgeAudio,
	}, true
}

func (audioProcessor *AudioProcessor) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	providerAudio := audioProcessor.applyAmbient(nil)
	if len(providerAudio) == 0 {
		providerAudio = append([]byte(nil), audioProcessor.silenceFrame...)
	}
	return internal_telephony_media.AssistantOutputFrame{ProviderAudio: providerAudio}, true
}

func (audioProcessor *AudioProcessor) ClearOutputBuffer() {
	audioProcessor.outputBuffer.Clear()
	audioProcessor.bridgeOutputBuffer.Clear()
}
