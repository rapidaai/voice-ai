// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import "github.com/emiago/sipgo/sip"

func (s *Server) handleBye(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	fromHdr := req.From()
	fromUser := ""
	if fromHdr != nil {
		fromUser = fromHdr.Address.User
	}

	s.mu.RLock()
	session, exists := s.sessions[callID]
	s.mu.RUnlock()

	if !exists {
		// Try the outbound dialog cache — maybe this BYE is for a dialog we created
		// but hasn't registered in sessions yet, or was already cleaned up.
		if err := s.dialogClientCache.ReadBye(req, tx); err == nil {
			return
		}
		s.logger.Warnw("BYE received for unknown session", "call_id", callID, "from", fromUser)
		s.sendResponse(tx, req, 481)
		return
	}

	disconnectMetadata := NewSIPReason(req).DisconnectMetadata()
	session.SetDisconnectMetadata(disconnectMetadata)

	if session.GetInfo().Direction == CallDirectionOutbound {
		if err := s.dialogClientCache.ReadBye(req, tx); err != nil {
			// If dialog cache can't handle it (dialog already gone), respond ourselves.
			s.logger.Warnw("Dialog cache ReadBye failed, responding directly",
				"error", err, "call_id", callID)
			s.sendResponse(tx, req, 200)
		}

		s.finishOutboundRemoteBye(session)
		return
	}

	// Inbound BYE must match the server-owned dialog so teardown cannot hide
	// route, tag, or CSeq errors behind a synthetic 200 OK.
	if err := s.dialogServerCache.ReadBye(req, tx); err != nil {
		s.logger.Warnw("Inbound BYE rejected by dialog cache",
			"error", err, "call_id", callID)
		s.sendResponse(tx, req, 481)
		return
	}

	s.finishInboundRemoteBye(session)
}

func (s *Server) finishInboundRemoteBye(session *Session) {
	// Remote BYE owns inbound SIP teardown. Clear local BYE sending before
	// waking listeners so cleanup cannot send BYE back to the peer.
	session.ClearOnDisconnect()
	session.NotifyBye()
	_ = s.EndInboundCall(session, LifecycleReasonRemoteBye)

	s.mu.RLock()
	onBye := s.onBye
	s.mu.RUnlock()

	if onBye != nil {
		if err := onBye(session); err != nil {
			s.logger.Warnw("BYE handler returned error", "error", err, "call_id", session.GetCallID())
		}
	}
}

func (s *Server) finishOutboundRemoteBye(session *Session) {
	// Remote BYE owns outbound SIP teardown. Clear local BYE sending before
	// waking listeners so their cleanup cannot race and send BYE back to the peer.
	session.ClearOnDisconnect()
	session.NotifyBye()
	_ = s.EndCallWithReason(session, LifecycleReasonRemoteBye)

	s.mu.RLock()
	onBye := s.onBye
	s.mu.RUnlock()
	if onBye != nil {
		if err := onBye(session); err != nil {
			s.logger.Warnw("BYE handler returned error", "error", err, "call_id", session.GetCallID())
		}
	}
}
