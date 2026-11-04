// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_plivo

import (
	"errors"
	"time"
)

const PlivoProvider = "plivo"

// EventType enumerates the Plivo bidirectional audio-stream WebSocket events.
type EventType string

const (
	// Inbound (Plivo -> server).
	EventTypeStart EventType = "start"
	EventTypeMedia EventType = "media"
	EventTypeStop  EventType = "stop"
	// Outbound (server -> Plivo).
	EventTypePlayAudio  EventType = "playAudio"
	EventTypeClearAudio EventType = "clearAudio"
)

// PlivoMediaEvent is an inbound WebSocket message from Plivo's bidirectional stream.
// Plivo delivers "start" (with call metadata), "media" (base64 mu-law audio), and "stop".
type PlivoMediaEvent struct {
	Event    EventType   `json:"event"`
	StreamID string      `json:"streamId,omitempty"`
	Start    *PlivoStart `json:"start,omitempty"`
	Media    *PlivoMedia `json:"media,omitempty"`
}

// PlivoStart carries call metadata sent with the "start" event.
type PlivoStart struct {
	CallID      string            `json:"callId"`
	StreamID    string            `json:"streamId"`
	AccountID   string            `json:"accountId,omitempty"`
	Tracks      []string          `json:"tracks,omitempty"`
	MediaFormat *PlivoMediaFormat `json:"mediaFormat,omitempty"`
}

// PlivoMediaFormat describes the negotiated audio format on the stream.
type PlivoMediaFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels,omitempty"`
}

// PlivoMedia is the audio payload of a "media" event (base64-encoded mu-law).
type PlivoMedia struct {
	Track     string `json:"track,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Payload   string `json:"payload"`
}

// PlivoOutboundMessage is a message sent from the server to Plivo: playAudio to
// transmit audio to the caller, clearAudio to flush buffered audio (barge-in).
// clearAudio carries the streamId of the stream it targets; playAudio carries no
// streamId and sets the codec via contentType/sampleRate on the media object.
type PlivoOutboundMessage struct {
	Event    EventType           `json:"event"`
	StreamID string              `json:"streamId,omitempty"`
	Media    *PlivoOutboundMedia `json:"media,omitempty"`
}

// PlivoOutboundMedia is the audio body of a playAudio message.
type PlivoOutboundMedia struct {
	ContentType string `json:"contentType"`
	SampleRate  int    `json:"sampleRate"`
	Payload     string `json:"payload"`
}

// Audio framing. Plivo streams mu-law 8kHz; a 20ms frame is 160 bytes, matching
// the other providers. Inbound mu-law is upsampled to 16kHz linear16 for the
// pipeline; assistant audio is downsampled back to 8kHz mu-law for Plivo.
const (
	ChunkDuration         = 20 * time.Millisecond
	MulawBytesPerMs       = 8
	Linear16BytesPerMs    = 32
	OutputChunkSize       = MulawBytesPerMs * 20
	BridgeOutputFrameSize = Linear16BytesPerMs * 20
	InputBufferThreshold  = Linear16BytesPerMs * 40
	MulawSilence          = 0xFF

	// Outbound playAudio media settings.
	OutboundContentType = "audio/x-mulaw"
	OutboundSampleRate  = 8000
)

var (
	ErrVaultCredentialMissing         = errors.New("vault credential is nil")
	ErrVaultCredentialValueMissing    = errors.New("vault credential value is nil")
	ErrVaultAuthIDMissing             = errors.New("illegal vault config auth_id not found")
	ErrVaultAuthTokenMissing          = errors.New("illegal vault config auth_token not found")
	ErrVaultAuthIDInvalid             = errors.New("illegal vault config auth_id is not a string")
	ErrVaultAuthTokenInvalid          = errors.New("illegal vault config auth_token is not a string")
	ErrRequestBodyReadFailed          = errors.New("failed to read request body")
	ErrRequestBodyParseFailed         = errors.New("failed to parse request body")
	ErrStatusCallbackCallUUIDMissing  = errors.New("call uuid not found in callback")
	ErrStatusCallbackStatusMissing    = errors.New("status not found in payload")
	ErrOutboundResponseMissingUUID    = errors.New("plivo response missing request_uuid")
	ErrProviderAPIError               = errors.New("plivo API error")
	ErrProviderHangupFailed           = errors.New("plivo hangup failed")
	ErrInboundFromMissing             = errors.New("missing or empty 'From' parameter")
	ErrAudioProcessorInitFailed       = errors.New("failed to initialize Plivo audio processor")
	ErrResamplerCreateFailed          = errors.New("failed to create resampler")
	ErrProviderAudioConversionFailed  = errors.New("audio conversion to 16kHz linear16 failed")
	ErrAssistantAudioConversionFailed = errors.New("audio conversion to mu-law 8kHz failed")
)
