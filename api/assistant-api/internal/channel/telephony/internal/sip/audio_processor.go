// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/protos"
	"github.com/zaf/g711"
	"google.golang.org/protobuf/proto"
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

type audioResamplers struct {
	provider       internal_type.AudioResampler
	assistant      internal_type.AudioResampler
	bridgeUser     internal_type.AudioResampler
	bridgeOperator internal_type.AudioResampler
	ambient        internal_type.AudioResampler
}

type audioResampleWriters struct {
	providerInput  internal_type.AudioStreamResampler
	assistant      internal_type.AudioStreamResampler
	bridgeUser     internal_type.AudioStreamResampler
	bridgeOperator internal_type.AudioStreamResampler
}

// AudioProcessor owns SIP RTP codec conversion, buffering, pacing, bridge audio,
// and ringback generation.
type AudioProcessor struct {
	resamplers audioResamplers
	writers    audioResampleWriters
	rtpHandler rtpHandler
	record     func(...observability.Record) error

	providerOutputBuffer internal_telephony_output.FrameBuffer
	bridgeOutputBuffer   internal_telephony_output.FrameBuffer
	outputMu             sync.Mutex
	providerInputMu      sync.Mutex
	providerInputAudio   []byte
	providerReceivedAt   time.Time

	// bridgeMu orders ForwardUserAudio with DisconnectTransferMedia so outbound RTP is
	// not closed while a bridge send is in flight.
	bridgeMu            sync.Mutex
	bridge              atomic.Pointer[bridgeState]
	bridgeUserCh        chan bridgeRecordingFrame
	bridgeOperatorCh    chan bridgeRecordingFrame
	bridgeUserAudio     []byte
	bridgeOperatorAudio []byte

	ringtoneMu sync.RWMutex
	ringtone   []byte

	ringbackMu         sync.Mutex
	ringbackOffset     int
	ringbackFileOffset int
	ringbackActive     atomic.Bool

	ambientMixer internal_ambient.Mixer

	droppedBridgeFrames atomic.Uint64
	transferActive      atomic.Bool
}

func NewAudioProcessor(config AudioProcessorConfig) *AudioProcessor {
	providerResampler := resampler_soxr.New(
		resampler_soxr.WithLogger(config.Logger),
		resampler_soxr.WithHighQuality(),
	)
	assistantResampler := resampler_soxr.New(
		resampler_soxr.WithLogger(config.Logger),
		resampler_soxr.WithHighQuality(),
	)
	bridgeUserResampler := resampler_soxr.New(
		resampler_soxr.WithLogger(config.Logger),
		resampler_soxr.WithHighQuality(),
	)
	bridgeOperatorResampler := resampler_soxr.New(
		resampler_soxr.WithLogger(config.Logger),
		resampler_soxr.WithHighQuality(),
	)
	processor := &AudioProcessor{
		resamplers: audioResamplers{
			provider:       providerResampler,
			assistant:      assistantResampler,
			bridgeUser:     bridgeUserResampler,
			bridgeOperator: bridgeOperatorResampler,
			ambient: resampler_soxr.NewChunk(
				resampler_soxr.WithLogger(config.Logger),
				resampler_soxr.WithHighQuality(),
			),
		},
		rtpHandler:           config.RTPHandler,
		record:               config.Record,
		providerOutputBuffer: internal_telephony_output.NewBytesFrameBuffer(MulawFrameSize * 8),
		bridgeOutputBuffer:   internal_telephony_output.NewBytesFrameBuffer(BridgeOutputFrameSize * 8),
		bridgeUserCh:         make(chan bridgeRecordingFrame, BridgeRecordingChannelCapacity),
		bridgeOperatorCh:     make(chan bridgeRecordingFrame, BridgeRecordingChannelCapacity),
	}
	if writer, err := providerResampler.NewWriter(Linear8kConfig, Rapida16kConfig, func(output []byte) error {
		processor.providerInputAudio = append(processor.providerInputAudio, output...)
		return nil
	}); err == nil {
		processor.writers.providerInput = writer
	}
	if writer, err := assistantResampler.NewWriter(Rapida16kConfig, Mulaw8kConfig, func(providerAudio []byte) error {
		if processor.currentCodec().Name == sip_runtime.CodecPCMA.Name {
			providerAudio = internal_audio.UlawToAlaw(providerAudio)
		}
		processor.providerOutputBuffer.Write(providerAudio)
		return nil
	}); err == nil {
		processor.writers.assistant = writer
	}
	if writer, err := bridgeUserResampler.NewWriter(Linear8kConfig, Rapida16kConfig, func(output []byte) error {
		processor.bridgeUserAudio = append(processor.bridgeUserAudio, output...)
		return nil
	}); err == nil {
		processor.writers.bridgeUser = writer
	}
	if writer, err := bridgeOperatorResampler.NewWriter(Linear8kConfig, Rapida16kConfig, func(output []byte) error {
		processor.bridgeOperatorAudio = append(processor.bridgeOperatorAudio, output...)
		return nil
	}); err == nil {
		processor.writers.bridgeOperator = writer
	}
	processor.SetRingtone(config.Ringtone)
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Resampler:         processor.resamplers.ambient,
		TargetAudioConfig: Linear8kConfig,
		FrameBytes:        MulawFrameSize * 2,
	})
	if err == nil {
		processor.ambientMixer = ambientMixer
		if config.Ambient != nil {
			_ = processor.ambientMixer.Configure(*config.Ambient)
		}
	}
	return processor
}

