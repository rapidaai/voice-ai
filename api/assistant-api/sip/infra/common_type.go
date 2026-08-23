// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"encoding/json"
	"errors"
	"time"

	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/protos"
)

var (
	ErrInvalidConfig              = internal_core.ErrInvalidConfig
	ErrSessionNotFound            = internal_core.ErrSessionNotFound
	ErrSessionClosed              = internal_core.ErrSessionClosed
	ErrRTPNotInitialized          = internal_core.ErrRTPNotInitialized
	ErrRTPHandlerStopped          = internal_core.ErrRTPHandlerStopped
	ErrRTPOutputQueueFull         = internal_core.ErrRTPOutputQueueFull
	ErrRTPPortRangeExhausted      = internal_core.ErrRTPPortRangeExhausted
	ErrSDPParseFailed             = internal_core.ErrSDPParseFailed
	ErrCodecNotSupported          = internal_core.ErrCodecNotSupported
	ErrConnectionFailed           = internal_core.ErrConnectionFailed
	ErrAuthRequired               = internal_core.ErrAuthRequired
	ErrOutboundFromUserRequired   = internal_core.ErrOutboundFromUserRequired
	ErrInboundACKTimeout          = internal_core.ErrInboundACKTimeout
	ErrInboundInviteCancelled     = internal_core.ErrInboundInviteCancelled
	ErrInboundAnswerPolicyTimeout = internal_core.ErrInboundAnswerPolicyTimeout
	ErrBridgeLifecycleRejected    = internal_core.ErrBridgeLifecycleRejected
)

type SIPError = internal_core.SIPError

func NewSIPError(op, callID, message string, err error) *SIPError {
	return &SIPError{Op: op, CallID: callID, Message: message, Err: err}
}

type Transport string

const (
	TransportUDP Transport = "udp"
	TransportTCP Transport = "tcp"
	TransportTLS Transport = "tls"
)

func (t Transport) String() string {
	return string(t)
}

func (t Transport) IsValid() bool {
	switch t {
	case TransportUDP, TransportTCP, TransportTLS:
		return true
	default:
		return false
	}
}

type Config struct {
	Server   string `json:"sip_server" mapstructure:"sip_server"`
	Username string `json:"sip_username" mapstructure:"sip_username"`
	Password string `json:"sip_password" mapstructure:"sip_password"`
	Realm    string `json:"sip_realm" mapstructure:"sip_realm"`
	Domain   string `json:"sip_domain,omitempty" mapstructure:"sip_domain"`

	CallerID      string            `json:"sip_caller_id,omitempty" mapstructure:"sip_caller_id"`
	CustomHeaders map[string]string `json:"sip_headers,omitempty" mapstructure:"sip_headers"`

	Port              int       `json:"sip_port" mapstructure:"sip_port"`
	Transport         Transport `json:"sip_transport" mapstructure:"sip_transport"`
	RTPPortRangeStart int       `json:"rtp_port_range_start" mapstructure:"rtp_port_range_start"`
	RTPPortRangeEnd   int       `json:"rtp_port_range_end" mapstructure:"rtp_port_range_end"`
	SRTPEnabled       bool      `json:"srtp_enabled" mapstructure:"srtp_enabled"`

	RegisterTimeout     time.Duration `json:"register_timeout,omitempty" mapstructure:"register_timeout"`
	InviteTimeout       time.Duration `json:"invite_timeout,omitempty" mapstructure:"invite_timeout"`
	SessionTimeout      time.Duration `json:"session_timeout,omitempty" mapstructure:"session_timeout"`
	MediaTimeoutInitial time.Duration `json:"media_timeout_initial,omitempty" mapstructure:"media_timeout_initial"`
	MediaTimeout        time.Duration `json:"media_timeout,omitempty" mapstructure:"media_timeout"`
	KeepAliveEnabled    bool          `json:"keepalive_enabled,omitempty" mapstructure:"keepalive_enabled"`

	InboundAnswerMode      InboundAnswerMode `json:"inbound_answer_mode,omitempty" mapstructure:"inbound_answer_mode"`
	InboundMinRingDuration time.Duration     `json:"inbound_min_ring_duration,omitempty" mapstructure:"inbound_min_ring_duration"`
	InboundMaxRingDuration time.Duration     `json:"inbound_max_ring_duration,omitempty" mapstructure:"inbound_max_ring_duration"`
	InboundACKTimeout      time.Duration     `json:"inbound_ack_timeout,omitempty" mapstructure:"inbound_ack_timeout"`
}

