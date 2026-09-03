// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_channel_input "github.com/rapidaai/api/assistant-api/internal/channel/input"
	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/protos"
	"github.com/zaf/g711"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type bridgeState struct {
	outputTarget        internal_type.SIPRTPBridgeTarget
	inputCodecName      string
	outputCodecName     string
	forwardingTranscode func([]byte) []byte
}

type bridgeRecordingFrame struct {
	audio     []byte
	codecName string
}

// AudioProcessor owns SIP RTP codec conversion, buffering, pacing, bridge audio,
// and ringback generation.
type AudioProcessor struct {
	resampler  internal_type.AudioResampler
	rtpHandler rtpHandler
	pushInput  func(internal_type.Stream)
	record     func(...observability.Record) error

	inputBuffer  internal_channel_input.InputBuffer
	outputBuffer internal_telephony_output.FrameBuffer

	// bridgeMu orders ForwardUserAudio with DisconnectTransferMedia so outbound RTP is
	// not closed while a bridge send is in flight.
	bridgeMu         sync.Mutex
	bridge           atomic.Pointer[bridgeState]
	bridgeUserCh     chan bridgeRecordingFrame
	bridgeOperatorCh chan bridgeRecordingFrame

	ringtoneMu sync.RWMutex
	ringtone   []byte

	ringbackMu         sync.Mutex
	ringbackOffset     int
	ringbackFileOffset int

	ambientMixer internal_ambient.Mixer

	outputHealth        *internal_telephony_output.HealthStats
	droppedOutputFrames atomic.Uint64
	droppedBridgeFrames atomic.Uint64
	transferActive      atomic.Bool
}

func NewAudioProcessor(cfg AudioProcessorConfig) *AudioProcessor {
	p := &AudioProcessor{
		resampler:        cfg.Resampler,
		rtpHandler:       cfg.RTPHandler,
		pushInput:        cfg.PushInput,
		record:           cfg.Record,
		inputBuffer:      internal_channel_input.NewBytesInputBuffer(InputBufferThreshold * 2),
		outputBuffer:     internal_telephony_output.NewBytesFrameBuffer(BridgeOutputFrameSize * 8),
		bridgeUserCh:     make(chan bridgeRecordingFrame, AudioChannelSize),
		bridgeOperatorCh: make(chan bridgeRecordingFrame, AudioChannelSize),
		outputHealth:     internal_telephony_output.NewHealthStats(),
	}
	p.SetRingtone(cfg.Ringtone)
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Resampler:         cfg.Resampler,
		TargetAudioConfig: Linear8kConfig,
		FrameBytes:        MulawFrameSize * 2,
	})
	if err == nil {
		p.ambientMixer = ambientMixer
		if cfg.Ambient != nil {
			_ = p.ambientMixer.Configure(*cfg.Ambient)
		}
	}
	return p
}

func (p *AudioProcessor) currentCodec() *sip_runtime.Codec {
	if p.rtpHandler == nil {
		return &sip_runtime.CodecPCMU
	}
	codec := p.rtpHandler.GetCodec()
	if codec == nil {
		return &sip_runtime.CodecPCMU
	}
	return codec
}

func (p *AudioProcessor) SetRingtone(name string) {
	audio := LoadRingtoneBytes(name)
	p.ringtoneMu.Lock()
	p.ringtone = audio
	p.ringtoneMu.Unlock()
}

func (p *AudioProcessor) ConfigureAmbient(cfg internal_ambient.Config) error {
	if p.ambientMixer == nil {
		return nil
	}
	return p.ambientMixer.Configure(cfg)
}

func (p *AudioProcessor) ringtoneBytes() []byte {
	p.ringtoneMu.RLock()
	defer p.ringtoneMu.RUnlock()
	return p.ringtone
}

func (p *AudioProcessor) decodeProviderAudioToLinear8k(audioData []byte) []byte {
	return decodeG711ToLinear8k(audioData, p.currentCodec().Name)
}

