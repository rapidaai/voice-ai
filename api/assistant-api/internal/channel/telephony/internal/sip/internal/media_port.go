// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MediaPortConfig struct {
	Context    context.Context
	Logger     commons.Logger
	Session    *sip_runtime.Session
	RTPHandler rtpHandler
	Resampler  internal_type.AudioResampler
	StreamSink func(internal_type.Stream)
	Record     func(...observability.Record) error
}

type MediaPort struct {
	logger commons.Logger

	session        *sip_runtime.Session
	rtpHandler     rtpHandler
	audioProcessor *AudioProcessor
	mediaSession   *internal_telephony_media.MediaSession
	streamSink     func(internal_type.Stream)
	record         func(...observability.Record) error

	ctx    context.Context
	cancel context.CancelFunc

	inputStarted          atomic.Bool
	outputStarted         atomic.Bool
	bridgeRecorderStarted atomic.Bool
	closed                atomic.Bool
	transferActive        atomic.Bool
}

func NewMediaPort(config MediaPortConfig) (*MediaPort, error) {
	rtpHandler := config.RTPHandler
	if rtpHandler == nil && config.Session != nil {
		rtpHandler = config.Session.GetRTPHandler()
	}
	if rtpHandler == nil {
		callID := ""
		if config.Session != nil {
			callID = config.Session.GetCallID()
		}
		return nil, sip_runtime.NewSIPError("NewMediaPort", callID, "session has no RTP handler", sip_runtime.ErrRTPNotInitialized)
	}
	if config.Session == nil {
		return nil, sip_runtime.NewSIPError("NewMediaPort", "", "session is required", sip_runtime.ErrRTPNotInitialized)
	}
	portContext := config.Context
	if portContext == nil {
		portContext = context.Background()
	}
	ctx, cancel := context.WithCancel(portContext)
	mediaPort := &MediaPort{
		logger:     config.Logger,
		session:    config.Session,
		rtpHandler: rtpHandler,
		streamSink: config.StreamSink,
		record:     config.Record,
		ctx:        ctx,
		cancel:     cancel,
	}
	mediaPort.audioProcessor = NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtpHandler,
		Resampler:  config.Resampler,
		PushInput:  config.StreamSink,
		Record:     config.Record,
		Ringtone:   DefaultRingtone,
		Ambient:    resolveAmbientConfig(config.Session),
	})
	mediaPort.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     ctx,
		Logger:      config.Logger,
		MediaEngine: mediaPort.audioProcessor,
		StreamSink:  config.StreamSink,
		OutputSink: func(frame internal_telephony_media.AssistantOutputFrame) error {
			return mediaPort.deliverAssistantFrame(frame)
		},
		Record: config.Record,
	})
	return mediaPort, nil
}

func resolveAmbientConfig(sipSession *sip_runtime.Session) *internal_ambient.Config {
	if sipSession == nil {
		return nil
	}
	assistant := sipSession.GetAssistant()
	if assistant == nil || assistant.AssistantPhoneDeployment == nil || assistant.AssistantPhoneDeployment.OutputAudio == nil {
		return nil
	}
	options := assistant.AssistantPhoneDeployment.OutputAudio.GetOptions()
	config, ok := internal_ambient.ParseFromOptions(options)
	if !ok {
		return nil
	}
	return &config
}

// Start activates the full media path for calls that are already answered.
func (port *MediaPort) Start() {
	if port == nil || port.closed.Load() {
		return
	}
	port.StartInput()
	port.StartOutput()
	port.StartBridgeRecorder()
}

// StartInput begins RTP input forwarding without enabling assistant RTP output.
func (port *MediaPort) StartInput() {
	if port == nil || port.closed.Load() {
		return
	}
	if !port.inputStarted.CompareAndSwap(false, true) {
		return
	}
	go port.forwardIncomingAudio()
}

// StartOutput enables paced assistant audio delivery to RTP.
func (port *MediaPort) StartOutput() {
	if port == nil || port.closed.Load() {
		return
	}
	if !port.outputStarted.CompareAndSwap(false, true) {
		return
	}
	port.mediaSession.Start()
}

