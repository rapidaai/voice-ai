// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"fmt"
	"time"

	"github.com/emiago/sipgo/sip"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

func (s *Server) handleInvite(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	fromURI := req.From().Address.String()
	toURI := req.To().Address.String()

	s.logger.Infow("Received INVITE", "call_id", callID, "from", fromURI, "to", toURI)
	s.setPendingInvite(callID, req, tx)
	defer s.clearPendingInvite(callID)
	defer s.clearInviteCancelled(callID)

	// Check if this is a re-INVITE for an existing session (e.g., codec renegotiation
	// or hold/resume from remote side, common after outbound calls are answered)
	s.mu.RLock()
	existingSession, isReInvite := s.sessions[callID]
	s.mu.RUnlock()

	if isReInvite && existingSession != nil {
		info := existingSession.GetInfo()
		s.logger.Infow("Routing as re-INVITE for existing session",
			"call_id", callID,
			"direction", info.Direction,
			"state", info.State)
		s.handleReInvite(req, tx, existingSession)
		return
	}

	// Parse SDP from incoming INVITE to get remote RTP address and codec preferences
	sdpInfo, err := s.ParseSDP(req.Body())
	if err != nil {
		s.logger.Warnw("Failed to parse SDP, using defaults", "error", err, "call_id", callID)
		sdpInfo = &SDPMediaInfo{PreferredCodec: &CodecPCMU}
	}

	// Resolve tenant-specific config via middleware chain.
	// The chain: routingMiddleware → assistantMiddleware → vaultConfigResolver
	// Each middleware enriches the SIPRequestContext; the final handler returns the InviteResult.
	s.mu.RLock()
	resolver := s.configResolver
	s.mu.RUnlock()

	var tenantConfig *Config
	var resolvedExtra map[string]interface{}

	if resolver != nil {
		reqCtx := &SIPRequestContext{
			Method:  "INVITE",
			CallID:  callID,
			FromURI: fromURI,
			ToURI:   toURI,
			SDPInfo: sdpInfo,
		}
		result, err := resolver(reqCtx)
		if err != nil {
			s.logger.Errorw("SIP authentication/config resolution failed", "error", err, "call_id", callID)
			s.sendResponse(tx, req, 500)
			return
		}
		if !result.ShouldAllow {
			s.logger.Warnw("Call rejected by authentication chain",
				"call_id", callID,
				"code", result.RejectCode,
				"reason", result.RejectMsg)
			s.sendResponse(tx, req, result.RejectCode)
			return
		}
		tenantConfig = result.Config
		resolvedExtra = result.Extra

		s.logger.Debugw("SIP INVITE authenticated",
			"call_id", callID,
			"assistant_id", reqCtx.AssistantID,
			"has_api_key", reqCtx.APIKey != "")
	}

	// Reject if no config was resolved — all config must be explicitly provided
	if tenantConfig == nil {
		s.logger.Errorw("No SIP config resolved for call, rejecting", "call_id", callID)
		s.sendResponse(tx, req, 500)
		return
	}

	// For inbound calls, ensure the server address is set from listen config
	// so RTP handler binds to the correct local IP
	if tenantConfig.Server == "" || tenantConfig.Server == "0.0.0.0" {
		tenantConfig.Server = s.listenConfig.GetExternalIP()
	}

	// Negotiate codec
	negotiatedCodec := sdpInfo.PreferredCodec
	if negotiatedCodec == nil {
		negotiatedCodec = &CodecPCMU
	}

	// Extract vault credential from resolved extra for direct session access
	var vaultCredential *protos.VaultCredential
	if vaultCredVal, ok := resolvedExtra["vault_credential"]; ok {
		if vaultCred, ok := vaultCredVal.(*protos.VaultCredential); ok {
			vaultCredential = vaultCred
		}
	}

	// Create session with resolved tenant config and middleware state
	auth, _ := resolvedExtra["auth"].(types.SimplePrinciple)
	assistant, _ := resolvedExtra["assistant"].(*internal_assistant_entity.Assistant)
	session, err := NewSession(s.ctx, &SessionConfig{
		Config:          tenantConfig,
		Direction:       CallDirectionInbound,
		CallID:          callID,
		Codec:           negotiatedCodec,
		Logger:          s.logger,
		Auth:            auth,
		Assistant:       assistant,
		VaultCredential: vaultCredential,
	})
	if err != nil {
		s.logger.Errorw("Failed to create session", "error", err, "call_id", callID)
		s.sendResponse(tx, req, 500)
		return
	}

	// Also propagate all middleware-resolved state to metadata for backward compatibility
	// so the onInvite handler can access it via session.GetMetadata() if needed.
	for k, v := range resolvedExtra {
		session.SetMetadata(k, v)
	}
	session.SetMetadata(MetadataCallFromURI, fromURI)
	session.SetMetadata(MetadataCallToURI, toURI)

	// Register session and lifecycle hooks.
	s.registerSession(session, callID)
	s.mu.RLock()
	onCreated := s.onCreated
	s.mu.RUnlock()
	if onCreated != nil {
		onCreated(session, fromURI, toURI)
	}

	if s.isInviteCancelled(callID) {
		s.terminatePendingInvite(callID, 487)
		session.ClearOnDisconnect()
		s.beginEnding(session, "invite_cancelled")
		session.End()
		return
	}

	// Create an inbound dialog session via the server dialog cache.
	// This tracks dialog state (To-tag, CSeq, Route) so we can later send
	// BYE to properly disconnect the call when the assistant ends the conversation.
	dialogSession, err := s.dialogServerCache.ReadInvite(req, tx)
	if err != nil {
		s.logger.Warnw("Failed to create inbound dialog session — BYE on disconnect will not work",
			"error", err, "call_id", callID)
		// Fall back to non-dialog response flow
		s.sendResponse(tx, req, 100)
		s.sendResponse(tx, req, 180)
	} else {
		session.SetDialogServerSession(dialogSession)
		// Send provisionals via dialog session (non-blocking, ensures consistent To-tag)
		if err := dialogSession.Respond(100, "Trying", nil); err != nil {
			s.logger.Warnw("Failed to send 100 via dialog", "error", err, "call_id", callID)
		}
		if err := dialogSession.Respond(180, "Ringing", nil); err != nil {
			s.logger.Warnw("Failed to send 180 via dialog", "error", err, "call_id", callID)
		}
	}
	s.setCallState(session, CallStateRinging, "inbound_invite_ringing")
	s.mu.RLock()
	onRinging := s.onRinging
	s.mu.RUnlock()
	if onRinging != nil {
		onRinging(session, fromURI, toURI)
	}

	s.logger.Debugw("Parsed remote SDP",
		"call_id", callID,
		"remote_rtp_ip", sdpInfo.ConnectionIP,
		"remote_rtp_port", sdpInfo.AudioPort,
		"codec", negotiatedCodec.Name)

	// Allocate an RTP port from the shared pool
	rtpPort, err := s.rtpAllocator.Allocate()
	if err != nil {
		s.logger.Errorw("No RTP ports available", "error", err, "call_id", callID)
		s.notifyError(session, err)
		s.removeSession(callID)
		s.sendResponse(tx, req, 503) // Service Unavailable
		return
	}

	// Create RTP handler with allocated port — bind to the local/bind address
	// (0.0.0.0), not the external IP. The external IP is only for SDP
	// advertisement. Binding to an external IP that isn't on a local interface
	// causes net.ListenUDP to fail.
	rtpBindIP := s.listenConfig.GetBindAddress()
	rtpHandler, err := NewRTPHandler(s.ctx, &RTPConfig{
		LocalIP:     rtpBindIP,
		LocalPort:   rtpPort,
		PayloadType: negotiatedCodec.PayloadType,
		ClockRate:   negotiatedCodec.ClockRate,
		Logger:      s.logger,
	})
	if err != nil {
		s.rtpAllocator.Release(rtpPort)
		s.logger.Errorw("Failed to create RTP handler", "error", err, "call_id", callID)
		s.notifyError(session, err)
		s.removeSession(callID)
		s.sendResponse(tx, req, 500)
		return
	}

	// Set remote RTP address from incoming SDP
	if sdpInfo.ConnectionIP != "" && sdpInfo.AudioPort > 0 {
		rtpHandler.SetRemoteAddr(sdpInfo.ConnectionIP, sdpInfo.AudioPort)
		session.SetRemoteRTP(sdpInfo.ConnectionIP, sdpInfo.AudioPort)
	}

	// Get local RTP address — use external IP for SDP so remote peer sends RTP to reachable address
	_, localPort := rtpHandler.LocalAddr()
	externalIP := s.listenConfig.GetExternalIP()
	session.SetLocalRTP(externalIP, localPort)
	session.SetNegotiatedCodec(negotiatedCodec.Name, int(negotiatedCodec.ClockRate))

	// Store the RTP handler in the session
	session.SetRTPHandler(rtpHandler)

	// Start RTP processing
	rtpHandler.Start()
	// NOTE: do not emit call.media.started here. Local RTP startup does not
	// guarantee that real media packets have started flowing yet.

	// Generate SDP for response — advertise the negotiated codec only.
	// Using NegotiatedSDPConfig ensures we confirm the codec we agreed upon,
	// rather than re-offering all codecs which can confuse some PBXes.
	sdpConfig := s.NegotiatedSDPConfig(externalIP, localPort, negotiatedCodec)
	sdpBody := s.GenerateSDP(sdpConfig)

	if s.isInviteCancelled(callID) {
		s.terminatePendingInvite(callID, 487)
		session.ClearOnDisconnect()
		s.beginEnding(session, "invite_cancelled_before_200ok")
		session.End()
		return
	}

	// Send 200 OK with SDP.
	// When a dialog session exists, use RespondSDP which blocks until ACK is
	// received (or timeout). This establishes the dialog in Confirmed state,
	// enabling us to send BYE later. Falls back to manual response if no dialog.
	if ds := session.GetDialogServerSession(); ds != nil {
		if err := ds.RespondSDP([]byte(sdpBody)); err != nil {
			s.logger.Warnw("Dialog RespondSDP failed — falling back to manual response",
				"error", err, "call_id", callID)
			s.sendSDPResponse(tx, req, sdpBody)
		}
	} else {
		s.sendSDPResponse(tx, req, sdpBody)
	}
	s.clearPendingInvite(callID)
	s.setCallState(session, CallStateConnected, "inbound_invite_answered")
	s.mu.RLock()
	onAnswered := s.onAnswered
	s.mu.RUnlock()
	if onAnswered != nil {
		onAnswered(session, fromURI, toURI)
	}

	s.logger.Infow("SIP call answered",
		"call_id", callID,
		"local_rtp", fmt.Sprintf("%s:%d", externalIP, localPort),
		"remote_rtp", fmt.Sprintf("%s:%d", sdpInfo.ConnectionIP, sdpInfo.AudioPort),
		"codec", negotiatedCodec.Name)

	// Call the invite handler (which will start the conversation)
	s.mu.RLock()
	onInvite := s.onInvite
	s.mu.RUnlock()

	if onInvite != nil {
		if err := onInvite(session, fromURI, toURI); err != nil {
			s.logger.Errorw("INVITE handler failed", "error", err, "call_id", callID)
			s.notifyError(session, err)
		}
	}
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

	// For outbound calls, validate the re-INVITE through the dialog cache.
	// This updates the remoteCSeqNo in the dialog so subsequent requests are accepted.
	if info.Direction == CallDirectionOutbound {
		if dialogSession := session.GetDialogClientSession(); dialogSession != nil {
			if err := dialogSession.ReadRequest(req, tx); err != nil {
				s.logger.Warnw("re-INVITE CSeq validation failed through dialog",
					"error", err, "call_id", callID)
				s.sendResponse(tx, req, 400) // Bad Request
				return
			}
		}
	}

	// If no SDP body, this is a session refresh (RFC 4028) — just respond with our SDP
	if len(req.Body()) == 0 {
		s.logger.Debugw("re-INVITE with no SDP body (session refresh)", "call_id", callID)
		s.respondWithCurrentSDP(tx, req, session)
		return
	}

	s.logger.Debugw("re-INVITE SDP body (raw)",
		"call_id", callID,
		"sdp_body", string(req.Body()))

	// Parse updated SDP from re-INVITE
	sdpInfo, err := s.ParseSDP(req.Body())
	if err != nil {
		s.logger.Warnw("Failed to parse re-INVITE SDP", "error", err, "call_id", callID)
		s.sendResponse(tx, req, 488) // Not Acceptable Here
		return
	}

	s.logger.Debugw("re-INVITE SDP parsed",
		"call_id", callID,
		"sdp_direction", string(sdpInfo.Direction),
		"sdp_ip", sdpInfo.ConnectionIP,
		"sdp_port", sdpInfo.AudioPort,
		"is_hold", sdpInfo.IsHold())

	// Only update remote RTP when SDP indicates active media (not hold).
	// Hold signals:
	//   - 0.0.0.0 connection IP (RFC 3264 §8.4) — used by Asterisk, FreeSWITCH
	//   - sendonly / inactive direction — used by Twilio, Telnyx, Vonage
	// During hold we keep the previous remote RTP address so audio resumes correctly.
	if !sdpInfo.IsHold() {
		rtpHandler := session.GetRTPHandler()
		if rtpHandler != nil && sdpInfo.ConnectionIP != "" && sdpInfo.AudioPort > 0 {
			rtpHandler.SetRemoteAddr(sdpInfo.ConnectionIP, sdpInfo.AudioPort)
			session.SetRemoteRTP(sdpInfo.ConnectionIP, sdpInfo.AudioPort)
			s.logger.Debugw("Updated remote RTP from re-INVITE",
				"call_id", callID,
				"remote_rtp_ip", sdpInfo.ConnectionIP,
				"remote_rtp_port", sdpInfo.AudioPort)
		}

		// Update codec if the re-INVITE proposes a different one.
		// Asterisk commonly sends re-INVITE after bridging to switch codecs
		// (e.g., direct_media or codec transcoding changes). If we ignore this
		// and keep sending the old payload type, Asterisk sees a PT mismatch
		// and tears down the call immediately.
		if sdpInfo.PreferredCodec != nil {
			currentCodec := session.GetNegotiatedCodec()
			if currentCodec == nil || currentCodec.PayloadType != sdpInfo.PreferredCodec.PayloadType {
				rtpHandler := session.GetRTPHandler()
				if rtpHandler != nil {
					rtpHandler.SetCodec(sdpInfo.PreferredCodec)
				}
				session.SetNegotiatedCodec(sdpInfo.PreferredCodec.Name, int(sdpInfo.PreferredCodec.ClockRate))
				s.logger.Infow("Codec updated from re-INVITE",
					"call_id", callID,
					"new_codec", sdpInfo.PreferredCodec.Name,
					"payload_type", sdpInfo.PreferredCodec.PayloadType)
			}
		}
	} else {
		s.logger.Infow("re-INVITE indicates hold — keeping current RTP target",
			"call_id", callID,
			"sdp_direction", string(sdpInfo.Direction),
			"sdp_ip", sdpInfo.ConnectionIP)
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
	localIP, localPort := session.GetLocalRTP()
	if localIP == "" {
		localIP = s.listenConfig.GetExternalIP()
	}
	codec := session.GetNegotiatedCodec()
	sdpConfig := s.NegotiatedSDPConfig(localIP, localPort, codec)
	sdpBody := s.GenerateSDP(sdpConfig)
	s.sendSDPResponse(tx, req, sdpBody)
}

func (s *Server) handleAck(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()

	s.mu.RLock()
	session, exists := s.sessions[callID]
	s.mu.RUnlock()

	if !exists {
		s.logger.Warnw("ACK received for unknown session", "call_id", callID)
		return
	}

	// For inbound calls with a dialog session, ReadAck confirms the dialog.
	// NOTE: When RespondSDP is used (which blocks until ACK), the dialog is
	// already confirmed by the time this handler fires. ReadAck is still called
	// for consistency — it's a no-op if CSeq matches.
	if ds := session.GetDialogServerSession(); ds != nil {
		if err := ds.ReadAck(req, tx); err != nil {
			s.logger.Warnw("Dialog ReadAck failed", "error", err, "call_id", callID)
		}
	}

	// Only promote to connected for inbound calls. For outbound calls,
	// handleOutboundDialog owns the state machine via WaitAnswer. ACK is also
	// sent for non-2xx final responses (e.g. 407), and promoting to connected
	// on those causes a spurious initializing→connected→failed transition.
	if session.GetInfo().Direction == CallDirectionInbound {
		s.setCallState(session, CallStateConnected, "reinvite_acknowledged")
		s.logger.Debugw("SIP call established (ACK received)", "call_id", callID)
	}
}

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
		// but haven't registered in sessions yet, or that was already cleaned up.
		if err := s.dialogClientCache.ReadBye(req, tx); err == nil {
			s.logger.Infow("BYE handled by dialog client cache (no session)", "call_id", callID)
			return
		}
		s.logger.Warnw("BYE received for unknown session", "call_id", callID, "from", fromUser)
		s.sendResponse(tx, req, 481) // Call/Transaction Does Not Exist
		return
	}

	info := session.GetInfo()
	connectedDuration := ""
	if info.ConnectedTime != nil {
		connectedDuration = time.Since(*info.ConnectedTime).String()
	}
	s.logger.Infow("BYE received — tearing down call",
		"call_id", callID,
		"from", fromUser,
		"direction", info.Direction,
		"state", info.State,
		"duration", info.Duration,
		"connected_duration", connectedDuration,
		"session_ended", session.IsEnded())

	// Signal BYE for both call directions before branch-specific teardown.
	session.NotifyBye()

	// For outbound calls, let the DialogClientCache handle the BYE.
	// ReadBye sends 200 OK, sets dialog state to Ended, and cancels the dialog's
	// context — which unblocks handleOutboundDialog's select{} loop.
	//
	// IMPORTANT: For outbound calls, we do NOT call session.End() or removeSession()
	// here. The handleOutboundDialog goroutine owns the session lifecycle. It calls
	// onInvite synchronously (which launches startCall), then waits on the dialog
	// context. If we called session.End() here, it would kill the session while
	// onInvite/startCall is still setting up, causing "Session already ended before
	// startCall". Instead, handleOutboundDialog will call session.End() after
	// onInvite returns and the select{} fires.
	if info.Direction == CallDirectionOutbound {
		if err := s.dialogClientCache.ReadBye(req, tx); err != nil {
			// If dialog cache can't handle it (dialog already gone), respond ourselves
			s.logger.Warnw("Dialog cache ReadBye failed, responding directly",
				"error", err, "call_id", callID)
			s.sendResponse(tx, req, 200)
		}
		s.logger.Infow("Outbound BYE processed via dialog cache — session lifecycle delegated to handleOutboundDialog",
			"call_id", callID,
			"duration", info.Duration)

		// Fire the onBye callback for application-level cleanup
		s.mu.RLock()
		onBye := s.onBye
		s.mu.RUnlock()
		if onBye != nil {
			if err := onBye(session); err != nil {
				s.logger.Warnw("BYE handler returned error", "error", err, "call_id", callID)
			}
		}
		return
	}

	// Inbound call — respond 200 OK and tear down.
	// Use the dialog server cache if available (handles To-tag matching and
	// sets dialog state to Ended). Fall back to manual 200 OK otherwise.
	if err := s.dialogServerCache.ReadBye(req, tx); err != nil {
		s.logger.Debugw("Dialog server cache ReadBye failed, responding directly",
			"error", err, "call_id", callID)
		s.sendResponse(tx, req, 200)
	}

	// Get callback before calling it
	s.mu.RLock()
	onBye := s.onBye
	s.mu.RUnlock()

	if onBye != nil {
		if err := onBye(session); err != nil {
			s.logger.Warnw("BYE handler returned error", "error", err, "call_id", callID)
		}
	}

	// Remote sent BYE — clear onDisconnect so session.End() does NOT send
	// BYE back. The remote already knows the call is over.
	session.ClearOnDisconnect()
	s.beginEnding(session, "remote_bye")
	session.End()
	s.logger.Infow("SIP call ended (BYE processed)", "call_id", callID, "duration", info.Duration)
}

