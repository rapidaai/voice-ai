// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_plivo

import (
	"fmt"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	internal_channel_input "github.com/rapidaai/api/assistant-api/internal/channel/input"
	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/zaf/g711"
)

// AudioProcessor handles audio conversion for Plivo mu-law 8kHz streams. Inbound
// provider audio (mu-law 8kHz) is resampled to linear16 16kHz for the pipeline,
// and assistant audio (linear16 16kHz) is resampled back to mu-law 8kHz for Plivo.
type AudioProcessor struct {
	logger commons.Logger

	resampler internal_type.AudioResampler

	plivoConfig      *protos.AudioConfig
	downstreamConfig *protos.AudioConfig

	inputBuffer internal_channel_input.InputBuffer

	outputBuffer       internal_telephony_output.FrameBuffer
	bridgeOutputBuffer internal_telephony_output.FrameBuffer

	silenceFrame []byte

	ambientMixer internal_ambient.Mixer

	outputHealth *internal_telephony_output.HealthStats
}

// NewAudioProcessor creates a new Plivo audio processor.
func NewAudioProcessor(logger commons.Logger) (*AudioProcessor, error) {
	resampler, err := internal_audio_resampler.GetResampler(logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResamplerCreateFailed, err)
	}

	audioProcessor := &AudioProcessor{
		logger:             logger,
		resampler:          resampler,
		plivoConfig:        internal_audio.NewMulaw8khzMonoAudioConfig(),
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
	if err != nil {
		logger.Warnf("ambient mixer unavailable, ambient audio disabled: %v", err)
	} else {
		audioProcessor.ambientMixer = ambientMixer
	}

	return audioProcessor, nil
}

// ConfigureAmbient applies an ambient audio configuration to the mixer, if any.
func (audioProcessor *AudioProcessor) ConfigureAmbient(ambientConfig internal_ambient.Config) error {
	if audioProcessor.ambientMixer == nil {
		return nil
	}
	return audioProcessor.ambientMixer.Configure(ambientConfig)
}

// ProcessProviderAudioFrame resamples an inbound mu-law 8kHz frame to linear16
// 16kHz, emitting bridge audio immediately and pipeline audio once buffered.
func (audioProcessor *AudioProcessor) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	inputFrame := internal_telephony_media.InputAudioFrame{
		ReceivedAt: frame.ReceivedAt,
	}
	if len(frame.Audio) == 0 {
		return inputFrame, nil
	}

	converted, err := audioProcessor.resampler.Resample(frame.Audio, audioProcessor.plivoConfig, audioProcessor.downstreamConfig)
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

// ProcessAssistantAudio resamples assistant linear16 16kHz audio into mu-law 8kHz
// output frames and, on completion, flushes both output and bridge buffers.
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
	return audioProcessor.resampler.Resample(audio, audioProcessor.downstreamConfig, audioProcessor.plivoConfig)
}

func (audioProcessor *AudioProcessor) createSilenceFrame() []byte {
	frame := make([]byte, OutputChunkSize)
	for i := range frame {
		frame[i] = MulawSilence
	}
	return frame
}

// OutputFrameDuration returns the pacing interval for outbound frames.
func (audioProcessor *AudioProcessor) OutputFrameDuration() time.Duration {
	return ChunkDuration
}

// OnTickHealth records a pacer tick for output health tracking.
func (audioProcessor *AudioProcessor) OnTickHealth(event internal_telephony_output.TickHealth) {
	if audioProcessor.outputHealth != nil {
		audioProcessor.outputHealth.OnTickHealth(event)
	}
}

// OutputHealthSnapshot returns the current output health snapshot.
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
	primaryPCM := g711.DecodeUlaw(frame)
	mixedPCM, err := audioProcessor.ambientMixer.Mix(primaryPCM)
	if err != nil || len(mixedPCM) == 0 {
		return frame
	}
	return g711.EncodeUlaw(mixedPCM)
}

// NextOutputFrame returns the next mu-law output frame if one is buffered.
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

// IdleOutputFrame returns a silence (or ambient) frame when no assistant audio
// is queued, keeping the outbound stream paced.
func (audioProcessor *AudioProcessor) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	providerAudio := audioProcessor.applyAmbient(nil)
	if len(providerAudio) == 0 {
		providerAudio = append([]byte(nil), audioProcessor.silenceFrame...)
	}
	return internal_telephony_media.AssistantOutputFrame{ProviderAudio: providerAudio}, true
}

// ClearOutputBuffer drops all buffered output audio (used on barge-in).
func (audioProcessor *AudioProcessor) ClearOutputBuffer() {
	audioProcessor.outputBuffer.Clear()
	audioProcessor.bridgeOutputBuffer.Clear()
}
