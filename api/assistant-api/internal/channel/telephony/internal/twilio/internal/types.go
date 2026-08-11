// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_twilio

import (
	"errors"
	"time"
)

const TwilioProvider = "twilio"

type EventType string

const (
	EventTypeConnected EventType = "connected"
	EventTypeStart     EventType = "start"
	EventTypeMedia     EventType = "media"
	EventTypeStop      EventType = "stop"
	EventTypeClear     EventType = "clear"
)

type OutboundFailureReason string

const (
	OutboundFailureReasonRequestCancelled           OutboundFailureReason = "twilio_outbound_request_cancelled"
	OutboundFailureReasonAuthenticationFailed       OutboundFailureReason = "twilio_outbound_authentication_failed"
	OutboundFailureReasonProviderAPIError           OutboundFailureReason = "twilio_outbound_provider_api_error"
	OutboundFailureReasonResponseMissingStatusOrSID OutboundFailureReason = "twilio_outbound_provider_response_missing_status_or_sid"
)

func (r OutboundFailureReason) String() string {
	return string(r)
}

type TwilioMediaEvent struct {
	Event     EventType    `json:"event"`
	Media     *TwilioMedia `json:"media,omitempty"`
	StreamSid string       `json:"streamSid"`
}

type TwilioMedia struct {
	Track     string `json:"track"`
	Chunk     string `json:"chunk"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"`
}

type TwilioOutboundMessage struct {
	Event    EventType            `json:"event"`
	StreamID string               `json:"streamSid"`
	Media    *TwilioOutboundMedia `json:"media,omitempty"`
}

type TwilioOutboundMedia struct {
	Payload string `json:"payload"`
}

const (
	ChunkDuration         = 20 * time.Millisecond
	MulawBytesPerMs       = 8
	Linear16BytesPerMs    = 32
	OutputChunkSize       = MulawBytesPerMs * 20
	BridgeOutputFrameSize = Linear16BytesPerMs * 20
	InputBufferThreshold  = Linear16BytesPerMs * 40
	MulawSilence          = 0xFF
)

var (
	ErrVaultCredentialValueMissing      = errors.New("vault credential value is nil")
	ErrVaultAccountSIDMissing           = errors.New("illegal vault config accountSid is not found")
	ErrVaultAccountTokenMissing         = errors.New("illegal vault config account_token not found")
	ErrVaultAccountSIDInvalid           = errors.New("illegal vault config account_sid is not a string")
	ErrVaultAccountTokenInvalid         = errors.New("illegal vault config account_token is not a string")
	ErrRequestBodyReadFailed            = errors.New("failed to read request body")
	ErrRequestBodyParseFailed           = errors.New("failed to parse request body")
	ErrStatusCallbackCallSIDMissing     = errors.New("call sid not found in callback")
	ErrStatusCallbackStatusMissing      = errors.New("status not found in payload")
	ErrOutboundResponseMissingStatusSID = errors.New("twilio response missing status or sid")
	ErrInboundFromMissing               = errors.New("missing or empty 'from' query parameter")
	ErrAudioProcessorInitFailed         = errors.New("failed to initialize Twilio audio processor")
	ErrResamplerCreateFailed            = errors.New("failed to create resampler")
	ErrProviderAudioConversionFailed    = errors.New("audio conversion to 16kHz linear16 failed")
	ErrAssistantAudioConversionFailed   = errors.New("audio conversion to mulaw 8kHz failed")
)