func (s *Server) handleCancel(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	s.markInviteCancelled(callID)

	s.mu.RLock()
	session, exists := s.sessions[callID]
	s.mu.RUnlock()

	inviteTerminated := s.terminatePendingInvite(callID, 487)

	if !exists && !inviteTerminated {
		s.logger.Warnw("CANCEL received for unknown session", "call_id", callID)
		s.sendResponse(tx, req, 481) // Call/Transaction Does Not Exist
		return
	}
	if exists && !inviteTerminated {
		state := session.GetState()
		if state != CallStateInitializing && state != CallStateRinging {
			s.logger.Warnw("CANCEL received for non-pending dialog", "call_id", callID, "state", state)
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
			s.logger.Warnw("CANCEL handler returned error", "error", err, "call_id", callID)
		}
	}

	// CANCEL is for an unanswered INVITE — clear onDisconnect so End()
	// does not attempt to send BYE (no dialog established yet).
	if exists {
		session.ClearOnDisconnect()
		s.beginEnding(session, "cancel_received")
		session.End()
	}
	s.sendResponse(tx, req, 200) // OK
	s.logger.Infow("SIP call cancelled", "call_id", callID)
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
// For in-dialog UPDATEs with SDP: update remote RTP (unless hold), respond with our SDP.
// For UPDATEs without SDP or unknown sessions: accept gracefully to keep dialog alive.
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
		s.logger.Debugw("UPDATE for unknown session, accepting", "call_id", callID)
		s.sendResponse(tx, req, 200)
		return
	}

	// If SDP body present, handle media renegotiation with hold detection
	if body := req.Body(); len(body) > 0 {
		sdpInfo, err := s.ParseSDP(body)
		if err != nil {
			s.logger.Warnw("Failed to parse UPDATE SDP", "error", err, "call_id", callID)
			s.sendResponse(tx, req, 200) // Accept anyway to keep dialog alive
			return
		}

		s.logger.Debugw("UPDATE SDP parsed",
			"call_id", callID,
			"sdp_direction", string(sdpInfo.Direction),
			"sdp_ip", sdpInfo.ConnectionIP,
			"sdp_port", sdpInfo.AudioPort,
			"is_hold", sdpInfo.IsHold())

		// Only update remote RTP for active media (not hold)
		if !sdpInfo.IsHold() {
			rtpHandler := session.GetRTPHandler()
			if rtpHandler != nil && sdpInfo.ConnectionIP != "" && sdpInfo.AudioPort > 0 {
				rtpHandler.SetRemoteAddr(sdpInfo.ConnectionIP, sdpInfo.AudioPort)
				session.SetRemoteRTP(sdpInfo.ConnectionIP, sdpInfo.AudioPort)
				s.logger.Debugw("Updated remote RTP from UPDATE",
					"call_id", callID,
					"remote_rtp_ip", sdpInfo.ConnectionIP,
					"remote_rtp_port", sdpInfo.AudioPort)
			}

			// Update codec if UPDATE proposes a different one
			if sdpInfo.PreferredCodec != nil {
				currentCodec := session.GetNegotiatedCodec()
				if currentCodec == nil || currentCodec.PayloadType != sdpInfo.PreferredCodec.PayloadType {
					rtpHandler := session.GetRTPHandler()
					if rtpHandler != nil {
						rtpHandler.SetCodec(sdpInfo.PreferredCodec)
					}
					session.SetNegotiatedCodec(sdpInfo.PreferredCodec.Name, int(sdpInfo.PreferredCodec.ClockRate))
					s.logger.Infow("Codec updated from UPDATE",
						"call_id", callID,
						"new_codec", sdpInfo.PreferredCodec.Name,
						"payload_type", sdpInfo.PreferredCodec.PayloadType)
				}
			}
		} else {
			s.logger.Infow("UPDATE indicates hold — keeping current RTP target",
				"call_id", callID,
				"sdp_direction", string(sdpInfo.Direction),
				"sdp_ip", sdpInfo.ConnectionIP)
		}

		s.respondWithCurrentSDP(tx, req, session)
	} else {
		s.sendResponse(tx, req, 200)
	}

	s.logger.Debugw("UPDATE handled", "call_id", callID)
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
// transfer via B2BUA bridge (INVITE-based), triggered by the LLM tool — not REFER.
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