// StartBridgeRecorder enables transfer bridge recording delivery.
func (port *MediaPort) StartBridgeRecorder() {
	if port == nil || port.closed.Load() {
		return
	}
	if !port.bridgeRecorderStarted.CompareAndSwap(false, true) {
		return
	}
	go port.audioProcessor.RunBridgeRecorder(port.ctx)
}

func (port *MediaPort) Close() error {
	if port == nil {
		return nil
	}
	if !port.closed.CompareAndSwap(false, true) {
		return nil
	}
	port.stopRingback(false)
	if port.mediaSession != nil {
		port.mediaSession.Shutdown()
	}
	if port.cancel != nil {
		port.cancel()
	}
	return nil
}

func (port *MediaPort) HandleInitialization(init *protos.ConversationInitialization) {
	if port == nil || port.mediaSession == nil {
		return
	}
	port.mediaSession.HandleInitialization(init)
}

func (port *MediaPort) HandleAssistantAudio(audio []byte, completed bool) error {
	if port == nil || port.mediaSession == nil {
		return nil
	}
	if err := port.mediaSession.HandleAssistantAudio(audio, completed); err != nil {
		return err
	}
	return nil
}

func (port *MediaPort) HandleInterrupt() {
	if port == nil || port.mediaSession == nil {
		return
	}
	port.mediaSession.HandleInterrupt()
}

func (port *MediaPort) EnterTransferMode(ringtone string) bool {
	if port == nil {
		return true
	}
	if !port.transferActive.CompareAndSwap(false, true) {
		return false
	}
	port.audioProcessor.SetTransferActive(true)
	port.audioProcessor.ClearInputBuffer()
	port.audioProcessor.ClearOutputBuffer()
	port.audioProcessor.SetRingtone(ringtone)
	port.audioProcessor.StartRingback()
	return true
}

func (port *MediaPort) ResumeAssistant() bool {
	if port == nil {
		return true
	}
	if !port.transferActive.CompareAndSwap(true, false) {
		return false
	}
	port.stopRingback(false)
	port.audioProcessor.SetTransferActive(false)
	port.audioProcessor.DisconnectTransferMedia()
	return true
}

func (port *MediaPort) StopTransferRingback() {
	if port == nil {
		return
	}
	port.stopRingback(true)
}

func (port *MediaPort) ConnectTransferMedia(target internal_type.SIPRTPBridgeTarget, outputCodecName string) {
	if port == nil || port.audioProcessor == nil {
		return
	}
	inCodec := port.rtpHandler.GetCodec()
	port.audioProcessor.ConnectTransferMedia(target, inCodec, outputCodecName)
}

func (port *MediaPort) DisconnectTransferMedia() {
	if port == nil || port.audioProcessor == nil {
		return
	}
	port.audioProcessor.DisconnectTransferMedia()
}

func (port *MediaPort) RecordTransferOperatorAudio(audio []byte) {
	if port == nil || port.audioProcessor == nil {
		return
	}
	port.audioProcessor.RecordTransferOperatorAudio(audio)
}

func (port *MediaPort) LocalAddr() (string, int) {
	if port == nil || port.rtpHandler == nil {
		return "", 0
	}
	return port.rtpHandler.LocalAddr()
}

func (port *MediaPort) CodecName() string {
	if port == nil || port.rtpHandler == nil || port.rtpHandler.GetCodec() == nil {
		return "PCMU"
	}
	return port.rtpHandler.GetCodec().Name
}

func (port *MediaPort) forwardIncomingAudio() {
	if port == nil || port.rtpHandler == nil {
		return
	}
	audioIn := port.rtpHandler.AudioIn()
	for {
		select {
		case <-port.ctx.Done():
			return
		case audioData, ok := <-audioIn:
			if !ok {
				return
			}
			if port.audioProcessor.ForwardUserAudio(audioData) {
				continue
			}
			if port.transferActive.Load() {
				continue
			}
			if err := port.handleProviderAudioFrame(audioData, time.Now()); err != nil && port.record != nil {
				_ = port.record(observability.RecordLog{
					Level:   observability.LevelError,
					Message: "SIP provider audio processing failed",
					Attributes: observability.Attributes{
						"component": observability.ComponentCall.String(),
						"provider":  Provider,
						"call_id":   port.session.GetCallID(),
						"error":     err.Error(),
					},
				})
			}
		}
	}
}

