// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emiago/sipgo"
	"github.com/google/uuid"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

// Session manages a single SIP call session
type Session struct {
	mu sync.RWMutex

	info   SessionInfo
	config *Config
	ended  atomic.Bool

	ctx    context.Context
	cancel context.CancelFunc

	// RTP handling
	rtpHandler    *RTPHandler
	rtpLocalPort  int
	rtpRemoteAddr string
	rtpRemotePort int

	// Codec negotiation result
	negotiatedCodec *Codec

	// Outbound dialog phase tracks SIP answer progress independently from app call state.
	outboundDialogPhase OutboundDialogPhase
	inboundSetupPhase   InboundSetupPhase
	inboundTimings      InboundSetupTimings

	// User metadata for passing context between layers (e.g., outbound call info)
	metadata map[string]interface{}

	// Authentication and authorization context - available in all session methods
	auth            *types.Authentication                // Authentication principal
	assistant       *internal_assistant_entity.Assistant // Assistant entity
	conversationID  uint64                               // Conversation ID
	contextID       string                               // Call context ID (outbound)
	vaultCredential *protos.VaultCredential              // Vault-resolved SIP provider credential

	// byeReceived notifies Talk about remote BYE without ending the session early.
	byeReceived        chan struct{}
	byeReceivedOnce    sync.Once
	disconnectMetadata DisconnectMetadata

	// Outbound dialog session — stored so BYE/re-INVITE handlers can access it.
	// nil for inbound calls.
	dialogClientSession *sipgo.DialogClientSession

	// Inbound dialog session — stored so we can send BYE when ending an inbound call.
	// nil for outbound calls.
	dialogServerSession    *sipgo.DialogServerSession
	initialACKReceived     bool
	initialACKReceivedOnce sync.Once
	initialACKSignal       chan struct{}
	reInviteACKPending     bool
	reInviteACKCount       uint64

	// onDisconnect is called via Disconnect() to perform transport-level call teardown
	// (e.g., sending SIP BYE). NOT called by End() — the caller must invoke
	// Disconnect() explicitly before End() if a SIP BYE should be sent.
	// Set by the server that owns this session.
	onDisconnect func(session *Session)
	// onPreAnswerCancel is called by lifecycle cancellation while an outbound
	// INVITE is still waiting for a final answer. The outbound call owner wires
	// this to the WaitAnswer context so sipgo sends SIP CANCEL instead of BYE.
	onPreAnswerCancel func()
	// onEnded is called once after End() completes local teardown (RTP stop,
	// context cancel, state transition).
	onEnded func(session *Session)
}

type SessionOption func(*Session)

func WithSessionConfig(config *Config) SessionOption {
	return func(session *Session) { session.config = config }
}

func WithSessionDirection(direction CallDirection) SessionOption {
	return func(session *Session) { session.info.Direction = direction }
}

func WithSessionCallID(callID string) SessionOption {
	return func(session *Session) { session.info.CallID = callID }
}

func WithSessionCodec(codec *Codec) SessionOption {
	return func(session *Session) {
		if codec == nil {
			return
		}
		session.negotiatedCodec = codec
		session.info.Codec = codec.Name
		session.info.SampleRate = int(codec.ClockRate)
	}
}

func WithSessionAuth(auth *types.Authentication) SessionOption {
	return func(session *Session) { session.auth = auth }
}

func WithSessionAssistant(assistant *internal_assistant_entity.Assistant) SessionOption {
	return func(session *Session) { session.assistant = assistant }
}

func WithSessionConversationID(conversationID uint64) SessionOption {
	return func(session *Session) { session.conversationID = conversationID }
}

func WithSessionContextID(contextID string) SessionOption {
	return func(session *Session) { session.contextID = contextID }
}

func WithSessionVaultCredential(vaultCredential *protos.VaultCredential) SessionOption {
	return func(session *Session) { session.vaultCredential = vaultCredential }
}

