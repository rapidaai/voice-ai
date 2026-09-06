// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

import (
	"fmt"
	"sync"
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

const (
	chunkDuration         = 20 * time.Millisecond
	defaultFrameSize      = 160
	maxOptimalFrameSize   = 320
	linear16BytesPerMs    = 32
	bridgeOutputFrameSize = linear16BytesPerMs * 20
	inputBufferThreshold  = linear16BytesPerMs * 40
)

type AudioProcessorConfig struct {
	AsteriskConfig   *protos.AudioConfig
	DownstreamConfig *protos.AudioConfig
	SilenceByte      byte
	FrameSize        int
	Ambient          *internal_ambient.Config
}

// AudioProcessor handles audio conversion between Asterisk and downstream formats.
type AudioProcessor struct {
	logger commons.Logger

	resampler        internal_type.AudioResampler
	asteriskConfig   *protos.AudioConfig
	downstreamConfig *protos.AudioConfig
	silenceByte      byte
	optimalFrameSize int
	stateMu          sync.RWMutex

	inputBuffer internal_channel_input.InputBuffer

	outputBuffer       internal_telephony_output.FrameBuffer
	bridgeOutputBuffer internal_telephony_output.FrameBuffer

	silenceFrame []byte

	ambientMixer internal_ambient.Mixer

	xoffActive bool
	xoffMu     sync.Mutex

	outputHealth *internal_telephony_output.HealthStats
}

func NewAudioProcessor(logger commons.Logger, cfg AudioProcessorConfig) (*AudioProcessor, error) {
	frameSize := cfg.FrameSize
	if frameSize <= 0 {
		frameSize = defaultFrameSize
	}

	audioProcessor := &AudioProcessor{
		logger: logger,
		resampler: resampler_soxr.New(
			resampler_soxr.WithLogger(logger),
			resampler_soxr.WithHighQuality(),
		),
		asteriskConfig:     cfg.AsteriskConfig,
		downstreamConfig:   cfg.DownstreamConfig,
		silenceByte:        cfg.SilenceByte,
		optimalFrameSize:   frameSize,
		inputBuffer:        internal_channel_input.NewBytesInputBuffer(inputBufferThreshold * 2),
		outputBuffer:       internal_telephony_output.NewBytesFrameBuffer(frameSize * 8),
		bridgeOutputBuffer: internal_telephony_output.NewBytesFrameBuffer(bridgeOutputFrameSize * 8),
		outputHealth:       internal_telephony_output.NewHealthStats(),
	}
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame(frameSize, audioProcessor.silenceByte)

	if cfg.AsteriskConfig != nil {
		switch cfg.AsteriskConfig.GetAudioFormat() {
		case protos.AudioConfig_MuLaw8:
			ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
				Resampler:         audioProcessor.resampler,
				TargetAudioConfig: internal_audio.NewLinear8khzMonoAudioConfig(),
				FrameBytes:        frameSize * 2,
			})
			if err == nil {
				audioProcessor.ambientMixer = ambientMixer
			}
		case protos.AudioConfig_LINEAR16:
			ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
				Resampler:         audioProcessor.resampler,
				TargetAudioConfig: cfg.AsteriskConfig,
				FrameBytes:        frameSize,
			})
			if err == nil {
				audioProcessor.ambientMixer = ambientMixer
			}
		}
	}
	if cfg.Ambient != nil {
		_ = audioProcessor.ConfigureAmbient(*cfg.Ambient)
	}
	return audioProcessor, nil
}

func (audioProcessor *AudioProcessor) ConfigureAmbient(ambientConfig internal_ambient.Config) error {
	if audioProcessor.ambientMixer == nil {
		return nil
	}
	return audioProcessor.ambientMixer.Configure(ambientConfig)
}

func (audioProcessor *AudioProcessor) SetOptimalFrameSize(size int) {
	size = normalizeOptimalFrameSize(size)
	if size == 0 {
		return
	}
	audioProcessor.stateMu.Lock()
	audioProcessor.optimalFrameSize = size
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame(size, audioProcessor.silenceByte)
	audioProcessor.stateMu.Unlock()
}

func (audioProcessor *AudioProcessor) GetOptimalFrameSize() int {
	audioProcessor.stateMu.RLock()
	defer audioProcessor.stateMu.RUnlock()
	return audioProcessor.optimalFrameSize
}

func (audioProcessor *AudioProcessor) GetDownstreamConfig() *protos.AudioConfig {
	return audioProcessor.downstreamConfig
}

func (audioProcessor *AudioProcessor) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	inputFrame := internal_telephony_media.InputAudioFrame{
		ReceivedAt: frame.ReceivedAt,
	}
	if len(frame.Audio) == 0 {
		return inputFrame, nil
	}

	converted, err := audioProcessor.resampler.Resample(frame.Audio, audioProcessor.asteriskConfig, audioProcessor.downstreamConfig)
	if err != nil {
		return inputFrame, fmt.Errorf("audio conversion from asterisk format to downstream failed: %w", err)
	}

	inputFrame.BridgeAudio = converted
	audioProcessor.inputBuffer.Write(converted)
	if pipelineAudio, ok := audioProcessor.inputBuffer.DrainIfReady(inputBufferThreshold); ok {
		inputFrame.PipelineAudio = pipelineAudio
	}
	return inputFrame, nil
}

