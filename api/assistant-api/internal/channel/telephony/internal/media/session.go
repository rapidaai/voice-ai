// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telephony_media

import (
	"context"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewMediaSession(config MediaSessionConfig) *MediaSession {
	sessionContext := config.Context
	if sessionContext == nil {
		sessionContext = context.Background()
	}
	ctx, cancel := context.WithCancel(sessionContext)
	mediaSession := &MediaSession{
		logger:            config.Logger,
		mediaEngine:       config.MediaEngine,
		sendProviderClear: config.SendProviderClear,
		streamSink:        config.StreamSink,
		outputSink:        config.OutputSink,
		record:            config.Record,
		ctx:               ctx,
		cancel:            cancel,
	}
	if config.MediaEngine == nil {
		return mediaSession
	}
	return mediaSession
}

func (mediaSession *MediaSession) Start() {
	if mediaSession == nil || !mediaSession.hasMediaEngine() || mediaSession.closed.Load() {
		return
	}
	mediaSession.startMu.Lock()
	defer mediaSession.startMu.Unlock()
	if !mediaSession.started.CompareAndSwap(false, true) {
		return
	}
	mediaSession.sinkMu.RLock()
	hasOutputSink := mediaSession.outputSink != nil
	mediaSession.sinkMu.RUnlock()
	if hasOutputSink {
		go mediaSession.runFrameOutputSender()
	}
}

func (mediaSession *MediaSession) HandleInitialization(init *protos.ConversationInitialization) {
	if mediaSession == nil || mediaSession.mediaEngine == nil || init == nil {
		return
	}
	ambientConfig, ok := internal_ambient.ParseFromInitialization(init)
	if !ok {
		return
	}
	if err := mediaSession.mediaEngine.ConfigureAmbient(ambientConfig); err != nil && mediaSession.logger != nil {
		mediaSession.logger.Warnw("Failed to configure ambient audio", "error", err.Error(), "profile", ambientConfig.Profile)
	}
}

func (mediaSession *MediaSession) HandleAssistantAudio(audio []byte, completed bool) error {
	if mediaSession == nil || !mediaSession.hasMediaEngine() {
		return nil
	}
	return mediaSession.mediaEngine.ProcessAssistantAudio(audio, completed)
}

func (mediaSession *MediaSession) HandleProviderAudioFrame(frame ProviderAudioFrame) error {
	if mediaSession == nil || !mediaSession.hasMediaEngine() {
		return nil
	}
	if frame.ReceivedAt.IsZero() {
		frame.ReceivedAt = time.Now()
	}
	inputFrame, err := mediaSession.mediaEngine.ProcessProviderAudioFrame(frame)
	if err != nil {
		return err
	}
	mediaSession.emitInputAudioFrame(inputFrame, frame.ReceivedAt)
	return nil
}

func (mediaSession *MediaSession) HandleInterrupt() {
	if mediaSession == nil || !mediaSession.hasMediaEngine() {
		return
	}
	mediaSession.outputFrameMu.Lock()
	mediaSession.mediaEngine.ClearOutputBuffer()
	mediaSession.currentOutputFrame = AssistantOutputFrame{}
	mediaSession.hasCurrentOutputFrame = false
	mediaSession.outputFrameMu.Unlock()
	if mediaSession.sendProviderClear != nil {
		if err := mediaSession.sendProviderClear(); err != nil && mediaSession.record != nil {
			_ = mediaSession.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to send telephony clear command",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"error":     err.Error(),
				},
			})
		}
	}
	if mediaSession.record != nil {
		_ = mediaSession.record(observability.RecordEvent{
			Component: observability.ComponentCall,
			Event:     observability.CallStatus,
			Attributes: observability.Attributes{
				"component": observability.ComponentCall.String(),
				"status":    "output_queue_cleared",
				"reason":    "interruption",
			},
		})
	}
}

func (mediaSession *MediaSession) Shutdown() {
	if mediaSession == nil {
		return
	}
	if !mediaSession.closed.CompareAndSwap(false, true) {
		return
	}
	if mediaSession.cancel != nil {
		mediaSession.cancel()
	}
}

