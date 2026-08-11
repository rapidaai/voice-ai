// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// Outbound owns the lifecycle for a single outbound SIP call.
type Outbound struct {
	server  *Server
	session *Session
	dialog  *outboundDialog
	media   *outboundMedia
	request OutboundInviteRequest

	answerContext  context.Context
	statusObserver internal_type.ProviderCallStatusReporter
	closeOnce      sync.Once
}

// NewOutbound creates an outbound SIP call handler.
func NewOutbound(server *Server, session *Session, dialog *outboundDialog, media *outboundMedia, request OutboundInviteRequest) *Outbound {
	return &Outbound{
		server:  server,
		session: session,
		dialog:  dialog,
		media:   media,
		request: request,
	}
}

// Start runs the outbound call lifecycle asynchronously.
func (outboundCall *Outbound) Start() {
	go outboundCall.HandleCall()
}

// HandleCall connects the outbound INVITE and then observes dialog teardown.
func (outboundCall *Outbound) HandleCall() {
	defer outboundCall.dialog.Close()
	answerTime, err := outboundCall.Connect()
	if err != nil {
		return
	}

	if err := outboundCall.callOutboundInviteHandler(answerTime); err != nil {
		outboundCall.failAfterAnswer(OutboundFailure{
			Class:           OutboundFailureApplication,
			Reason:          err.Error(),
			Termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_application"},
			LifecycleReason: LifecycleReasonPipelineSetupFailed,
			Err:             err,
		})
	}
	outboundCall.waitForSessionEnd(answerTime)
}

// Connect waits for the outbound INVITE answer, prepares media, starts RTP, and sends ACK.
// It owns setup failure side effects; steady-state call handling remains in HandleCall.
func (outboundCall *Outbound) Connect() (time.Time, error) {
	outboundConfig := outboundCall.session.config.ToOutboundConfig()
	ringingTimeout := outboundConfig.EffectiveRingingTimeout()
	assistantID := uint64(0)
	if assistant := outboundCall.session.GetAssistant(); assistant != nil {
		assistantID = assistant.Id
	}

	outboundCall.server.logger.Debugw("Outbound call waiting for answer",
		"call_id", outboundCall.session.GetCallID(),
		"context_id", outboundCall.session.GetContextID(),
		"assistant_id", assistantID,
		"conversation_id", outboundCall.session.GetConversationID(),
		"mode", outboundCall.request.Config.Mode,
		"to_user", outboundCall.request.Identity.ToUser,
		"from_user", outboundCall.request.Identity.FromUser,
		"trunk_address", outboundCall.request.Config.Address,
		"ringing_timeout_ms", ringingTimeout.Milliseconds(),
		"auth_username", outboundConfig.Auth.Username,
		"auth_realm", outboundConfig.Auth.Realm,
		"digest_uri", outboundCall.dialog.InviteRequest().Recipient.Addr(),
		"request_uri", outboundCall.dialog.InviteRequest().Recipient.String())

	if err := outboundCall.waitForAnswer(outboundConfig, ringingTimeout); err != nil {
		outboundCall.failBeforeAnswer(NewOutboundInviteFailure(err), outboundConfig.Auth)
		return time.Time{}, err
	}

	answerTime := time.Now()
	outboundCall.session.SetOutboundDialogPhase(OutboundDialogPhaseAnswered)
	outboundCall.server.logger.Infow("Outbound call 200 OK received; setting up RTP before ACK",
		"call_id", outboundCall.session.GetCallID())

	if failure := outboundCall.answerOutboundInvite(answerTime); failure != nil {
		outboundCall.failAfterAnswer(*failure)
		return time.Time{}, failure.Err
	}

	outboundCall.session.SetOutboundDialogPhase(OutboundDialogPhaseConfirmed)
	outboundCall.server.TransitionCall(outboundCall.session, CallStateConnected, LifecycleReasonOutboundACKSent)
	outboundCall.ReportStatus(internal_type.ProviderCallStatusUpdate{
		CallStatus:       string(OutboundCallStatusAnswered),
		DisconnectReason: LifecycleReasonOutboundACKSent.String(),
	})
	return answerTime, nil
}