func (c *Config) Validate() error {
	return c.toCore().Validate()
}

func (c *Config) ApplyOperationalDefaults(port int, transport Transport, rtpStart, rtpEnd int) {
	if c == nil {
		return
	}
	coreConfig := c.toCore()
	coreConfig.ApplyOperationalDefaults(port, internal_core.Transport(transport), rtpStart, rtpEnd)
	*c = configFromCore(coreConfig)
}

func (c *Config) ApplyTimeoutDefaults(registerTimeout, inviteTimeout, sessionTimeout time.Duration) {
	if c == nil {
		return
	}
	coreConfig := c.toCore()
	coreConfig.ApplyTimeoutDefaults(registerTimeout, inviteTimeout, sessionTimeout)
	*c = configFromCore(coreConfig)
}

func (c *Config) ApplyMediaTimeoutDefaults(initialTimeout, mediaTimeout time.Duration) {
	if c == nil {
		return
	}
	coreConfig := c.toCore()
	coreConfig.ApplyMediaTimeoutDefaults(initialTimeout, mediaTimeout)
	*c = configFromCore(coreConfig)
}

func (c *Config) ApplyInboundAnswerDefaults(
	mode InboundAnswerMode,
	minRingDuration time.Duration,
	maxRingDuration time.Duration,
	ackTimeout time.Duration,
) {
	if c == nil {
		return
	}
	coreConfig := c.toCore()
	coreConfig.ApplyInboundAnswerDefaults(
		internal_core.InboundAnswerMode(mode),
		minRingDuration,
		maxRingDuration,
		ackTimeout,
	)
	*c = configFromCore(coreConfig)
}

func (c *Config) EffectiveRegisterTimeout() time.Duration {
	return c.toCore().EffectiveRegisterTimeout()
}

func (c *Config) ValidateRTP() error {
	return c.toCore().ValidateRTP()
}

func (c *Config) GetTransport() Transport {
	return Transport(c.toCore().GetTransport())
}

func (c *Config) GetSIPURI() string {
	return c.toCore().GetSIPURI()
}

func (c *Config) GetListenAddr() string {
	return c.toCore().GetListenAddr()
}

func (c *Config) toCore() *internal_core.Config {
	if c == nil {
		return nil
	}
	return &internal_core.Config{
		Server:                 c.Server,
		Username:               c.Username,
		Password:               c.Password,
		Realm:                  c.Realm,
		Domain:                 c.Domain,
		CallerID:               c.CallerID,
		CustomHeaders:          c.CustomHeaders,
		Port:                   c.Port,
		Transport:              internal_core.Transport(c.Transport),
		RTPPortRangeStart:      c.RTPPortRangeStart,
		RTPPortRangeEnd:        c.RTPPortRangeEnd,
		SRTPEnabled:            c.SRTPEnabled,
		RegisterTimeout:        c.RegisterTimeout,
		InviteTimeout:          c.InviteTimeout,
		SessionTimeout:         c.SessionTimeout,
		MediaTimeoutInitial:    c.MediaTimeoutInitial,
		MediaTimeout:           c.MediaTimeout,
		KeepAliveEnabled:       c.KeepAliveEnabled,
		InboundAnswerMode:      internal_core.InboundAnswerMode(c.InboundAnswerMode),
		InboundMinRingDuration: c.InboundMinRingDuration,
		InboundMaxRingDuration: c.InboundMaxRingDuration,
		InboundACKTimeout:      c.InboundACKTimeout,
	}
}

