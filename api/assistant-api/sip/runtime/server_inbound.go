// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"fmt"
	"time"

	"github.com/emiago/sipgo/sip"
)

func (s *Server) handleInvite(req *sip.Request, tx sip.ServerTransaction) {
	NewInbound(s, req, tx).HandleInvite()
}

// handleReInvite processes a re-INVITE for an existing session.
// Re-INVITEs are sent by the remote side for:
//   - Codec renegotiation (all providers)
//   - Hold/resume (Twilio: sendonly/inactive, Asterisk: 0.0.0.0, Vonage: inactive)
//   - Direct media / session refresh (Asterisk, FreeSWITCH)
//   - ICE restart (WebRTC-based providers)
//
// We update the remote RTP address only when the SDP represents active media.
// Hold signals (0.0.0.0, sendonly, inactive) are acknowledged but don't redirect RTP.
func (s *Server) handleReInvite(req *sip.Request, tx sip.ServerTransaction, session *Session) {
	callID := req.CallID().Value()
	info := session.GetInfo()
	s.logger.Infow("Handling re-INVITE for existing session",
		"call_id", callID,
		"direction", info.Direction)

	if !s.validateInDialogRequest(tx, req, session, "re-INVITE") {
		return
	}

	// If no SDP body, this is a session refresh (RFC 4028), so respond with our SDP.
	if len(req.Body()) == 0 {
		s.logger.Debugw("re-INVITE with no SDP body (session refresh)", "call_id", callID)
		s.respondWithCurrentSDP(tx, req, session)
		return
	}

	s.logger.Debugw("re-INVITE SDP body (raw)",
		"call_id", callID,
		"sdp_body", string(req.Body()))

	mediaOffer, failure := newInboundMediaOffer(
		s,
		req,
		"re-INVITE",
		LifecycleReasonInboundReinviteSDPRejected,
		true,
	)
	if failure != nil {
		s.rejectInboundDialogMediaOffer(tx, req, callID, "re-INVITE", *failure)
		return
	}
	sdpInfo := mediaOffer.sdpInfo

	s.logger.Debugw("re-INVITE SDP parsed",
		"call_id", callID,
		"sdp_direction", string(sdpInfo.Direction),
		"sdp_ip", sdpInfo.RTPAddress.IP,
		"sdp_port", sdpInfo.RTPAddress.Port,
		"is_hold", sdpInfo.IsHold())

	// Only update remote RTP when SDP indicates active media (not hold).
	// Hold signals:
	//   - 0.0.0.0 connection IP (RFC 3264 §8.4), used by Asterisk, FreeSWITCH
	//   - sendonly / inactive direction, used by Twilio, Telnyx, Vonage
	// During hold we keep the previous remote RTP address so audio resumes correctly.
	if !sdpInfo.IsHold() {
		rtpHandler := session.GetRTPHandler()
		if rtpHandler != nil && sdpInfo.RTPAddress.IP != "" && sdpInfo.RTPAddress.Port > 0 {
			rtpHandler.SetSymmetricRTP(s.useSymmetricRTPForRemoteIP(sdpInfo.RTPAddress.IP))
			rtpHandler.setRemoteMediaAddress(remoteMediaAddress{
				remoteRTPAddress:  sdpInfo.RTPAddress,
				remoteRTCPAddress: sdpInfo.RTCPAddress,
			})
			session.SetRemoteRTPAddress(sdpInfo.RTPAddress)
			s.logger.Debugw("Updated remote RTP from re-INVITE",
				"call_id", callID,
				"remote_rtp_ip", sdpInfo.RTPAddress.IP,
				"remote_rtp_port", sdpInfo.RTPAddress.Port)
		}

		negotiatedCodec := mediaOffer.negotiatedCodec
		currentCodec := session.GetNegotiatedCodec()
		if currentCodec == nil || currentCodec.PayloadType != negotiatedCodec.PayloadType {
			rtpHandler := session.GetRTPHandler()
			if rtpHandler != nil {
				rtpHandler.SetInboundMediaFormat(negotiatedCodec, sdpInfo.PacketizationDuration())
			}
			session.SetNegotiatedCodec(negotiatedCodec.Name, int(negotiatedCodec.ClockRate))
			s.logger.Infow("Codec updated from re-INVITE",
				"call_id", callID,
				"new_codec", negotiatedCodec.Name,
				"payload_type", negotiatedCodec.PayloadType)
		} else if rtpHandler := session.GetRTPHandler(); rtpHandler != nil {
			rtpHandler.SetInboundMediaFormat(negotiatedCodec, sdpInfo.PacketizationDuration())
		}
	} else {
		s.logger.Infow("re-INVITE indicates hold, keeping current RTP target",
			"call_id", callID,
			"sdp_direction", string(sdpInfo.Direction),
			"sdp_ip", sdpInfo.RTPAddress.IP)
	}

	// Always respond with our SDP (sendrecv) to signal we're ready for media.
	// respondWithCurrentSDP uses the session's negotiated codec, so after any
	// codec switch above, the response will advertise only the correct codec.
	s.respondWithCurrentSDP(tx, req, session)
	s.logger.Infow("re-INVITE handled", "call_id", callID)
}

