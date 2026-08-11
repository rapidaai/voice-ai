// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz

import (
	"errors"
	"fmt"
	"time"
)

const (
	VobizProvider = "vobiz"
	WebhookEvent  = "webhook"
)

type EventType string

const (
	// Inbound (vobiz -> us)
	EventTypeStart        EventType = "start"
	EventTypeMedia        EventType = "media"
	EventTypePlayedStream EventType = "playedStream"
	EventTypeClearedAudio EventType = "clearedAudio"
	// Outbound (us -> vobiz)
	EventTypePlayAudio  EventType = "playAudio"
	EventTypeCheckpoint EventType = "checkpoint"
	EventTypeClearAudio EventType = "clearAudio"
	EventTypeStop       EventType = "stop"
)

type OutboundFailureReason string

const (
	OutboundFailureReasonRequestCancelled           OutboundFailureReason = "vobiz_outbound_request_cancelled"
	OutboundFailureReasonMissingVaultCredential     OutboundFailureReason = "vobiz_outbound_missing_vault_credential"
	OutboundFailureReasonAuthIDMissing              OutboundFailureReason = "vobiz_outbound_auth_id_missing"
	OutboundFailureReasonAuthTokenMissing           OutboundFailureReason = "vobiz_outbound_auth_token_missing"
	OutboundFailureReasonContextIDMissing           OutboundFailureReason = "vobiz_outbound_context_id_missing"
	OutboundFailureReasonProviderAPIError           OutboundFailureReason = "vobiz_outbound_provider_api_error"
	OutboundFailureReasonResponseMissingRequestUUID OutboundFailureReason = "vobiz_outbound_provider_response_missing_request_uuid"
)

func (r OutboundFailureReason) String() string {
	return string(r)
}

// VobizMediaEvent is the inbound JSON envelope vobiz sends over the WebSocket.
// Identifiers live inside the nested `start` object on the start event; the
// `media`/control events carry a top-level streamId.
type VobizMediaEvent struct {
	SequenceNumber int         `json:"sequenceNumber"`
	StreamId       string      `json:"streamId"`
	Event          EventType   `json:"event"`
	Start          *VobizStart `json:"start,omitempty"`
	Media          *VobizMedia `json:"media,omitempty"`
	Name           string      `json:"name,omitempty"`
}

type VobizStart struct {
	CallId      string           `json:"callId"`
	StreamId    string           `json:"streamId"`
	AccountId   string           `json:"accountId"`
	Tracks      []string         `json:"tracks"`
	MediaFormat VobizMediaFormat `json:"mediaFormat"`
}

type VobizMediaFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sampleRate"`
}

type VobizMedia struct {
	Track     string `json:"track"`
	Timestamp string `json:"timestamp"`
	Chunk     int    `json:"chunk"`
	Payload   string `json:"payload"`
}

// VobizPlayAudioMessage queues a 20ms audio chunk for playback to the caller.
// Note: playAudio carries no streamId (per the vobiz spec).
type VobizPlayAudioMessage struct {
	Event EventType          `json:"event"`
	Media VobizOutboundMedia `json:"media"`
}

type VobizOutboundMedia struct {
	ContentType string `json:"contentType"`
	SampleRate  int    `json:"sampleRate"`
	Payload     string `json:"payload"`
}

// VobizControlMessage is used for clearAudio (barge-in), checkpoint, and stop.
type VobizControlMessage struct {
	Event    EventType `json:"event"`
	StreamID string    `json:"streamId,omitempty"`
	Name     string    `json:"name,omitempty"`
}

// MakeCallRequest is the body for POST /api/v1/Account/{auth_id}/Call/.
// answer_url must return XML; for the websocket integration it returns a
// <Stream> verb pointing at our WebSocket.
type MakeCallRequest struct {
	From         string `json:"from"`
	To           string `json:"to"`
	AnswerURL    string `json:"answer_url"`
	AnswerMethod string `json:"answer_method,omitempty"`
	RingURL      string `json:"ring_url,omitempty"`
	RingMethod   string `json:"ring_method,omitempty"`
	HangupURL    string `json:"hangup_url,omitempty"`
	HangupMethod string `json:"hangup_method,omitempty"`
	CallerName   string `json:"caller_name,omitempty"`
}

// CallResponse is the response of a fired outbound call. RequestUUID is the
// call identifier (equivalent to call_uuid) used to correlate callbacks.
type CallResponse struct {
	APIID       string `json:"api_id"`
	Message     string `json:"message"`
	RequestUUID string `json:"request_uuid"`
}

// VobizAPIError is returned for non-2xx Vobiz API responses. Message is a
// best-effort human-readable message extracted from the response body.
type VobizAPIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *VobizAPIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("vobiz api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("vobiz api error (status %d): %s", e.StatusCode, e.Body)
}

// Outbound media format negotiated via the <Stream contentType> attribute.
const (
	OutputContentType = "audio/x-mulaw"
	OutputSampleRate  = 8000
)

// Audio framing constants (mulaw 8kHz provider <-> linear16 16kHz engine).
// Mirrors the Twilio provider since the codec path is identical.
const (
	ChunkDuration         = 20 * time.Millisecond
	MulawBytesPerMs       = 8
	Linear16BytesPerMs    = 32
	OutputChunkSize       = MulawBytesPerMs * 20    // 160 bytes = 20ms @ 8kHz mulaw
	BridgeOutputFrameSize = Linear16BytesPerMs * 20 // 640 bytes = 20ms @ 16kHz L16
	InputBufferThreshold  = Linear16BytesPerMs * 40 // 1280 bytes = 40ms @ 16kHz
	MulawSilence          = 0xFF
)

var (
	ErrVaultCredentialValueMissing    = errors.New("vault credential value is nil")
	ErrVaultAuthIDMissing             = errors.New("illegal vault config auth_id not found")
	ErrVaultAuthTokenMissing          = errors.New("illegal vault config auth_token not found")
	ErrStatusCallbackStatusMissing    = errors.New("status not found in payload")
	ErrCatchAllChannelUUIDMissing     = errors.New("call uuid not found in catch-all callback")
	ErrOutboundResponseMissingUUID    = errors.New("vobiz call response missing request_uuid")
	ErrAudioProcessorInitFailed       = errors.New("failed to initialize vobiz audio processor")
	ErrResamplerCreateFailed          = errors.New("failed to create resampler")
	ErrProviderAudioConversionFailed  = errors.New("audio conversion to 16kHz linear16 failed")
	ErrAssistantAudioConversionFailed = errors.New("audio conversion to mulaw 8kHz failed")
)