func configFromCore(c *internal_core.Config) Config {
	if c == nil {
		return Config{}
	}
	return Config{
		Server:                 c.Server,
		Username:               c.Username,
		Password:               c.Password,
		Realm:                  c.Realm,
		Domain:                 c.Domain,
		CallerID:               c.CallerID,
		CustomHeaders:          c.CustomHeaders,
		Port:                   c.Port,
		Transport:              Transport(c.Transport),
		RTPPortRangeStart:      c.RTPPortRangeStart,
		RTPPortRangeEnd:        c.RTPPortRangeEnd,
		SRTPEnabled:            c.SRTPEnabled,
		RegisterTimeout:        c.RegisterTimeout,
		InviteTimeout:          c.InviteTimeout,
		SessionTimeout:         c.SessionTimeout,
		MediaTimeoutInitial:    c.MediaTimeoutInitial,
		MediaTimeout:           c.MediaTimeout,
		KeepAliveEnabled:       c.KeepAliveEnabled,
		InboundAnswerMode:      InboundAnswerMode(c.InboundAnswerMode),
		InboundMinRingDuration: c.InboundMinRingDuration,
		InboundMaxRingDuration: c.InboundMaxRingDuration,
		InboundACKTimeout:      c.InboundACKTimeout,
	}
}

type CallState string

const (
	CallStateInitializing    CallState = "initializing"
	CallStateRinging         CallState = "ringing"
	CallStateConnected       CallState = "connected"
	CallStateOnHold          CallState = "on_hold"
	CallStateTransferring    CallState = "transferring"
	CallStateBridgeConnected CallState = "bridge_connected"
	CallStateEnding          CallState = "ending"
	CallStateEnded           CallState = "ended"
	CallStateFailed          CallState = "failed"
	CallStateCancelled       CallState = "cancelled"
)

func (s CallState) String() string {
	return string(s)
}

func (s CallState) IsTerminal() bool {
	return s == CallStateEnded || s == CallStateFailed || s == CallStateCancelled
}

func (s CallState) IsActive() bool {
	return s == CallStateConnected || s == CallStateRinging || s == CallStateOnHold || s == CallStateTransferring || s == CallStateBridgeConnected
}

func callStateFromCore(state internal_core.CallState) CallState {
	return CallState(state)
}

func (s CallState) toCore() internal_core.CallState {
	return internal_core.CallState(s)
}

type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

func (d CallDirection) toCore() internal_core.CallDirection {
	return internal_core.CallDirection(d)
}

type InboundSetupPhase string

const (
	InboundSetupPhaseInviteReceived   InboundSetupPhase = "invite_received"
	InboundSetupPhaseTryingSent       InboundSetupPhase = "trying_sent"
	InboundSetupPhaseRingingSent      InboundSetupPhase = "ringing_sent"
	InboundSetupPhaseAuthenticated    InboundSetupPhase = "authenticated"
	InboundSetupPhaseRouted           InboundSetupPhase = "routed"
	InboundSetupPhaseMediaAllocated   InboundSetupPhase = "media_allocated"
	InboundSetupPhaseApplicationReady InboundSetupPhase = "application_ready"
	InboundSetupPhaseAnswerReady      InboundSetupPhase = "answer_ready"
	InboundSetupPhaseAnswered         InboundSetupPhase = "answered"
	InboundSetupPhaseACKConfirmed     InboundSetupPhase = "ack_confirmed"
	InboundSetupPhaseMediaFlowing     InboundSetupPhase = "media_flowing"
)

type InboundAnswerMode string

const (
	InboundAnswerModeImmediate            InboundAnswerMode = "answer_immediately"
	InboundAnswerModeAfterMinRingDuration InboundAnswerMode = "answer_after_min_ring_ms"
)

