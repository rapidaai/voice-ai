// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

// Pipeline identifies a dispatchable SIP orchestration stage.
type Pipeline interface {
	CallID() string
}

// SessionEstablishedPipeline carries the resolved call context into setup.
type SessionEstablishedPipeline struct {
	ID              string
	Session         *sip_runtime.Session
	Config          *sip_runtime.Config
	VaultCredential *protos.VaultCredential
	Direction       sip_runtime.CallDirection
	AssistantID     uint64
	Auth            *types.Authentication
	RequestURI      string
	CallAddress     sip_runtime.CallAddress
	ConversationID  uint64
}

func (p SessionEstablishedPipeline) CallID() string { return p.ID }

// TransferInitiatedPipeline carries a transfer request and its media callbacks.
type TransferInitiatedPipeline struct {
	ID                 string
	Session            *sip_runtime.Session
	TargetURI          string
	Targets            []string
	Config             *sip_runtime.Config
	PostTransferAction string
	OnAttempt          func(target string, attempt int, total int)
	OnConnected        func(outboundRTP *sip_runtime.RTPHandler)
	OnFailed           func()
	OnTeardown         func()
	OnResumeAI         func()
	OnOperatorAudio    func([]byte)
}

func (p TransferInitiatedPipeline) CallID() string { return p.ID }

// CallFailedPipeline reports a SIP failure before normal media processing.
type CallFailedPipeline struct {
	ID      string
	Session *sip_runtime.Session
	Error   error
	SIPCode int
}

func (p CallFailedPipeline) CallID() string { return p.ID }
