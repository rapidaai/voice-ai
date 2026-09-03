// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// GetSession returns the session for a call ID.
func (s *Server) GetSession(callID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[callID]
	return session, exists
}

// EndCall terminates a call using lifecycle-aware signaling.
func (s *Server) EndCall(session *Session) error {
	return s.EndCallWithReason(session, LifecycleReasonEndCall)
}

func (s *Server) TransitionCall(session *Session, next CallState, reason LifecycleReason) bool {
	if session == nil || session.IsEnded() {
		return false
	}
	return s.setCallState(session, next, reason.String())
}

func (s *Server) EndCallWithReason(session *Session, reason LifecycleReason) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	if session.IsEnded() {
		return nil
	}
	if s.shouldCancelBeforeAnswer(session) {
		return s.CancelCall(session, reason)
	}
	s.logLifecycleTeardown(session, reason, "bye")
	s.beginEnding(session, reason.String())
	session.End()
	return nil
}

func (s *Server) FailCall(session *Session, reason LifecycleReason, err error) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	if session.IsEnded() {
		return nil
	}
	preAnswer := s.shouldCancelBeforeAnswer(session) || s.shouldCancelPendingInvite(session)
	s.setCallState(session, CallStateFailed, reason.String())
	if err != nil {
		s.notifyError(session, err)
	}
	s.beginEnding(session, reason.String())
	if preAnswer {
		s.logLifecycleTeardown(session, reason, "cancel")
		session.ClearOnDisconnect()
	} else {
		s.logLifecycleTeardown(session, reason, "bye")
	}
	session.End()
	return nil
}

func (s *Server) CancelCall(session *Session, reason LifecycleReason) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	if session.IsEnded() {
		return nil
	}
	if !s.shouldCancelPendingInvite(session) {
		return s.EndCallWithReason(session, reason)
	}
	return s.cancelPendingInvite(session, reason)
}

func (s *Server) rejectInboundInvite(req *sip.Request, tx sip.ServerTransaction, callID string, statusCode int, failureClass inboundFailureClass, reason LifecycleReason, err error) {
	if callID != "" {
		s.recordDetachedInboundReject(callID, reason)
	}
	if s.logger != nil {
		s.logger.Warnw("Inbound INVITE rejected",
			"call_id", callID,
			"status_code", statusCode,
			"failure_class", string(failureClass),
			"reason", reason,
			"error", err)
	}
	response := s.sendResponse(tx, req, statusCode)
	s.recordRejectedInboundInvite(req, response)
}

func (s *Server) ConnectInboundCall(session *Session, reason LifecycleReason) bool {
	if !isInboundSession(session) {
		return false
	}
	return s.TransitionCall(session, CallStateConnected, reason)
}

func (s *Server) CancelInboundCall(session *Session, reason LifecycleReason) error {
	if !isInboundSession(session) {
		return fmt.Errorf("inbound session is required")
	}
	return s.CancelCall(session, reason)
}

func (s *Server) FailInboundCall(session *Session, reason LifecycleReason, err error) error {
	if !isInboundSession(session) {
		return fmt.Errorf("inbound session is required")
	}
	return s.FailCall(session, reason, err)
}

func (s *Server) EndInboundCall(session *Session, reason LifecycleReason) error {
	if !isInboundSession(session) {
		return fmt.Errorf("inbound session is required")
	}
	return s.EndCallWithReason(session, reason)
}

func (s *Server) cancelPendingInvite(session *Session, reason LifecycleReason) error {
	s.logLifecycleTeardown(session, reason, "cancel")
	s.setCallState(session, CallStateCancelled, reason.String())
	session.CancelPreAnswer()
	session.ClearOnDisconnect()
	session.End()
	return nil
}

func (s *Server) shouldCancelPendingInvite(session *Session) bool {
	if session == nil {
		return false
	}
	info := session.GetInfo()
	if info.Direction == CallDirectionInbound {
		return info.State == CallStateInitializing || info.State == CallStateRinging
	}
	return s.shouldCancelBeforeAnswer(session)
}