type InboundSetupTimings struct {
	InviteReceivedAt           time.Time
	TryingSentAt               time.Time
	RingingSentAt              time.Time
	AnsweredAt                 time.Time
	ACKConfirmedAt             time.Time
	FirstRTPReceivedAt         time.Time
	FirstAssistantAudioReadyAt time.Time
	FirstAssistantAudioSentAt  time.Time
}

func inboundSetupTimingsFromCore(t internal_core.InboundSetupTimings) InboundSetupTimings {
	return InboundSetupTimings{
		InviteReceivedAt:           t.InviteReceivedAt,
		TryingSentAt:               t.TryingSentAt,
		RingingSentAt:              t.RingingSentAt,
		AnsweredAt:                 t.AnsweredAt,
		ACKConfirmedAt:             t.ACKConfirmedAt,
		FirstRTPReceivedAt:         t.FirstRTPReceivedAt,
		FirstAssistantAudioReadyAt: t.FirstAssistantAudioReadyAt,
		FirstAssistantAudioSentAt:  t.FirstAssistantAudioSentAt,
	}
}

type SessionInfo struct {
	CallID           string        `json:"call_id"`
	LocalTag         string        `json:"local_tag"`
	RemoteTag        string        `json:"remote_tag"`
	LocalURI         string        `json:"local_uri"`
	RemoteURI        string        `json:"remote_uri"`
	State            CallState     `json:"state"`
	Direction        CallDirection `json:"direction"`
	StartTime        time.Time     `json:"start_time"`
	ConnectedTime    *time.Time    `json:"connected_time,omitempty"`
	EndTime          *time.Time    `json:"end_time,omitempty"`
	LocalRTPAddress  string        `json:"local_rtp_address"`
	RemoteRTPAddress string        `json:"remote_rtp_address"`
	Codec            string        `json:"codec"`
	SampleRate       int           `json:"sample_rate"`
	Duration         time.Duration `json:"duration,omitempty"`
}

func (s *SessionInfo) GetDuration() time.Duration {
	if s.EndTime != nil && s.ConnectedTime != nil {
		return s.EndTime.Sub(*s.ConnectedTime)
	}
	if s.ConnectedTime != nil {
		return time.Since(*s.ConnectedTime)
	}
	return 0
}

func sessionInfoFromCore(info internal_core.SessionInfo) SessionInfo {
	return SessionInfo{
		CallID:           info.CallID,
		LocalTag:         info.LocalTag,
		RemoteTag:        info.RemoteTag,
		LocalURI:         info.LocalURI,
		RemoteURI:        info.RemoteURI,
		State:            CallState(info.State),
		Direction:        CallDirection(info.Direction),
		StartTime:        info.StartTime,
		ConnectedTime:    info.ConnectedTime,
		EndTime:          info.EndTime,
		LocalRTPAddress:  info.LocalRTPAddress,
		RemoteRTPAddress: info.RemoteRTPAddress,
		Codec:            info.Codec,
		SampleRate:       info.SampleRate,
		Duration:         info.Duration,
	}
}

type EventType string

const (
	EventTypeInvite     EventType = "invite"
	EventTypeRinging    EventType = "ringing"
	EventTypeConnected  EventType = "connected"
	EventTypeBye        EventType = "bye"
	EventTypeCancel     EventType = "cancel"
	EventTypeDTMF       EventType = "dtmf"
	EventTypeError      EventType = "error"
	EventTypeRTPStarted EventType = "rtp_started"
	EventTypeRTPStopped EventType = "rtp_stopped"
)

