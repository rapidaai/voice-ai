// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"time"

	"github.com/emiago/sipgo/sip"
)

func (s *Server) replayRejectedInboundInvite(request *sip.Request, transaction sip.ServerTransaction) bool {
	if request == nil || request.CallID() == nil || request.From() == nil || request.From().Params == nil {
		return false
	}
	fromTag, ok := request.From().Params.Get("tag")
	if !ok || fromTag == "" || request.CallID().Value() == "" {
		return false
	}
	key := inboundInviteKey{callID: request.CallID().Value(), fromTag: fromTag}

	now := time.Now()
	s.mu.Lock()
	rejectedInvite, exists := s.rejectedInvites[key]
	if exists && now.After(rejectedInvite.expiresAt) {
		delete(s.rejectedInvites, key)
		exists = false
	}
	s.mu.Unlock()
	if !exists {
		return false
	}

	if s.logger != nil {
		s.logger.Debugw("Replaying cached inbound INVITE rejection",
			"call_id", key.callID,
			"from_tag", key.fromTag,
			"status_code", rejectedInvite.statusCode)
	}
	response := sip.NewResponseFromRequest(request, rejectedInvite.statusCode, rejectedInvite.reason, nil)
	if rejectedInvite.includeContact && response.Contact() == nil && s.listenConfig != nil {
		contactHeader := s.listenConfig.SIPContactHeader()
		response.AppendHeader(&contactHeader)
	}
	if err := transaction.Respond(response); err != nil {
		s.logger.Errorw("Failed to replay cached inbound INVITE rejection",
			"error", err,
			"call_id", key.callID,
			"status_code", rejectedInvite.statusCode)
	}
	return true
}

func (s *Server) recordRejectedInboundInvite(request *sip.Request, response *sip.Response) {
	if response == nil || response.StatusCode < 300 {
		return
	}
	if request == nil || request.CallID() == nil || request.From() == nil || request.From().Params == nil {
		return
	}
	fromTag, ok := request.From().Params.Get("tag")
	if !ok || fromTag == "" || request.CallID().Value() == "" {
		return
	}
	now := time.Now()
	key := inboundInviteKey{callID: request.CallID().Value(), fromTag: fromTag}

	s.mu.Lock()
	if s.rejectedInvites == nil {
		s.rejectedInvites = make(map[inboundInviteKey]inboundRejectedInvite)
	}
	if len(s.rejectedInvites) >= MaxInboundRejectedInvites {
		s.pruneRejectedInboundInvitesLocked(now)
	}
	s.rejectedInvites[key] = inboundRejectedInvite{
		statusCode:     response.StatusCode,
		reason:         response.Reason,
		includeContact: response.Contact() != nil,
		expiresAt:      now.Add(InboundRejectedInviteTTL),
	}
	s.mu.Unlock()
}

func (s *Server) pruneRejectedInboundInvitesLocked(now time.Time) {
	for key, rejectedInvite := range s.rejectedInvites {
		if now.After(rejectedInvite.expiresAt) {
			delete(s.rejectedInvites, key)
		}
	}
	if len(s.rejectedInvites) < MaxInboundRejectedInvites {
		return
	}
	for key := range s.rejectedInvites {
		delete(s.rejectedInvites, key)
		if len(s.rejectedInvites) < MaxInboundRejectedInvites {
			return
		}
	}
}