func (s *Server) shouldCancelBeforeAnswer(session *Session) bool {
	if session == nil {
		return false
	}
	info := session.GetInfo()
	if info.Direction != CallDirectionOutbound {
		return false
	}
	dialogPhase := session.GetOutboundDialogPhase()
	if dialogPhase != "" {
		return dialogPhase.IsPreAnswer()
	}
	return info.State == CallStateInitializing || info.State == CallStateRinging
}

func isInboundSession(session *Session) bool {
	if session == nil {
		return false
	}
	return session.GetInfo().Direction == CallDirectionInbound
}

func (s *Server) recordDetachedInboundReject(callID string, reason LifecycleReason) {
	lifecycle := newCallLifecycle(callID, CallStateInitializing, s.logger)
	_ = lifecycle.Transition(CallStateFailed, reason.String())
	_ = lifecycle.Transition(CallStateEnded, LifecycleReasonSessionEnd.String())
}

func (s *Server) logLifecycleTeardown(session *Session, reason LifecycleReason, teardownMethod string) {
	if s.logger == nil || session == nil {
		return
	}
	info := session.GetInfo()
	s.logger.Infow("SIP lifecycle teardown selected",
		"call_id", info.CallID,
		"state", info.State,
		"direction", info.Direction,
		"dialog_phase", session.GetOutboundDialogPhase(),
		"reason", reason,
		"teardown_method", teardownMethod)
}

// sendBye sends SIP BYE to the remote party via the active dialog session.
func (s *Server) sendBye(session *Session) {
	callID := session.GetCallID()

	if ds := session.GetDialogClientSession(); ds != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.sendOutboundBye(ctx, ds); err != nil {
			s.logger.Warnw("Failed to send BYE for outbound call",
				"call_id", callID, "error", err)
		} else {
			s.logger.Infow("Sent BYE for outbound call", "call_id", callID)
		}
	}

	if ds := session.GetDialogServerSession(); ds != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ds.Bye(ctx); err != nil {
			s.logger.Warnw("Failed to send BYE for inbound call",
				"call_id", callID, "error", err)
		} else {
			s.logger.Infow("Sent BYE for inbound call", "call_id", callID)
		}
	}
}

func (s *Server) sendOutboundBye(ctx context.Context, dialogSession *sipgo.DialogClientSession) error {
	return (&outboundDialog{dialogSession: dialogSession}).SendBye(ctx)
}

// removeSession removes a session from memory.
func (s *Server) removeSession(callID string) {
	s.mu.Lock()
	_, exists := s.sessions[callID]
	if exists {
		delete(s.sessions, callID)
		s.sessionCount.Add(-1)
	}
	delete(s.lifecycles, callID)
	s.mu.Unlock()
}

func (s *Server) getOrCreateLifecycle(session *Session) *CallLifecycle {
	if session == nil {
		return nil
	}
	callID := session.GetCallID()
	current := session.GetState()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lifecycles == nil {
		s.lifecycles = make(map[string]*CallLifecycle)
	}
	if lc, ok := s.lifecycles[callID]; ok && lc != nil {
		return lc
	}
	lc := newCallLifecycle(callID, current, s.logger)
	s.lifecycles[callID] = lc
	return lc
}

func (s *Server) transitionLifecycle(session *Session, next CallState, reason string) bool {
	lc := s.getOrCreateLifecycle(session)
	if lc == nil {
		return false
	}
	if err := lc.Transition(next, reason); err != nil {
		s.logger.Warnw("Call lifecycle transition rejected",
			"call_id", session.GetCallID(),
			"from", lc.State(),
			"to", next,
			"reason", reason,
			"error", err)
		return false
	}
	return true
}

func (s *Server) setCallState(session *Session, next CallState, reason string) bool {
	if session == nil {
		return false
	}
	if !s.transitionLifecycle(session, next, reason) {
		return false
	}
	session.SetState(next)
	s.syncOutboundDialogPhase(session, next)
	return true
}