const (
	BridgeCallTimeout                    = 30 * time.Second
	BridgeSafetyTimeout                  = 5 * time.Minute
	MetadataBridgeTransferTarget         = "bridge_transfer_target"
	MetadataBridgeTransferStatus         = "bridge_transfer_status"
	MetadataBridgeTransferDuration       = "bridge_transfer_duration"
	MetadataBridgeTransferOutboundCallID = "bridge_transfer_outbound_call_id"
	MetadataDisconnectReason             = "disconnect_reason"
	MetadataDisconnectRawReason          = "disconnect_raw_reason"
	PostTransferActionEndCall            = "end_call"
	PostTransferActionResumeAI           = "resume_ai"
)

type Event struct {
	Type      EventType              `json:"type"`
	CallID    string                 `json:"call_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

const (
	DisconnectReasonRemoteHangup   = "remote_hangup"
	DisconnectReasonNormalClearing = "normal_clearing"
	DisconnectReasonBusy           = "busy"
	DisconnectReasonNoAnswer       = "no_answer"
	DisconnectReasonRejected       = "rejected"
	DisconnectReasonCancelled      = "cancelled"
	DisconnectReasonNetworkFailure = "network_failure"
	DisconnectReasonRemoteError    = "remote_error"
)

type DisconnectMetadata struct {
	Reason             string
	Text               string
	Raw                string
	ProviderStatusCode int
}

func (m DisconnectMetadata) toCore() internal_core.DisconnectMetadata {
	return internal_core.DisconnectMetadata{
		Reason:             m.Reason,
		Text:               m.Text,
		Raw:                m.Raw,
		ProviderStatusCode: m.ProviderStatusCode,
	}
}

func disconnectMetadataFromCore(m internal_core.DisconnectMetadata) DisconnectMetadata {
	return DisconnectMetadata{
		Reason:             m.Reason,
		Text:               m.Text,
		Raw:                m.Raw,
		ProviderStatusCode: m.ProviderStatusCode,
	}
}

func NewEvent(eventType EventType, callID string, data map[string]interface{}) Event {
	return Event{
		Type:      eventType,
		CallID:    callID,
		Timestamp: time.Now(),
		Data:      data,
	}
}

type DTMFEvent struct {
	Digit    string `json:"digit"`
	Duration int    `json:"duration_ms"`
}

type RTPStats struct {
	PacketsSent     uint64        `json:"packets_sent"`
	PacketsReceived uint64        `json:"packets_received"`
	BytesSent       uint64        `json:"bytes_sent"`
	BytesReceived   uint64        `json:"bytes_received"`
	PacketsLost     uint64        `json:"packets_lost"`
	PacketsDropped  uint64        `json:"packets_dropped"`
	Jitter          time.Duration `json:"jitter"`
}

func rtpStatsFromCore(stats internal_core.RTPStats) RTPStats {
	return RTPStats{
		PacketsSent:     stats.PacketsSent,
		PacketsReceived: stats.PacketsReceived,
		BytesSent:       stats.BytesSent,
		BytesReceived:   stats.BytesReceived,
		PacketsLost:     stats.PacketsLost,
		PacketsDropped:  stats.PacketsDropped,
		Jitter:          stats.Jitter,
	}
}

func ParseConfigFromVault(vaultCredential *protos.VaultCredential) (*Config, error) {
	coreConfig, err := internal_core.ParseConfigFromVault(vaultCredential)
	if err != nil {
		return nil, err
	}
	config := configFromCore(coreConfig)
	return &config, nil
}

func ExtractDIDFromURI(uri string) string {
	return internal_core.ExtractDIDFromURI(uri)
}

func cloneMap(values map[string]interface{}) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]interface{}, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func copyJSONCompatibleMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	copied := make(map[string]string, len(raw))
	for key, value := range raw {
		copied[key] = value
	}
	return copied
}

func marshalJSONMap(value interface{}) map[string]interface{} {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func isCoreSIPError(err error) (*internal_core.SIPError, bool) {
	var sipErr *internal_core.SIPError
	if errors.As(err, &sipErr) {
		return sipErr, true
	}
	return nil, false
}