// Close releases resamplers owned by this audio processor.
func (processor *AudioProcessor) Close() {
	if processor == nil {
		return
	}
	if processor.writers.providerInput != nil {
		processor.writers.providerInput.Close()
	}
	if processor.writers.assistant != nil {
		processor.writers.assistant.Close()
	}
	if processor.writers.bridgeUser != nil {
		processor.writers.bridgeUser.Close()
	}
	if processor.writers.bridgeOperator != nil {
		processor.writers.bridgeOperator.Close()
	}
	processor.resamplers.provider.Close()
	processor.resamplers.assistant.Close()
	processor.resamplers.bridgeUser.Close()
	processor.resamplers.bridgeOperator.Close()
	processor.resamplers.ambient.Close()
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
	return p.resamplers.provider.Resample(linearPCM8k, Linear8kConfig, Rapida16kConfig)
}

func (p *AudioProcessor) ProcessProviderAudioFrame(frame internal_telephony_media.ProviderAudioFrame) (internal_telephony_media.InputAudioFrame, error) {
	inputFrame := internal_telephony_media.InputAudioFrame{
		ReceivedAt: frame.ReceivedAt,
	}
	if len(frame.Audio) == 0 {
		return inputFrame, nil
	}
	linearPCM8k := p.decodeProviderAudioToLinear8k(frame.Audio)
	p.providerInputMu.Lock()
	defer p.providerInputMu.Unlock()
	if p.providerReceivedAt.IsZero() {
		p.providerReceivedAt = frame.ReceivedAt
	}
	receivedAt := p.providerReceivedAt

	var resampled []byte
	if p.writers.providerInput != nil {
		p.providerInputAudio = p.providerInputAudio[:0]
		if err := p.writers.providerInput.Write(linearPCM8k); err != nil {
			return inputFrame, fmt.Errorf("%w: %w", ErrProviderAudioConversionFailed, err)
		}
		resampled = append([]byte(nil), p.providerInputAudio...)
	} else {
		var err error
		resampled, err = p.resampleProviderPCMToInternal(linearPCM8k)
		if err != nil {
			return inputFrame, fmt.Errorf("%w: %w", ErrProviderAudioConversionFailed, err)
		}
	}
	if len(resampled) == 0 {
		return inputFrame, nil
	}
	if !receivedAt.IsZero() {
		inputFrame.ReceivedAt = receivedAt
	}
	p.providerReceivedAt = time.Time{}
	inputFrame.BridgeAudio = resampled
	inputFrame.PipelineAudio = append([]byte(nil), resampled...)
	return inputFrame, nil
}

func (p *AudioProcessor) ProcessAssistantAudio(audio []byte, completed bool) error {
	if p.bridge.Load() != nil || p.transferActive.Load() {
		return nil
	}

	p.outputMu.Lock()
	defer p.outputMu.Unlock()
	if p.bridge.Load() != nil || p.transferActive.Load() {
		return nil
	}
	if len(audio) > 0 || completed {
		if p.writers.assistant != nil {
			if len(audio) > 0 {
				if err := p.writers.assistant.Write(audio); err != nil {
					return fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
				}
			}
			if completed {
				if err := p.writers.assistant.Flush(); err != nil {
					return fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
				}
			}
		} else {
			writeProviderAudio := func(providerAudio []byte) {
				if p.currentCodec().Name == sip_runtime.CodecPCMA.Name {
					providerAudio = internal_audio.UlawToAlaw(providerAudio)
				}
				p.providerOutputBuffer.Write(providerAudio)
			}
			if len(audio) > 0 {
				providerAudio, err := p.resamplers.assistant.Resample(audio, Rapida16kConfig, Mulaw8kConfig)
				if err != nil {
					return fmt.Errorf("%w: %w", ErrAssistantAudioConversionFailed, err)
				}
				if len(providerAudio) > 0 {
					writeProviderAudio(providerAudio)
				}
			}
		}
	}
	if len(audio) > 0 {
		p.bridgeOutputBuffer.Write(audio)
	}
	if completed {
		p.completeOutputLocked()
	}
	return nil
}

func (p *AudioProcessor) Complete() {
	p.outputMu.Lock()
	defer p.outputMu.Unlock()
	p.completeOutputLocked()
}

func (p *AudioProcessor) completeOutputLocked() {
	silenceByte := byte(MulawSilenceByte)
	if p.currentCodec().Name == sip_runtime.CodecPCMA.Name {
		silenceByte = 0xD5
	}
	p.providerOutputBuffer.Complete(MulawFrameSize, silenceByte)
	p.bridgeOutputBuffer.Complete(BridgeOutputFrameSize, 0)
}

