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
	"time"

	"github.com/emiago/sipgo/sip"
)

type inboundInviteIdentity struct {
	callID  string
	fromTag string
	fromURI string
	toURI   string
}

// Inbound owns the lifecycle for a single inbound SIP INVITE.
type Inbound struct {
	server      *Server
	request     *sip.Request
	transaction sip.ServerTransaction

	identity       inboundInviteIdentity
	inviteKey      inboundInviteKey
	mediaOffer     inboundMediaOffer
	resolvedConfig inboundConfig

	session           *Session
	dialog            *inboundDialog
	media             *inboundMedia
	finalResponseOnce sync.Once
}

// NewInbound creates an inbound SIP call handler.
func NewInbound(server *Server, request *sip.Request, transaction sip.ServerTransaction) *Inbound {
	return &Inbound{
		server:      server,
		request:     request,
		transaction: transaction,
	}
}

func inboundInviteIdentityFromRequest(request *sip.Request) (inboundInviteIdentity, bool) {
	if request == nil ||
		request.CallID() == nil ||
		request.CallID().Value() == "" ||
		request.From() == nil ||
		request.From().Params == nil ||
		request.To() == nil {
		return inboundInviteIdentity{}, false
	}
	fromTag, ok := request.From().Params.Get("tag")
	if !ok || fromTag == "" {
		return inboundInviteIdentity{}, false
	}

	return inboundInviteIdentity{
		callID:  request.CallID().Value(),
		fromTag: fromTag,
		fromURI: request.From().Address.String(),
		toURI:   request.To().Address.String(),
	}, true
}

// newInboundSession creates the call session but does not register it.
// HandleInvite owns registration, cancellation checks, and terminal cleanup.
func newInboundSession(
	ctx context.Context,
	resolvedConfig inboundConfig,
	identity inboundInviteIdentity,
	mediaOffer inboundMediaOffer,
	setupPhase InboundSetupPhase,
	setupTimings InboundSetupTimings,
) (*Session, *inboundFailure) {
	session, err := NewSession(ctx, &SessionConfig{
		Config:          resolvedConfig.config,
		Direction:       CallDirectionInbound,
		CallID:          identity.callID,
		Codec:           mediaOffer.negotiatedCodec,
		Auth:            resolvedConfig.auth,
		Assistant:       resolvedConfig.assistant,
		VaultCredential: resolvedConfig.vaultCredential,
	})
	if err != nil {
		sessionErr := fmt.Errorf("failed to create inbound session: %w", err)
		failure := newInboundSessionFailure(sessionErr)
		return nil, &failure
	}

	if setupPhase != "" {
		session.SetInboundSetupPhase(setupPhase)
	}
	session.SetInboundSetupTimings(setupTimings)
	return session, nil
}