// handleUnknownRequest is the catch-all handler for SIP methods without an explicit handler.
// This is critical for provider compatibility:
//   - Asterisk may send PUBLISH, MESSAGE, SUBSCRIBE in certain configurations
//   - Twilio sends SUBSCRIBE for dialog-info events
//   - FreeSWITCH sends MESSAGE for T.38 fax negotiation
//   - Vonage may send PRACK for reliable provisional responses
//
// For in-dialog requests (known Call-ID): respond 200 OK to prevent dialog teardown.
// For out-of-dialog SUBSCRIBE: respond 489 Bad Event (no event package supported).
// For other out-of-dialog requests: respond 405 Method Not Allowed.
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
		s.logger.Warnw("Unhandled SIP method for active session — accepting to keep dialog alive",
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
			s.logger.Warnw("Unknown SIP method received (no session) — rejecting",
				"method", method,
				"call_id", callID,
				"from", fromUser)
			s.sendResponse(tx, req, 405) // Method Not Allowed
		}
	}
}

func (s *Server) sendResponse(tx sip.ServerTransaction, req *sip.Request, statusCode int) {
	resp := sip.NewResponseFromRequest(req, statusCode, "", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Errorw("Failed to send SIP response",
			"error", err,
			"status", statusCode,
			"call_id", req.CallID().Value())
	}
}

