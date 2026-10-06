// Copyright (c) 2023-2025 RapidaAI
// Author: RapidaAI Team <team@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telnyx

import "errors"

const (
	Provider                   = "telnyx"
	WebhookEvent               = "webhook"
	ChannelEventConnected      = "connected"
	ChannelEventStreamStarted  = "stream_started"
	ChannelEventDTMF           = "dtmf"
	StreamTrackBoth            = "both_tracks"
	InboundStreamingStart      = "streaming.start"
	HTTPHeaderAuthorization    = "Authorization"
	HTTPHeaderContentType      = "Content-Type"
	HTTPContentTypeApplication = "application/json"
)

type EventType string

const (
	EventTypeConnected EventType = "connected"
	EventTypeStart     EventType = "start"
	EventTypeMedia     EventType = "media"
	EventTypeDTMF      EventType = "dtmf"
	EventTypeStop      EventType = "stop"
	EventTypeClear     EventType = "clear"
)

type OutboundFailureReason string

const (
	OutboundFailureReasonRequestCancelled     OutboundFailureReason = "telnyx_outbound_request_cancelled"
	OutboundFailureReasonAuthenticationFailed OutboundFailureReason = "telnyx_outbound_authentication_failed"
	OutboundFailureReasonRequestPayloadFailed OutboundFailureReason = "telnyx_outbound_request_payload_failed"
	OutboundFailureReasonRequestCreateFailed  OutboundFailureReason = "telnyx_outbound_request_create_failed"
	OutboundFailureReasonProviderAPIError     OutboundFailureReason = "telnyx_outbound_provider_api_error"
	OutboundFailureReasonResponseReadFailed   OutboundFailureReason = "telnyx_outbound_provider_response_read_failed"
	OutboundFailureReasonResponseDecodeFailed OutboundFailureReason = "telnyx_outbound_provider_response_decode_failed"
	OutboundFailureReasonHTTPStatusFailed     OutboundFailureReason = "telnyx_outbound_provider_http_status_failed"
)

func (r OutboundFailureReason) String() string {
	return string(r)
}

type TelnyxWebSocketEvent struct {
	Event    EventType         `json:"event"`
	StreamID string            `json:"stream_id"`
	Start    *TelnyxStartEvent `json:"start,omitempty"`
	Media    *TelnyxMediaEvent `json:"media,omitempty"`
	Stop     *TelnyxStopEvent  `json:"stop,omitempty"`
}

type TelnyxStartEvent struct {
	CallControlID string            `json:"call_control_id"`
	MediaFormat   TelnyxMediaFormat `json:"media_format"`
}

type TelnyxMediaFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

type TelnyxMediaEvent struct {
	Track   string `json:"track"`
	Payload string `json:"payload"`
}

type TelnyxStopEvent struct {
	CallControlID string `json:"call_control_id"`
}

type TelnyxOutboundMessage struct {
	Event    EventType            `json:"event"`
	StreamID string               `json:"stream_id"`
	Media    *TelnyxOutboundMedia `json:"media,omitempty"`
}

type TelnyxOutboundMedia struct {
	Payload string `json:"payload"`
}

var (
	ErrAudioProcessorInitFailed       = errors.New("failed to initialize Telnyx audio processor")
	ErrProviderAudioConversionFailed  = errors.New("audio conversion to 16kHz linear16 failed")
	ErrAssistantAudioConversionFailed = errors.New("audio conversion to mulaw 8kHz failed")

	ErrRequestBodyReadFailed          = errors.New("failed to read request body")
	ErrRequestBodyParseFailed         = errors.New("failed to parse request body")
	ErrStatusCallbackDataMissing      = errors.New("data field not found in payload")
	ErrStatusCallbackEventTypeMissing = errors.New("event_type not found in payload")
	ErrCatchAllCallControlIDMissing   = errors.New("call control id not found in callback")
	ErrInboundFromMissing             = errors.New("missing or empty 'from' query parameter")
	ErrVaultCredentialMissing         = errors.New("vault credential is nil")
	ErrVaultCredentialValueMissing    = errors.New("vault credential value is nil")
	ErrVaultAPIKeyMissing             = errors.New("api_key not found in vault credential")
	ErrVaultConnectionIDMissing       = errors.New("connection_id not found in vault credential")
	ErrProviderHangupFailed           = errors.New("hangup failed")
	ErrProviderAPIError               = errors.New("provider API error")
)