func (s *Server) syncOutboundDialogPhase(session *Session, state CallState) {
	if session == nil || session.GetInfo().Direction != CallDirectionOutbound {
		return
	}
	switch state {
	case CallStateConnected:
		phase := session.GetOutboundDialogPhase()
		if phase == "" || phase.IsPreAnswer() || phase == OutboundDialogPhaseAnswered {
			session.SetOutboundDialogPhase(OutboundDialogPhaseConfirmed)
		}
	case CallStateEnded:
		session.SetOutboundDialogPhase(OutboundDialogPhaseTerminated)
	}
}

func (s *Server) beginEnding(session *Session, reason string) {
	if session == nil {
		return
	}
	_ = s.transitionLifecycle(session, CallStateEnding, reason)
}

func (s *Server) setPendingInvite(key inboundInviteKey, req *sip.Request, tx sip.ServerTransaction) {
	s.mu.Lock()
	if s.pendingInvites == nil {
		s.pendingInvites = make(map[inboundInviteKey]*pendingInvite)
	}
	s.pendingInvites[key] = &pendingInvite{req: req, tx: tx}
	s.mu.Unlock()
}

func (s *Server) clearPendingInvite(key inboundInviteKey) {
	s.mu.Lock()
	delete(s.pendingInvites, key)
	s.mu.Unlock()
}

func (s *Server) terminatePendingInvite(key inboundInviteKey, status int) bool {
	s.mu.Lock()
	pending, ok := s.pendingInvites[key]
	if ok {
		if pending != nil && pending.finalResponseStarted {
			ok = false
		} else {
			delete(s.pendingInvites, key)
		}
	}
	s.mu.Unlock()

	if !ok || pending == nil || pending.req == nil || pending.tx == nil {
		return false
	}
	s.sendResponse(pending.tx, pending.req, status)
	return true
}

func (s *Server) beginPendingInviteFinalResponse(key inboundInviteKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelledInvites != nil && s.cancelledInvites[key] {
		return false
	}
	pending, ok := s.pendingInvites[key]
	if !ok || pending == nil {
		return true
	}
	pending.finalResponseStarted = true
	return true
}

func (s *Server) isPendingInviteFinalResponseStarted(key inboundInviteKey) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pending := s.pendingInvites[key]
	return pending != nil && pending.finalResponseStarted
}

func (s *Server) markInviteCancelled(key inboundInviteKey) {
	s.mu.Lock()
	if s.cancelledInvites == nil {
		s.cancelledInvites = make(map[inboundInviteKey]bool)
	}
	s.cancelledInvites[key] = true
	s.mu.Unlock()
}

func (s *Server) isInviteCancelled(key inboundInviteKey) bool {
	s.mu.RLock()
	cancelled := s.cancelledInvites[key]
	s.mu.RUnlock()
	return cancelled
}

func (s *Server) clearInviteCancelled(key inboundInviteKey) {
	s.mu.Lock()
	delete(s.cancelledInvites, key)
	s.mu.Unlock()
}

// notifyError notifies the configured error handler.
func (s *Server) notifyError(session *Session, err error) {
	s.mu.RLock()
	onError := s.onError
	s.mu.RUnlock()

	if onError != nil {
		onError(session, err)
	}
}

// registerSession registers a session and installs disconnect cleanup.
func (s *Server) registerSession(session *Session, callID string) {
	initialState := session.GetState()
	lifecycle := newCallLifecycle(callID, initialState, s.logger)

	session.SetOnDisconnect(func(sess *Session) {
		s.sendBye(sess)
	})
	session.SetOnEnded(func(sess *Session) {
		_ = s.transitionLifecycle(sess, CallStateEnded, "session_end")
		if sess.GetInfo().Direction == CallDirectionOutbound {
			sess.SetOutboundDialogPhase(OutboundDialogPhaseTerminated)
		}
		s.removeSession(callID)
	})
	s.mu.Lock()
	if s.lifecycles == nil {
		s.lifecycles = make(map[string]*CallLifecycle)
	}
	s.sessions[callID] = session
	s.lifecycles[callID] = lifecycle
	s.sessionCount.Add(1)
	s.mu.Unlock()
}