func decodeG711ToLinear8k(audioData []byte, codecName string) []byte {
	if len(audioData) == 0 {
		return nil
	}
	switch codecName {
	case sip_runtime.CodecPCMA.Name:
		return g711.DecodeAlaw(audioData)
	default:
		return g711.DecodeUlaw(audioData)
	}
}

func (p *AudioProcessor) resampleProviderPCMToInternal(linearPCM8k []byte) ([]byte, error) {
	if len(linearPCM8k) == 0 {
		return nil, nil
	}
	return p.resampler.Resample(linearPCM8k, Linear8kConfig, Rapida16kConfig)
}

func (p *AudioProcessor) convertOutputAudio(audioData []byte) ([]byte, error) {
	convertedAudio, err := p.resampler.Resample(audioData, Rapida16kConfig, Mulaw8kConfig)
	if err != nil {
		return nil, err
	}
	if p.currentCodec().Name == sip_runtime.CodecPCMA.Name {
		convertedAudio = internal_audio.UlawToAlaw(convertedAudio)
	}
	return convertedAudio, nil
}

func (p *AudioProcessor) encodeAssistantOutputFrame(assistantPCM16k []byte) ([]byte, error) {
	if len(assistantPCM16k) == 0 {
		return nil, nil
	}
	convertedAudio, err := p.convertOutputAudio(assistantPCM16k)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
	}
	return p.applyAmbient(convertedAudio), nil
}

func (p *AudioProcessor) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	inputFrame := internal_telephony_media.InputAudioFrame{
		ReceivedAt: frame.ReceivedAt,
	}
	if len(frame.Audio) == 0 {
		return inputFrame, nil
	}
	linearPCM8k := p.decodeProviderAudioToLinear8k(frame.Audio)
	resampled, err := p.resampleProviderPCMToInternal(linearPCM8k)
	if err != nil {
		return inputFrame, fmt.Errorf("%w: %w", ErrProviderAudioConversionFailed, err)
	}
	if len(resampled) == 0 {
		return inputFrame, nil
	}
	inputFrame.BridgeAudio = resampled
	p.inputBuffer.Write(resampled)
	if pipelineAudio, ok := p.inputBuffer.DrainIfReady(InputBufferThreshold); ok {
		inputFrame.PipelineAudio = pipelineAudio
	}
	return inputFrame, nil
}

func (p *AudioProcessor) ProcessAssistantAudio(audio []byte, completed bool) error {
	if p.bridge.Load() != nil || p.transferActive.Load() {
		return nil
	}
	if len(audio) > 0 {
		p.outputBuffer.Write(audio)
	}
	if completed {
		p.Complete()
	}
	return nil
}

func (p *AudioProcessor) Complete() {
	p.outputBuffer.Complete(BridgeOutputFrameSize, 0)
}

func (p *AudioProcessor) nextAssistantPCM16kFrame() []byte {
	chunk, ok := p.outputBuffer.Next(BridgeOutputFrameSize)
	if !ok {
		return nil
	}
	return chunk
}

func (p *AudioProcessor) ClearOutputBuffer() {
	p.outputBuffer.Clear()
	p.rtpHandler.FlushAudioOut()
}

func (p *AudioProcessor) ClearInputBuffer() {
	p.inputBuffer.Clear()
}

func (p *AudioProcessor) OutputHealthSnapshot() internal_telephony_output.HealthSnapshot {
	if p.outputHealth == nil {
		return internal_telephony_output.HealthSnapshot{}
	}
	return p.outputHealth.Snapshot()
}

func (p *AudioProcessor) OnTickHealth(event internal_telephony_output.TickHealth) {
	if p.outputHealth != nil {
		p.outputHealth.OnTickHealth(event)
	}
}

func (p *AudioProcessor) OutputFrameDuration() time.Duration {
	return ChunkDuration
}