// HandleInvite processes the inbound INVITE lifecycle.
func (inboundCall *Inbound) HandleInvite() {
	identity, ok := inboundInviteIdentityFromRequest(inboundCall.request)
	if !ok {
		inboundCall.server.sendResponse(inboundCall.transaction, inboundCall.request, 400)
		return
	}
	inboundCall.identity = identity
	inboundCall.inviteKey = inboundInviteKey{callID: identity.callID, fromTag: identity.fromTag}
	setupPhase := InboundSetupPhaseInviteReceived
	setupTimings := InboundSetupTimings{InviteReceivedAt: time.Now()}

	if inboundCall.server.replayRejectedInboundInvite(inboundCall.request, inboundCall.transaction) {
		return
	}

	setupTimings.TryingSentAt = time.Now()
	inboundCall.server.sendResponse(inboundCall.transaction, inboundCall.request, 100)

	inboundCall.server.setPendingInvite(inboundCall.inviteKey, inboundCall.request, inboundCall.transaction)
	defer func() {
		inboundCall.server.clearPendingInvite(inboundCall.inviteKey)
		inboundCall.server.clearInviteCancelled(inboundCall.inviteKey)
	}()

	inboundCall.server.mu.RLock()
	existingSession, isReInvite := inboundCall.server.sessions[inboundCall.identity.callID]
	inboundCall.server.mu.RUnlock()
	if isReInvite && existingSession != nil {
		inboundCall.server.handleReInvite(inboundCall.request, inboundCall.transaction, existingSession)
		return
	}

	if inboundCall.server.isInviteCancelled(inboundCall.inviteKey) {
		inboundCall.server.terminatePendingInvite(inboundCall.inviteKey, 487)
		return
	}

	mediaOffer, mediaFailure := NewInboundMediaOffer(
		inboundCall.server,
		inboundCall.request,
		"inbound INVITE",
		LifecycleReasonInboundInviteFailed,
		false,
	)
	if mediaFailure != nil {
		inboundCall.server.RejectInboundInvite(
			inboundCall.request,
			inboundCall.transaction,
			inboundCall.identity.callID,
			mediaFailure.statusCode,
			mediaFailure.responseClass,
			mediaFailure.lifecycleReason,
			mediaFailure.err,
		)
		return
	}
	inboundCall.mediaOffer = mediaOffer
	if inboundCall.server.isInviteCancelled(inboundCall.inviteKey) {
		inboundCall.server.terminatePendingInvite(inboundCall.inviteKey, 487)
		return
	}

	resolvedConfig, configFailure := NewInboundConfig(inboundCall.server, inboundCall.identity, inboundCall.mediaOffer)
	if configFailure != nil {
		inboundCall.server.RejectInboundInvite(
			inboundCall.request,
			inboundCall.transaction,
			inboundCall.identity.callID,
			configFailure.statusCode,
			configFailure.responseClass,
			configFailure.lifecycleReason,
			configFailure.err,
		)
		return
	}
	inboundCall.resolvedConfig = resolvedConfig
	setupPhase = inboundCall.resolvedConfig.setupPhase
	if inboundCall.server.isInviteCancelled(inboundCall.inviteKey) {
		inboundCall.server.terminatePendingInvite(inboundCall.inviteKey, 487)
		return
	}

	session, sessionFailure := newInboundSession(
		inboundCall.server.ctx,
		inboundCall.resolvedConfig,
		inboundCall.identity,
		inboundCall.mediaOffer,
		setupPhase,
		setupTimings,
	)
	if sessionFailure != nil {
		inboundCall.server.RejectInboundInvite(
			inboundCall.request,
			inboundCall.transaction,
			inboundCall.identity.callID,
			sessionFailure.statusCode,
			sessionFailure.responseClass,
			sessionFailure.lifecycleReason,
			sessionFailure.err,
		)
		return
	}
	inboundCall.session = session

	inboundCall.server.registerSession(inboundCall.session, inboundCall.identity.callID)
	if inboundCall.cancelBeforeAnswer(LifecycleReasonInviteCancelled) {
		return
	}

	dialog, dialogFailure := NewInboundDialog(
		inboundCall.server,
		inboundCall.session,
		inboundCall.request,
		inboundCall.transaction,
		inboundCall.inviteKey,
	)
	if dialogFailure != nil {
		inboundCall.failBeforeAnswer(*dialogFailure)
		return
	}
	inboundCall.dialog = dialog
	inboundCall.session.SetDialogServerSession(inboundCall.dialog.DialogSession())

	if err := inboundCall.dialog.StartRinging(inboundCall.server.ctx); err != nil {
		inboundCall.failBeforeAnswer(newInboundDialogFailure(500, err))
		return
	}
	inboundCall.session.SetInboundSetupPhase(InboundSetupPhaseRingingSent)
	inboundCall.session.MarkInboundSetupTimestamp(InboundSetupPhaseRingingSent, time.Now())
	inboundCall.server.logger.Infow("Inbound SIP setup phase",
		"call_id", inboundCall.identity.callID,
		"phase", InboundSetupPhaseRingingSent,
		"reason", LifecycleReasonInboundInviteRinging)
	inboundCall.server.TransitionCall(inboundCall.session, CallStateRinging, LifecycleReasonInboundInviteRinging)

	inboundCall.media = NewInboundMedia(inboundCall.server, inboundCall.session, inboundCall.mediaOffer)
	if err := inboundCall.media.Prepare(); err != nil {
		inboundCall.failBeforeAnswer(newInboundRTPUnavailableFailure(err, LifecycleReasonInboundInviteFailed))
		return
	}
	inboundCall.session.SetInboundSetupPhase(InboundSetupPhaseMediaAllocated)
	if inboundCall.cancelBeforeAnswer(LifecycleReasonInviteCancelledBeforeAnswer) {
		return
	}

	if err := inboundCall.callInboundApplicationReadyHandler(); err != nil {
		applicationErr := fmt.Errorf("application readiness failed: %w", err)
		failure := newInboundApplicationFailure(applicationErr)
		failure.statusCode = 503
		inboundCall.failBeforeAnswer(failure)
		return
	}
	inboundCall.session.SetInboundSetupPhase(InboundSetupPhaseApplicationReady)
	if inboundCall.cancelBeforeAnswer(LifecycleReasonInviteCancelledBeforeAnswer) {
		return
	}
	if err := inboundCall.waitUntilAnswerReady(); err != nil {
		inboundCall.failBeforeAnswer(newInboundNoAnswerFailure(err))
		return
	}
	inboundCall.session.SetInboundSetupPhase(InboundSetupPhaseAnswerReady)
	inboundCall.server.logger.Infow("Inbound SIP setup phase",
		"call_id", inboundCall.identity.callID,
		"phase", InboundSetupPhaseAnswerReady,
		"reason", LifecycleReasonInboundAnswerPolicyReady)
	if inboundCall.cancelBeforeAnswer(LifecycleReasonInviteCancelledBeforeAnswer) {
		return
	}

	cancelled, answerFailure := inboundCall.answerInboundInvite()
	if cancelled {
		return
	}
	if answerFailure != nil {
		if answerFailure.class == inboundFailureNoACK || answerFailure.statusCode == 0 {
			inboundCall.finalResponseOnce.Do(func() {
				inboundCall.recordFailure(*answerFailure)
				inboundCall.cleanupApplication()
				_ = inboundCall.server.FailInboundCall(inboundCall.session, answerFailure.lifecycleReason, answerFailure.err)
			})
			return
		}
		inboundCall.failBeforeAnswer(*answerFailure)
		return
	}
	if mediaStartFailure := inboundCall.startInboundMedia(); mediaStartFailure != nil {
		inboundCall.finalResponseOnce.Do(func() {
			inboundCall.recordFailure(*mediaStartFailure)
			inboundCall.cleanupApplication()
			_ = inboundCall.server.FailInboundCall(inboundCall.session, mediaStartFailure.lifecycleReason, mediaStartFailure.err)
		})
		return
	}
	inboundCall.session.SetInboundSetupPhase(InboundSetupPhaseMediaFlowing)

	if err := inboundCall.callInboundInviteHandler(); err != nil {
		inboundCall.server.notifyError(inboundCall.session, err)
		failure := newInboundApplicationFailure(err)
		inboundCall.finalResponseOnce.Do(func() {
			inboundCall.recordFailure(failure)
			inboundCall.cleanupApplication()
			_ = inboundCall.server.FailInboundCall(inboundCall.session, failure.lifecycleReason, failure.err)
		})
	}
}