// NewSession creates a new SIP session.
func NewSession(ctx context.Context, opts ...SessionOption) (*Session, error) {
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &Session{
		info: SessionInfo{
			CallID:     uuid.New().String(),
			LocalTag:   uuid.New().String()[:8],
			State:      CallStateInitializing,
			StartTime:  time.Now(),
			Codec:      CodecPCMU.Name,
			SampleRate: int(CodecPCMU.ClockRate),
		},
		ctx:              sessionCtx,
		cancel:           cancel,
		negotiatedCodec:  &CodecPCMU,
		byeReceived:      make(chan struct{}),
		initialACKSignal: make(chan struct{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(session)
		}
	}
	if session.config == nil {
		cancel()
		return nil, fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}
	// Outbound identity/auth is validated before the INVITE is built.
	if session.info.Direction == CallDirectionOutbound {
		if err := session.config.Validate(); err != nil {
			cancel()
			return nil, err
		}
	} else if err := session.config.ValidateRTP(); err != nil {
		cancel()
		return nil, err
	}
	if session.info.Direction == CallDirectionOutbound {
		session.outboundDialogPhase = OutboundDialogPhaseInviting
	}

	return session, nil
}

// GetInfo returns the current session information
func (s *Session) GetInfo() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info := s.info
	info.Duration = info.GetDuration()
	return info
}

// GetCallID returns the call ID
func (s *Session) GetCallID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info.CallID
}

// SetState updates the session state with proper state machine transitions
func (s *Session) SetState(state CallState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previousState := s.info.State

	// Validate state transitions
	if !s.isValidTransition(previousState, state) {
		return
	}

	s.info.State = state

	switch state {
	case CallStateConnected:
		now := time.Now()
		s.info.ConnectedTime = &now
	case CallStateEnded:
		now := time.Now()
		s.info.EndTime = &now
	case CallStateFailed:
		now := time.Now()
		s.info.EndTime = &now
	case CallStateCancelled:
		now := time.Now()
		s.info.EndTime = &now
	}

}

// isValidTransition checks if a state transition is valid
func (s *Session) isValidTransition(from, to CallState) bool {
	// Terminal states are immutable.
	if from.IsTerminal() {
		return false
	}

	// Allow active states to transition to terminal cleanup states.
	if to == CallStateEnded || to == CallStateFailed || to == CallStateCancelled {
		return true
	}

	// Define valid transitions
	validTransitions := map[CallState][]CallState{
		CallStateInitializing:    {CallStateRinging, CallStateConnected},
		CallStateRinging:         {CallStateConnected, CallStateEnding},
		CallStateConnected:       {CallStateOnHold, CallStateTransferring, CallStateEnding},
		CallStateOnHold:          {CallStateConnected, CallStateEnding},
		CallStateTransferring:    {CallStateConnected, CallStateBridgeConnected, CallStateEnding},
		CallStateBridgeConnected: {CallStateConnected, CallStateEnding},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, validTo := range allowed {
		if validTo == to {
			return true
		}
	}
	return false
}

// SetRemoteRTP sets the remote RTP address after SDP negotiation
func (s *Session) SetRemoteRTP(addr string, port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rtpRemoteAddr = addr
	s.rtpRemotePort = port
	s.info.RemoteRTPAddress = fmt.Sprintf("%s:%d", addr, port)
}

// SetLocalRTP sets the local RTP address
func (s *Session) SetLocalRTP(addr string, port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rtpLocalPort = port
	s.info.LocalRTPAddress = fmt.Sprintf("%s:%d", addr, port)
}

// GetLocalRTP returns the local RTP IP and port for this session.
func (s *Session) GetLocalRTP() (string, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Parse IP from the stored LocalRTPAddress ("ip:port" format)
	addr := s.info.LocalRTPAddress
	if addr == "" {
		return "", s.rtpLocalPort
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, s.rtpLocalPort
	}
	return host, s.rtpLocalPort
}

// GetRTPLocalPort returns the local RTP port bound for this session.
func (s *Session) GetRTPLocalPort() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rtpLocalPort
}