func (outboundCall *Outbound) waitForAnswer(outboundConfig OutboundConfig, ringingTimeout time.Duration) error {
	answerParentContext := outboundCall.session.Context()
	if outboundCall.answerContext != nil {
		answerParentContext = outboundCall.answerContext
	}
	answerContext, cancelAnswerWithCause := context.WithCancelCause(answerParentContext)
	defer cancelAnswerWithCause(nil)

	var answerCompleted atomic.Bool
	var preAnswerCancelOnce sync.Once
	var ringingTimeoutReached atomic.Bool
	var ringingReported atomic.Bool
	sendPreAnswerCancel := func() {
		preAnswerCancelOnce.Do(func() {
			outboundCall.cancelBeforeAnswer()
			cancelAnswerWithCause(sipgo.WaitAnswerForceCancelErr)
		})
	}

	stopParentCancel := context.AfterFunc(answerParentContext, func() {
		if !answerCompleted.Load() {
			sendPreAnswerCancel()
		}
	})
	defer stopParentCancel()

	if ringingTimeout > 0 {
		ringingTimer := time.AfterFunc(ringingTimeout, func() {
			if answerCompleted.Load() {
				return
			}
			ringingTimeoutReached.Store(true)
			sendPreAnswerCancel()
		})
		defer ringingTimer.Stop()
	}
	outboundCall.session.SetOnPreAnswerCancel(sendPreAnswerCancel)
	defer outboundCall.session.ClearOnPreAnswerCancel()

	err := outboundCall.dialog.WaitAnswer(answerContext, sipgo.AnswerOptions{
		Username: outboundConfig.Auth.Username,
		Password: outboundConfig.Auth.Password,
		OnResponse: func(response *sip.Response) error {
			outboundCall.server.logger.Debugw("Outbound call response",
				"call_id", outboundCall.session.GetCallID(),
				"status", response.StatusCode)

			if outboundAuthMissingForChallenge(outboundConfig.Auth, response.StatusCode) {
				return ErrAuthRequired
			}
			switch response.StatusCode {
			case 180, 183:
				outboundCall.session.SetOutboundDialogPhase(OutboundDialogPhaseProceeding)
				outboundCall.server.TransitionCall(outboundCall.session, CallStateRinging, LifecycleReasonOutboundProgressRinging)
				if ringingReported.CompareAndSwap(false, true) {
					outboundCall.ReportStatus(internal_type.ProviderCallStatusUpdate{
						CallStatus:         string(OutboundCallStatusRinging),
						DisconnectReason:   LifecycleReasonOutboundProgressRinging.String(),
						ProviderStatusCode: response.StatusCode,
					})
				}
			}

			outboundCall.dialog.LogAuthChallenge(response, outboundConfig.Auth)
			return nil
		},
	})
	answerCompleted.Store(true)
	if err == nil {
		return nil
	}
	if ringingTimeoutReached.Load() {
		return context.DeadlineExceeded
	}
	if parentErr := answerParentContext.Err(); parentErr != nil {
		return parentErr
	}
	return err
}

