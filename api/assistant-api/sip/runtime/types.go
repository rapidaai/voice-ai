// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emiago/sipgo/sip"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
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
	ErrInvalidCallRoute           = errors.New("invalid SIP call route")
	ErrMiddlewareChainIncomplete  = errors.New("SIP middleware chain incomplete")
	ErrPhoneDeploymentRequired    = errors.New("SIP phone deployment is required")
	ErrVaultResolverRequired      = errors.New("SIP vault resolver is required")
	ErrCredentialIDRequired       = errors.New("SIP credential ID is required")
	ErrVaultCredentialResolution  = errors.New("SIP vault credential resolution failed")
	ErrVaultConfigInvalid         = errors.New("SIP vault configuration is invalid")
	ErrSIPCallCapacityExceeded    = errors.New("SIP call capacity exceeded")
	ErrSIPCallRateExceeded        = errors.New("SIP call setup rate exceeded")
)

// ServerState represents the state of the SIP server.
type ServerState int32

const (
	ServerStateCreated ServerState = iota
	ServerStateRunning
	ServerStateStopped

	InboundRejectedInviteTTL  = time.Minute
	MaxInboundRejectedInvites = 1024
)

// CallAddress contains exact SIP parties, resolved phone values, and non-credential headers.
// Header names are lowercase and repeated values preserve arrival order.
type CallAddress struct {
	From    string
	To      string
	FromURI string
	ToURI   string
	Headers map[string]string
}

// CallRoute identifies the route encoded in an inbound SIP Request-URI.
type CallRoute interface {
	Kind() string
}

// AgentCallRoute identifies an assistant-targeted SIP route.
type AgentCallRoute struct {
	AssistantID uint64
}

func (AgentCallRoute) Kind() string {
	return "agent"
}

// DIDCallRoute identifies a phone-number-targeted SIP route.
type DIDCallRoute struct {
	DID string
}

func (DIDCallRoute) Kind() string {
	return "did"
}

// NewCallAddress reads an inbound party snapshot and non-credential headers.
func NewCallAddress(request *sip.Request) CallAddress {
	if request == nil {
		return CallAddress{}
	}

	address := CallAddress{Headers: make(map[string]string)}
	if from := request.From(); from != nil {
		address.FromURI = from.Address.String()
		if validator.Phone(from.Address.User) {
			address.From = from.Address.User
		}
	}
	if to := request.To(); to != nil {
		address.ToURI = to.Address.String()
	}

	for _, header := range request.Headers() {
		if header == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(header.Name()))
		if name == "" || name == "authorization" || name == "proxy-authorization" {
			continue
		}
		value := strings.TrimSpace(header.Value())
		if previous := address.Headers[name]; previous != "" {
			address.Headers[name] = previous + "," + value
		} else {
			address.Headers[name] = value
		}
	}

	return address
}

// SIPRequestContext contains information about an incoming SIP request.
// Used by the middleware chain to resolve config for every SIP request.
//
// Middleware enriches this context as it flows through the chain:
//
//	RouteMiddleware → resolves assistant route, sets Auth and Assistant
//	VaultMiddleware → fetches SIP config from vault, sets VaultCredential
type SIPRequestContext struct {
	Method      string // SIP method (INVITE, REGISTER, BYE, etc.)
	CallID      string
	RequestURI  string
	CallAddress CallAddress
	SDPInfo     *SDPMediaInfo

	Auth            *types.Authentication
	Assistant       *internal_assistant_entity.Assistant
	VaultCredential *protos.VaultCredential
	Config          *Config
}