// callInboundApplicationReadyHandler invokes the optional application readiness hook.
// The hook result is returned raw so HandleInvite can own failure classification.
func (inboundCall *Inbound) callInboundApplicationReadyHandler() error {
	// Snapshot callbacks under lock, then invoke outside it so handlers can call Server APIs.
	inboundCall.server.mu.RLock()
	applicationReadyHandler := inboundCall.server.onApplicationReady
	inboundCall.server.mu.RUnlock()
	if applicationReadyHandler == nil {
		return nil
	}
	return applicationReadyHandler(inboundCall.session, inboundCall.identity.fromURI, inboundCall.identity.toURI)
}

// waitUntilAnswerReady applies the configured answer policy without changing call state.
// HandleInvite records the answer-ready phase only after this wait succeeds.
func (inboundCall *Inbound) waitUntilAnswerReady() error {
	answerPolicy := inboundCall.resolvedConfig.answerPolicy
	if answerPolicy.Mode == "" {
		answerPolicy = DefaultInboundAnswerPolicy()
	}
	if !answerPolicy.Mode.IsValid() {
		return fmt.Errorf("%w: invalid inbound answer mode %q", ErrInvalidConfig, answerPolicy.Mode)
	}

	switch answerPolicy.Mode {
	case InboundAnswerModeImmediate:
	case InboundAnswerModeAfterMinRingDuration:
		if answerPolicy.MinRingDuration <= 0 {
			return fmt.Errorf("%w: min_ring_duration is required for answer_after_min_ring_ms", ErrInvalidConfig)
		}
		if err := inboundCall.waitForMinimumRingDuration(inboundCall.server.ctx, answerPolicy.MinRingDuration); err != nil {
			return err
		}
	}
	return nil
}