// respondWithCurrentSDP builds a 200 OK response with the session's current local SDP.
// Used by re-INVITE and UPDATE handlers.
// IMPORTANT: Uses the session's negotiated codec (not all supported codecs) so the
// remote side sees a confirmation of the agreed codec, not a new offer. Advertising
// multiple codecs in a re-INVITE answer confuses Asterisk/FreeSWITCH and can cause
// immediate call teardown ("remote codecs: None" in the peer's logs).
func (s *Server) respondWithCurrentSDP(tx sip.ServerTransaction, req *sip.Request, session *Session) {
	localAddress := session.LocalRTPAddress()
	if localAddress.IP == "" {
		localAddress.IP = s.listenConfig.GetExternalIP()
	}
	codec := session.GetNegotiatedCodec()
	sdpConfig := s.NegotiatedSDPConfig(localAddress, codec)
	sdpBody := s.GenerateSDP(sdpConfig)
	if req.Method == sip.INVITE {
		session.BeginReInviteACKWait()
		if err := s.sendSDPResponseAndWaitACK(tx, req, session, sdpBody, LifecycleReasonInboundReinviteACKReceived, s.effectiveInboundACKTimeout(), nil); err != nil {
			s.logger.Warnw("re-INVITE ACK wait failed",
				"call_id", session.GetCallID(),
				"error", err)
		}
		return
	}
	s.sendSDPResponse(tx, req, sdpBody)
}

func (s *Server) rejectInboundDialogMediaOffer(
	tx sip.ServerTransaction,
	req *sip.Request,
	callID string,
	requestName string,
	failure inboundFailure,
) {
	s.logger.Warnw("Inbound SIP dialog SDP rejected",
		"call_id", callID,
		"request", requestName,
		"status_code", failure.statusCode,
		"failure_class", string(failure.class),
		"failure_response_class", string(failure.responseClass),
		"failure_reason", failure.reason,
		"sli_result", string(failure.termination.Result),
		"sli_reason", failure.termination.Reason,
		"retryable", failure.retryable,
		"reason", failure.lifecycleReason,
		"error", failure.err)
	s.sendResponse(tx, req, failure.statusCode)
}

func (s *Server) handleAck(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()

	s.mu.RLock()
	session, exists := s.sessions[callID]
	s.mu.RUnlock()

	if !exists {
		s.logger.Warnw("ACK received for unknown or closed session",
			"call_id", callID,
			"reason", LifecycleReasonInboundLateACK)
		return
	}

	if session.GetInfo().Direction != CallDirectionInbound {
		s.logger.Debugw("ACK for outbound dialog ignored by inbound ACK handler", "call_id", callID)
		return
	}
	s.acceptInboundACK(req, tx, session)
}

