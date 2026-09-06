// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_vonage

import (
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

// AudioProcessor handles audio for Vonage linear16 16kHz streams.
type AudioProcessor struct {
	logger commons.Logger

	resampler   internal_type.AudioResampler
	audioConfig *protos.AudioConfig

	inputBuffer        internal_channel_input.InputBuffer
	outputBuffer       internal_telephony_output.FrameBuffer
	bridgeOutputBuffer internal_telephony_output.FrameBuffer

	silenceFrame []byte
	ambientMixer internal_ambient.Mixer
	outputHealth *internal_telephony_output.HealthStats
}

// NewAudioProcessor creates a new Vonage audio processor
func NewAudioProcessor(logger commons.Logger) (*AudioProcessor, error) {
	resampler := resampler_soxr.New(
		resampler_soxr.WithLogger(logger),
		resampler_soxr.WithHighQuality(),
	)

	audioProcessor := &AudioProcessor{
		logger:             logger,
		resampler:          resampler,
		audioConfig:        internal_audio.NewLinear16khzMonoAudioConfig(),
		inputBuffer:        internal_channel_input.NewBytesInputBuffer(InputBufferThreshold * 2),
		outputBuffer:       internal_telephony_output.NewBytesFrameBuffer(OutputChunkSize * 8),
		bridgeOutputBuffer: internal_telephony_output.NewBytesFrameBuffer(OutputChunkSize * 8),
		outputHealth:       internal_telephony_output.NewHealthStats(),
	}
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame()
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Resampler:         audioProcessor.resampler,
		TargetAudioConfig: audioProcessor.audioConfig,
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

	providerAudio := append([]byte(nil), frame.Audio...)
	inputFrame.BridgeAudio = providerAudio
	audioProcessor.inputBuffer.Write(providerAudio)
	if pipelineAudio, ok := audioProcessor.inputBuffer.DrainIfReady(InputBufferThreshold); ok {
		inputFrame.PipelineAudio = pipelineAudio
	}
	return inputFrame, nil
}

func (audioProcessor *AudioProcessor) ProcessAssistantAudio(audio []byte, completed bool) error {
	if len(audio) > 0 {
		audioProcessor.outputBuffer.Write(audio)
		audioProcessor.bridgeOutputBuffer.Write(audio)
	}
	if completed {
		audioProcessor.outputBuffer.Complete(OutputChunkSize, 0x00)
		audioProcessor.bridgeOutputBuffer.Complete(OutputChunkSize, 0x00)
	}
	return nil
}

func (audioProcessor *AudioProcessor) createSilenceFrame() []byte {
	return make([]byte, OutputChunkSize)
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
	bridgeAudio, _ := audioProcessor.bridgeOutputBuffer.Next(OutputChunkSize)
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
