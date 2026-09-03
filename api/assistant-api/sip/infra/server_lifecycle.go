// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

func (s *Server) GetSession(callID string) (*Session, bool) {
	session, ok := s.inner.GetSession(callID)
	return wrapSession(session), ok
}

func (s *Server) EndCall(session *Session) error {
	return s.inner.EndCall(session.unwrap())
}

func (s *Server) TransitionCall(session *Session, next CallState, reason LifecycleReason) bool {
	return s.inner.TransitionCall(session.unwrap(), next, reason)
}

func (s *Server) EndCallWithReason(session *Session, reason LifecycleReason) error {
	return s.inner.EndCallWithReason(session.unwrap(), reason)
}

func (s *Server) FailCall(session *Session, reason LifecycleReason, err error) error {
	return s.inner.FailCall(session.unwrap(), reason, err)
}

func (s *Server) CancelCall(session *Session, reason LifecycleReason) error {
	return s.inner.CancelCall(session.unwrap(), reason)
}

func (s *Server) ConnectInboundCall(session *Session, reason LifecycleReason) bool {
	return s.inner.ConnectInboundCall(session.unwrap(), reason)
}

func (s *Server) CancelInboundCall(session *Session, reason LifecycleReason) error {
	return s.inner.CancelInboundCall(session.unwrap(), reason)
}

func (s *Server) FailInboundCall(session *Session, reason LifecycleReason, err error) error {
	return s.inner.FailInboundCall(session.unwrap(), reason, err)
}

func (s *Server) EndInboundCall(session *Session, reason LifecycleReason) error {
	return s.inner.EndInboundCall(session.unwrap(), reason)
}
