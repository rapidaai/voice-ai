// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"bytes"
	"fmt"
	"time"

	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newWebRTCAudioBufferState() webrtc_internal.WebRTCAudioBufferState {
	return webrtc_internal.WebRTCAudioBufferState{
		InputAudioBuffer:  bytes.NewBuffer(make([]byte, 0, webrtc_internal.InputBufferThreshold*2)),
		OutputAudioBuffer: bytes.NewBuffer(make([]byte, 0, webrtc_internal.WebRTCOutputPCM16kFrameBytes*2)),
	}
}

func (s *webrtcStreamer) bufferAndSendInput(audio []byte, inputAudioReceivedAt time.Time) {
	if inputAudioReceivedAt.IsZero() {
		inputAudioReceivedAt = time.Now()
	}
	s.Input(&protos.ConversationBridgeUserAudio{
		Audio: audio,
		Time:  timestamppb.New(inputAudioReceivedAt),
	})

	s.audioBufferState.InputAudioBufferMu.Lock()
	s.audioBufferState.InputAudioBuffer.Write(audio)
	if s.audioBufferState.InputAudioBuffer.Len() < webrtc_internal.InputBufferThreshold {
		s.audioBufferState.InputAudioBufferMu.Unlock()
		return
	}

	audioData := s.audioBufferState.InputAudioBuffer.Bytes()
	s.audioBufferState.InputAudioBuffer = bytes.NewBuffer(make([]byte, 0, webrtc_internal.InputBufferThreshold*2))
	s.audioBufferState.InputAudioBufferMu.Unlock()

	s.Input(&protos.ConversationUserMessage{
		Message: &protos.ConversationUserMessage_Audio{Audio: audioData},
		Time:    timestamppb.New(inputAudioReceivedAt),
	})
}

func (s *webrtcStreamer) bufferAndSendOutput(contextID string, audio []byte) {
	s.outputStateMu.Lock()
	if contextID == "" && s.outputFlushed {
		s.outputStateMu.Unlock()
		return
	}
	if contextID != "" {
		if contextID == s.flushedOutputContextID {
			s.outputStateMu.Unlock()
			return
		}
		if s.outputContextID != contextID {
			s.audioBufferState.OutputAudioBufferMu.Lock()
			s.audioBufferState.OutputAudioBuffer.Reset()
			s.audioBufferState.OutputAudioBufferMu.Unlock()
			s.outputContextID = contextID
		}
		s.outputFlushed = false
	}

	s.audioBufferState.OutputAudioBufferMu.Lock()
	s.audioBufferState.OutputAudioBuffer.Write(audio)
	if s.audioBufferState.OutputAudioBuffer.Len() < webrtc_internal.WebRTCOutputPCM16kFrameBytes {
		s.audioBufferState.OutputAudioBufferMu.Unlock()
		s.outputStateMu.Unlock()
		return
	}

	var audioEnqueueResults []struct {
		queuedAt      time.Time
		droppedFrames int
		queueDepth    int
	}
	for s.audioBufferState.OutputAudioBuffer.Len() >= webrtc_internal.WebRTCOutputPCM16kFrameBytes {
		frame := make([]byte, webrtc_internal.WebRTCOutputPCM16kFrameBytes)
		s.audioBufferState.OutputAudioBuffer.Read(frame)
		assistantAudioQueuedAt := time.Now()
		outputFrame := webrtc_internal.OutputAudioFrame{
			Audio:    frame,
			QueuedAt: assistantAudioQueuedAt,
		}
		droppedFrames := 0
		s.outputAudioQueueMu.Lock()
		if webrtc_internal.OutputAudioQueueMaxFrames > 0 && len(s.outputAudioQueue) >= webrtc_internal.OutputAudioQueueMaxFrames {
			s.outputAudioQueue[0] = webrtc_internal.OutputAudioFrame{}
			copy(s.outputAudioQueue, s.outputAudioQueue[1:])
			s.outputAudioQueue[len(s.outputAudioQueue)-1] = outputFrame
			droppedFrames = webrtc_internal.OutputAudioDropOldestSize
		} else {
			s.outputAudioQueue = append(s.outputAudioQueue, outputFrame)
		}
		outputQueueDepth := len(s.outputAudioQueue)
		s.outputAudioQueueMu.Unlock()
		audioEnqueueResults = append(audioEnqueueResults, struct {
			queuedAt      time.Time
			droppedFrames int
			queueDepth    int
		}{assistantAudioQueuedAt, droppedFrames, outputQueueDepth})
	}
	s.audioBufferState.OutputAudioBufferMu.Unlock()
	s.outputStateMu.Unlock()

	for _, audioEnqueueResult := range audioEnqueueResults {
		s.Mu.Lock()
		s.mediaHealthState.RecordAssistantAudioQueued(audioEnqueueResult.queuedAt)
		s.Mu.Unlock()
		if audioEnqueueResult.droppedFrames == 0 {
			continue
		}
		totalDroppedFrames := s.sessionState.AddOutputAudioDroppedFrames(audioEnqueueResult.droppedFrames)
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "WebRTC output queue overflow dropped the oldest assistant audio frame; this keeps playback current when audio is produced faster than WebRTC can send it.",
			Attributes: observability.Attributes{
				"component":                            observability.ComponentWebRTC.String(),
				webrtc_internal.DataType:               webrtc_internal.EventOutputQueueOverflow,
				webrtc_internal.DataSessionID:          s.sessionID,
				webrtc_internal.DataPolicy:             webrtc_internal.OutputQueuePolicyDropOldest,
				webrtc_internal.DataDroppedFrames:      fmt.Sprintf("%d", audioEnqueueResult.droppedFrames),
				webrtc_internal.DataLimitFrames:        fmt.Sprintf("%d", webrtc_internal.OutputAudioQueueMaxFrames),
				webrtc_internal.DataQueueDepthFrames:   fmt.Sprintf("%d", audioEnqueueResult.queueDepth),
				webrtc_internal.DataTotalDroppedFrames: fmt.Sprintf("%d", totalDroppedFrames),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricWebRTCOutputQueueDrops,
				Value:       fmt.Sprintf("%d", audioEnqueueResult.droppedFrames),
				Description: "WebRTC output queue dropped frames",
			}},
		})
	}
}

func (s *webrtcStreamer) clearBufferedOutputAudio() {
	s.outputWriteMu.Lock()
	defer s.outputWriteMu.Unlock()
	s.outputStateMu.Lock()
	s.audioBufferState.OutputAudioBufferMu.Lock()
	s.audioBufferState.OutputAudioBuffer.Reset()
	s.audioBufferState.OutputAudioBufferMu.Unlock()
	s.currentOutputFrame = nil
	s.currentOutputGeneration = 0
	s.clearOutputAudio()
	s.outputPaused = false
	s.outputFlushed = false
	s.outputClearPending = false
	s.outputContextID = ""
	s.flushedOutputContextID = ""
	s.outputGeneration++
	s.pendingClearGeneration = 0
	s.outputStateMu.Unlock()
}

func (s *webrtcStreamer) withOutputAudioBuffer(fn func(buf *bytes.Buffer)) {
	s.audioBufferState.OutputAudioBufferMu.Lock()
	defer s.audioBufferState.OutputAudioBufferMu.Unlock()
	fn(s.audioBufferState.OutputAudioBuffer)
}