// SetNegotiatedCodec sets the negotiated codec
func (s *Session) SetNegotiatedCodec(codecName string, sampleRate int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	codec := GetCodecByName(codecName)
	if codec == nil {
		codec = &CodecPCMU
	}
	s.negotiatedCodec = codec
	s.info.Codec = codec.Name
	s.info.SampleRate = sampleRate
}

// GetNegotiatedCodec returns the negotiated codec
func (s *Session) GetNegotiatedCodec() *Codec {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.negotiatedCodec
}

func (s *Session) SetOutboundDialogPhase(phase OutboundDialogPhase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outboundDialogPhase = phase
}

func (s *Session) GetOutboundDialogPhase() OutboundDialogPhase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.outboundDialogPhase
}

func (s *Session) SetInboundSetupPhase(phase InboundSetupPhase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inboundSetupPhase = phase
}

func (s *Session) GetInboundSetupPhase() InboundSetupPhase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inboundSetupPhase
}

func (s *Session) MarkInboundSetupTimestamp(phase InboundSetupPhase, at time.Time) {
	if at.IsZero() {
		at = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch phase {
	case InboundSetupPhaseInviteReceived:
		s.inboundTimings.InviteReceivedAt = at
	case InboundSetupPhaseTryingSent:
		s.inboundTimings.TryingSentAt = at
	case InboundSetupPhaseRingingSent:
		s.inboundTimings.RingingSentAt = at
	case InboundSetupPhaseAnswered:
		s.inboundTimings.AnsweredAt = at
	case InboundSetupPhaseACKConfirmed:
		s.inboundTimings.ACKConfirmedAt = at
	}
}

func (s *Session) MarkInboundFirstRTPReceived() bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.inboundTimings.FirstRTPReceivedAt.IsZero() {
		return false
	}
	s.inboundTimings.FirstRTPReceivedAt = now
	return true
}

func (s *Session) MarkInboundAssistantAudioReady() bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.inboundTimings.FirstAssistantAudioReadyAt.IsZero() {
		return false
	}
	s.inboundTimings.FirstAssistantAudioReadyAt = now
	return true
}

func (s *Session) MarkInboundFirstAssistantAudioSent() bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.inboundTimings.FirstAssistantAudioSentAt.IsZero() {
		return false
	}
	s.inboundTimings.FirstAssistantAudioSentAt = now
	return true
}

func (s *Session) GetInboundSetupTimings() InboundSetupTimings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inboundTimings
}

func (s *Session) GetInboundLatencyMetrics() map[string]int64 {
	return s.GetInboundSetupTimings().LatencyMetrics()
}

func (s *Session) SetInboundSetupTimings(timings InboundSetupTimings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inboundTimings = timings
}

// SetRTPHandler sets the RTP handler for this session.
// Once adopted by the session, RTP teardown is owned by Session.End.
// Replacing a different handler stops the previous one so sockets cannot leak.
func (s *Session) SetRTPHandler(handler *RTPHandler) {
	s.mu.Lock()
	previous := s.rtpHandler
	if previous == handler {
		s.mu.Unlock()
		return
	}
	s.rtpHandler = handler
	s.mu.Unlock()

	if previous != nil {
		_ = previous.Stop()
	}
}

// GetRTPHandler returns the RTP handler for this session
func (s *Session) GetRTPHandler() *RTPHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rtpHandler
}

// Context returns the session context
func (s *Session) Context() context.Context {
	return s.ctx
}

// SetMetadata stores a key-value pair on the session
func (s *Session) SetMetadata(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.metadata == nil {
		s.metadata = make(map[string]interface{})
	}
	s.metadata[key] = value
}

// GetMetadata retrieves a value by key from session metadata
func (s *Session) GetMetadata(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.metadata == nil {
		return nil, false
	}
	v, ok := s.metadata[key]
	return v, ok
}

// SetDialogClientSession stores the outbound DialogClientSession on this session.
// This allows BYE and re-INVITE handlers to interact with the sipgo dialog.
func (s *Session) SetDialogClientSession(ds *sipgo.DialogClientSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dialogClientSession = ds
}

