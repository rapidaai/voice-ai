// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vonage

import (
	"errors"
	"time"
)

const (
	Provider              = "vonage"
	WebhookEvent          = "webhook"
	NCCOEventTypeSync     = "synchronous"
	WebSocketContentType  = "audio/l16;rate=16000"
	WebSocketEndpointType = "websocket"
	ChannelEventConnected = "connected"
	ClearAction           = "clear"
)

type EventType string

const (
	EventTypeWebSocketConnected EventType = "websocket:connected"
	EventTypeStop               EventType = "stop"
)

type OutboundFailureReason string

const (
	OutboundFailureReasonRequestCancelled     OutboundFailureReason = "vonage_outbound_request_cancelled"
	OutboundFailureReasonAuthenticationFailed OutboundFailureReason = "vonage_outbound_authentication_failed"
	OutboundFailureReasonProviderAPIError     OutboundFailureReason = "vonage_outbound_provider_api_error"
	OutboundFailureReasonProviderCallCreate   OutboundFailureReason = "vonage_outbound_provider_call_create_failed"
)

func (r OutboundFailureReason) String() string {
	return string(r)
}

type VonageWebSocketEvent struct {
	Event EventType `json:"event"`
}

type VonageClearMessage struct {
	Action string `json:"action"`
}

var (
	ErrVaultCredentialValueMissing = errors.New("vault credential value is nil")
	ErrVaultPrivateKeyMissing      = errors.New("illegal vault config privateKey is not found")
	ErrVaultApplicationIDMissing   = errors.New("illegal vault config application_id is not found")
	ErrVaultPrivateKeyInvalid      = errors.New("illegal vault config private_key is not a string")
	ErrVaultApplicationIDInvalid   = errors.New("illegal vault config application_id is not a string")

	ErrRequestBodyReadFailed       = errors.New("failed to read request body")
	ErrRequestBodyParseFailed      = errors.New("failed to parse request body")
	ErrStatusCallbackStatusMissing = errors.New("status not found in payload")
	ErrCatchAllChannelUUIDMissing  = errors.New("uuid not found in callback")
	ErrProviderCallCreateFailed    = errors.New("failed to create call")
	ErrInboundFromMissing          = errors.New("missing or empty 'from' query parameter")

	ErrAudioProcessorInitFailed = errors.New("failed to initialize Vonage audio processor")
)

const (
	ChunkDuration        = 20 * time.Millisecond
	Linear16BytesPerMs   = 32
	OutputChunkSize      = Linear16BytesPerMs * 20
	InputBufferThreshold = Linear16BytesPerMs * 40
)
