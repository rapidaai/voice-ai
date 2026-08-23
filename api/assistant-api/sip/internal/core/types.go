// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/rapidaai/protos"
)

var (
	ErrInvalidConfig              = errors.New("invalid SIP configuration")
	ErrSessionNotFound            = errors.New("SIP session not found")
	ErrSessionClosed              = errors.New("SIP session is closed")
	ErrRTPNotInitialized          = errors.New("RTP handler not initialized")
	ErrRTPHandlerStopped          = errors.New("RTP handler is stopped")
	ErrRTPMediaTimeout            = errors.New("RTP media timeout")
	ErrRTPOutputQueueFull         = errors.New("RTP output queue is full")
	ErrRTPPortRangeExhausted      = errors.New("no RTP ports available")
	ErrSDPParseFailed             = errors.New("failed to parse SDP")
	ErrCodecNotSupported          = errors.New("codec not supported")
	ErrConnectionFailed           = errors.New("SIP connection failed")
	ErrAuthRequired               = errors.New("SIP auth required but credentials are missing")
	ErrOutboundFromUserRequired   = errors.New("outbound From user is required")
	ErrInboundACKTimeout          = errors.New("inbound ACK timeout")
	ErrInboundInviteCancelled     = errors.New("inbound INVITE cancelled")
	ErrInboundAnswerPolicyTimeout = errors.New("inbound answer policy timeout")
	ErrBridgeLifecycleRejected    = errors.New("bridge lifecycle transition rejected")
)

// SIPError adds operation and call context to SIP failures.
type SIPError struct {
	Op      string
	CallID  string
	Code    int
	Message string
	Err     error
}

func (e *SIPError) Error() string {
	if e.CallID != "" {
		return fmt.Sprintf("sip %s [call_id=%s]: %s: %v", e.Op, e.CallID, e.Message, e.Err)
	}
	return fmt.Sprintf("sip %s: %s: %v", e.Op, e.Message, e.Err)
}

func (e *SIPError) Unwrap() error {
	return e.Err
}

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