func (s *Server) handleCancel(req *sip.Request, tx sip.ServerTransaction) {
	identity, ok := inboundInviteIdentityFromRequest(req)
	if !ok {
		s.sendResponse(tx, req, 400)
		return
	}
	key := inboundInviteKey{callID: identity.callID, fromTag: identity.fromTag}
	s.markInviteCancelled(key)

	s.mu.RLock()
	session, exists := s.sessions[identity.callID]
	s.mu.RUnlock()

	inviteTerminated := s.terminatePendingInvite(key, 487)

	if !exists && !inviteTerminated {
		s.logger.Warnw("CANCEL received for unknown session", "call_id", identity.callID)
		s.sendResponse(tx, req, 481) // Call/Transaction Does Not Exist
		return
	}
	if exists && !inviteTerminated {
		state := session.GetState()
		setupPhase := session.GetInboundSetupPhase()
		if (state != CallStateInitializing && state != CallStateRinging) || inboundFinalAnswerStarted(setupPhase) || s.isPendingInviteFinalResponseStarted(key) {
			s.logger.Warnw("CANCEL received for non-pending dialog", "call_id", identity.callID, "state", state, "inbound_setup_phase", setupPhase)
			s.sendResponse(tx, req, 481)
			return
		}
	}

	// Get callback before calling it
	s.mu.RLock()
	onCancel := s.onCancel
	s.mu.RUnlock()

	if exists && onCancel != nil {
		if err := onCancel(session); err != nil {
			s.logger.Warnw("CANCEL handler returned error", "error", err, "call_id", identity.callID)
		}
	}

	// CANCEL is for an unanswered INVITE, clear onDisconnect so End()
	// does not attempt to send BYE (no dialog established yet).
	if exists {
		_ = s.CancelInboundCall(session, LifecycleReasonCancelReceived)
	}
	s.sendResponse(tx, req, 200) // OK
	s.logger.Infow("SIP call cancelled", "call_id", identity.callID)
}

func inboundFinalAnswerStarted(phase InboundSetupPhase) bool {
	switch phase {
	case InboundSetupPhaseAnswered, InboundSetupPhaseACKConfirmed:
		return true
	default:
		return false
	}
}

func (s *Server) handleRegister(req *sip.Request, tx sip.ServerTransaction) {
	s.logger.Debugw("REGISTER received")
	s.sendResponse(tx, req, 200) // OK
}

func (s *Server) handleOptions(req *sip.Request, tx sip.ServerTransaction) {
	s.logger.Debugw("OPTIONS received")
	s.sendResponse(tx, req, 200) // OK
}

// handleUpdate processes SIP UPDATE requests (RFC 3311).
// Used by various providers for:
//   - Asterisk/FreeSWITCH: direct_media negotiation, session timers, codec changes
//   - Twilio/Telnyx: early media SDP updates, session parameter changes
//   - Vonage: codec renegotiation during call setup
//
// For in-dialog UPDATEs with SDP: update remote RTP when media remains active,
// then respond with our SDP. Unknown or invalid dialogs are rejected.
func (s *Server) handleUpdate(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	fromUser := ""
	if fromHdr := req.From(); fromHdr != nil {
		fromUser = fromHdr.Address.User
	}

	s.logger.Infow("UPDATE received",
		"call_id", callID,
		"from", fromUser)

	s.mu.RLock()
	session, exists := s.sessions[callID]
	s.mu.RUnlock()

	if !exists || session == nil {
		s.logger.Warnw("UPDATE received for unknown session", "call_id", callID)
		s.sendResponse(tx, req, 481)
		return
	}
	if !s.validateInDialogRequest(tx, req, session, "UPDATE") {
		return
	}

	if body := req.Body(); len(body) > 0 {
		mediaOffer, failure := newInboundMediaOffer(
			s,
			req,
			"UPDATE",
			LifecycleReasonInboundUpdateSDPRejected,
			true,
		)
		if failure != nil {
			s.rejectInboundDialogMediaOffer(tx, req, callID, "UPDATE", *failure)
			return
		}
		sdpInfo := mediaOffer.sdpInfo

		s.logger.Debugw("UPDATE SDP parsed",
			"call_id", callID,
			"sdp_direction", string(sdpInfo.Direction),
			"sdp_ip", sdpInfo.RTPAddress.IP,
			"sdp_port", sdpInfo.RTPAddress.Port,
			"is_hold", sdpInfo.IsHold())

		// Only update remote RTP for active media (not hold)
		if !sdpInfo.IsHold() {
			rtpHandler := session.GetRTPHandler()
			if rtpHandler != nil && sdpInfo.RTPAddress.IP != "" && sdpInfo.RTPAddress.Port > 0 {
				rtpHandler.SetSymmetricRTP(s.useSymmetricRTPForRemoteIP(sdpInfo.RTPAddress.IP))
				rtpHandler.setRemoteMediaAddress(remoteMediaAddress{
					remoteRTPAddress:  sdpInfo.RTPAddress,
					remoteRTCPAddress: sdpInfo.RTCPAddress,
				})
				session.SetRemoteRTPAddress(sdpInfo.RTPAddress)
				s.logger.Debugw("Updated remote RTP from UPDATE",
					"call_id", callID,
					"remote_rtp_ip", sdpInfo.RTPAddress.IP,
					"remote_rtp_port", sdpInfo.RTPAddress.Port)
			}

			negotiatedCodec := mediaOffer.negotiatedCodec
			currentCodec := session.GetNegotiatedCodec()
			if currentCodec == nil || currentCodec.PayloadType != negotiatedCodec.PayloadType {
				rtpHandler := session.GetRTPHandler()
				if rtpHandler != nil {
					rtpHandler.SetInboundMediaFormat(negotiatedCodec, sdpInfo.PacketizationDuration())
				}
				session.SetNegotiatedCodec(negotiatedCodec.Name, int(negotiatedCodec.ClockRate))
				s.logger.Infow("Codec updated from UPDATE",
					"call_id", callID,
					"new_codec", negotiatedCodec.Name,
					"payload_type", negotiatedCodec.PayloadType)
			} else if rtpHandler := session.GetRTPHandler(); rtpHandler != nil {
				rtpHandler.SetInboundMediaFormat(negotiatedCodec, sdpInfo.PacketizationDuration())
			}
		} else {
			s.logger.Infow("UPDATE indicates hold, keeping current RTP target",
				"call_id", callID,
				"sdp_direction", string(sdpInfo.Direction),
				"sdp_ip", sdpInfo.RTPAddress.IP)
		}

		s.respondWithCurrentSDP(tx, req, session)
	} else {
		s.sendResponse(tx, req, 200)
	}

	s.logger.Debugw("UPDATE handled", "call_id", callID)
}