// ResolveRoute resolves the route encoded in the SIP Request-URI.
func (c *SIPRequestContext) ResolveRoute() (CallRoute, error) {
	if c == nil {
		return nil, ErrInvalidCallRoute
	}

	requestURI := strings.TrimSpace(c.RequestURI)
	requestURI = strings.TrimPrefix(strings.TrimPrefix(requestURI, "sip:"), "sips:")
	routeUserWithParameters, _, _ := strings.Cut(requestURI, "@")
	routeUser, _, _ := strings.Cut(strings.TrimSpace(routeUserWithParameters), ";")
	routeUser = strings.TrimSpace(routeUser)
	if routeUser == "" {
		return nil, ErrInvalidCallRoute
	}

	if strings.HasPrefix(routeUser, "agent-") {
		routeValue := strings.TrimSpace(routeUser[len("agent-"):])
		if routeValue == "" || strings.Contains(routeValue, ":") {
			return nil, ErrInvalidCallRoute
		}
		assistantID, err := strconv.ParseUint(routeValue, 10, 64)
		if err != nil || assistantID == 0 {
			return nil, ErrInvalidCallRoute
		}
		return AgentCallRoute{AssistantID: assistantID}, nil
	}
	if strings.HasPrefix(routeUser, "did-") {
		routeUser = strings.TrimSpace(routeUser[len("did-"):])
	}
	if !validator.Phone(routeUser) {
		return nil, ErrInvalidCallRoute
	}
	return DIDCallRoute{DID: routeUser}, nil
}

// Middleware processes a SIP request context and mutates it in place.
// Returning nil continues to the next middleware by index. Returning an error
// stops execution.
//
// Example chain for INVITE:
//
//	RouteMiddleware → VaultMiddleware
type Middleware func(ctx *SIPRequestContext) error

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
	if c == nil {
		return
	}
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
	if c == nil {
		return
	}
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
	if c == nil {
		return
	}
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
	if c == nil {
		return
	}
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

type RTPStats struct {
	PacketsSent                    uint64        `json:"packets_sent"`
	PacketsReceived                uint64        `json:"packets_received"`
	PacketsDelivered               uint64        `json:"packets_delivered"`
	BytesSent                      uint64        `json:"bytes_sent"`
	BytesReceived                  uint64        `json:"bytes_received"`
	PacketsLost                    uint64        `json:"packets_lost"`
	PacketsDropped                 uint64        `json:"packets_dropped"`
	AudioInputDropped              uint64        `json:"audio_input_dropped"`
	NetworkPacketsLost             uint64        `json:"network_packets_lost"`
	LateOrDuplicatePackets         uint64        `json:"late_or_duplicate_packets"`
	InvalidPackets                 uint64        `json:"invalid_packets"`
	JitterBufferResyncDropped      uint64        `json:"jitter_buffer_resync_dropped"`
	RTPIngressQueueDropped         uint64        `json:"rtp_ingress_queue_dropped"`
	SilenceSuppressionFrames       uint64        `json:"silence_suppression_frames"`
	LastRTPReceivedAt              time.Time     `json:"last_rtp_received_at,omitempty"`
	LastAudioDeliveredAt           time.Time     `json:"last_audio_delivered_at,omitempty"`
	InboundQuality                 string        `json:"inbound_quality"`
	InboundQualityScore            uint8         `json:"inbound_quality_score"`
	InboundQualityWindow           time.Duration `json:"inbound_quality_window"`
	InboundWindowPacketsReceived   uint64        `json:"inbound_window_packets_received"`
	InboundWindowPacketsDelivered  uint64        `json:"inbound_window_packets_delivered"`
	InboundWindowPacketsLost       uint64        `json:"inbound_window_packets_lost"`
	InboundWindowPacketsDropped    uint64        `json:"inbound_window_packets_dropped"`
	InboundWindowAudioInputDropped uint64        `json:"inbound_window_audio_input_dropped"`
	InboundLossRate                float64       `json:"inbound_loss_rate"`
	InboundDropRate                float64       `json:"inbound_drop_rate"`
	InboundDeliveryRate            float64       `json:"inbound_delivery_rate"`
	RTCPEnabled                    bool          `json:"rtcp_enabled"`
	LocalRTCPPort                  int           `json:"local_rtcp_port"`
	RemoteRTCPPort                 int           `json:"remote_rtcp_port"`
	RTCPPacketsSent                uint64        `json:"rtcp_packets_sent"`
	RTCPPacketsReceived            uint64        `json:"rtcp_packets_received"`
	RTCPReportsSent                uint64        `json:"rtcp_reports_sent"`
	RTCPSenderReportsSent          uint64        `json:"rtcp_sender_reports_sent"`
	RTCPReceiverReportsSent        uint64        `json:"rtcp_receiver_reports_sent"`
	RTCPSenderReportsReceived      uint64        `json:"rtcp_sender_reports_received"`
	RTCPReceiverReportsReceived    uint64        `json:"rtcp_receiver_reports_received"`
	RTCPFractionLost               uint8         `json:"rtcp_fraction_lost"`
	RTCPPacketsLost                uint32        `json:"rtcp_packets_lost"`
	RTCPJitter                     uint32        `json:"rtcp_jitter"`
	RTCPRemoteFractionLost         uint8         `json:"rtcp_remote_fraction_lost"`
	RTCPRemotePacketsLost          uint32        `json:"rtcp_remote_packets_lost"`
	RTCPRemoteJitter               uint32        `json:"rtcp_remote_jitter"`
	RTCPRoundTripTime              time.Duration `json:"rtcp_round_trip_time"`
	Jitter                         time.Duration `json:"jitter"`
}