// sendSDPResponse sends a SIP 200 OK response with the given SDP body.
// Adds a Contact header (required by RFC 3261 §13.3.1.1 for INVITE/re-INVITE responses)
// so that Asterisk, Twilio, and other providers know where to send subsequent requests.
func (s *Server) sendSDPResponse(tx sip.ServerTransaction, req *sip.Request, sdpBody string) {
	s.logger.Debugw("Sending SIP response with SDP",
		"call_id", req.CallID().Value(),
		"method", req.Method,
		"sdp_body", sdpBody)
	resp := sip.NewSDPResponseFromRequest(req, []byte(sdpBody))

	// Add Contact header if not already present — mandatory for INVITE/re-INVITE 200 OK.
	// Without this, Asterisk and other providers cannot route subsequent in-dialog requests
	// (re-INVITEs, BYEs) back to us, causing immediate call teardown.
	if resp.Contact() == nil {
		externalIP := s.listenConfig.GetExternalIP()
		scheme := "sip"
		contactHdr := &sip.ContactHeader{
			Address: sip.Uri{
				Scheme: scheme,
				Host:   externalIP,
				Port:   s.listenConfig.Port,
			},
		}
		resp.AppendHeader(contactHdr)
	}

	if err := tx.Respond(resp); err != nil {
		s.logger.Errorw("Failed to send SIP response with SDP",
			"error", err,
			"call_id", req.CallID().Value())
	}
}