func (mediaSession *MediaSession) hasMediaEngine() bool {
	return mediaSession.mediaEngine != nil
}

func (mediaSession *MediaSession) runFrameOutputSender() {
	frameDuration := 20 * time.Millisecond
	if duration := mediaSession.mediaEngine.OutputFrameDuration(); duration > 0 {
		frameDuration = duration
	}
	(&internal_output.Pacer{
		Logger:        mediaSession.logger,
		FrameDuration: frameDuration,
		Provider:      mediaSession,
		Consumer:      mediaSession,
	}).Run(mediaSession.ctx)
}

func (mediaSession *MediaSession) NextFrame() []byte {
	if mediaSession.mediaEngine == nil {
		return nil
	}
	mediaSession.outputFrameMu.Lock()
	defer mediaSession.outputFrameMu.Unlock()
	outputFrame, ok := mediaSession.mediaEngine.NextOutputFrame()
	if !ok || len(outputFrame.ProviderAudio) == 0 {
		return nil
	}
	mediaSession.currentOutputFrame = outputFrame
	mediaSession.hasCurrentOutputFrame = true
	return outputFrame.ProviderAudio
}

func (mediaSession *MediaSession) IdleFrame() []byte {
	if mediaSession.mediaEngine == nil {
		return nil
	}
	mediaSession.outputFrameMu.Lock()
	defer mediaSession.outputFrameMu.Unlock()
	outputFrame, ok := mediaSession.mediaEngine.IdleOutputFrame()
	if !ok || len(outputFrame.ProviderAudio) == 0 {
		return nil
	}
	outputFrame.Idle = true
	mediaSession.currentOutputFrame = outputFrame
	mediaSession.hasCurrentOutputFrame = true
	return outputFrame.ProviderAudio
}

func (mediaSession *MediaSession) ConsumeFrame(_ []byte) error {
	mediaSession.sinkMu.RLock()
	outputSink := mediaSession.outputSink
	mediaSession.sinkMu.RUnlock()
	if outputSink == nil {
		return nil
	}

	mediaSession.outputFrameMu.Lock()
	outputFrame := mediaSession.currentOutputFrame
	hasCurrentOutputFrame := mediaSession.hasCurrentOutputFrame
	mediaSession.currentOutputFrame = AssistantOutputFrame{}
	mediaSession.hasCurrentOutputFrame = false
	if !hasCurrentOutputFrame {
		mediaSession.outputFrameMu.Unlock()
		return nil
	}
	err := outputSink(outputFrame)
	mediaSession.outputFrameMu.Unlock()
	if err != nil {
		if mediaSession.record != nil {
			_ = mediaSession.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Telephony output send failed",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"error":     err.Error(),
				},
			})
		}
		return err
	}
	if outputFrame.Idle || len(outputFrame.BridgeAudio) == 0 {
		return nil
	}
	mediaSession.emitStream(&protos.ConversationBridgeOperatorAudio{
		Audio: outputFrame.BridgeAudio,
		Time:  timestamppb.Now(),
	})
	return nil
}

func (mediaSession *MediaSession) emitInputAudioFrame(inputFrame InputAudioFrame, fallbackReceivedAt time.Time) {
	receivedAt := inputFrame.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = fallbackReceivedAt
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now()
	}
	if len(inputFrame.BridgeAudio) > 0 {
		mediaSession.emitStream(&protos.ConversationBridgeUserAudio{
			Audio: inputFrame.BridgeAudio,
			Time:  timestamppb.New(receivedAt),
		})
	}
	if len(inputFrame.PipelineAudio) == 0 {
		return
	}
	userAudio := &protos.ConversationUserMessage{
		Message: &protos.ConversationUserMessage_Audio{Audio: inputFrame.PipelineAudio},
		Time:    timestamppb.New(receivedAt),
	}
	mediaSession.emitStream(userAudio)
}

func (mediaSession *MediaSession) emitStream(stream internal_type.Stream) {
	mediaSession.sinkMu.RLock()
	streamSink := mediaSession.streamSink
	mediaSession.sinkMu.RUnlock()
	if streamSink == nil {
		return
	}
	streamSink(stream)
}