func (s *Server) validateInDialogRequest(tx sip.ServerTransaction, req *sip.Request, session *Session, requestName string) bool {
	callID := req.CallID().Value()
	info := session.GetInfo()
	switch info.Direction {
	case CallDirectionInbound:
		dialogSession := session.GetDialogServerSession()
		if dialogSession == nil {
			s.logger.Warnw("Inbound SIP dialog request missing dialog ownership",
				"call_id", callID,
				"request", requestName)
			s.sendResponse(tx, req, 481)
			return false
		}
		matchedDialog, err := s.dialogServerCache.MatchDialogRequest(req)
		if err != nil || matchedDialog != dialogSession {
			s.logger.Warnw("Inbound SIP dialog request rejected",
				"call_id", callID,
				"request", requestName,
				"error", err)
			s.sendResponse(tx, req, 481)
			return false
		}
		if err := dialogSession.ReadRequest(req, tx); err != nil {
			s.logger.Warnw("Inbound SIP dialog CSeq validation failed",
				"call_id", callID,
				"request", requestName,
				"error", err)
			s.sendResponse(tx, req, 400)
			return false
		}
		return true
	case CallDirectionOutbound:
		dialogSession := session.GetDialogClientSession()
		if dialogSession == nil {
			s.logger.Warnw("Outbound SIP dialog request missing dialog ownership",
				"call_id", callID,
				"request", requestName)
			s.sendResponse(tx, req, 481)
			return false
		}
		matchedDialog, err := s.dialogClientCache.MatchRequestDialog(req)
		if err != nil || matchedDialog != dialogSession {
			s.logger.Warnw("Outbound SIP dialog request rejected",
				"call_id", callID,
				"request", requestName,
				"error", err)
			s.sendResponse(tx, req, 481)
			return false
		}
		if err := dialogSession.ReadRequest(req, tx); err != nil {
			s.logger.Warnw("Outbound SIP dialog CSeq validation failed",
				"call_id", callID,
				"request", requestName,
				"error", err)
			s.sendResponse(tx, req, 400)
			return false
		}
		return true
	default:
		s.logger.Warnw("SIP dialog request rejected for unknown direction",
			"call_id", callID,
			"request", requestName,
			"direction", info.Direction)
		s.sendResponse(tx, req, 481)
		return false
	}
}

// handleInfo processes SIP INFO requests (RFC 6086).
// Used by providers for:
//   - Asterisk/FreeSWITCH: DTMF relay (application/dtmf-relay), call recording
//   - Twilio: session metadata, custom headers
//   - Generic: application/ooh323 info, broadsoft call center events
func (s *Server) handleInfo(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	contentType := ""
	if ct := req.GetHeader("Content-Type"); ct != nil {
		contentType = ct.Value()
	}
	s.logger.Debugw("INFO received",
		"call_id", callID,
		"content_type", contentType)
	s.sendResponse(tx, req, 200)
}