func (inboundCall *Inbound) waitForMinimumRingDuration(ctx context.Context, minRingDuration time.Duration) error {
	setupTimings := inboundCall.session.GetInboundSetupTimings()
	if minRingDuration <= 0 || setupTimings.RingingSentAt.IsZero() {
		return nil
	}
	if remainingDuration := minRingDuration - time.Since(setupTimings.RingingSentAt); remainingDuration > 0 {
		ringTimer := time.NewTimer(remainingDuration)
		defer ringTimer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-inboundCall.server.ctx.Done():
			return inboundCall.server.ctx.Err()
		case <-inboundCall.session.Context().Done():
			return inboundCall.session.Context().Err()
		case <-ringTimer.C:
		}
	}
	return nil
}

// answerInboundInvite sends the final SIP answer and waits for the initial ACK.
// It classifies ACK/cancel/dialog failures without closing the session itself.
func (inboundCall *Inbound) answerInboundInvite() (bool, *inboundFailure) {
	finalResponseSent := false

	err := inboundCall.dialog.AnswerAndWaitACK(
		inboundCall.server.ctx,
		inboundCall.server.GenerateSDP(inboundCall.media.SDPConfig()),
		inboundCall.resolvedConfig.answerPolicy.ACKTimeout,
		func(answeredAt time.Time) {
			finalResponseSent = true
			inboundCall.session.SetInboundSetupPhase(InboundSetupPhaseAnswered)
			inboundCall.session.MarkInboundSetupTimestamp(InboundSetupPhaseAnswered, answeredAt)
			inboundCall.server.logger.Infow("Inbound SIP setup phase",
				"call_id", inboundCall.identity.callID,
				"phase", InboundSetupPhaseAnswered,
				"reason", LifecycleReasonInboundInviteAnswered)
		},
	)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, ErrInboundInviteCancelled) {
		return true, nil
	}
	if errors.Is(err, ErrInboundACKTimeout) {
		failure := newInboundNoACKFailure(err)
		return false, &failure
	}
	statusCode := 500
	if finalResponseSent {
		statusCode = 0
	}
	failure := newInboundDialogFailure(statusCode, err)
	return false, &failure
}

// startInboundMedia starts RTP after the final response is established.
// Media timeout handling is attached here, but terminal failure remains in HandleInvite.
func (inboundCall *Inbound) startInboundMedia() *inboundFailure {
	if err := inboundCall.media.Start(inboundCall.mediaTimeout); err != nil {
		failure := newInboundRTPUnavailableFailure(err, LifecycleReasonInboundMediaFailed)
		return &failure
	}
	return nil
}

