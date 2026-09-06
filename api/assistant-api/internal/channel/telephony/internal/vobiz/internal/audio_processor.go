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

	resampler internal_type.AudioResampler

	inputBuffer        internal_channel_input.InputBuffer
	outputBuffer       internal_telephony_output.FrameBuffer
	bridgeOutputBuffer internal_telephony_output.FrameBuffer

	silenceFrame []byte
	ambientMixer internal_ambient.Mixer
	outputHealth *internal_telephony_output.HealthStats
}

func NewAudioProcessor(logger commons.Logger) (*AudioProcessor, error) {
	resampler := resampler_soxr.New(
		resampler_soxr.WithLogger(logger),
		resampler_soxr.WithHighQuality(),
	)

	audioProcessor := &AudioProcessor{
		logger:             logger,
		resampler:          resampler,
		providerConfig:     internal_audio.NewMulaw8khzMonoAudioConfig(),
		downstreamConfig:   internal_audio.NewLinear16khzMonoAudioConfig(),
		inputBuffer:        internal_channel_input.NewBytesInputBuffer(InputBufferThreshold * 2),
		outputBuffer:       internal_telephony_output.NewBytesFrameBuffer(OutputChunkSize * 8),
		bridgeOutputBuffer: internal_telephony_output.NewBytesFrameBuffer(BridgeOutputFrameSize * 8),
		outputHealth:       internal_telephony_output.NewHealthStats(),
	}
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame()
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Resampler:         audioProcessor.resampler,
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

	converted, err := audioProcessor.resampler.Resample(frame.Audio, audioProcessor.providerConfig, audioProcessor.downstreamConfig)
	if err != nil {
		return inputFrame, fmt.Errorf("%w: %w", ErrProviderAudioConversionFailed, err)
	}

	inputFrame.BridgeAudio = converted
	audioProcessor.inputBuffer.Write(converted)
	if pipelineAudio, ok := audioProcessor.inputBuffer.DrainIfReady(InputBufferThreshold); ok {
		inputFrame.PipelineAudio = pipelineAudio
	}
	return inputFrame, nil
}

func (audioProcessor *AudioProcessor) ProcessAssistantAudio(audio []byte, completed bool) error {
	if len(audio) > 0 {
		converted, err := audioProcessor.convertOutputAudio(audio)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
		}
		audioProcessor.outputBuffer.Write(converted)
		audioProcessor.bridgeOutputBuffer.Write(audio)
	}
	if completed {
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

func (audioProcessor *AudioProcessor) OnTickHealth(event internal_telephony_output.TickHealth) {
	if audioProcessor.outputHealth != nil {
		audioProcessor.outputHealth.OnTickHealth(event)
	}
}

func (audioProcessor *AudioProcessor) OutputHealthSnapshot() internal_telephony_output.HealthSnapshot {
	if audioProcessor.outputHealth == nil {
		return internal_telephony_output.HealthSnapshot{}
	}
	return audioProcessor.outputHealth.Snapshot()
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
