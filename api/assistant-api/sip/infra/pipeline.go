// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

// Pipeline is the base interface for all SIP call lifecycle stages.
// Each concrete type represents a distinct stage in the pipeline.
// Handlers receive a typed Pipeline, apply logic, and emit the next stage(s)
// via OnPipeline — forming chains without explicit wiring.
type Pipeline interface {
	CallID() string
}

// =============================================================================
// Media pipeline — RTP, codec, session establishment
// =============================================================================

// SessionEstablishedPipeline is emitted after RTP allocation, session creation,
// and 200 OK is sent. Converges inbound and outbound flows. FromURI/ToURI
// carry the INVITE addresses so downstream stages can build a CallContext
// without re-parsing SIP headers.
type SessionEstablishedPipeline struct {
	ID              string
	Session         *Session
	Config          *Config
	VaultCredential *protos.VaultCredential
	Direction       CallDirection
	AssistantID     uint64
	Auth            types.SimplePrinciple
	FromURI         string
	ToURI           string
	ConversationID  uint64 // Non-zero for outbound (already created by channel pipeline)
}

func (p SessionEstablishedPipeline) CallID() string { return p.ID }

// CallCreatedPipeline is emitted when SIP session identity is created and
// registered, before ringing/answer lifecycle transitions.
type CallCreatedPipeline struct {
	ID      string
	Session *Session
	FromURI string
	ToURI   string
}

func (p CallCreatedPipeline) CallID() string { return p.ID }

// CallRingingPipeline is emitted when an inbound call reaches ringing state
// (180 Ringing), before conversation startup.
type CallRingingPipeline struct {
	ID      string
	Session *Session
	FromURI string
	ToURI   string
}

func (p CallRingingPipeline) CallID() string { return p.ID }

// CallAnsweredPipeline is emitted when an inbound call is answered
// (200 OK / connected), before conversation startup.
type CallAnsweredPipeline struct {
	ID      string
	Session *Session
	FromURI string
	ToURI   string
}

func (p CallAnsweredPipeline) CallID() string { return p.ID }

// CallMediaStartedPipeline represents confirmed media flow start.
// Emit only when RTP/media start can be signaled reliably.
type CallMediaStartedPipeline struct {
	ID      string
	Session *Session
}

func (p CallMediaStartedPipeline) CallID() string { return p.ID }

// =============================================================================
// Signal pipeline — BYE, CANCEL, transfer (preempts everything)
// =============================================================================

type ByeReceivedPipeline struct {
	ID      string
	Session *Session
	Reason  string
}

func (p ByeReceivedPipeline) CallID() string { return p.ID }

type CancelReceivedPipeline struct {
	ID      string
	Session *Session
}

func (p CancelReceivedPipeline) CallID() string { return p.ID }

type TransferInitiatedPipeline struct {
	ID                 string
	Session            *Session
	TransferID         string
	TargetURI          string
	Targets            []string
	RoutingMode        string
	Config             *Config
	PostTransferAction string
	OnAttempt          func(target string, attempt int, total int)
	OnConnected        func(outboundRTP *RTPHandler)
	OnFailed           func()
	OnTeardown         func()
	OnResumeAI         func()
	OnOperatorAudio    func([]byte)
}

func (p TransferInitiatedPipeline) CallID() string { return p.ID }

type TransferConnectedPipeline struct {
	ID              string
	InboundSession  *Session
	OutboundSession *Session
	TargetURI       string
	Attempt         int
	TotalAttempts   int
	TransferID      string
	RoutingMode     string
}

func (p TransferConnectedPipeline) CallID() string { return p.ID }

type TransferFailedPipeline struct {
	ID          string
	Session     *Session
	TransferID  string
	RoutingMode string
	Error       error
	Reason      string
}

func (p TransferFailedPipeline) CallID() string { return p.ID }

type TransferAttemptStartedPipeline struct {
	ID        string
	Session   *Session
	TransferID string
	TargetURI string
	Attempt   int
	Total     int
	RoutingMode string
}

func (p TransferAttemptStartedPipeline) CallID() string { return p.ID }

// TransferRequestedPipeline is emitted when transfer routing starts, before
// target attempts begin.
type TransferRequestedPipeline struct {
	ID                 string
	Session            *Session
	TransferID         string
	Targets            []string
	RoutingMode        string
	PostTransferAction string
}

func (p TransferRequestedPipeline) CallID() string { return p.ID }

// TransferTargetRingingPipeline is emitted when a transfer target is known to
// be ringing from an outbound progress signal.
type TransferTargetRingingPipeline struct {
	ID          string
	Session     *Session
	TransferID  string
	TargetURI   string
	Attempt     int
	Total       int
	RoutingMode string
}

func (p TransferTargetRingingPipeline) CallID() string { return p.ID }

// TransferAttemptEndedPipeline is emitted once per attempted transfer target
// with terminal state (connected/no_answer/busy/rejected/failed/cancelled).
type TransferAttemptEndedPipeline struct {
	ID             string
	Session        *Session
	TransferID     string
	AttemptID      string
	TargetURI      string
	OutboundCallID string
	Attempt        int
	Total          int
	RoutingMode    string
	State          string
	Reason         string
	AnsweredBy     string
	Metadata       map[string]interface{}
}

func (p TransferAttemptEndedPipeline) CallID() string { return p.ID }

// TransferCancelledPipeline is emitted when pending transfer targets are
// cancelled (for example answered_by_other in parallel routing).
type TransferCancelledPipeline struct {
	ID          string
	Session     *Session
	TransferID  string
	TargetURI   string
	Attempt     int
	Total       int
	RoutingMode string
	Reason      string
	AnsweredBy  string
	Metadata    map[string]interface{}
}

func (p TransferCancelledPipeline) CallID() string { return p.ID }

type CallEndedPipeline struct {
	ID       string
	Duration time.Duration
	Reason   string
}

func (p CallEndedPipeline) CallID() string { return p.ID }

type CallFailedPipeline struct {
	ID      string
	Session *Session
	Error   error
	SIPCode int
}

func (p CallFailedPipeline) CallID() string { return p.ID }

// =============================================================================
// Control pipeline — metrics, events, recording, DTMF, registration
// =============================================================================

// EventEmittedPipeline is a generic event for logging and observability.
type EventEmittedPipeline struct {
	ID    string
	Event string
	Data  map[string]string
}

func (p EventEmittedPipeline) CallID() string { return p.ID }

// MetricEmittedPipeline carries metrics for a call.
type MetricEmittedPipeline struct {
	ID      string
	Metrics []*protos.Metric
}

func (p MetricEmittedPipeline) CallID() string { return p.ID }

// DTMFReceivedPipeline is emitted when a DTMF digit is detected via RTP (RFC 4733).
type DTMFReceivedPipeline struct {
	ID       string
	Digit    string
	Duration int // milliseconds
}

func (p DTMFReceivedPipeline) CallID() string { return p.ID }
