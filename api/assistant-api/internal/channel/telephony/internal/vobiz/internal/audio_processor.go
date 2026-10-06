// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_vobiz

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
	"github.com/zaf/g711"
)

// AudioProcessor handles audio conversion for vobiz mulaw 8kHz streams.
// Mirrors the Twilio processor since the codec path (mulaw 8k <-> linear16 16k)
// is identical.
type AudioProcessor struct {
	logger           commons.Logger
	providerConfig   *protos.AudioConfig
	downstreamConfig *protos.AudioConfig

	resampler          internal_type.AudioResampler
	inputWriter        internal_type.AudioStreamResampler
	outputWriter       internal_type.AudioStreamResampler
	providerInputAudio []byte

	inputBuffer        internal_channel_input.InputBuffer
	outputBuffer       internal_telephony_output.FrameBuffer
	bridgeOutputBuffer internal_telephony_output.FrameBuffer

	silenceFrame []byte
	ambientMixer internal_ambient.Mixer
}

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
		providerConfig:     internal_audio.NewMulaw8khzMonoAudioConfig(),
		downstreamConfig:   internal_audio.NewLinear16khzMonoAudioConfig(),
		inputBuffer:        internal_channel_input.NewBytesInputBuffer(InputBufferThreshold * 2),
		outputBuffer:       internal_telephony_output.NewBytesFrameBuffer(OutputChunkSize * 8),
		bridgeOutputBuffer: internal_telephony_output.NewBytesFrameBuffer(BridgeOutputFrameSize * 8),
	}
	inputWriter, err := inputResampler.NewWriter(audioProcessor.providerConfig, audioProcessor.downstreamConfig, func(output []byte) error {
		audioProcessor.providerInputAudio = append(audioProcessor.providerInputAudio, output...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	audioProcessor.inputWriter = inputWriter
	outputWriter, err := outputResampler.NewWriter(audioProcessor.downstreamConfig, audioProcessor.providerConfig, func(output []byte) error {
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
		TargetAudioConfig: internal_audio.NewLinear8khzMonoAudioConfig(),
		FrameBytes:        OutputChunkSize * 2,
	})
	if err == nil {
		audioProcessor.ambientMixer = ambientMixer
	}

	return audioProcessor, nil
}

func (audioProcessor *AudioProcessor) ConfigureAmbient(cfg internal_ambient.Config) error {
	if audioProcessor.ambientMixer == nil {
		return nil
	}
	return audioProcessor.ambientMixer.Configure(cfg)
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
		converted, err = audioProcessor.resampler.Resample(frame.Audio, audioProcessor.providerConfig, audioProcessor.downstreamConfig)
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
		audioProcessor.outputBuffer.Complete(OutputChunkSize, MulawSilence)
		audioProcessor.bridgeOutputBuffer.Complete(BridgeOutputFrameSize, 0)
	}
	return nil
}

func (audioProcessor *AudioProcessor) convertOutputAudio(audio []byte) ([]byte, error) {
	return audioProcessor.resampler.Resample(audio, audioProcessor.downstreamConfig, audioProcessor.providerConfig)
}

func (audioProcessor *AudioProcessor) createSilenceFrame() []byte {
	frame := make([]byte, OutputChunkSize)
	for i := range frame {
		frame[i] = MulawSilence
	}
	return frame
}

func (audioProcessor *AudioProcessor) OutputFrameDuration() time.Duration {
	return ChunkDuration
}

func (audioProcessor *AudioProcessor) applyAmbient(chunk []byte) []byte {
	if audioProcessor.ambientMixer == nil {
		return chunk
	}
	primaryPCM := g711.DecodeUlaw(chunk)
	mixedPCM, err := audioProcessor.ambientMixer.Mix(primaryPCM)
	if err != nil || len(mixedPCM) == 0 {
		return chunk
	}
	return g711.EncodeUlaw(mixedPCM)
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

func (audioProcessor *AudioProcessor) OutputDrained() bool {
	return audioProcessor.outputBuffer.Len() == 0
}

func (audioProcessor *AudioProcessor) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	providerAudio := audioProcessor.applyAmbient(nil)
	if len(providerAudio) == 0 {
		providerAudio = append([]byte(nil), audioProcessor.silenceFrame...)
	}
	return internal_telephony_media.AssistantOutputFrame{ProviderAudio: providerAudio}, true
}

func (audioProcessor *AudioProcessor) ClearOutputBuffer() {
	if audioProcessor.outputWriter != nil {
		_ = audioProcessor.outputWriter.Flush()
	}
	audioProcessor.outputBuffer.Clear()
	audioProcessor.bridgeOutputBuffer.Clear()
}