// GetDialogClientSession returns the outbound DialogClientSession, or nil for inbound calls.
func (s *Session) GetDialogClientSession() *sipgo.DialogClientSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dialogClientSession
}

// SetDialogServerSession stores the inbound DialogServerSession on this session.
// This allows the server to send BYE when ending an inbound call.
func (s *Session) SetDialogServerSession(ds *sipgo.DialogServerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dialogServerSession = ds
}

// GetDialogServerSession returns the inbound DialogServerSession, or nil for outbound calls.
func (s *Session) GetDialogServerSession() *sipgo.DialogServerSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dialogServerSession
}

func (s *Session) MarkInitialACKReceived() bool {
	s.mu.Lock()
	if s.initialACKReceived {
		s.mu.Unlock()
		return false
	}
	s.initialACKReceived = true
	initialACKSignal := s.initialACKSignal
	s.mu.Unlock()
	if initialACKSignal != nil {
		s.initialACKReceivedOnce.Do(func() {
			close(initialACKSignal)
		})
	}
	return true
}

func (s *Session) HasInitialACKReceived() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialACKReceived
}

func (s *Session) InitialACKSignal() <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialACKSignal
}

func (s *Session) BeginReInviteACKWait() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reInviteACKPending = true
}

func (s *Session) HasReInviteACKPending() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reInviteACKPending
}

func (s *Session) CompleteReInviteACKWait() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.reInviteACKPending {
		return false
	}
	s.reInviteACKPending = false
	s.reInviteACKCount++
	return true
}

func (s *Session) ClearReInviteACKWait() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.reInviteACKPending {
		return false
	}
	s.reInviteACKPending = false
	return true
}

func (s *Session) ReInviteACKCount() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reInviteACKCount
}

// SetOnDisconnect registers a callback that is invoked when the session is disconnected.
// This allows the SIP server to inject transport-level call teardown (e.g., sending BYE)
// without the session needing to know about SIP signaling internals.
func (s *Session) SetOnDisconnect(fn func(session *Session)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onDisconnect = fn
}

// ClearOnDisconnect removes the disconnect callback without invoking it.
// Used when the remote party initiated teardown (BYE/CANCEL) so session.End()
// does not send BYE back to a party that already knows the call is over.
func (s *Session) ClearOnDisconnect() {
	s.mu.Lock()
	s.onDisconnect = nil
	s.mu.Unlock()
}

// SetOnPreAnswerCancel registers a callback for lifecycle-owned cancellation
// before an outbound INVITE has received a final answer.
func (s *Session) SetOnPreAnswerCancel(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onPreAnswerCancel = fn
}

// ClearOnPreAnswerCancel removes the pre-answer cancel callback.
func (s *Session) ClearOnPreAnswerCancel() {
	s.mu.Lock()
	s.onPreAnswerCancel = nil
	s.mu.Unlock()
}

// CancelPreAnswer invokes the registered pre-answer cancel callback once.
func (s *Session) CancelPreAnswer() {
	s.mu.Lock()
	fn := s.onPreAnswerCancel
	s.onPreAnswerCancel = nil
	s.mu.Unlock()

	if fn != nil {
		fn()
	}
}

// SetOnEnded registers a callback that is invoked once End() completes.
func (s *Session) SetOnEnded(fn func(session *Session)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onEnded = fn
}

// Disconnect performs transport-level call teardown by invoking the onDisconnect callback.
// This sends a SIP BYE (or equivalent) to the remote party before local cleanup.
// Safe to call multiple times — the callback is cleared after first invocation.
func (s *Session) Disconnect() {
	s.mu.Lock()
	fn := s.onDisconnect
	s.onDisconnect = nil // Clear to prevent double-disconnect
	s.mu.Unlock()

	if fn != nil {
		fn(s)
	}
}

// GetAuth returns the authentication principal for this session.
func (s *Session) GetAuth() *types.Authentication {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auth
}

// SetAuth sets the authentication principal for this session.
func (s *Session) SetAuth(auth *types.Authentication) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auth = auth
}