// handleNotify processes SIP NOTIFY requests (RFC 6665).
// Used by providers for:
//   - Twilio/Telnyx: REFER progress (sipfrag), subscription state updates
//   - Asterisk: MWI (message-summary), dialog-info, presence
//   - Vonage: session progress events
func (s *Server) handleNotify(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	eventHdr := ""
	if ev := req.GetHeader("Event"); ev != nil {
		eventHdr = ev.Value()
	}
	s.logger.Debugw("NOTIFY received",
		"call_id", callID,
		"event", eventHdr)
	s.sendResponse(tx, req, 200)
}

// handleRefer processes SIP REFER requests (RFC 3515).
// Inbound REFER (provider-initiated transfer) is declined. The platform supports
// transfer via B2BUA bridge (INVITE-based), triggered by the LLM tool, not REFER.
func (s *Server) handleRefer(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	referTo := ""
	if rt := req.GetHeader("Refer-To"); rt != nil {
		referTo = rt.Value()
	}
	s.logger.Warnw("REFER received (call transfer not supported)",
		"call_id", callID,
		"refer_to", referTo)
	s.sendResponse(tx, req, 603) // Decline
}

// handleSubscribe processes SIP SUBSCRIBE requests (RFC 6665).
// Twilio and some SIP trunks send SUBSCRIBE for dialog-info, presence, or MWI.
// We don't support event subscriptions, so respond with 489 Bad Event to
// signal this cleanly. Using 489 instead of 405/603 prevents Twilio from
// retrying the subscription endlessly.
func (s *Server) handleSubscribe(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	eventHdr := ""
	if ev := req.GetHeader("Event"); ev != nil {
		eventHdr = ev.Value()
	}
	s.logger.Debugw("SUBSCRIBE received (event subscriptions not supported)",
		"call_id", callID,
		"event", eventHdr)
	resp := sip.NewResponseFromRequest(req, 489, "Bad Event", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Errorw("Failed to send 489 for SUBSCRIBE", "error", err, "call_id", callID)
	}
}

// handleMessage processes SIP MESSAGE requests (RFC 3428).
// Used by FreeSWITCH for text events and by some SIP providers for out-of-band data.
func (s *Server) handleMessage(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	s.logger.Debugw("MESSAGE received", "call_id", callID)
	s.sendResponse(tx, req, 200)
}

// handleUnknownRequest handles provider-specific SIP methods that do not have
// dedicated handlers while preserving active dialogs.
func (s *Server) handleUnknownRequest(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	method := string(req.Method)
	fromUser := ""
	if fromHdr := req.From(); fromHdr != nil {
		fromUser = fromHdr.Address.User
	}

	s.mu.RLock()
	_, inDialog := s.sessions[callID]
	s.mu.RUnlock()

	if inDialog {
		// In-dialog: accept unknown methods to keep the dialog alive.
		// Rejecting with 405 causes Asterisk/FreeSWITCH/Twilio to tear down the call.
		s.logger.Warnw("Unhandled SIP method for active session, accepting to keep dialog alive",
			"method", method,
			"call_id", callID,
			"from", fromUser)
		s.sendResponse(tx, req, 200)
	} else {
		// Out-of-dialog: use RFC-appropriate rejection codes.
		// SUBSCRIBE without a matching event package → 489 Bad Event
		// to prevent subscription loops (Twilio retries on 405).
		if req.Method == sip.SUBSCRIBE {
			s.logger.Debugw("Out-of-dialog SUBSCRIBE rejected",
				"call_id", callID,
				"from", fromUser)
			resp := sip.NewResponseFromRequest(req, 489, "Bad Event", nil)
			if err := tx.Respond(resp); err != nil {
				s.logger.Errorw("Failed to send 489 response", "error", err)
			}
		} else {
			s.logger.Warnw("Unknown SIP method received (no session), rejecting",
				"method", method,
				"call_id", callID,
				"from", fromUser)
			s.sendResponse(tx, req, 405) // Method Not Allowed
		}
	}
}

func (s *Server) sendResponse(tx sip.ServerTransaction, req *sip.Request, statusCode int) *sip.Response {
	return s.sendResponseWithHeaders(tx, req, statusCode)
}