// callInboundInviteHandler invokes the post-answer application hook.
// Errors are returned raw so HandleInvite owns notification and call failure.
func (inboundCall *Inbound) callInboundInviteHandler() error {
	// Snapshot callbacks under lock, then invoke outside it so handlers can call Server APIs.
	inboundCall.server.mu.RLock()
	inviteHandler := inboundCall.server.onInvite
	inboundCall.server.mu.RUnlock()
	if inviteHandler == nil {
		return nil
	}
	return inviteHandler(inboundCall.session, inboundCall.identity.fromURI, inboundCall.identity.toURI)
}

func (inboundCall *Inbound) cancelBeforeAnswer(reason LifecycleReason) bool {
	if inboundCall.session == nil || !inboundCall.server.isInviteCancelled(inboundCall.inviteKey) {
		return false
	}
	if inboundCall.dialog != nil {
		inboundCall.dialog.CancelBeforeAnswer()
	} else {
		inboundCall.server.terminatePendingInvite(inboundCall.inviteKey, 487)
	}
	inboundCall.cleanupApplication()
	_ = inboundCall.server.CancelInboundCall(inboundCall.session, reason)
	return true
}

func (inboundCall *Inbound) mediaTimeout() {
	if inboundCall.session == nil || inboundCall.session.IsEnded() {
		return
	}

	failure := newInboundMediaTimeoutFailure()

	// Once ACK is confirmed, media timeout usually means the peer stopped
	// sending RTP or the final BYE was lost. Preserve the server-error
	// termination signal, but close the call as an ended session.
	inboundCall.finalResponseOnce.Do(func() {
		if inboundCall.session == nil {
			return
		}
		inboundCall.recordFailure(failure)
		inboundCall.cleanupApplication()
		_ = inboundCall.server.EndInboundCall(inboundCall.session, failure.lifecycleReason)
	})
}

func (inboundCall *Inbound) recordFailure(failure inboundFailure) {
	if inboundCall.session == nil {
		return
	}
	inboundCall.session.SetMetadata("sip.failure_class", string(failure.class))
	inboundCall.session.SetMetadata("sip.failure_response_class", string(failure.responseClass))
	inboundCall.session.SetMetadata("sip.failure_reason", failure.reason)
	inboundCall.session.SetMetadata("sip.sli_result", string(failure.termination.Result))
	inboundCall.session.SetMetadata("sip.sli_reason", failure.termination.Reason)
	inboundCall.session.SetMetadata("sip.failure_retryable", failure.retryable)
	if failure.statusCode > 0 {
		inboundCall.session.SetMetadata("sip.failure_status_code", failure.statusCode)
	}
}

// failBeforeAnswer owns terminal setup failure side effects before the final INVITE answer.
// It sends one final SIP response and then fails the session.
func (inboundCall *Inbound) failBeforeAnswer(failure inboundFailure) {
	inboundCall.recordFailure(failure)
	if failure.statusCode > 0 {
		if inboundCall.dialog != nil {
			inboundCall.dialog.RejectBeforeAnswer(failure.statusCode)
		} else {
			response := inboundCall.server.sendResponse(inboundCall.transaction, inboundCall.request, failure.statusCode)
			inboundCall.server.recordRejectedInboundInvite(inboundCall.request, response)
		}
	}
	inboundCall.cleanupApplication()
	_ = inboundCall.server.FailInboundCall(inboundCall.session, failure.lifecycleReason, failure.err)
}

func (inboundCall *Inbound) cleanupApplication() {
	inboundCall.server.mu.RLock()
	onApplicationCleanup := inboundCall.server.onApplicationCleanup
	inboundCall.server.mu.RUnlock()
	if onApplicationCleanup != nil && inboundCall.session != nil {
		onApplicationCleanup(inboundCall.session)
	}
}