func (p *AudioProcessor) applyAmbient(chunk []byte) []byte {
	if p.ambientMixer == nil {
		return chunk
	}
	codec := p.currentCodec()
	var primaryPCM []byte
	if len(chunk) > 0 {
		switch codec.Name {
		case sip_runtime.CodecPCMA.Name:
			primaryPCM = g711.DecodeAlaw(chunk)
		default:
			primaryPCM = g711.DecodeUlaw(chunk)
		}
	}
	mixedPCM, err := p.ambientMixer.Mix(primaryPCM)
	if err != nil || len(mixedPCM) == 0 {
		return chunk
	}
	switch codec.Name {
	case sip_runtime.CodecPCMA.Name:
		return g711.EncodeAlaw(mixedPCM)
	default:
		return g711.EncodeUlaw(mixedPCM)
	}
}

func (p *AudioProcessor) NextOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	if p.transferActive.Load() {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	assistantPCM16k := p.nextAssistantPCM16kFrame()
	if len(assistantPCM16k) == 0 {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	return internal_telephony_media.AssistantOutputFrame{
		ProviderAudio: assistantPCM16k,
	}, true
}

func (p *AudioProcessor) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	if p.transferActive.Load() {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	if p.ambientMixer == nil || !p.ambientMixer.CurrentConfig().Enabled {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	return internal_telephony_media.AssistantOutputFrame{
		ProviderAudio: make([]byte, BridgeOutputFrameSize),
		Idle:          true,
	}, true
}

func (p *AudioProcessor) SetTransferActive(active bool) {
	p.transferActive.Store(active)
}

func (p *AudioProcessor) rtpOutputQueueFullError() error {
	dropped := p.droppedOutputFrames.Add(1)
	return fmt.Errorf("%w: dropped_frames_total=%d", ErrRTPOutputQueueFull, dropped)
}

func (p *AudioProcessor) IsBridgeActive() bool {
	return p.bridge.Load() != nil
}

func (p *AudioProcessor) ConnectTransferMedia(target internal_type.SIPRTPBridgeTarget, inputCodec *sip_runtime.Codec, outputCodecName string) {
	if target == nil {
		return
	}
	inputCodecName := sip_runtime.CodecPCMU.Name
	if inputCodec != nil && inputCodec.Name != "" {
		inputCodecName = inputCodec.Name
	}
	if outputCodecName == "" {
		outputCodecName = inputCodecName
	}
	state := &bridgeState{
		outputTarget:    target,
		inputCodecName:  inputCodecName,
		outputCodecName: outputCodecName,
	}
	if inputCodecName != outputCodecName {
		if inputCodecName == sip_runtime.CodecPCMA.Name && outputCodecName == sip_runtime.CodecPCMU.Name {
			state.forwardingTranscode = internal_audio.AlawToUlaw
		} else if inputCodecName == sip_runtime.CodecPCMU.Name && outputCodecName == sip_runtime.CodecPCMA.Name {
			state.forwardingTranscode = internal_audio.UlawToAlaw
		}
	}
	p.bridgeMu.Lock()
	p.bridge.Store(state)
	p.bridgeMu.Unlock()
}

// DisconnectTransferMedia returns only after in-flight bridge sends finish.
func (p *AudioProcessor) DisconnectTransferMedia() {
	p.bridgeMu.Lock()
	p.bridge.Store(nil)
	p.bridgeMu.Unlock()
}

// ForwardUserAudio routes caller audio to the bridge target and recorder.
func (p *AudioProcessor) ForwardUserAudio(audioData []byte) bool {
	p.bridgeMu.Lock()
	defer p.bridgeMu.Unlock()
	state := p.bridge.Load()
	if state == nil {
		return false
	}
	sourceAudio := audioData
	if state.forwardingTranscode != nil {
		audioData = state.forwardingTranscode(audioData)
	}
	if err := state.outputTarget.EnqueueAudio(audioData); err != nil {
		dropped := p.droppedBridgeFrames.Add(1)
		if p.record != nil && errors.Is(err, sip_runtime.ErrRTPOutputQueueFull) && (dropped == 1 || dropped%100 == 0) {
			_ = p.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "SIP bridge audio output queue full",
				Attributes: observability.Attributes{
					"component":            observability.ComponentCall.String(),
					"provider":             Provider,
					"reason":               "bridge_audio_out_full",
					"dropped_frames_total": fmt.Sprintf("%d", dropped),
				},
			})
		}
		return true
	}
	select {
	case p.bridgeUserCh <- bridgeRecordingFrame{audio: sourceAudio, codecName: state.inputCodecName}:
	default:
	}
	return true
}