func (s *Server) sendResponseWithHeaders(
	tx sip.ServerTransaction,
	req *sip.Request,
	statusCode int,
	headers ...sip.Header,
) *sip.Response {
	resp := sip.NewResponseFromRequest(req, statusCode, "", nil)
	if req != nil && req.Method == sip.INVITE && statusCode >= 200 && resp.Contact() == nil && s.listenConfig != nil {
		contactHeader := s.listenConfig.SIPContactHeader()
		resp.AppendHeader(&contactHeader)
	}
	for _, header := range headers {
		if header != nil {
			resp.AppendHeader(header)
		}
	}
	if err := tx.Respond(resp); err != nil {
		callID := ""
		if req != nil && req.CallID() != nil {
			callID = req.CallID().Value()
		}
		s.logger.Errorw("Failed to send SIP response",
			"error", err,
			"status", statusCode,
			"call_id", callID)
	}
	return resp
}

// sendSDPResponse sends a SIP 200 OK response with the given SDP body.
// Adds a Contact header (required by RFC 3261 §13.3.1.1 for INVITE/re-INVITE responses)
// so that Asterisk, Twilio, and other providers know where to send subsequent requests.
func (s *Server) sendSDPResponse(tx sip.ServerTransaction, req *sip.Request, sdpBody string) {
	s.logger.Debugw("Sending SIP response with SDP",
		"call_id", req.CallID().Value(),
		"method", req.Method,
		"sdp_body", sdpBody)
	resp := s.newSDPResponse(req, sdpBody)

	if err := tx.Respond(resp); err != nil {
		s.logger.Errorw("Failed to send SIP response with SDP",
			"error", err,
			"call_id", req.CallID().Value())
	}
}

func (s *Server) sendSDPResponseAndWaitACK(
	tx sip.ServerTransaction,
	req *sip.Request,
	session *Session,
	sdpBody string,
	ackReason LifecycleReason,
	ackTimeout time.Duration,
	onResponseSent func(time.Time),
) error {
	ackAccepted := false
	defer func() {
		if !ackAccepted && ackReason == LifecycleReasonInboundReinviteACKReceived {
			session.ClearReInviteACKWait()
		}
	}()

	s.logger.Debugw("Sending SIP response with SDP and waiting for ACK",
		"call_id", req.CallID().Value(),
		"method", req.Method,
		"ack_timeout_ms", ackTimeout.Milliseconds(),
		"sdp_body", sdpBody)

	responseRequest := req
	dialogSession := session.GetDialogServerSession()
	if ackReason == LifecycleReasonInboundInviteACKReceived {
		if dialogSession == nil || dialogSession.InviteRequest == nil {
			return fmt.Errorf("initial inbound INVITE answer requires dialog ownership")
		}
		responseRequest = dialogSession.InviteRequest
	}
	resp := s.newSDPResponse(responseRequest, sdpBody)
	if dialogSession != nil {
		dialogSession.InviteResponse = resp
	}
	if err := tx.Respond(resp); err != nil {
		return err
	}
	if onResponseSent != nil {
		onResponseSent(time.Now())
	}

	var initialACKReceived <-chan struct{}
	if ackReason == LifecycleReasonInboundInviteACKReceived {
		initialACKReceived = session.InitialACKSignal()
	}
	if ackTimeout <= 0 {
		ackTimeout = s.effectiveInboundACKTimeout()
	}
	ackTimer := time.NewTimer(ackTimeout)
	defer ackTimer.Stop()
	retryFinalResponse := ackReason == LifecycleReasonInboundInviteACKReceived && s.shouldRetryInboundFinalResponse()
	retryInterval := s.effectiveInboundFinalResponseRetryInitial()
	var retryTimer *time.Timer
	var retryC <-chan time.Time
	if retryFinalResponse {
		retryTimer = time.NewTimer(retryInterval)
		retryC = retryTimer.C
		defer retryTimer.Stop()
	}

	for {
		select {
		case ackRequest := <-tx.Acks():
			if ackRequest == nil {
				return fmt.Errorf("inbound ACK request is nil")
			}
			s.acceptInboundACKWithReason(ackRequest, tx, session, ackReason)
			ackAccepted = true
			return nil
		case <-initialACKReceived:
			ackAccepted = true
			return nil
		case <-tx.Done():
			if inboundACKAlreadyAccepted(session, ackReason) {
				ackAccepted = true
				return nil
			}
			if err := tx.Err(); err != nil {
				return err
			}
			return ErrInboundACKTimeout
		case <-ackTimer.C:
			if inboundACKAlreadyAccepted(session, ackReason) {
				ackAccepted = true
				return nil
			}
			return ErrInboundACKTimeout
		case <-retryC:
			if err := tx.Respond(resp); err != nil {
				return err
			}
			retryInterval *= 2
			if maxInterval := s.effectiveInboundFinalResponseRetryMax(); retryInterval > maxInterval {
				retryInterval = maxInterval
			}
			retryTimer.Reset(retryInterval)
		}
	}
}