func (p *AudioProcessor) ClearOutputBuffer() {
	p.outputMu.Lock()
	defer p.outputMu.Unlock()
	if p.writers.assistant != nil {
		// Flush filter history before discarding the response, including its tail.
		_ = p.writers.assistant.Flush()
	}
	p.providerOutputBuffer.Clear()
	p.bridgeOutputBuffer.Clear()
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
	p.outputMu.Lock()
	defer p.outputMu.Unlock()
	if p.transferActive.Load() {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	providerAudio, ok := p.providerOutputBuffer.Next(MulawFrameSize)
	if !ok {
		return internal_telephony_media.AssistantOutputFrame{}, false
	}
	bridgeAudio, _ := p.bridgeOutputBuffer.Next(BridgeOutputFrameSize)
	return internal_telephony_media.AssistantOutputFrame{
		ProviderAudio: p.applyAmbient(providerAudio),
		BridgeAudio:   bridgeAudio,
	}, true
}

func (p *AudioProcessor) OutputDrained() bool {
	p.outputMu.Lock()
	defer p.outputMu.Unlock()
	return !p.transferActive.Load() && p.providerOutputBuffer.Len() == 0
}

func (p *AudioProcessor) IdleOutputFrame() (internal_telephony_media.AssistantOutputFrame, bool) {
	if p.transferActive.Load() {
		if !p.ringbackActive.Load() {
			return internal_telephony_media.AssistantOutputFrame{}, false
		}
		return internal_telephony_media.AssistantOutputFrame{
			ProviderAudio: p.nextRingbackFrame(MulawFrameSize),
			Idle:          true,
		}, true
	}
	providerAudio := p.applyAmbient(nil)
	if len(providerAudio) == 0 {
		silenceByte := byte(MulawSilenceByte)
		if p.currentCodec().Name == sip_runtime.CodecPCMA.Name {
			silenceByte = 0xD5
		}
		providerAudio = make([]byte, MulawFrameSize)
		for index := range providerAudio {
			providerAudio[index] = silenceByte
		}
	}
	return internal_telephony_media.AssistantOutputFrame{
		ProviderAudio: providerAudio,
		Idle:          true,
	}, true
}

func (p *AudioProcessor) SetTransferActive(active bool) {
	p.transferActive.Store(active)
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
	if err := state.outputTarget.WriteAudio(audioData); err != nil {
		dropped := p.droppedBridgeFrames.Add(1)
		if p.record != nil && (dropped == 1 || dropped%100 == 0) {
			_ = p.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "SIP bridge audio write failed",
				Attributes: observability.Attributes{
					"component":            observability.ComponentCall.String(),
					"provider":             Provider,
					"reason":               "bridge_audio_write_failed",
					"dropped_frames_total": fmt.Sprintf("%d", dropped),
					"error":                err.Error(),
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
func (p *AudioProcessor) RunBridgeRecorder(ctx context.Context, streamSink func(proto.Message)) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-p.bridgeUserCh:
			if resampled, err := p.resampleBridgeRecordingFrame(frame, p.writers.bridgeUser, &p.bridgeUserAudio, p.resamplers.bridgeUser); err == nil && len(resampled) > 0 && streamSink != nil {
				streamSink(&protos.ConversationBridgeUserAudio{
					Audio: resampled,
					Time:  timestamppb.Now(),
				})
			}
		case frame := <-p.bridgeOperatorCh:
			if resampled, err := p.resampleBridgeRecordingFrame(frame, p.writers.bridgeOperator, &p.bridgeOperatorAudio, p.resamplers.bridgeOperator); err == nil && len(resampled) > 0 && streamSink != nil {
				streamSink(&protos.ConversationBridgeOperatorAudio{
					Audio: resampled,
					Time:  timestamppb.Now(),
				})
			}
		}
	}
}

func (p *AudioProcessor) resampleBridgeRecordingFrame(
	frame bridgeRecordingFrame,
	writer internal_type.AudioStreamResampler,
	outputAudio *[]byte,
	resampler internal_type.AudioResampler,
) ([]byte, error) {
	linearPCM8k := decodeG711ToLinear8k(frame.audio, frame.codecName)
	if len(linearPCM8k) == 0 {
		return nil, nil
	}
	if writer != nil {
		*outputAudio = (*outputAudio)[:0]
		if err := writer.Write(linearPCM8k); err != nil {
			return nil, err
		}
		return append([]byte(nil), (*outputAudio)...), nil
	}
	return resampler.Resample(linearPCM8k, Linear8kConfig, Rapida16kConfig)
}

func (p *AudioProcessor) StartRingback() {
	if p == nil {
		return
	}
	p.ringbackMu.Lock()
	p.ringbackOffset = 0
	p.ringbackFileOffset = 0
	p.ringbackMu.Unlock()
	p.ringbackActive.Store(true)
}

func (p *AudioProcessor) StopRingback() {
	if p == nil {
		return
	}
	p.ringbackActive.Store(false)
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
