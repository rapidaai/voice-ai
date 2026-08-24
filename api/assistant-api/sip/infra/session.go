// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"

	"github.com/emiago/sipgo"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

func NewSession(ctx context.Context, opts ...SessionOption) (*Session, error) {
	inner, err := internal_core.NewSession(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return wrapSession(inner), nil
}

func wrapSession(inner *internal_core.Session) *Session {
	if inner == nil {
		return nil
	}
	return &Session{inner: inner}
}

func (s *Session) unwrap() *internal_core.Session {
	if s == nil {
		return nil
	}
	return s.inner
}

func (s *Session) GetInfo() SessionInfo {
	return sessionInfoFromCore(s.inner.GetInfo())
}

func (s *Session) GetCallID() string {
	return s.inner.GetCallID()
}

func (s *Session) SetState(state CallState) {
	s.inner.SetState(state.toCore())
}

func (s *Session) SetRemoteRTP(addr string, port int) {
	s.inner.SetRemoteRTP(addr, port)
}

func (s *Session) SetLocalRTP(addr string, port int) {
	s.inner.SetLocalRTP(addr, port)
}

func (s *Session) GetLocalRTP() (string, int) {
	return s.inner.GetLocalRTP()
}

func (s *Session) GetRTPLocalPort() int {
	return s.inner.GetRTPLocalPort()
}

func (s *Session) SetNegotiatedCodec(codecName string, sampleRate int) {
	s.inner.SetNegotiatedCodec(codecName, sampleRate)
}

func (s *Session) GetNegotiatedCodec() *Codec {
	return codecFromCore(s.inner.GetNegotiatedCodec())
}

func (s *Session) SetOutboundDialogPhase(phase OutboundDialogPhase) {
	s.inner.SetOutboundDialogPhase(internal_core.OutboundDialogPhase(phase))
}

func (s *Session) GetOutboundDialogPhase() OutboundDialogPhase {
	return OutboundDialogPhase(s.inner.GetOutboundDialogPhase())
}

func (s *Session) SetInboundSetupPhase(phase InboundSetupPhase) {
	s.inner.SetInboundSetupPhase(internal_core.InboundSetupPhase(phase))
}

func (s *Session) GetInboundSetupPhase() InboundSetupPhase {
	return InboundSetupPhase(s.inner.GetInboundSetupPhase())
}

func (s *Session) MarkInboundAssistantAudioReady() bool {
	return s.inner.MarkInboundAssistantAudioReady()
}

func (s *Session) MarkInboundFirstAssistantAudioSent() bool {
	return s.inner.MarkInboundFirstAssistantAudioSent()
}

func (s *Session) GetInboundSetupTimings() InboundSetupTimings {
	return inboundSetupTimingsFromCore(s.inner.GetInboundSetupTimings())
}

func (s *Session) GetInboundLatencyMetrics() map[string]int64 {
	return s.inner.GetInboundLatencyMetrics()
}

func (s *Session) SetRTPHandler(handler *RTPHandler) {
	s.inner.SetRTPHandler(handler.unwrap())
}

func (s *Session) GetRTPHandler() *RTPHandler {
	return wrapRTPHandler(s.inner.GetRTPHandler())
}

func (s *Session) Events() <-chan Event {
	out := make(chan Event)
	go func() {
		defer close(out)
		for event := range s.inner.Events() {
			out <- Event{
				Type:      EventType(event.Type),
				CallID:    event.CallID,
				Timestamp: event.Timestamp,
				Data:      event.Data,
			}
		}
	}()
	return out
}

func (s *Session) Errors() <-chan error {
	return s.inner.Errors()
}

func (s *Session) Context() context.Context {
	return s.inner.Context()
}

func (s *Session) SetMetadata(key string, value interface{}) {
	s.inner.SetMetadata(key, value)
}

func (s *Session) GetMetadata(key string) (interface{}, bool) {
	return s.inner.GetMetadata(key)
}

func (s *Session) SetDialogClientSession(ds *sipgo.DialogClientSession) {
	s.inner.SetDialogClientSession(ds)
}

func (s *Session) GetDialogClientSession() *sipgo.DialogClientSession {
	return s.inner.GetDialogClientSession()
}

func (s *Session) SetDialogServerSession(ds *sipgo.DialogServerSession) {
	s.inner.SetDialogServerSession(ds)
}

func (s *Session) GetDialogServerSession() *sipgo.DialogServerSession {
	return s.inner.GetDialogServerSession()
}

func (s *Session) MarkInitialACKReceived() bool {
	return s.inner.MarkInitialACKReceived()
}

func (s *Session) HasInitialACKReceived() bool {
	return s.inner.HasInitialACKReceived()
}

func (s *Session) BeginReInviteACKWait() {
	s.inner.BeginReInviteACKWait()
}

func (s *Session) HasReInviteACKPending() bool {
	return s.inner.HasReInviteACKPending()
}

func (s *Session) CompleteReInviteACKWait() bool {
	return s.inner.CompleteReInviteACKWait()
}

func (s *Session) ClearReInviteACKWait() bool {
	return s.inner.ClearReInviteACKWait()
}

func (s *Session) ReInviteACKCount() uint64 {
	return s.inner.ReInviteACKCount()
}

func (s *Session) SetOnDisconnect(fn func(session *Session)) {
	if fn == nil {
		s.inner.SetOnDisconnect(nil)
		return
	}
	s.inner.SetOnDisconnect(func(session *internal_core.Session) {
		fn(wrapSession(session))
	})
}

func (s *Session) ClearOnDisconnect() {
	s.inner.ClearOnDisconnect()
}

func (s *Session) SetOnPreAnswerCancel(fn func()) {
	s.inner.SetOnPreAnswerCancel(fn)
}

func (s *Session) ClearOnPreAnswerCancel() {
	s.inner.ClearOnPreAnswerCancel()
}

func (s *Session) CancelPreAnswer() {
	s.inner.CancelPreAnswer()
}

func (s *Session) SetOnEnded(fn func(session *Session)) {
	if fn == nil {
		s.inner.SetOnEnded(nil)
		return
	}
	s.inner.SetOnEnded(func(session *internal_core.Session) {
		fn(wrapSession(session))
	})
}

func (s *Session) Disconnect() {
	s.inner.Disconnect()
}

func (s *Session) GetAuth() *types.Authentication {
	return s.inner.GetAuth()
}

func (s *Session) SetAuth(auth *types.Authentication) {
	s.inner.SetAuth(auth)
}

func (s *Session) GetAssistant() *internal_assistant_entity.Assistant {
	return s.inner.GetAssistant()
}

func (s *Session) SetAssistant(assistant *internal_assistant_entity.Assistant) {
	s.inner.SetAssistant(assistant)
}

func (s *Session) GetConversationID() uint64 {
	return s.inner.GetConversationID()
}

func (s *Session) GetContextID() string {
	return s.inner.GetContextID()
}

func (s *Session) SetConversationID(id uint64) {
	s.inner.SetConversationID(id)
}

func (s *Session) GetVaultCredential() *protos.VaultCredential {
	return s.inner.GetVaultCredential()
}

func (s *Session) SendEvent(event Event) {
	s.inner.SendEvent(internal_core.Event{
		Type:      internal_core.EventType(event.Type),
		CallID:    event.CallID,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	})
}

func (s *Session) SendError(err error) {
	s.inner.SendError(err)
}

func (s *Session) End() {
	s.inner.End()
}

func (s *Session) IsActive() bool {
	return s.inner.IsActive()
}

func (s *Session) IsEnded() bool {
	return s.inner.IsEnded()
}

func (s *Session) NotifyBye() {
	s.inner.NotifyBye()
}

func (s *Session) SetDisconnectMetadata(metadata DisconnectMetadata) {
	s.inner.SetDisconnectMetadata(metadata.toCore())
}

func (s *Session) GetDisconnectMetadata() DisconnectMetadata {
	return disconnectMetadataFromCore(s.inner.GetDisconnectMetadata())
}

func (s *Session) ByeReceived() <-chan struct{} {
	return s.inner.ByeReceived()
}

func (s *Session) GetConfig() *Config {
	coreConfig := s.inner.GetConfig()
	if coreConfig == nil {
		return nil
	}
	config := configFromCore(coreConfig)
	return &config
}

func (s *Session) GetState() CallState {
	return callStateFromCore(s.inner.GetState())
}

func (s *Session) GetRTPStats() *RTPStats {
	stats := s.inner.GetRTPStats()
	if stats == nil {
		return nil
	}
	out := rtpStatsFromCore(*stats)
	return &out
}