func (s *Server) shouldRetryInboundFinalResponse() bool {
	if s.listenConfig == nil || s.listenConfig.Transport == "" {
		return true
	}
	return s.listenConfig.Transport == TransportUDP
}

func inboundACKAlreadyAccepted(session *Session, reason LifecycleReason) bool {
	switch reason {
	case LifecycleReasonInboundInviteACKReceived:
		return session.HasInitialACKReceived()
	case LifecycleReasonInboundReinviteACKReceived:
		return !session.HasReInviteACKPending()
	default:
		return false
	}
}

func (s *Server) newSDPResponse(req *sip.Request, sdpBody string) *sip.Response {
	resp := sip.NewSDPResponseFromRequest(req, []byte(sdpBody))
	if resp.Contact() == nil {
		contactHeader := s.listenConfig.SIPContactHeader()
		resp.AppendHeader(&contactHeader)
	}
	return resp
}

func (s *Server) effectiveInboundACKTimeout() time.Duration {
	if s.inboundACKTimeout > 0 {
		return s.inboundACKTimeout
	}
	return defaultInboundACKTimeout
}

func (s *Server) effectiveInboundFinalResponseRetryInitial() time.Duration {
	if s.inboundFinalResponseRetryInitial > 0 {
		return s.inboundFinalResponseRetryInitial
	}
	return defaultInboundFinalResponseRetryInitial
}

func (s *Server) effectiveInboundFinalResponseRetryMax() time.Duration {
	if s.inboundFinalResponseRetryMax > 0 {
		return s.inboundFinalResponseRetryMax
	}
	return defaultInboundFinalResponseRetryMax
}

func (s *Server) effectiveInboundRingingInterval() time.Duration {
	if s.inboundRingingInterval > 0 {
		return s.inboundRingingInterval
	}
	return defaultInboundRingingInterval
}

func (s *Server) acceptInboundACK(req *sip.Request, tx sip.ServerTransaction, session *Session) {
	if !session.HasInitialACKReceived() {
		s.acceptInboundACKWithReason(req, tx, session, LifecycleReasonInboundInviteACKReceived)
		return
	}
	if session.HasReInviteACKPending() {
		s.acceptInboundACKWithReason(req, tx, session, LifecycleReasonInboundReinviteACKReceived)
		return
	}
	s.logger.Debugw("Late or duplicate inbound ACK ignored",
		"call_id", session.GetCallID(),
		"reason", LifecycleReasonInboundLateACK)
}

func (s *Server) acceptInboundACKWithReason(req *sip.Request, tx sip.ServerTransaction, session *Session, reason LifecycleReason) {
	callID := session.GetCallID()
	if dialogSession := session.GetDialogServerSession(); dialogSession != nil {
		if err := dialogSession.ReadAck(req, tx); err != nil {
			s.logger.Warnw("Dialog ReadAck failed",
				"error", err,
				"call_id", callID,
				"reason", reason)
		}
	}
	switch reason {
	case LifecycleReasonInboundInviteACKReceived:
		session.MarkInitialACKReceived()
		session.SetInboundSetupPhase(InboundSetupPhaseACKConfirmed)
		session.MarkInboundSetupTimestamp(InboundSetupPhaseACKConfirmed, time.Now())
		s.ConnectInboundCall(session, reason)
		s.logger.Infow("Inbound SIP latency metrics",
			"call_id", callID,
			"metrics", session.GetInboundLatencyMetrics())
		s.logger.Infow("Initial inbound INVITE ACK received",
			"call_id", callID,
			"reason", reason)
	case LifecycleReasonInboundReinviteACKReceived:
		session.CompleteReInviteACKWait()
		s.logger.Infow("Inbound re-INVITE ACK received",
			"call_id", callID,
			"reason", reason,
			"reinvite_ack_count", session.ReInviteACKCount())
	}
}