// Config combines provider SIP settings from vault with platform runtime settings.
type Config struct {
	Server   string `json:"sip_server" mapstructure:"sip_server"`
	Username string `json:"sip_username" mapstructure:"sip_username"`
	Password string `json:"sip_password" mapstructure:"sip_password"`
	Realm    string `json:"sip_realm" mapstructure:"sip_realm"`
	Domain   string `json:"sip_domain,omitempty" mapstructure:"sip_domain"`

	// CallerID overrides the From header user in outbound calls.
	CallerID string `json:"sip_caller_id,omitempty" mapstructure:"sip_caller_id"`

	// CustomHeaders are added to outbound INVITE requests.
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

// Validate validates the shared SIP network configuration.
func (c *Config) Validate() error {
	return c.ValidateRTP()
}

// ApplyOperationalDefaults fills unset platform-owned SIP runtime settings.
func (c *Config) ApplyOperationalDefaults(port int, transport Transport, rtpStart, rtpEnd int) {
	if c.Port <= 0 && port > 0 {
		c.Port = port
	}
	if c.Transport == "" && transport != "" {
		c.Transport = transport
	}
	if c.RTPPortRangeStart <= 0 && rtpStart > 0 {
		c.RTPPortRangeStart = rtpStart
	}
	if c.RTPPortRangeEnd <= 0 && rtpEnd > 0 {
		c.RTPPortRangeEnd = rtpEnd
	}
}

func (c *Config) ApplyTimeoutDefaults(registerTimeout, inviteTimeout, sessionTimeout time.Duration) {
	if c.RegisterTimeout <= 0 && registerTimeout > 0 {
		c.RegisterTimeout = registerTimeout
	}
	if c.InviteTimeout <= 0 && inviteTimeout > 0 {
		c.InviteTimeout = inviteTimeout
	}
	if c.SessionTimeout <= 0 && sessionTimeout > 0 {
		c.SessionTimeout = sessionTimeout
	}
}

func (c *Config) ApplyMediaTimeoutDefaults(initialTimeout, mediaTimeout time.Duration) {
	if c.MediaTimeoutInitial <= 0 && initialTimeout > 0 {
		c.MediaTimeoutInitial = initialTimeout
	}
	if c.MediaTimeout <= 0 && mediaTimeout > 0 {
		c.MediaTimeout = mediaTimeout
	}
}

func (c *Config) ApplyInboundAnswerDefaults(
	mode InboundAnswerMode,
	minRingDuration time.Duration,
	maxRingDuration time.Duration,
	ackTimeout time.Duration,
) {
	if c.InboundAnswerMode == "" && mode != "" {
		c.InboundAnswerMode = mode
	}
	if c.InboundMinRingDuration <= 0 && minRingDuration > 0 {
		c.InboundMinRingDuration = minRingDuration
	}
	if c.InboundMaxRingDuration <= 0 && maxRingDuration > 0 {
		c.InboundMaxRingDuration = maxRingDuration
	}
	if c.InboundACKTimeout <= 0 && ackTimeout > 0 {
		c.InboundACKTimeout = ackTimeout
	}
}

func (c *Config) EffectiveRegisterTimeout() time.Duration {
	if c != nil && c.RegisterTimeout > 0 {
		return c.RegisterTimeout
	}
	return defaultRegisterTimeout
}

type InboundAnswerMode string

const (
	InboundAnswerModeImmediate            InboundAnswerMode = "answer_immediately"
	InboundAnswerModeAfterMinRingDuration InboundAnswerMode = "answer_after_min_ring_ms"
)

func (m InboundAnswerMode) IsValid() bool {
	switch m {
	case "", InboundAnswerModeImmediate, InboundAnswerModeAfterMinRingDuration:
		return true
	default:
		return false
	}
}

type InboundAnswerPolicy struct {
	Mode            InboundAnswerMode
	MinRingDuration time.Duration
	ACKTimeout      time.Duration
}

func DefaultInboundAnswerPolicy() InboundAnswerPolicy {
	return InboundAnswerPolicy{
		Mode:       InboundAnswerModeImmediate,
		ACKTimeout: defaultInboundACKTimeout,
	}
}

func (c *Config) EffectiveInboundAnswerPolicy(defaultACKTimeout time.Duration) InboundAnswerPolicy {
	policy := DefaultInboundAnswerPolicy()
	if defaultACKTimeout > 0 {
		policy.ACKTimeout = defaultACKTimeout
	}
	if c == nil {
		return policy
	}
	if c.InboundAnswerMode != "" {
		policy.Mode = c.InboundAnswerMode
	}
	if c.InboundMinRingDuration > 0 {
		policy.MinRingDuration = c.InboundMinRingDuration
	}
	if c.InboundACKTimeout > 0 {
		policy.ACKTimeout = c.InboundACKTimeout
	}
	return policy
}

func (c *Config) ValidateRTP() error {
	if c.Server == "" {
		return fmt.Errorf("%w: sip_server is required", ErrInvalidConfig)
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("%w: sip_port must be between 1 and 65535", ErrInvalidConfig)
	}
	if c.RTPPortRangeStart <= 0 || c.RTPPortRangeEnd <= 0 {
		return fmt.Errorf("%w: rtp_port_range must be specified", ErrInvalidConfig)
	}
	if c.RTPPortRangeStart > c.RTPPortRangeEnd {
		return fmt.Errorf("%w: rtp_port_range_start must be less than or equal to rtp_port_range_end", ErrInvalidConfig)
	}
	if c.RTPPortRangeStart < 1024 {
		return fmt.Errorf("%w: rtp_port_range_start must be >= 1024 (non-privileged port)", ErrInvalidConfig)
	}
	if !c.Transport.IsValid() && c.Transport != "" {
		return fmt.Errorf("%w: invalid transport: %s", ErrInvalidConfig, c.Transport)
	}
	if !c.InboundAnswerMode.IsValid() {
		return fmt.Errorf("%w: invalid inbound_answer_mode: %s", ErrInvalidConfig, c.InboundAnswerMode)
	}
	if c.InboundAnswerMode == InboundAnswerModeAfterMinRingDuration && c.InboundMinRingDuration <= 0 {
		return fmt.Errorf("%w: min_ring_duration is required for answer_after_min_ring_ms", ErrInvalidConfig)
	}
	return nil
}

func (c *Config) GetTransport() Transport {
	if c.Transport == "" {
		return TransportUDP
	}
	return c.Transport
}

func (c *Config) GetSIPURI() string {
	domain := c.Domain
	if domain == "" {
		domain = c.Server
	}
	return fmt.Sprintf("sip:%s@%s:%d", c.Username, domain, c.Port)
}

func (c *Config) GetListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Server, c.Port)
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

