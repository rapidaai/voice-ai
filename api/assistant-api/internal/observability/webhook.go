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
	CallID    string               `json:"callId"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallRingingWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	CallID    string               `json:"callId"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"contextId"`
	Source    string               `json:"source"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallProviderAnsweredWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	CallID    string               `json:"callId"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"contextId"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallFailedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider         string                      `json:"provider,omitempty"`
	CallID           string                      `json:"callId,omitempty"`
	To               string                      `json:"to,omitempty"`
	From             string                      `json:"from,omitempty"`
	Direction        WebhookCallDirection        `json:"direction,omitempty"`
	ContextID        string                      `json:"contextId,omitempty"`
	Stage            string                      `json:"stage,omitempty"`
	Source           string                      `json:"source,omitempty"`
	Error            string                      `json:"error,omitempty"`
	DurationMs       string                      `json:"durationMs,omitempty"`
	Status           WebhookCallStatus           `json:"status"`
	DisconnectReason WebhookCallDisconnectReason `json:"disconnectReason,omitempty"`
}

type CallStartedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	CallID    string               `json:"callId"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"contextId"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallEndedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider         string                      `json:"provider"`
	CallID           string                      `json:"callId"`
	To               string                      `json:"to"`
	From             string                      `json:"from"`
	Direction        WebhookCallDirection        `json:"direction"`
	ContextID        string                      `json:"contextId"`
	DurationMs       string                      `json:"durationMs"`
	Status           WebhookCallStatus           `json:"status"`
	DisconnectReason WebhookCallDisconnectReason `json:"disconnectReason,omitempty"`
}

type CallOutboundRequestedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"contextId"`
	Status    WebhookCallStatus    `json:"status"`
}

type CallOutboundDispatchedWebhookPayload struct {
	V1WebhookPayloadBase

	Provider  string               `json:"provider"`
	CallID    string               `json:"callId"`
	To        string               `json:"to"`
	From      string               `json:"from"`
	Direction WebhookCallDirection `json:"direction"`
	ContextID string               `json:"contextId"`
	Status    WebhookCallStatus    `json:"status"`
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
	MessageCount string `json:"messageCount"`
	Status       string `json:"status"`
}

type ConversationCompletedWebhookPayload struct {
	V1WebhookPayloadBase

	Source           string                   `json:"source"`
	Reason           string                   `json:"reason"`
	Status           string                   `json:"status"`
	DisconnectReason string                   `json:"disconnectReason,omitempty"`
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
	DisconnectReason string `json:"disconnectReason,omitempty"`
}

type WebRTCConnectedWebhookPayload struct {
	V1WebhookPayloadBase

	SessionID           string `json:"sessionId"`
	MediaSessionID      uint64 `json:"mediaSessionId"`
	ICELatencyMs        int64  `json:"iceLatencyMs"`
	PeerConnectionState string `json:"peerConnectionState"`
}

type WebRTCAudioTrackReceivedWebhookPayload struct {
	V1WebhookPayloadBase

	SessionID      string `json:"sessionId"`
	MediaSessionID uint64 `json:"mediaSessionId"`
	Codec          string `json:"codec"`
}

type WebRTCReconnectingWebhookPayload struct {
	V1WebhookPayloadBase

	Type           string `json:"type"`
	SessionID      string `json:"sessionId"`
	MediaSessionID uint64 `json:"mediaSessionId"`
	Reason         string `json:"reason"`
	RestartAttempt uint64 `json:"restartAttempt"`
	RestartLimit   uint64 `json:"restartLimit"`
}

type WebRTCFailedWebhookPayload struct {
	V1WebhookPayloadBase

	Type                string `json:"type"`
	SessionID           string `json:"sessionId"`
	MediaSessionID      uint64 `json:"mediaSessionId,omitempty"`
	Reason              string `json:"reason"`
	PeerConnectionState string `json:"peerConnectionState,omitempty"`
	Error               string `json:"error,omitempty"`
	Fallback            string `json:"fallback,omitempty"`
}

type WebRTCDisconnectedWebhookPayload struct {
	V1WebhookPayloadBase

	Type                string `json:"type,omitempty"`
	SessionID           string `json:"sessionId"`
	MediaSessionID      uint64 `json:"mediaSessionId"`
	Reason              string `json:"reason"`
	PeerConnectionState string `json:"peerConnectionState,omitempty"`
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