// ParseConfigFromVault extracts provider-owned SIP settings from vault.
func ParseConfigFromVault(vaultCredential *protos.VaultCredential) (*Config, error) {
	if vaultCredential == nil || vaultCredential.GetValue() == nil {
		return nil, fmt.Errorf("vault credential is required")
	}

	options := utils.Option(vaultCredential.GetValue().AsMap())
	cfg := &Config{}

	for _, key := range []string{"sip_uri", "host", "host_port"} {
		value, err := options.GetString(key)
		if err != nil || strings.TrimSpace(value) == "" {
			continue
		}

		raw := strings.TrimSpace(value)
		if !strings.HasPrefix(raw, "sip:") && !strings.HasPrefix(raw, "sips:") {
			raw = "sip:" + raw
		}
		var uri sip.Uri
		if err := sip.ParseUri(raw, &uri); err == nil && uri.Host != "" {
			cfg.Server = uri.Host
			if uri.Port > 0 && uri.Port <= 65535 {
				cfg.Port = uri.Port
			}
		}
	}
	if server, err := options.GetString("sip_server"); err == nil && strings.TrimSpace(server) != "" {
		cfg.Server = strings.TrimSpace(server)
	}
	if cfg.Port <= 0 {
		if port, err := options.GetUint32("sip_port"); err == nil && port > 0 && port <= 65535 {
			cfg.Port = int(port)
		}
	}
	if username, err := options.GetString("user"); err == nil && strings.TrimSpace(username) != "" {
		cfg.Username = strings.TrimSpace(username)
	}
	if username, err := options.GetString("sip_username"); err == nil && strings.TrimSpace(username) != "" {
		cfg.Username = strings.TrimSpace(username)
	}
	if password, err := options.GetString("password"); err == nil && strings.TrimSpace(password) != "" {
		cfg.Password = strings.TrimSpace(password)
	}
	if password, err := options.GetString("sip_password"); err == nil && strings.TrimSpace(password) != "" {
		cfg.Password = strings.TrimSpace(password)
	}
	if realm, err := options.GetString("sip_realm"); err == nil && strings.TrimSpace(realm) != "" {
		cfg.Realm = strings.TrimSpace(realm)
	}
	if domain, err := options.GetString("sip_domain"); err == nil && strings.TrimSpace(domain) != "" {
		cfg.Domain = strings.TrimSpace(domain)
	}
	if callerID, err := options.GetString("sip_caller_id"); err == nil && strings.TrimSpace(callerID) != "" {
		cfg.CallerID = strings.TrimSpace(callerID)
	}
	if headers, err := options.GetStringMap("headers"); err == nil && len(headers) > 0 {
		cfg.CustomHeaders = headers
	}
	if headers, err := options.GetStringMap("sip_headers"); err == nil && len(headers) > 0 {
		cfg.CustomHeaders = headers
	}

	return cfg, nil
}