func (port *MediaPort) handleProviderAudioFrame(providerAudio []byte, receivedAt time.Time) error {
	if receivedAt.IsZero() {
		receivedAt = time.Now()
	}
	inputFrame, err := port.audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      providerAudio,
		ReceivedAt: receivedAt,
	})
	if err != nil {
		return err
	}
	port.recordReceivedUserAudio(inputFrame.BridgeAudio, receivedAt)
	port.sendPipelineUserAudio(inputFrame.PipelineAudio, receivedAt)
	return nil
}

func (port *MediaPort) recordReceivedUserAudio(userPCM16k []byte, receivedAt time.Time) {
	if len(userPCM16k) == 0 || port.streamSink == nil {
		return
	}
	port.streamSink(&protos.ConversationBridgeUserAudio{
		Audio: userPCM16k,
		Time:  timestamppb.New(receivedAt),
	})
}

func (port *MediaPort) sendPipelineUserAudio(userPCM16k []byte, receivedAt time.Time) {
	if len(userPCM16k) == 0 || port.streamSink == nil {
		return
	}
	port.streamSink(&protos.ConversationUserMessage{
		Message: &protos.ConversationUserMessage_Audio{Audio: userPCM16k},
		Time:    timestamppb.New(receivedAt),
	})
}

func (port *MediaPort) deliverAssistantFrame(outputFrame internal_telephony_media.AssistantOutputFrame) error {
	if port == nil || port.closed.Load() {
		return sip_runtime.ErrSessionClosed
	}
	providerAudio, err := port.audioProcessor.encodeAssistantOutputFrame(outputFrame.ProviderAudio)
	if err != nil {
		if port.record != nil {
			_ = port.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "SIP assistant audio encoding failed",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"provider":  Provider,
					"call_id":   port.session.GetCallID(),
					"error":     err.Error(),
				},
			})
		}
		return err
	}
	if len(providerAudio) == 0 {
		return nil
	}
	if err := port.rtpHandler.EnqueueAudio(providerAudio); err != nil {
		if port.record != nil {
			_ = port.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "SIP assistant audio output failed",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"provider":  Provider,
					"call_id":   port.session.GetCallID(),
					"error":     err.Error(),
				},
			})
		}
		if errors.Is(err, sip_runtime.ErrRTPOutputQueueFull) {
			return port.audioProcessor.rtpOutputQueueFullError()
		}
		return err
	}
	if outputFrame.Idle || len(outputFrame.ProviderAudio) == 0 {
		return nil
	}
	port.recordDeliveredAssistantAudio(outputFrame.ProviderAudio)
	return nil
}

func (port *MediaPort) recordDeliveredAssistantAudio(assistantPCM16k []byte) {
	if port.streamSink != nil {
		port.streamSink(&protos.ConversationBridgeOperatorAudio{
			Audio: assistantPCM16k,
			Time:  timestamppb.Now(),
		})
	}
	if port.session == nil || port.session.GetInfo().Direction != sip_runtime.CallDirectionInbound {
		return
	}
	if port.session.MarkInboundFirstAssistantAudioSent() && port.record != nil {
		_ = port.record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallStatus,
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"provider":  Provider,
				"call_id":   port.session.GetCallID(),
				"status":    "first_assistant_audio_sent",
			},
		})
	}
}

func (port *MediaPort) stopRingback(clearOutput bool) {
	if port == nil || port.audioProcessor == nil {
		return
	}
	port.audioProcessor.StopRingback()
	if clearOutput && port.audioProcessor != nil {
		port.audioProcessor.ClearOutputBuffer()
	}
}
