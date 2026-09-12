// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emiago/sipgo/sip"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	sip_config "github.com/rapidaai/api/assistant-api/sip/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// ServerState represents the state of the SIP server.
type ServerState int32

// CallLifecycle is the single owner of call-state transitions inside the SIP runtime.
// It validates transitions and emits structured transition logs.
type CallLifecycle struct {
	mu     sync.Mutex
	callID string
	state  CallState
	logger commons.Logger
}

type CallTerminationResult string

type CallTermination struct {
	Result CallTerminationResult
	Reason string
}

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
	Config          *sip_config.Config
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

// RTPAddress identifies an RTP or RTCP network endpoint.
type RTPAddress struct {
	// IP is the endpoint IP address.
	IP string

	// Port is the endpoint UDP port.
	Port int
}

func (address RTPAddress) String() string {
	if address.IP == "" && address.Port == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", address.IP, address.Port)
}

func (address RTPAddress) Validate() bool {
	return !utils.IsEmpty(address.IP) && validator.Between(address.Port, 1, rtpMaxPort)
}

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

// RTPConfig holds the socket, codec, timeout, and packet timing settings for an RTP handler.
type RTPConfig struct {
	// LocalAddress is the local RTP endpoint used to bind the RTP socket. A zero port chooses from the configured range.
	LocalAddress RTPAddress

	// PayloadType is the RTP payload type used to select the initial codec.
	PayloadType uint8

	// ClockRate is the RTP clock rate for the initial codec. Zero defaults to G.711 8 kHz.
	ClockRate uint32

	// RTPPortRangeStart is the first candidate RTP port when LocalAddress.Port is zero.
	RTPPortRangeStart int

	// RTPPortRangeEnd is the last candidate RTP port when LocalAddress.Port is zero.
	RTPPortRangeEnd int

	// SymmetricRTP sends RTP back to the source address observed on inbound RTP packets.
	SymmetricRTP bool

	// MediaTimeoutInitial is the maximum wait for the first inbound RTP packet.
	MediaTimeoutInitial time.Duration

	// MediaTimeout is the maximum gap allowed after inbound RTP has started.
	MediaTimeout time.Duration

	// PacketizationTime is the expected inbound packet duration used by the jitter buffer.
	PacketizationTime time.Duration

	// portStats records RTP port allocation counters owned by the server.
	portStats *RTPPortStats
}

// Validate validates the RTP handler configuration and fills runtime defaults.
func (c *RTPConfig) Validate() error {
	if !validator.NonNil(c) {
		return errRTPConfigRequired
	}
	if utils.IsEmpty(c.LocalAddress.IP) {
		return errRTPLocalAddressIPRequired
	}
	if !validator.Between(c.LocalAddress.Port, 0, rtpMaxPort) {
		return fmt.Errorf(rtpErrorIntFormat, errRTPInvalidLocalAddressPort, c.LocalAddress.Port)
	}
	if c.LocalAddress.Port == 0 {
		if !validator.Between(c.RTPPortRangeStart, 1, rtpMaxPort) ||
			!validator.Between(c.RTPPortRangeEnd, 1, rtpMaxPort) {
			if c.RTPPortRangeStart <= 0 || c.RTPPortRangeEnd <= 0 {
				return errRTPPortRangeRequired
			}
			if c.RTPPortRangeStart > c.RTPPortRangeEnd {
				return errRTPPortRangeInvalidOrder
			}
			return fmt.Errorf(rtpErrorMaxPortFormat, errRTPPortRangeValue, rtpMaxPort)
		}
		if c.RTPPortRangeStart > c.RTPPortRangeEnd {
			return errRTPPortRangeInvalidOrder
		}
	}
	if c.ClockRate == 0 {
		c.ClockRate = rtpDefaultClockRate
	}
	if c.MediaTimeoutInitial <= 0 {
		c.MediaTimeoutInitial = rtpMediaTimeoutInitial
	}
	if c.MediaTimeout <= 0 {
		c.MediaTimeout = rtpMediaTimeout
	}
	if c.PacketizationTime <= 0 {
		c.PacketizationTime = rtpDefaultPacketizationTime
	}
	if c.PacketizationTime < rtpMinPacketizationTime ||
		c.PacketizationTime > rtpMaxPacketizationTime ||
		c.PacketizationTime%time.Millisecond != 0 {
		return fmt.Errorf(rtpErrorIntFormat, errRTPInvalidPacketizationTime, c.PacketizationTime.Milliseconds())
	}
	return nil
}

type CallState string

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

type InboundSetupPhase string

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

type DisconnectMetadata struct {
	Reason             string
	Text               string
	Raw                string
	ProviderStatusCode int
}

type RTPStats struct {
	PacketsSent                 uint64        `json:"packets_sent"`
	PacketsReceived             uint64        `json:"packets_received"`
	PacketsDelivered            uint64        `json:"packets_delivered"`
	BytesSent                   uint64        `json:"bytes_sent"`
	BytesReceived               uint64        `json:"bytes_received"`
	PacketsLost                 uint64        `json:"packets_lost"`
	PacketsDropped              uint64        `json:"packets_dropped"`
	LateOrDuplicatePackets      uint64        `json:"late_or_duplicate_packets"`
	InvalidPackets              uint64        `json:"invalid_packets"`
	JitterBufferResyncDropped   uint64        `json:"jitter_buffer_resync_dropped"`
	SilenceSuppressionFrames    uint64        `json:"silence_suppression_frames"`
	LastRTPReceivedAt           time.Time     `json:"last_rtp_received_at,omitempty"`
	LastAudioDeliveredAt        time.Time     `json:"last_audio_delivered_at,omitempty"`
	RTCPEnabled                 bool          `json:"rtcp_enabled"`
	LocalRTCPPort               int           `json:"local_rtcp_port"`
	RemoteRTCPPort              int           `json:"remote_rtcp_port"`
	RTCPPacketsSent             uint64        `json:"rtcp_packets_sent"`
	RTCPPacketsReceived         uint64        `json:"rtcp_packets_received"`
	RTCPReportsSent             uint64        `json:"rtcp_reports_sent"`
	RTCPSenderReportsSent       uint64        `json:"rtcp_sender_reports_sent"`
	RTCPReceiverReportsSent     uint64        `json:"rtcp_receiver_reports_sent"`
	RTCPSenderReportsReceived   uint64        `json:"rtcp_sender_reports_received"`
	RTCPReceiverReportsReceived uint64        `json:"rtcp_receiver_reports_received"`
	RTCPFractionLost            uint8         `json:"rtcp_fraction_lost"`
	RTCPPacketsLost             uint32        `json:"rtcp_packets_lost"`
	RTCPJitter                  uint32        `json:"rtcp_jitter"`
	RTCPRemoteFractionLost      uint8         `json:"rtcp_remote_fraction_lost"`
	RTCPRemotePacketsLost       uint32        `json:"rtcp_remote_packets_lost"`
	RTCPRemoteJitter            uint32        `json:"rtcp_remote_jitter"`
	RTCPRoundTripTime           time.Duration `json:"rtcp_round_trip_time"`
	Jitter                      time.Duration `json:"jitter"`
}
