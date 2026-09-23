// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"errors"

	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/pkg/channel"
	"github.com/rapidaai/protos"
)

// runOutputWriter routes assistant audio to the pacer and non-audio messages to gRPC.
func (s *webrtcStreamer) runOutputWriter() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		case clearGeneration := <-s.outputClearCh:
			s.outputWriteMu.Lock()
			s.outputStateMu.Lock()
			shouldSendOutputClear := s.outputClearPending && s.pendingClearGeneration == clearGeneration
			s.outputStateMu.Unlock()
			if !shouldSendOutputClear {
				s.outputWriteMu.Unlock()
				continue
			}
			s.Mu.Lock()
			signalingSessionID := s.signalingSessionID
			s.Mu.Unlock()
			if signalingSessionID == "" {
				signalingSessionID = s.sessionID
			}
			// The pending clear fences audio while signaling runs without teardown locks.
			s.outputWriteMu.Unlock()
			if !s.dispatchOutput(s.buildGRPCResponse(&protos.ServerSignaling{
				SessionId: signalingSessionID,
				Message:   &protos.ServerSignaling_Clear{Clear: true},
			})) {
				return
			}
			s.outputWriteMu.Lock()
			s.outputStateMu.Lock()
			if s.pendingClearGeneration == clearGeneration {
				s.outputClearPending = false
				s.pendingClearGeneration = 0
			}
			s.outputStateMu.Unlock()
			s.outputWriteMu.Unlock()

		case <-s.OutputCh.Ready():
			msg, err := s.OutputCh.TryReceive()
			if errors.Is(err, channel.ErrClosed) {
				return
			}
			if err != nil {
				continue
			}
			if m, ok := msg.(*protos.ConversationAssistantMessage); ok {
				if _, ok := m.Message.(*protos.ConversationAssistantMessage_Audio); ok {
					s.bufferAndSendOutput(m.GetId(), m.GetAudio(), m.GetCompleted())
					continue
				}
			}

			if resp := s.buildGRPCResponse(msg); resp != nil {
				if !s.dispatchOutput(resp) {
					return
				}
			}
		}
	}
}

func (s *webrtcStreamer) runAudioPacer() {
	(&internal_output.Pacer{
		Logger:        s.Logger,
		FrameDuration: webrtc_internal.OutputPaceDuration,
		Provider:      s,
		Consumer:      s,
		Health:        s.outputHealth,
	}).Run(s.Ctx)
}