func (audioProcessor *AudioProcessor) ProcessAssistantAudio(audio []byte, completed bool) error {
	frameSize, _ := audioProcessor.getOutputState()
	if len(audio) > 0 {
		converted, err := audioProcessor.convertOutputAudio(audio)
		if err != nil {
			return fmt.Errorf("audio conversion from downstream to asterisk format failed: %w", err)
		}
		audioProcessor.outputBuffer.Write(converted)
		audioProcessor.bridgeOutputBuffer.Write(audio)
	}
	if completed {
		audioProcessor.outputBuffer.Complete(frameSize, audioProcessor.silenceByte)
		audioProcessor.bridgeOutputBuffer.Complete(bridgeOutputFrameSize, 0)
	}
	return nil
}

func (audioProcessor *AudioProcessor) convertOutputAudio(audio []byte) ([]byte, error) {
	return audioProcessor.resampler.Resample(audio, audioProcessor.downstreamConfig, audioProcessor.asteriskConfig)
}

func (audioProcessor *AudioProcessor) ClearOutputBuffer() {
	audioProcessor.outputBuffer.Clear()
	audioProcessor.bridgeOutputBuffer.Clear()
}

func (audioProcessor *AudioProcessor) createSilenceFrame(frameSize int, silenceByte byte) []byte {
	frameSize = normalizeFrameAllocationSize(frameSize)
	frame := make([]byte, frameSize)
	for i := range frame {
		frame[i] = silenceByte
	}
	return frame
}

func normalizeFrameAllocationSize(frameSize int) int {
	if frameSize <= 0 || frameSize > maxOptimalFrameSize {
		frameSize = defaultFrameSize
	}
	return frameSize
}

func normalizeOptimalFrameSize(frameSize int) int {
	if frameSize <= 0 || frameSize > maxOptimalFrameSize {
		return 0
	}
	return frameSize
}

func (audioProcessor *AudioProcessor) OutputFrameDuration() time.Duration {
	return chunkDuration
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

func (audioProcessor *AudioProcessor) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	if audioProcessor.IsXOFF() {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}

	frameSize, _ := audioProcessor.getOutputState()
	providerAudio, ok := audioProcessor.outputBuffer.Next(frameSize)
	if !ok {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	bridgeAudio, _ := audioProcessor.bridgeOutputBuffer.Next(bridgeOutputFrameSize)
	return internal_telephony_media.AssistantOutputFrame{
		ProviderAudio: audioProcessor.applyAmbient(providerAudio),
		BridgeAudio:   bridgeAudio,
	}, true
}

func (audioProcessor *AudioProcessor) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	if audioProcessor.IsXOFF() {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}

	_, silenceFrame := audioProcessor.getOutputState()
	providerAudio := audioProcessor.applyAmbient(nil)
	if len(providerAudio) == 0 {
		providerAudio = append([]byte(nil), silenceFrame...)
	}
	return internal_telephony_media.AssistantOutputFrame{ProviderAudio: providerAudio}, true
}

func (audioProcessor *AudioProcessor) applyAmbient(frame []byte) []byte {
	if audioProcessor.ambientMixer == nil {
		return frame
	}
	switch audioProcessor.asteriskConfig.GetAudioFormat() {
	case protos.AudioConfig_MuLaw8:
		primaryPCM := g711.DecodeUlaw(frame)
		mixedPCM, err := audioProcessor.ambientMixer.Mix(primaryPCM)
		if err != nil || len(mixedPCM) == 0 {
			return frame
		}
		return g711.EncodeUlaw(mixedPCM)
	case protos.AudioConfig_LINEAR16:
		mixedPCM, err := audioProcessor.ambientMixer.Mix(frame)
		if err != nil {
			return frame
		}
		return mixedPCM
	default:
		return frame
	}
}

func (audioProcessor *AudioProcessor) SetXOFF() {
	audioProcessor.xoffMu.Lock()
	audioProcessor.xoffActive = true
	audioProcessor.xoffMu.Unlock()
}

func (audioProcessor *AudioProcessor) SetXON() {
	audioProcessor.xoffMu.Lock()
	audioProcessor.xoffActive = false
	audioProcessor.xoffMu.Unlock()
}

func (audioProcessor *AudioProcessor) IsXOFF() bool {
	audioProcessor.xoffMu.Lock()
	defer audioProcessor.xoffMu.Unlock()
	return audioProcessor.xoffActive
}

func (audioProcessor *AudioProcessor) getOutputState() (int, []byte) {
	audioProcessor.stateMu.RLock()
	defer audioProcessor.stateMu.RUnlock()
	frameSize := audioProcessor.optimalFrameSize
	if frameSize <= 0 {
		frameSize = defaultFrameSize
	}
	return frameSize, audioProcessor.silenceFrame
}
