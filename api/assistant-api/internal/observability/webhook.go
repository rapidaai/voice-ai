// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

const WebhookPayloadVersionV1 = "1"

// WebhookCallStatus identifies the lifecycle state of a call.
type WebhookCallStatus string

const (
	WebhookCallStatusPending    WebhookCallStatus = "pending"
	WebhookCallStatusRinging    WebhookCallStatus = "ringing"
	WebhookCallStatusInProgress WebhookCallStatus = "in_progress"
	WebhookCallStatusCompleted  WebhookCallStatus = "completed"
	WebhookCallStatusFailed     WebhookCallStatus = "failed"
	WebhookCallStatusCancelled  WebhookCallStatus = "cancelled"
)

func (s WebhookCallStatus) String() string {
	return string(s)
}

// WebhookCallDirection identifies the direction of a call.
type WebhookCallDirection string

const (
	WebhookCallDirectionInbound  WebhookCallDirection = "inbound"
	WebhookCallDirectionOutbound WebhookCallDirection = "outbound"
)

func (d WebhookCallDirection) String() string {
	return string(d)
}

// WebhookCallDisconnectReason identifies why a call terminated.
type WebhookCallDisconnectReason string

const (
	WebhookCallDisconnectReasonUnknown              WebhookCallDisconnectReason = "unknown"
	WebhookCallDisconnectReasonRemoteHangup         WebhookCallDisconnectReason = "remote_hangup"
	WebhookCallDisconnectReasonAssistantEnded       WebhookCallDisconnectReason = "assistant_ended"
	WebhookCallDisconnectReasonToolEnded            WebhookCallDisconnectReason = "tool_ended"
	WebhookCallDisconnectReasonTransferred          WebhookCallDisconnectReason = "transferred"
	WebhookCallDisconnectReasonIdleTimeout          WebhookCallDisconnectReason = "idle_timeout"
	WebhookCallDisconnectReasonMaxDuration          WebhookCallDisconnectReason = "max_duration"
	WebhookCallDisconnectReasonNoAnswer             WebhookCallDisconnectReason = "no_answer"
	WebhookCallDisconnectReasonBusy                 WebhookCallDisconnectReason = "busy"
	WebhookCallDisconnectReasonRejected             WebhookCallDisconnectReason = "rejected"
	WebhookCallDisconnectReasonCancelled            WebhookCallDisconnectReason = "cancelled"
	WebhookCallDisconnectReasonAuthenticationFailed WebhookCallDisconnectReason = "authentication_failed"
	WebhookCallDisconnectReasonConfigurationFailed  WebhookCallDisconnectReason = "configuration_failed"
	WebhookCallDisconnectReasonProviderFailed       WebhookCallDisconnectReason = "provider_failed"
	WebhookCallDisconnectReasonNetworkFailed        WebhookCallDisconnectReason = "network_failed"
	WebhookCallDisconnectReasonMediaFailed          WebhookCallDisconnectReason = "media_failed"
	WebhookCallDisconnectReasonCapacityExceeded     WebhookCallDisconnectReason = "capacity_exceeded"
	WebhookCallDisconnectReasonInternalError        WebhookCallDisconnectReason = "internal_error"
)

func (r WebhookCallDisconnectReason) String() string {
	return string(r)
}

type V1WebhookPayload interface {
	isV1WebhookPayload()
}

type V1WebhookPayloadBase struct {
	Version string                 `json:"version"`
	Extra   map[string]interface{} `json:"extra"`
}

type CallReceivedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	CallID    string               `json:"call_id"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallRingingWebhookPayload struct {
	V1WebhookPayloadBase

	Provider    string               `json:"provider"`
	CallID      string               `json:"call_id"`
	To          string               `json:"to"`
	From        string               `json:"from"`
	Direction   WebhookCallDirection `json:"direction"`
	ContextID   string               `json:"context_id"`
	Source      string               `json:"source"`
	StatusEvent string               `json:"status_event"`
	Status      WebhookCallStatus    `json:"status"`
}

type CallProviderAnsweredWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	CallID    string               `json:"call_id"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"context_id"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallFailedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider         string                      `json:"provider,omitempty"`
	CallID           string                      `json:"call_id,omitempty"`
	To               string                      `json:"to,omitempty"`
	From             string                      `json:"from,omitempty"`
	Direction        WebhookCallDirection        `json:"direction,omitempty"`
	ContextID        string                      `json:"context_id,omitempty"`
	Stage            string                      `json:"stage,omitempty"`
	Source           string                      `json:"source,omitempty"`
	StatusEvent      string                      `json:"status_event,omitempty"`
	Error            string                      `json:"error,omitempty"`
	DurationMs       string                      `json:"duration_ms,omitempty"`
	Status           WebhookCallStatus           `json:"status"`
	DisconnectReason WebhookCallDisconnectReason `json:"disconnect_reason,omitempty"`
}

type CallStartedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	CallID    string               `json:"call_id"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"context_id"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallEndedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider         string                      `json:"provider"`
	CallID           string                      `json:"call_id"`
	To               string                      `json:"to"`
	From             string                      `json:"from"`
	Direction        WebhookCallDirection        `json:"direction"`
	ContextID        string                      `json:"context_id"`
	DurationMs       string                      `json:"duration_ms"`
	Status           WebhookCallStatus           `json:"status"`
	DisconnectReason WebhookCallDisconnectReason `json:"disconnect_reason,omitempty"`
}

type CallOutboundRequestedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"context_id"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallOutboundDispatchedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider    string               `json:"provider"`
	CallID      string               `json:"call_id"`
	To          string               `json:"to"`
	From        string               `json:"from"`
	Direction   WebhookCallDirection `json:"direction"`
	ContextID   string               `json:"context_id"`
	StatusEvent string               `json:"status_event"`
	Status      WebhookCallStatus    `json:"status"`
}

type ConversationBeginWebhookPayload struct {
	V1WebhookPayloadBase

	Source     string `json:"source"`
	Identifier string `json:"identifier"`
	Status     string `json:"status"`
}

type ConversationResumeWebhookPayload struct {
	V1WebhookPayloadBase

	Source       string `json:"source"`
	Identifier   string `json:"identifier"`
	MessageCount string `json:"message_count"`
	Status       string `json:"status"`
}

type ConversationCompletedWebhookPayload struct {
	V1WebhookPayloadBase

	Source           string                   `json:"source"`
	Reason           string                   `json:"reason"`
	Status           string                   `json:"status"`
	DisconnectReason string                   `json:"disconnect_reason,omitempty"`
	Messages         []map[string]interface{} `json:"messages"`
	Metadata         map[string]interface{}   `json:"metadata"`
	Metrics          []map[string]interface{} `json:"metrics"`
}

type ConversationErrorWebhookPayload struct {
	V1WebhookPayloadBase

	Source           string `json:"source"`
	Reason           string `json:"reason"`
	Message          string `json:"message"`
	Status           string `json:"status"`
	DisconnectReason string `json:"disconnect_reason,omitempty"`
}

type WebRTCConnectedWebhookPayload struct {
	V1WebhookPayloadBase

	SessionID           string `json:"session_id"`
	MediaSessionID      uint64 `json:"media_session_id"`
	ICELatencyMs        int64  `json:"ice_latency_ms"`
	PeerConnectionState string `json:"peer_connection_state"`
}

type WebRTCAudioTrackReceivedWebhookPayload struct {
	V1WebhookPayloadBase

	SessionID      string `json:"session_id"`
	MediaSessionID uint64 `json:"media_session_id"`
	Codec          string `json:"codec"`
}

type WebRTCReconnectingWebhookPayload struct {
	V1WebhookPayloadBase

	Type           string `json:"type"`
	SessionID      string `json:"session_id"`
	MediaSessionID uint64 `json:"media_session_id"`
	Reason         string `json:"reason"`
	RestartAttempt uint64 `json:"restart_attempt"`
	RestartLimit   uint64 `json:"restart_limit"`
}

type WebRTCFailedWebhookPayload struct {
	V1WebhookPayloadBase

	Type                string `json:"type"`
	SessionID           string `json:"session_id"`
	MediaSessionID      uint64 `json:"media_session_id,omitempty"`
	Reason              string `json:"reason"`
	PeerConnectionState string `json:"peer_connection_state,omitempty"`
	Error               string `json:"error,omitempty"`
	Fallback            string `json:"fallback,omitempty"`
}

type WebRTCDisconnectedWebhookPayload struct {
	V1WebhookPayloadBase

	Type                string `json:"type,omitempty"`
	SessionID           string `json:"session_id"`
	MediaSessionID      uint64 `json:"media_session_id"`
	Reason              string `json:"reason"`
	PeerConnectionState string `json:"peer_connection_state,omitempty"`
}

func NewV1WebhookPayload(extra map[string]interface{}) V1WebhookPayloadBase {
	if extra == nil {
		extra = map[string]interface{}{}
	}
	return V1WebhookPayloadBase{
		Version: WebhookPayloadVersionV1,
		Extra:   extra,
	}
}

func (V1WebhookPayloadBase) isV1WebhookPayload() {}
