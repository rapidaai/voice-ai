// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type TelephonyEvent string

const (
	TelephonyEventCompleted        TelephonyEvent = "completed"
	TelephonyEventRinging          TelephonyEvent = "ringing"
	TelephonyEventAnswered         TelephonyEvent = "answered"
	TelephonyEventStreamStarted    TelephonyEvent = "stream-started"
	TelephonyEventChannelDestroyed TelephonyEvent = "channel_destroyed"
)

func (e TelephonyEvent) String() string {
	return string(e)
}

type TelephonyStatus string

const (
	TelephonyStatusFailed  TelephonyStatus = "FAILED"
	TelephonyStatusSuccess TelephonyStatus = "SUCCESS"
)

func (s TelephonyStatus) String() string {
	return string(s)
}

// StatusInfo is the structured response returned by status/event callbacks.
// It carries the event name and raw payload from the provider.
type StatusInfo struct {
	// Event is the status/event name from the provider callback
	// (e.g. "completed", "ringing", "answered", "stream-started", "channel_destroyed").
	//
	Event TelephonyEvent

	// ChannelUUID is the provider-specific call identifier from the callback.
	ChannelUUID string

	// Error is set when the provider callback represents a failed terminal state.
	Error *StatusError

	// Completed is set by the provider parser when the callback represents a successful terminal state.
	Completed bool

	// Duration is the provider-reported call duration, when present.
	// A pointer keeps explicit zero-duration callbacks distinguishable from
	// callbacks that did not include a duration.
	Duration *time.Duration

	// Price is the provider-reported call price, when present.
	Price string

	// RawPayload is the provider callback payload before parsing.
	RawPayload string

	// Payload is the raw event payload from the provider (parsed body, form data, etc.).
	Payload interface{}
}

type StatusError struct {
	Error  string
	Reason string
}

// CallInfo is the structured response returned by ReceiveCall and OutboundCall.
// Providers populate plain data fields; the dispatcher owns telemetry construction.
// When a new provider needs extra data, add a field here (or use Extra).
type CallInfo struct {
	// ChannelUUID is the provider-specific call identifier
	// (Twilio CallSid, Vonage UUID, Asterisk channel ID, SIP Call-ID, etc.)
	ChannelUUID string

	// CallerNumber is the resolved caller/client phone number.
	// Inbound: the caller's number. Outbound: the "to" number.
	CallerNumber string

	// FromNumber is the originating number for outbound calls.
	FromNumber string

	// Direction is "inbound" or "outbound".
	Direction string

	// Status is the call status string (e.g. "SUCCESS", "FAILED", "initiated").
	Status TelephonyStatus

	// StatusInfo carries the event name and payload for the call.
	// For OutboundCall: the initial event (e.g. "initiated", "channel_created").
	// For ReceiveCall: typically "webhook" with query params as payload.
	StatusInfo StatusInfo

	// ErrorMessage is set when the provider call fails. The dispatcher uses this
	// to build a telephony.error metadata entry.
	ErrorMessage string

	// Provider is the telephony provider name (twilio, vonage, exotel, asterisk, sip).
	Provider string

	// Extra holds provider-specific fields that don't warrant a top-level field.
	// Examples: vonage "conversation_uuid", sip "call.status".
	// If a field is used by multiple providers, promote it to a top-level field.
	Extra map[string]string
}

// ProviderCallStatusUpdate is the provider-neutral status payload persisted for telephony call setup.
type ProviderCallStatusUpdate struct {
	// ChannelUUID is the provider-owned call identifier persisted on the call context.
	ChannelUUID string
	// CallStatus is the normalized provider call state, such as initiated, failed, or cancelled.
	CallStatus string
	// ExpectedCallStatus is an optional compare-and-set guard for watchdog/race-sensitive updates.
	ExpectedCallStatus string
	// ErrorMessage is the operator-facing error detail for failed setup or terminal failure.
	ErrorMessage string
	// FailureClass is the normalized failure category used for filtering and alerting.
	FailureClass string
	// FailureReason is the provider or lifecycle reason that explains the failure class.
	FailureReason string
	// SLIResult is the reliability bucket for the terminal outcome, such as client_error or server_error.
	SLIResult string
	// SLIReason is the prefixed reason used for reliability dashboards and SLO grouping.
	SLIReason string
	// DisconnectReason is the normalized terminal reason persisted with call metadata.
	DisconnectReason string
	// Retryable indicates whether the provider failure may succeed on a later attempt.
	Retryable bool
	// ProviderStatusCode is the provider protocol or HTTP status code when one exists.
	ProviderStatusCode int
}

// ProviderCallStatusReporter receives provider-neutral call status updates from telephony implementations.
type ProviderCallStatusReporter func(update ProviderCallStatusUpdate)

// Telephony defines the interface that all telephony providers must implement.
// Providers return structured data — they never construct telemetry.
// The dispatcher is responsible for converting CallInfo/StatusInfo into telemetry.
type Telephony interface {
	// StatusCallback handles a status/event callback for a conversation.
	StatusCallback(ctx *gin.Context, auth *types.Authentication, assistantId, assistantConversationId uint64) (*StatusInfo, error)
	// CatchAllStatusCallback handles a catch-all event callback.
	CatchAllStatusCallback(ctx *gin.Context) (*StatusInfo, error)
	// ReceiveCall processes an incoming call webhook and returns structured call info.
	ReceiveCall(c *gin.Context) (*CallInfo, error)
	// OutboundCall places an outbound call and returns structured call info.
	OutboundCall(ctx context.Context, auth *types.Authentication, toPhone string, fromPhone string, assistant *internal_assistant_entity.Assistant, assistantConversationId uint64, vaultCredential *protos.VaultCredential, statusReporter ProviderCallStatusReporter, opts utils.Option) (*CallInfo, error)
	// InboundCall instructs the provider to answer/connect the inbound call.
	InboundCall(c *gin.Context, auth *types.Authentication, assistantId uint64, clientNumber string, assistantConversationId uint64) error
}

// GetContextAnswerPath returns the contextId-based WebSocket path for media streaming.
// Route: GET /:telephony/ctx/:contextId
func GetContextAnswerPath(provider, contextID string) string {
	return fmt.Sprintf("v1/talk/%s/ctx/%s", provider, contextID)
}

// GetContextEventPath returns the contextId-based event callback path for status updates.
// Route: GET/POST /:telephony/ctx/:contextId/event
func GetContextEventPath(provider, contextID string) string {
	return fmt.Sprintf("v1/talk/%s/ctx/%s/event", provider, contextID)
}