type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

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

func (t InboundSetupTimings) LatencyMetrics() map[string]int64 {
	metrics := make(map[string]int64)
	addMetric := func(name string, start, end time.Time) {
		if start.IsZero() || end.IsZero() {
			return
		}
		metrics[name] = end.Sub(start).Milliseconds()
	}
	addMetric("invite_to_100_ms", t.InviteReceivedAt, t.TryingSentAt)
	addMetric("invite_to_180_ms", t.InviteReceivedAt, t.RingingSentAt)
	addMetric("180_to_200_ms", t.RingingSentAt, t.AnsweredAt)
	addMetric("200_to_ack_ms", t.AnsweredAt, t.ACKConfirmedAt)
	addMetric("answer_to_first_rtp_ms", t.AnsweredAt, t.FirstRTPReceivedAt)
	addMetric("assistant_audio_ready_to_answer_ms", t.FirstAssistantAudioReadyAt, t.AnsweredAt)
	addMetric("answer_to_first_assistant_audio_sent_ms", t.AnsweredAt, t.FirstAssistantAudioSentAt)
	return metrics
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
	// BridgeCallTimeout is the maximum time to wait for the transfer target to answer.
	BridgeCallTimeout = 30 * time.Second

	// BridgeSafetyTimeout tears down the bridge if neither side hangs up.
	BridgeSafetyTimeout = 5 * time.Minute

	// MetadataBridgeTransferTarget is the session metadata key set by the streamer
	// when a TRANSFER_CONVERSATION directive is received. The engine reads this
	// after Talk() returns to orchestrate the bridge.
	MetadataBridgeTransferTarget = "bridge_transfer_target"

	// MetadataBridgeTransferStatus is set by executeBridgeTransfer to indicate
	// the outcome. Values: "completed" or "failed". Read by media.go to emit
	// the correct transfer event.
	MetadataBridgeTransferStatus = "bridge_transfer_status"

	// MetadataBridgeTransferDuration holds the bridge duration as a string
	// (time.Duration.String()). Set after BridgeTransfer returns.
	MetadataBridgeTransferDuration = "bridge_transfer_duration"

	// MetadataBridgeTransferOutboundCallID holds the SIP Call-ID of the
	// outbound (B-leg) call created for the transfer.
	MetadataBridgeTransferOutboundCallID = "bridge_transfer_outbound_call_id"

	// MetadataDisconnectReason holds the normalized terminal disconnect reason.
	MetadataDisconnectReason = "disconnect_reason"

	// MetadataDisconnectRawReason holds the raw provider Reason header.
	MetadataDisconnectRawReason = "disconnect_raw_reason"

	// PostTransferActionEndCall ends the inbound caller's session when the
	// operator (transfer target) hangs up.
	PostTransferActionEndCall = "end_call"

	// PostTransferActionResumeAI hands the caller back to the AI when the
	// operator (transfer target) hangs up.
	PostTransferActionResumeAI = "resume_ai"
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

// ParseConfigFromVault extracts provider-owned SIP settings from vault.
func ParseConfigFromVault(vaultCredential *protos.VaultCredential) (*Config, error) {
	if vaultCredential == nil || vaultCredential.GetValue() == nil {
		return nil, fmt.Errorf("vault credential is required")
	}

	credMap := vaultCredential.GetValue().AsMap()
	cfg := &Config{}

	if sipURI, ok := stringValue(credMap, "sip_uri"); ok {
		cfg.Server, cfg.Port = parseHostPort(sipURI, cfg.Port)
	}

	if host, ok := stringValue(credMap, "host"); ok {
		cfg.Server, cfg.Port = parseHostPort(host, cfg.Port)
	}
	if host, ok := stringValue(credMap, "host_port"); ok {
		cfg.Server, cfg.Port = parseHostPort(host, cfg.Port)
	}
	if server, ok := stringValue(credMap, "sip_server"); ok {
		cfg.Server = server
	}
	if cfg.Port <= 0 {
		cfg.Port = parsePortValue(credMap["sip_port"])
	}
	if username, ok := stringValue(credMap, "user"); ok {
		cfg.Username = username
	}
	if username, ok := stringValue(credMap, "sip_username"); ok {
		cfg.Username = username
	}
	if password, ok := stringValue(credMap, "password"); ok {
		cfg.Password = password
	}
	if password, ok := stringValue(credMap, "sip_password"); ok {
		cfg.Password = password
	}
	if realm, ok := stringValue(credMap, "sip_realm"); ok {
		cfg.Realm = realm
	}
	if domain, ok := stringValue(credMap, "sip_domain"); ok {
		cfg.Domain = domain
	}
	if callerID, ok := stringValue(credMap, "sip_caller_id"); ok {
		cfg.CallerID = callerID
	}
	if headers := parseHeadersValue(credMap["headers"]); len(headers) > 0 {
		cfg.CustomHeaders = headers
	}
	if headers := parseHeadersValue(credMap["sip_headers"]); len(headers) > 0 {
		cfg.CustomHeaders = headers
	}

	return cfg, nil
}

func stringValue(values map[string]interface{}, key string) (string, bool) {
	value, ok := values[key].(string)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

func parseHostPort(value string, currentPort int) (string, int) {
	raw := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(value), "sips:"), "sip:")
	host, portStr, err := net.SplitHostPort(raw)
	if err != nil {
		return raw, currentPort
	}
	if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port <= 65535 {
		return host, port
	}
	return host, currentPort
}