// answerOutboundInvite validates the answer, starts media, and ACKs the 200 OK.
// It returns a classified failure; the caller owns terminal side effects.
func (outboundCall *Outbound) answerOutboundInvite(answerTime time.Time) *OutboundFailure {
	mediaAnswer, err := NewOutboundMediaAnswer(outboundCall.server, outboundCall.dialog)
	if err != nil {
		if ackErr := outboundCall.ackAnswer(); ackErr != nil {
			outboundCall.server.logger.Warnw("Failed to ACK rejected outbound answer",
				"call_id", outboundCall.session.GetCallID(),
				"error", ackErr)
		} else {
			outboundCall.session.SetOutboundDialogPhase(OutboundDialogPhaseConfirmed)
		}
		return &OutboundFailure{
			Class:           OutboundFailureMedia,
			Reason:          err.Error(),
			Termination:     CallTermination{Result: CallTerminationClientError, Reason: "outbound_answer_sdp_failed"},
			LifecycleReason: LifecycleReasonOutboundAnswerSDPFailed,
			Err:             err,
		}
	}
	if err := outboundCall.media.ApplyAnswer(mediaAnswer); err != nil {
		return &OutboundFailure{
			Class:           OutboundFailureMedia,
			Reason:          err.Error(),
			Termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_media_apply_failed"},
			LifecycleReason: LifecycleReasonOutboundAnswerSDPFailed,
			Err:             err,
		}
	}
	if mediaAnswer.negotiatedCodec != nil {
		outboundCall.server.logger.Infow("Outbound call codec negotiated from 200 OK",
			"call_id", outboundCall.session.GetCallID(),
			"codec", mediaAnswer.negotiatedCodec.Name,
			"payload_type", mediaAnswer.negotiatedCodec.PayloadType,
			"clock_rate", mediaAnswer.negotiatedCodec.ClockRate)
	}

	if failure := outboundCall.startOutboundMedia(mediaAnswer, answerTime); failure != nil {
		return failure
	}
	if err := outboundCall.ackAnswer(); err != nil {
		outboundCall.server.logger.Errorw("Failed to send ACK", "error", err, "call_id", outboundCall.session.GetCallID())
		return &OutboundFailure{
			Class:           OutboundFailureUnknown,
			Reason:          err.Error(),
			Termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_failure"},
			LifecycleReason: LifecycleReasonOutboundACKFailed,
			Err:             err,
		}
	}
	outboundCall.server.logger.Infow("ACK sent (RTP already flowing)",
		"call_id", outboundCall.session.GetCallID(),
		"elapsed_since_200ok_ms", time.Since(answerTime).Milliseconds())
	return nil
}

func (outboundCall *Outbound) startOutboundMedia(mediaAnswer OutboundMediaAnswer, answerTime time.Time) *OutboundFailure {
	if err := outboundCall.media.Start(outboundCall.mediaTimeout); err != nil {
		return &OutboundFailure{
			Class:           OutboundFailureMedia,
			Reason:          err.Error(),
			Termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_media_start_failed"},
			LifecycleReason: LifecycleReasonOutboundAnswerSDPFailed,
			Err:             err,
		}
	}

	localIP, localPort := outboundCall.media.LocalAddr()
	outboundCall.server.logger.Infow("RTP started (pre-ACK)",
		"call_id", outboundCall.session.GetCallID(),
		"local_rtp", fmt.Sprintf("%s:%d", localIP, localPort),
		"remote_rtp", fmt.Sprintf("%s:%d", mediaAnswer.remoteIP, mediaAnswer.remotePort),
		"remote_addr_set", outboundCall.media.RemoteAddrConfigured(),
		"elapsed_since_200ok_ms", time.Since(answerTime).Milliseconds())
	return nil
}

func (outboundCall *Outbound) ackAnswer() error {
	ackContext, cancelAck := context.WithTimeout(outboundCall.session.Context(), 5*time.Second)
	defer cancelAck()
	return outboundCall.dialog.AckAnswer(ackContext)
}

func (outboundCall *Outbound) cancelBeforeAnswer() {
	cancelContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := outboundCall.dialog.CancelBeforeAnswer(cancelContext); err != nil {
		outboundCall.server.logger.Warnw("Failed to send outbound SIP CANCEL",
			"call_id", outboundCall.session.GetCallID(),
			"error", err)
	}
}

func (outboundCall *Outbound) callOutboundInviteHandler(answerTime time.Time) error {
	outboundCall.server.mu.RLock()
	inviteHandler := outboundCall.server.onInvite
	outboundCall.server.mu.RUnlock()
	if inviteHandler == nil {
		return nil
	}

	info := outboundCall.session.GetInfo()
	outboundCall.server.logger.Infow("Starting onInvite handler for outbound call",
		"call_id", outboundCall.session.GetCallID())
	if err := inviteHandler(outboundCall.session, info.LocalURI, info.RemoteURI); err != nil {
		return err
	}
	outboundCall.server.logger.Infow("onInvite handler completed",
		"call_id", outboundCall.session.GetCallID(),
		"total_elapsed_ms", time.Since(answerTime).Milliseconds())
	return nil
}

