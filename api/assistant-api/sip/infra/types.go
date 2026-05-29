// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

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

// SIP-specific errors
var (
	ErrInvalidConfig     = errors.New("invalid SIP configuration")
	ErrSessionNotFound   = errors.New("SIP session not found")
	ErrSessionClosed     = errors.New("SIP session is closed")
	ErrRTPNotInitialized = errors.New("RTP handler not initialized")
	ErrSDPParseFailed    = errors.New("failed to parse SDP")
	ErrCodecNotSupported = errors.New("codec not supported")
	ErrConnectionFailed  = errors.New("SIP connection failed")
)

// SIPError wraps SIP-specific errors with context
type SIPError struct {
	Op      string // Operation that failed
	CallID  string // SIP Call-ID if available
	Code    int    // SIP response code if applicable
	Message string // Human-readable message
	Err     error  // Underlying error
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

// NewSIPError creates a new SIP error
func NewSIPError(op, callID, message string, err error) *SIPError {
	return &SIPError{Op: op, CallID: callID, Message: message, Err: err}
}

// Transport represents the transport protocol for SIP
type Transport string

const (
	TransportUDP Transport = "udp"
	TransportTCP Transport = "tcp"
	TransportTLS Transport = "tls"
)

// String returns the string representation of the transport
func (t Transport) String() string {
	return string(t)
}

// IsValid checks if the transport is valid
func (t Transport) IsValid() bool {
	switch t {
	case TransportUDP, TransportTCP, TransportTLS:
		return true
	default:
		return false
	}
}

// Config holds the full SIP configuration, combining:
//   - Provider credentials (from vault/Twilio): Server, Username, Password, Realm, Domain
//   - Platform operational settings (from app config): Port, Transport, RTP range, timeouts
type Config struct {
	// Provider credentials — from vault (Twilio, SIP trunk provider, etc.)
	Server   string `json:"sip_server" mapstructure:"sip_server"`
	Username string `json:"sip_username" mapstructure:"sip_username"`
	Password string `json:"sip_password" mapstructure:"sip_password"`
	Realm    string `json:"sip_realm" mapstructure:"sip_realm"`
	Domain   string `json:"sip_domain,omitempty" mapstructure:"sip_domain"`

	// CallerID overrides the From header user in outbound calls.
	// For cloud providers (Twilio, Vonage, Telnyx), this should be the E.164 DID number.
	// For self-hosted PBX (Asterisk, FreeSWITCH), leave empty — defaults to Username
	// so the From URI matches the auth endpoint (required for PJSIP endpoint resolution).
	CallerID string `json:"sip_caller_id,omitempty" mapstructure:"sip_caller_id"`

	// CustomHeaders are user-defined SIP headers added to outbound INVITE requests.
	// Stored in vault as comma-separated key=value pairs (e.g. "X-Custom=foo,X-Other=bar").
	CustomHeaders map[string]string `json:"sip_headers,omitempty" mapstructure:"sip_headers"`

	// Platform operational settings — from app config (not from vault)
	Port              int       `json:"sip_port" mapstructure:"sip_port"`
	Transport         Transport `json:"sip_transport" mapstructure:"sip_transport"`
	RTPPortRangeStart int       `json:"rtp_port_range_start" mapstructure:"rtp_port_range_start"`
	RTPPortRangeEnd   int       `json:"rtp_port_range_end" mapstructure:"rtp_port_range_end"`
	SRTPEnabled       bool      `json:"srtp_enabled" mapstructure:"srtp_enabled"`

	// Timeout settings — from app config
	RegisterTimeout  time.Duration `json:"register_timeout,omitempty" mapstructure:"register_timeout"`
	InviteTimeout    time.Duration `json:"invite_timeout,omitempty" mapstructure:"invite_timeout"`
	SessionTimeout   time.Duration `json:"session_timeout,omitempty" mapstructure:"session_timeout"`
	KeepAliveEnabled bool          `json:"keepalive_enabled,omitempty" mapstructure:"keepalive_enabled"`
}

// Validate validates the full SIP configuration (for outbound calls / registration)
func (c *Config) Validate() error {
	if err := c.ValidateRTP(); err != nil {
		return err
	}
	if c.Username == "" {
		return fmt.Errorf("%w: sip_username is required", ErrInvalidConfig)
	}
	if c.Password == "" {
		return fmt.Errorf("%w: sip_password is required", ErrInvalidConfig)
	}
	return nil
}

// ApplyOperationalDefaults fills in unset operational fields (port, transport, RTP range)
// from the platform's app-level SIP config. These are infrastructure settings, not provider credentials.
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

// ValidateRTP validates the minimum config needed for inbound calls (server + RTP ports)
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
	if c.RTPPortRangeStart >= c.RTPPortRangeEnd {
		return fmt.Errorf("%w: rtp_port_range_start must be less than rtp_port_range_end", ErrInvalidConfig)
	}
	if c.RTPPortRangeStart < 1024 {
		return fmt.Errorf("%w: rtp_port_range_start must be >= 1024 (non-privileged port)", ErrInvalidConfig)
	}
	if !c.Transport.IsValid() && c.Transport != "" {
		return fmt.Errorf("%w: invalid transport: %s", ErrInvalidConfig, c.Transport)
	}
	return nil
}

