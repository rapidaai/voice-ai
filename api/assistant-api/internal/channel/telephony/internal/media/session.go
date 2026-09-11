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
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
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

func (mediaSession *MediaSession) HandleAssistantAudio(outputID string, audio []byte, completed bool) (bool, error) {
	if mediaSession == nil || !mediaSession.hasMediaEngine() {
		return false, nil
	}
	mediaSession.outputFrameMu.Lock()
	defer mediaSession.outputFrameMu.Unlock()
	if mediaSession.closed.Load() || mediaSession.ctx.Err() != nil {
		return false, nil
	}
	if mediaSession.blockedOutputID != "" && (outputID == "" || outputID == mediaSession.blockedOutputID) {
		return false, nil
	}
	if outputID == "" {
		outputID = mediaSession.currentOutputID
	}
	if outputID != "" {
		if _, flushed := mediaSession.flushedOutputIDs[outputID]; flushed {
			return false, nil
		}
		if _, closed := mediaSession.closedOutputIDs[outputID]; closed {
			return false, nil
		}
	}
	if outputID != "" && outputID != mediaSession.currentOutputID {
		mediaSession.currentOutputID = outputID
		mediaSession.outputFlushed = false
	}
	_, discarded := mediaSession.discardedOutputIDs[outputID]
	var response *responsePlayback
	for index := len(mediaSession.responses) - 1; index >= 0; index-- {
		queued := mediaSession.responses[index]
		if queued.id == outputID {
			if queued.completed {
				return false, nil
			}
			response = queued
			break
		}
	}
	if response == nil {
		response = &responsePlayback{id: outputID, failed: discarded}
		mediaSession.responses = append(mediaSession.responses, response)
	} else if discarded {
		response.failed = true
	}
	response.completed = completed
	if response != mediaSession.responses[0] {
		response.audio = append(response.audio, audio...)
		return true, nil
	}
	if len(response.audio) > 0 {
		audio = append(response.audio, audio...)
		response.audio = nil
	}
	err := mediaSession.mediaEngine.ProcessAssistantAudio(audio, completed)
	response.processed = true
	response.failed = response.failed || err != nil
	return true, err
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

func (mediaSession *MediaSession) HandleOutputControl(control proto.Message) (bool, error) {
	switch control.(type) {
	case *protos.ConversationPlaybackPause:
		if mediaSession == nil || !mediaSession.hasMediaEngine() {
			return true, nil
		}
		mediaSession.outputFrameMu.Lock()
		if mediaSession.outputPaused {
			mediaSession.outputFrameMu.Unlock()
			return true, nil
		}
		mediaSession.outputPaused = true
		mediaSession.outputFrameMu.Unlock()
		if mediaSession.record != nil {
			_ = mediaSession.record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallStatus,
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"status":    "output_paused",
					"reason":    "pause",
				},
			})
		}
		return true, nil
	case *protos.ConversationPlaybackContinue:
		if mediaSession == nil || !mediaSession.hasMediaEngine() {
			return true, nil
		}
		mediaSession.outputFrameMu.Lock()
		mediaSession.outputPaused = false
		mediaSession.outputFrameMu.Unlock()
		return true, nil
	case *protos.ConversationPlaybackFlush:
		flush := control.(*protos.ConversationPlaybackFlush)
		if mediaSession == nil || !mediaSession.hasMediaEngine() {
			return true, nil
		}
		mediaSession.outputFrameMu.Lock()
		var providerClearError error
		mediaSession.outputPaused = false
		if flush.GetId() != "" {
			if mediaSession.flushedOutputIDs == nil {
				mediaSession.flushedOutputIDs = make(map[string]struct{})
			}
			mediaSession.flushedOutputIDs[flush.GetId()] = struct{}{}
		}
		if !mediaSession.outputFlushed {
			mediaSession.mediaEngine.ClearOutputBuffer()
			mediaSession.currentOutputFrame = AssistantOutputFrame{}
			mediaSession.hasCurrentOutputFrame = false
			mediaSession.outputFlushed = true
			mediaSession.blockedOutputID = mediaSession.currentOutputID
			if mediaSession.flushedOutputIDs == nil {
				mediaSession.flushedOutputIDs = make(map[string]struct{})
			}
			if mediaSession.currentOutputID != "" {
				mediaSession.flushedOutputIDs[mediaSession.currentOutputID] = struct{}{}
			}
			for _, response := range mediaSession.responses {
				if response.id != "" {
					mediaSession.flushedOutputIDs[response.id] = struct{}{}
				}
			}
			mediaSession.responses = nil
			if mediaSession.sendProviderClear != nil {
				providerClearError = mediaSession.sendProviderClear()
			}
		}
		mediaSession.outputFrameMu.Unlock()
		if providerClearError != nil && mediaSession.record != nil {
			_ = mediaSession.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to send telephony clear command",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"error":     providerClearError.Error(),
				},
			})
		}
		if mediaSession.record != nil {
			_ = mediaSession.record(observability.RecordEvent{
				Component: observability.ComponentCall,
				Event:     observability.CallStatus,
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"status":    "output_queue_cleared",
					"reason":    "flush",
				},
			})
		}
		return true, providerClearError
	default:
		return false, nil
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