// ReportStatus sends an outbound provider status update for this call.
func (outboundCall *Outbound) ReportStatus(update internal_type.ProviderCallStatusUpdate) {
	if outboundCall.statusObserver == nil {
		return
	}
	update.ChannelUUID = outboundCall.session.GetCallID()
	outboundCall.statusObserver(update)
}

func (outboundCall *Outbound) reportFailure(failure OutboundFailure) {
	outboundCall.ReportStatus(failure.StatusUpdate(outboundCall.session.GetCallID()))
}

func (outboundCall *Outbound) reportCompleted(disconnectReason string) {
	outboundCall.ReportStatus(internal_type.ProviderCallStatusUpdate{
		CallStatus:       string(OutboundCallStatusCompleted),
		DisconnectReason: disconnectReason,
	})
}

func (outboundCall *Outbound) recordFailure(failure OutboundFailure) {
	failure.Record(outboundCall.session)
}

func (outboundCall *Outbound) failBeforeAnswer(failure OutboundFailure, auth SIPAuthConfig) {
	outboundCall.recordFailure(failure)
	outboundCall.reportFailure(failure)
	if outboundCall.session.IsEnded() {
		return
	}

	err := failure.Err
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		outboundCall.server.logger.Warnw("Outbound call ringing timeout reached; INVITE cancelled",
			"call_id", outboundCall.session.GetCallID(),
			"ringing_timeout_ms", outboundCall.request.Config.EffectiveRingingTimeout().Milliseconds())
	case errors.Is(err, context.Canceled):
		outboundCall.server.logger.Infow("Outbound call cancelled before answer",
			"call_id", outboundCall.session.GetCallID(),
			"reason", "context_cancelled")
		_ = outboundCall.server.CancelCall(outboundCall.session, LifecycleReasonOutboundCancelledBeforeAnswer)
		outboundCall.dialog.CloseAfter(2 * time.Second)
		return
	case errors.Is(err, ErrAuthRequired):
		outboundCall.server.logger.Errorw("Outbound call authentication required but credentials are missing",
			"call_id", outboundCall.session.GetCallID(),
			"auth_username_set", auth.Username != "",
			"auth_password_set", auth.Password != "",
			"failure_class", failure.Class)
	}

	var dialogErr *sipgo.ErrDialogResponse
	if errors.As(err, &dialogErr) {
		if dialogErr.Res.StatusCode == 401 || dialogErr.Res.StatusCode == 407 {
			outboundCall.server.logger.Errorw("Outbound call authentication failed; check SIP credentials in vault",
				"call_id", outboundCall.session.GetCallID(),
				"status", dialogErr.Res.StatusCode,
				"reason", dialogErr.Res.Reason,
				"auth_username", auth.Username,
				"auth_password_set", len(auth.Password) > 0,
				"digest_uri", outboundCall.dialog.InviteRequest().Recipient.Addr(),
				"failure_class", failure.Class,
				"hint", "Verify sip_username and sip_password in vault match the SIP provider's auth credentials")
		} else {
			outboundCall.server.logger.Warnw("Outbound call rejected by remote",
				"call_id", outboundCall.session.GetCallID(),
				"status", dialogErr.Res.StatusCode,
				"reason", dialogErr.Res.Reason,
				"failure_class", failure.Class,
				"retryable", failure.Retryable)
		}
	} else if !errors.Is(err, context.Canceled) {
		outboundCall.server.logger.Warnw("Outbound call failed",
			"call_id", outboundCall.session.GetCallID(),
			"error", err,
			"failure_class", failure.Class,
			"failure_reason", failure.Reason,
			"retryable", failure.Retryable)
	}
	_ = outboundCall.server.FailCall(outboundCall.session, failure.LifecycleReason, err)
	outboundCall.dialog.CloseAfter(2 * time.Second)
}