// RecordTransferOperatorAudio queues transfer target audio for recording.
func (p *AudioProcessor) RecordTransferOperatorAudio(audio []byte) {
	codecName := sip_runtime.CodecPCMU.Name
	if state := p.bridge.Load(); state != nil && state.outputCodecName != "" {
		codecName = state.outputCodecName
	}
	select {
	case p.bridgeOperatorCh <- bridgeRecordingFrame{audio: audio, codecName: codecName}:
	default:
	}
}

// RunBridgeRecorder pushes bridge audio into the Talk pipeline.
func (p *AudioProcessor) RunBridgeRecorder(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-p.bridgeUserCh:
			if resampled, err := p.resampleBridgeRecordingFrame(frame); err == nil {
				p.pushInput(&protos.ConversationBridgeUserAudio{
					Audio: resampled,
					Time:  timestamppb.Now(),
				})
			}
		case frame := <-p.bridgeOperatorCh:
			if resampled, err := p.resampleBridgeRecordingFrame(frame); err == nil {
				p.pushInput(&protos.ConversationBridgeOperatorAudio{
					Audio: resampled,
					Time:  timestamppb.Now(),
				})
			}
		}
	}
}

func (p *AudioProcessor) resampleBridgeRecordingFrame(frame bridgeRecordingFrame) ([]byte, error) {
	linearPCM8k := decodeG711ToLinear8k(frame.audio, frame.codecName)
	if len(linearPCM8k) == 0 {
		return nil, nil
	}
	return p.resampler.Resample(linearPCM8k, Linear8kConfig, Rapida16kConfig)
}

func (p *AudioProcessor) StartRingback() {
	if p == nil || p.rtpHandler == nil {
		return
	}
	p.ringbackMu.Lock()
	p.ringbackOffset = 0
	p.ringbackFileOffset = 0
	p.ringbackMu.Unlock()
	p.rtpHandler.SetFallbackAudioSource(p.nextRingbackFrame)
}

func (p *AudioProcessor) StopRingback() {
	if p == nil || p.rtpHandler == nil {
		return
	}
	p.rtpHandler.ClearFallbackAudioSource()
}

func (p *AudioProcessor) nextRingbackFrame(frameSize int) []byte {
	if frameSize <= 0 {
		return nil
	}
	codec := p.currentCodec()
	ringtone := p.ringtoneBytes()
	useFile := len(ringtone) >= frameSize
	p.ringbackMu.Lock()
	defer p.ringbackMu.Unlock()
	var frame []byte
	if useFile {
		end := p.ringbackFileOffset + frameSize
		if end > len(ringtone) {
			p.ringbackFileOffset = 0
			end = frameSize
		}
		if end <= len(ringtone) {
			frame = ringtone[p.ringbackFileOffset:end]
			p.ringbackFileOffset = end
		}
	}
	if len(frame) == 0 {
		frame, p.ringbackOffset = internal_audio.GenerateRingbackMulawFrame(p.ringbackOffset)
	}
	if codec.Name == sip_runtime.CodecPCMA.Name {
		frame = internal_audio.UlawToAlaw(frame)
	}
	return frame
}
