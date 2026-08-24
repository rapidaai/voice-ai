// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
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

// SessionEstablishedPipeline carries a SIP session into application readiness
// and call start. Inbound uses it twice: prepare while ringing, then start
// after 200 OK and ACK. Outbound starts after the remote answer is accepted.
type SessionEstablishedPipeline struct {
	ID              string
	Session         *Session
	Config          *Config
	VaultCredential *protos.VaultCredential
	Direction       CallDirection
	AssistantID     uint64
	Auth            *types.Authentication
	RequestURI      string
	CallAddress     CallAddress
	ConversationID  uint64 // Non-zero for outbound (already created by channel pipeline)
}

func (p SessionEstablishedPipeline) CallID() string { return p.ID }

// =============================================================================
// Signal pipeline — BYE, CANCEL, transfer (preempts everything)
// =============================================================================

type TransferInitiatedPipeline struct {
	ID                 string
	Session            *Session
	TargetURI          string
	Targets            []string
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
}

func (p TransferConnectedPipeline) CallID() string { return p.ID }

type TransferFailedPipeline struct {
	ID     string
	Error  error
	Reason string
}

func (p TransferFailedPipeline) CallID() string { return p.ID }

type CallFailedPipeline struct {
	ID      string
	Session *Session
	Error   error
	SIPCode int
}

func (p CallFailedPipeline) CallID() string { return p.ID }