func (outboundCall *Outbound) failAfterAnswer(failure OutboundFailure) {
	outboundCall.closeOnce.Do(func() {
		outboundCall.recordFailure(failure)
		outboundCall.reportFailure(failure)
		_ = outboundCall.server.FailCall(outboundCall.session, failure.LifecycleReason, failure.Err)
	})
}

func (outboundCall *Outbound) mediaTimeout() {
	if outboundCall.session == nil || outboundCall.session.IsEnded() {
		return
	}
	if outboundCall.hasReceivedRTP() {
		outboundCall.completeAfterEstablishedMediaTimeout()
		return
	}
	if outboundCall.server.logger != nil {
		outboundCall.server.logger.Warnw("Outbound SIP RTP media timed out",
			"call_id", outboundCall.session.GetCallID(),
			"reason", LifecycleReasonOutboundMediaTimeout)
	}
	outboundCall.failAfterAnswer(OutboundFailure{
		Class:           OutboundFailureMedia,
		Reason:          ErrRTPMediaTimeout.Error(),
		Termination:     CallTermination{Result: CallTerminationServerError, Reason: "outbound_media_timeout"},
		Retryable:       true,
		LifecycleReason: LifecycleReasonOutboundMediaTimeout,
		Err:             ErrRTPMediaTimeout,
	})
}

func (outboundCall *Outbound) hasReceivedRTP() bool {
	if outboundCall == nil || outboundCall.session == nil {
		return false
	}
	rtpHandler := outboundCall.session.GetRTPHandler()
	if rtpHandler == nil && outboundCall.media != nil {
		rtpHandler = outboundCall.media.rtpHandler
	}
	if rtpHandler == nil {
		return false
	}
	_, received := rtpHandler.GetStats()
	return received > 0
}

func (outboundCall *Outbound) completeAfterEstablishedMediaTimeout() {
	outboundCall.closeOnce.Do(func() {
		if outboundCall.session == nil || outboundCall.session.IsEnded() {
			return
		}
		outboundCall.session.SetDisconnectMetadata(DisconnectMetadata{
			Reason: DisconnectReasonRemoteHangup,
		})
		outboundCall.session.SetMetadata("sip.media_timeout", true)
		outboundCall.session.SetMetadata("sip.media_timeout_after_established_media", true)
		outboundCall.reportCompleted(DisconnectReasonRemoteHangup)
		_ = outboundCall.server.EndCallWithReason(outboundCall.session, LifecycleReasonOutboundMediaTimeout)
	})
}

func (outboundCall *Outbound) waitForSessionEnd(answerTime time.Time) {
	maxCallDuration := outboundCall.request.Config.EffectiveMaxCallDuration()

	var maxDurationC <-chan time.Time
	var maxDurationTimer *time.Timer
	if maxCallDuration > 0 {
		maxDurationTimer = time.NewTimer(maxCallDuration)
		maxDurationC = maxDurationTimer.C
		defer maxDurationTimer.Stop()
	}

	select {
	case <-maxDurationC:
		outboundCall.server.logger.Infow("Outbound call max duration reached; ending dialog",
			"call_id", outboundCall.session.GetCallID(),
			"max_call_duration_ms", maxCallDuration.Milliseconds())
		outboundCall.closeOnce.Do(func() {
			_ = outboundCall.server.EndCallWithReason(outboundCall.session, LifecycleReasonOutboundMaxDuration)
		})
	case <-outboundCall.session.Context().Done():
		return
	case <-outboundCall.dialog.Done():
		select {
		case <-outboundCall.session.Context().Done():
			return
		case <-time.After(30 * time.Second):
			outboundCall.server.logger.Warnw("Outbound dialog session did not end within 30s after BYE; forcing teardown",
				"call_id", outboundCall.session.GetCallID())
			if !outboundCall.session.IsEnded() {
				_ = outboundCall.server.EndCallWithReason(outboundCall.session, LifecycleReasonOutboundTeardownTimeout)
			}
		}
	}
}