func parsePortValue(v any) int {
	switch p := v.(type) {
	case float64:
		if int(p) > 0 && int(p) <= 65535 {
			return int(p)
		}
	case string:
		if port, err := strconv.Atoi(p); err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	return 0
}

func parseHeadersValue(value any) map[string]string {
	switch headers := value.(type) {
	case map[string]interface{}:
		parsed := make(map[string]string, len(headers))
		for name, value := range headers {
			if stringValue, ok := value.(string); ok {
				parsed[name] = stringValue
			}
		}
		if len(parsed) > 0 {
			return parsed
		}
	case string:
		if strings.TrimSpace(headers) == "" {
			return nil
		}
		parsed := make(map[string]string)
		if err := json.Unmarshal([]byte(headers), &parsed); err == nil && len(parsed) > 0 {
			return parsed
		}
	}
	return nil
}

// ExtractDIDFromURI extracts the user part from a SIP URI as a phone number (DID).
// Strips URI parameters (e.g. ;user=phone) that some providers append.
func ExtractDIDFromURI(uri string) string {
	raw := strings.TrimPrefix(strings.TrimPrefix(uri, "sip:"), "sips:")
	parts := strings.SplitN(raw, "@", 2)
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	user := parts[0]
	// Strip URI parameters (e.g. "+15551234567;user=phone" → "+15551234567")
	if idx := strings.IndexByte(user, ';'); idx >= 0 {
		user = user[:idx]
	}
	// Skip credential pairs (assistantID:apiKey)
	if strings.Contains(user, ":") {
		return ""
	}
	// Normalize to E.164: add "+" prefix for phone numbers
	if len(user) > 5 && user[0] != '+' {
		user = "+" + user
	}

	return user
}
