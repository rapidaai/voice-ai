// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// MakeTransferBridgeCall dials a transfer B-leg and returns after the leg answers.
func (s *Server) MakeTransferBridgeCall(ctx context.Context, cfg *Config, toUser, fromUser string, opts TransferBridgeCallOptions) (*Session, error) {
	outboundCall, err := s.prepareOutboundCallLeg(ctx, cfg, toUser, fromUser, outboundCallLegOptions{
		purpose:         OutboundLegPurposeTransferBridge,
		makeCallOptions: opts.makeCallOptions(),
		parentCallID:    opts.ParentCallID,
		parentContextID: opts.ContextID,
		parentConvID:    opts.ConversationID,
		transferTarget:  toUser,
		transferAttempt: opts.Attempt,
		transferTotal:   opts.TotalAttempts,
	})
	if err != nil {
		return nil, NewSIPError("MakeTransferBridgeCall", "", "outbound setup failed", err)
	}

	outboundCall.answerContext = ctx
	outboundCall.ReportStatus(internal_type.ProviderCallStatusUpdate{CallStatus: string(OutboundCallStatusInitiated)})
	if _, err := outboundCall.Connect(); err != nil {
		return nil, NewSIPError("MakeTransferBridgeCall", outboundCall.session.GetCallID(), "call not answered", err)
	}
	return outboundCall.session, nil
}

type BridgeEndReason int

// BridgeTransfer forwards operator audio to the caller and observes bridge teardown.
// The caller owns session cleanup so streamer-side bridge sends can drain first.
func (s *Server) BridgeTransfer(ctx context.Context, inbound, outbound *Session, onOperatorAudio func([]byte)) (BridgeEndReason, error) {
	inCallID := inbound.GetCallID()
	outCallID := outbound.GetCallID()

	inRTP := inbound.GetRTPHandler()
	outRTP := outbound.GetRTPHandler()
	if inRTP == nil || outRTP == nil {
		err := NewSIPError("BridgeTransfer", inCallID, "RTP handler unavailable", ErrRTPNotInitialized)
		if !outbound.IsEnded() {
			_ = s.FailCall(outbound, LifecycleReasonBridgeRTPUnavailable, err)
		}
		if !inbound.IsEnded() {
			_ = s.FailCall(inbound, LifecycleReasonBridgeRTPUnavailable, err)
		}
		return BridgeEndContext, err
	}

	inCodec := inbound.GetNegotiatedCodec()
	outCodec := outbound.GetNegotiatedCodec()
	needsTranscode := inCodec != nil && outCodec != nil && inCodec.Name != outCodec.Name
	if err := s.beginBridgeLifecycle(inbound, outbound); err != nil {
		if !outbound.IsEnded() {
			_ = s.FailCall(outbound, LifecycleReasonBridgeSetupFailed, err)
		}
		if !inbound.IsEnded() {
			_ = s.FailCall(inbound, LifecycleReasonBridgeSetupFailed, err)
		}
		return BridgeEndContext, err
	}

	audioCtx, audioCancel := context.WithCancel(ctx)
	defer audioCancel()
	go s.forwardBridgeAudio(audioCtx, outRTP.AudioIn(), inRTP, needsTranscode, outCodec, inCodec, onOperatorAudio)

	var reason BridgeEndReason
	select {
	case <-ctx.Done():
		reason = BridgeEndContext
		s.logger.Infow("Bridge: context cancelled",
			"inbound_call_id", inCallID, "outbound_call_id", outCallID, "error", ctx.Err())
	case <-inbound.ByeReceived():
		reason = BridgeEndInboundBye
		s.logger.Infow("Bridge: inbound caller hung up", "inbound_call_id", inCallID)
	case <-outbound.ByeReceived():
		reason = BridgeEndOutboundBye
		s.logger.Infow("Bridge: transfer target hung up", "outbound_call_id", outCallID)
	case <-inbound.Context().Done():
		reason = BridgeEndInboundBye
		s.logger.Infow("Bridge: inbound session ended", "inbound_call_id", inCallID)
	case <-outbound.Context().Done():
		reason = BridgeEndOutboundBye
		s.logger.Infow("Bridge: outbound session ended", "outbound_call_id", outCallID)
	case <-time.After(BridgeSafetyTimeout):
		reason = BridgeEndTimeout
		s.logger.Warnw("Bridge: safety timeout reached, tearing down",
			"inbound_call_id", inCallID, "outbound_call_id", outCallID)
	}

	audioCancel()

	s.logger.Infow("Audio bridge completed",
		"inbound_call_id", inCallID, "outbound_call_id", outCallID,
		"reason", reason)
	return reason, nil
}