// GetAssistant returns the assistant entity for this session.
func (s *Session) GetAssistant() *internal_assistant_entity.Assistant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.assistant
}

// SetAssistant sets the assistant entity for this session.
func (s *Session) SetAssistant(assistant *internal_assistant_entity.Assistant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assistant = assistant
}

// GetConversationID returns the conversation ID for this session.
func (s *Session) GetConversationID() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conversationID
}

func (s *Session) GetContextID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.contextID
}

// SetConversationID sets the conversation ID for this session.
func (s *Session) SetConversationID(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversationID = id
}

// GetVaultCredential returns the vault-resolved SIP provider credential for this session.
func (s *Session) GetVaultCredential() *protos.VaultCredential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vaultCredential
}

// End terminates the SIP session gracefully. This is the single teardown function —
// all triggers (BYE, pipeline end, streamer close) route here. Owns all side effects:
// 1. Send BYE via onDisconnect callback
// 2. Stop RTP
// 3. Cancel context
// 4. Set terminal state
func (s *Session) End() {
	if !s.ended.CompareAndSwap(false, true) {
		return
	}

	s.mu.RLock()
	terminal := s.info.State.IsTerminal()
	s.mu.RUnlock()
	if !terminal {
		s.SetState(CallStateEnding)
	}

	// Send BYE to remote party (clears callback to prevent double-send)
	s.Disconnect()

	// Stop RTP
	s.mu.Lock()
	rtpHandler := s.rtpHandler
	s.rtpHandler = nil
	s.mu.Unlock()
	if rtpHandler != nil {
		_ = rtpHandler.Stop()
	}

	// Cancel context — unblocks anything waiting on session.Context()
	s.cancel()

	s.mu.RLock()
	terminal = s.info.State.IsTerminal()
	s.mu.RUnlock()
	if !terminal {
		s.SetState(CallStateEnded)
	}

	s.mu.RLock()
	onEnded := s.onEnded
	s.mu.RUnlock()
	if onEnded != nil {
		onEnded(s)
	}
}

// IsActive returns whether the session is still active
func (s *Session) IsActive() bool {
	if s.ended.Load() {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info.State.IsActive()
}

// IsEnded returns whether the session has ended
func (s *Session) IsEnded() bool {
	return s.ended.Load()
}

// NotifyBye signals that a SIP BYE has been received for this session.
// This is safe to call multiple times — only the first call has effect.
// It does NOT end the session; it merely notifies listeners (e.g., startCall)
// that a BYE was received so they can shut down gracefully.
func (s *Session) NotifyBye() {
	s.byeReceivedOnce.Do(func() {
		close(s.byeReceived)
	})
}

func (s *Session) SetDisconnectMetadata(metadata DisconnectMetadata) {
	if metadata.Reason == "" {
		metadata.Reason = DisconnectReasonRemoteHangup
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disconnectMetadata = metadata
	if s.metadata == nil {
		s.metadata = make(map[string]interface{})
	}
	s.metadata[MetadataDisconnectReason] = metadata.Reason
	if metadata.Raw != "" {
		s.metadata[MetadataDisconnectRawReason] = metadata.Raw
	}
}

func (s *Session) GetDisconnectMetadata() DisconnectMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	metadata := s.disconnectMetadata
	if metadata.Reason == "" {
		metadata.Reason = DisconnectReasonRemoteHangup
	}
	return metadata
}

// ByeReceived returns a channel that is closed when a SIP BYE is received.
// Use this in select{} to detect early BYE without relying on session.End().
func (s *Session) ByeReceived() <-chan struct{} {
	return s.byeReceived
}

// GetConfig returns the SIP configuration for this session.
func (s *Session) GetConfig() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetState returns the current session state
func (s *Session) GetState() CallState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info.State
}

// GetRTPStats returns RTP statistics if available
func (s *Session) GetRTPStats() *RTPStats {
	s.mu.RLock()
	rtpHandler := s.rtpHandler
	s.mu.RUnlock()

	if rtpHandler == nil {
		return nil
	}

	stats := rtpHandler.GetDetailedStats()
	return &stats
}