// DiscardForTransfer abandons queued output without blocking same-context speech after resume.
func (mediaSession *MediaSession) DiscardForTransfer() {
	if mediaSession == nil || !mediaSession.hasMediaEngine() {
		return
	}
	mediaSession.outputFrameMu.Lock()
	defer mediaSession.outputFrameMu.Unlock()
	mediaSession.mediaEngine.ClearOutputBuffer()
	mediaSession.currentOutputFrame = AssistantOutputFrame{}
	mediaSession.hasCurrentOutputFrame = false
	for _, response := range mediaSession.responses {
		if response.id == "" {
			continue
		}
		if mediaSession.discardedOutputIDs == nil {
			mediaSession.discardedOutputIDs = make(map[string]struct{})
		}
		mediaSession.discardedOutputIDs[response.id] = struct{}{}
	}
	mediaSession.responses = nil
	mediaSession.outputPaused = false
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
	var completion *protos.ConversationPlaybackComplete
	var conversionError error
	defer func() {
		mediaSession.outputFrameMu.Unlock()
		if conversionError != nil && mediaSession.record != nil {
			_ = mediaSession.record(observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Queued telephony audio conversion failed",
				Attributes: observability.Attributes{
					"component": observability.ComponentCall.String(),
					"error":     conversionError.Error(),
				},
			})
		}
		if completion != nil {
			mediaSession.emitStream(completion)
		}
	}()
	if mediaSession.outputPaused || mediaSession.closed.Load() || mediaSession.ctx.Err() != nil {
		return nil
	}
	if mediaSession.hasCurrentOutputFrame {
		return mediaSession.currentOutputFrame.ProviderAudio
	}
	if len(mediaSession.responses) > 0 {
		response := mediaSession.responses[0]
		if !response.processed {
			conversionError = mediaSession.mediaEngine.ProcessAssistantAudio(response.audio, response.completed)
			response.failed = conversionError != nil
			response.audio = nil
			response.processed = true
		}
	}
	outputFrame, ok := mediaSession.mediaEngine.NextOutputFrame()
	if !ok || len(outputFrame.ProviderAudio) == 0 {
		if len(mediaSession.responses) == 0 || !mediaSession.responses[0].completed {
			return nil
		}
		if !mediaSession.responses[0].failed && !mediaSession.mediaEngine.OutputDrained() {
			return nil
		}
		response := mediaSession.responses[0]
		mediaSession.responses[0] = nil
		mediaSession.responses = mediaSession.responses[1:]
		if response.failed {
			mediaSession.mediaEngine.ClearOutputBuffer()
		}
		if response.id != "" && !response.failed && response.sent {
			completion = &protos.ConversationPlaybackComplete{
				Id:   response.id,
				Time: timestamppb.Now(),
			}
		}
		if response.id != "" {
			if mediaSession.closedOutputIDs == nil {
				mediaSession.closedOutputIDs = make(map[string]struct{})
			}
			mediaSession.closedOutputIDs[response.id] = struct{}{}
		}
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
	if mediaSession.outputPaused || mediaSession.closed.Load() || mediaSession.ctx.Err() != nil {
		return nil
	}
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
	if mediaSession.outputPaused || mediaSession.closed.Load() || mediaSession.ctx.Err() != nil {
		mediaSession.outputFrameMu.Unlock()
		return nil
	}
	outputFrame := mediaSession.currentOutputFrame
	hasCurrentOutputFrame := mediaSession.hasCurrentOutputFrame
	mediaSession.currentOutputFrame = AssistantOutputFrame{}
	mediaSession.hasCurrentOutputFrame = false
	if !hasCurrentOutputFrame {
		mediaSession.outputFrameMu.Unlock()
		return nil
	}
	err := outputSink(outputFrame)
	if !outputFrame.Idle && len(mediaSession.responses) > 0 {
		response := mediaSession.responses[0]
		response.failed = response.failed || err != nil
		response.sent = response.sent || err == nil
	}
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

func (mediaSession *MediaSession) emitStream(stream proto.Message) {
	mediaSession.sinkMu.RLock()
	streamSink := mediaSession.streamSink
	mediaSession.sinkMu.RUnlock()
	if streamSink == nil {
		return
	}
	streamSink(stream)
}