func (s *Server) beginBridgeLifecycle(inbound, outbound *Session) error {
	if err := s.beginBridgeLegLifecycle(inbound, "inbound"); err != nil {
		return err
	}
	if err := s.beginBridgeLegLifecycle(outbound, "outbound"); err != nil {
		return err
	}
	return nil
}

func (s *Server) beginBridgeLegLifecycle(session *Session, legRole string) error {
	if session == nil {
		return fmt.Errorf("%w: %s session is nil", ErrBridgeLifecycleRejected, legRole)
	}
	callID := session.GetCallID()
	currentState := session.GetState()
	switch currentState {
	case CallStateConnected:
		if !s.TransitionCall(session, CallStateTransferring, LifecycleReasonBridgeTransferStarted) {
			return fmt.Errorf("%w: %s call %s could not enter transfer state from %s",
				ErrBridgeLifecycleRejected, legRole, callID, currentState)
		}
	case CallStateTransferring:
	default:
		return fmt.Errorf("%w: %s call %s is in %s, expected %s or %s",
			ErrBridgeLifecycleRejected, legRole, callID, currentState, CallStateConnected, CallStateTransferring)
	}
	if !s.TransitionCall(session, CallStateBridgeConnected, LifecycleReasonBridgeMediaConnected) {
		return fmt.Errorf("%w: %s call %s could not enter bridge state from %s",
			ErrBridgeLifecycleRejected, legRole, callID, session.GetState())
	}
	return nil
}

// forwardBridgeAudio reads audio from src and enqueues it to dst, transcoding if needed.
func (s *Server) forwardBridgeAudio(ctx context.Context, src <-chan InboundAudioFrame, dst internal_type.SIPRTPBridgeTarget, needsTranscode bool, srcCodec, dstCodec *Codec, onAudio func([]byte)) {
	var droppedFrames uint64
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-src:
			if !ok {
				return
			}
			data := frame.Audio
			rawData := data
			if needsTranscode {
				data = s.transcodeG711(data, srcCodec, dstCodec)
			}
			if err := dst.EnqueueAudio(data); err != nil {
				droppedFrames++
				if errors.Is(err, ErrRTPHandlerStopped) {
					return
				}
				if s.logger != nil && errors.Is(err, ErrRTPOutputQueueFull) && (droppedFrames == 1 || droppedFrames%100 == 0) {
					s.logger.Warnw("Bridge RTP output queue full; dropping frame",
						"dropped_frames_total", droppedFrames)
				}
				continue
			}
			if onAudio != nil {
				onAudio(rawData)
			}
		}
	}
}

// transcodeG711 converts audio between PCMU and PCMA codecs.
func (s *Server) transcodeG711(data []byte, from, to *Codec) []byte {
	if from.Name == CodecPCMA.Name && to.Name == CodecPCMU.Name {
		return internal_audio.AlawToUlaw(data)
	}
	if from.Name == CodecPCMU.Name && to.Name == CodecPCMA.Name {
		return internal_audio.UlawToAlaw(data)
	}
	return data
}

func (s *Server) codecName(c *Codec) string {
	if c != nil {
		return c.Name
	}
	return CodecPCMU.Name
}
