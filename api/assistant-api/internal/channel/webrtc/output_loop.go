// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
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
			if !s.dispatchOutput(s.buildGRPCResponse(&protos.ServerSignaling{
				SessionId: signalingSessionID,
				Message:   &protos.ServerSignaling_Clear{Clear: true},
			})) {
				s.outputWriteMu.Unlock()
				return
			}
			s.outputStateMu.Lock()
			if s.pendingClearGeneration == clearGeneration {
				s.outputClearPending = false
				s.pendingClearGeneration = 0
			}
			s.outputStateMu.Unlock()
			s.outputWriteMu.Unlock()

		case msg := <-s.OutputCh:
			if m, ok := msg.(*protos.ConversationAssistantMessage); ok {
				if audio, ok := m.Message.(*protos.ConversationAssistantMessage_Audio); ok {
					s.bufferAndSendOutput(m.GetId(), audio.Audio)
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