// GetTransport returns the transport, defaulting to UDP if not set
func (c *Config) GetTransport() Transport {
	if c.Transport == "" {
		return TransportUDP
	}
	return c.Transport
}

// GetSIPURI returns the full SIP URI for the server
func (c *Config) GetSIPURI() string {
	domain := c.Domain
	if domain == "" {
		domain = c.Server
	}
	return fmt.Sprintf("sip:%s@%s:%d", c.Username, domain, c.Port)
}

// GetListenAddr returns the listen address string
func (c *Config) GetListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Server, c.Port)
}

// CallState represents the state of a SIP call
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
)

// String returns the string representation of the call state
func (s CallState) String() string {
	return string(s)
}

// IsTerminal returns true if the call state is terminal (ended or failed)
func (s CallState) IsTerminal() bool {
	return s == CallStateEnded || s == CallStateFailed
}

// IsActive returns true if the call is in an active state
func (s CallState) IsActive() bool {
	return s == CallStateConnected || s == CallStateRinging || s == CallStateOnHold || s == CallStateTransferring || s == CallStateBridgeConnected
}

// CallDirection represents the direction of the call
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// SessionInfo contains information about an active SIP session
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

// GetDuration calculates the call duration
func (s *SessionInfo) GetDuration() time.Duration {
	if s.EndTime != nil && s.ConnectedTime != nil {
		return s.EndTime.Sub(*s.ConnectedTime)
	}
	if s.ConnectedTime != nil {
		return time.Since(*s.ConnectedTime)
	}
	return 0
}

// EventType represents the type of SIP event
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

// =============================================================================
// Bridge Transfer Constants
// =============================================================================

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

	// MetadataCallFromURI stores the raw SIP From URI for lifecycle events.
	MetadataCallFromURI = "call_from_uri"

	// MetadataCallToURI stores the raw SIP To URI for lifecycle events.
	MetadataCallToURI = "call_to_uri"

	// PostTransferActionEndCall ends the inbound caller's session when the
	// operator (transfer target) hangs up.
	PostTransferActionEndCall = "end_call"

	// PostTransferActionResumeAI hands the caller back to the AI when the
	// operator (transfer target) hangs up.
	PostTransferActionResumeAI = "resume_ai"
)

// Event represents events from SIP stack
type Event struct {
	Type      EventType              `json:"type"`
	CallID    string                 `json:"call_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewEvent creates a new SIP event
func NewEvent(eventType EventType, callID string, data map[string]interface{}) Event {
	return Event{
		Type:      eventType,
		CallID:    callID,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// DTMFEvent represents DTMF input
type DTMFEvent struct {
	Digit    string `json:"digit"`
	Duration int    `json:"duration_ms"`
}

// RTPStats contains RTP statistics
type RTPStats struct {
	PacketsSent     uint64        `json:"packets_sent"`
	PacketsReceived uint64        `json:"packets_received"`
	BytesSent       uint64        `json:"bytes_sent"`
	BytesReceived   uint64        `json:"bytes_received"`
	PacketsLost     uint64        `json:"packets_lost"`
	Jitter          time.Duration `json:"jitter"`
}

// ParseConfigFromVault extracts SIP provider credentials from a vault credential.
// Handles: sip_uri, sip_server, sip_port, sip_username, sip_password, sip_realm,
// sip_domain, sip_caller_id, sip_headers (JSON string).
// Does NOT set operational fields (transport, RTP range) — call ApplyOperationalDefaults after.
func ParseConfigFromVault(vaultCredential *protos.VaultCredential) (*Config, error) {
	if vaultCredential == nil || vaultCredential.GetValue() == nil {
		return nil, fmt.Errorf("vault credential is required")
	}

	credMap := vaultCredential.GetValue().AsMap()
	cfg := &Config{}

	// Parse sip_uri → server + port (e.g. "sip:192.168.1.5:5060")
	if sipURI, ok := credMap["sip_uri"].(string); ok && sipURI != "" {
		raw := strings.TrimPrefix(strings.TrimPrefix(sipURI, "sips:"), "sip:")
		host, portStr, err := net.SplitHostPort(raw)
		if err != nil {
			cfg.Server = raw
		} else {
			cfg.Server = host
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
				cfg.Port = p
			}
		}
	}

	if server, ok := credMap["sip_server"].(string); ok && server != "" {
		cfg.Server = server
	}
	if cfg.Port <= 0 {
		cfg.Port = parsePortValue(credMap["sip_port"])
	}
	if username, ok := credMap["sip_username"].(string); ok {
		cfg.Username = username
	}
	if password, ok := credMap["sip_password"].(string); ok {
		cfg.Password = password
	}
	if realm, ok := credMap["sip_realm"].(string); ok {
		cfg.Realm = realm
	}
	if domain, ok := credMap["sip_domain"].(string); ok {
		cfg.Domain = domain
	}
	if callerID, ok := credMap["sip_caller_id"].(string); ok {
		cfg.CallerID = callerID
	}
	if headersRaw, ok := credMap["sip_headers"].(string); ok && headersRaw != "" {
		parsed := make(map[string]string)
		if err := json.Unmarshal([]byte(headersRaw), &parsed); err == nil {
			cfg.CustomHeaders = parsed
		}
	}

	return cfg, nil
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
