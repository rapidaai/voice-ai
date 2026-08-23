// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type Pipeline interface {
	CallID() string
}

type SessionEstablishedPipeline struct {
	ID              string
	Session         *Session
	Config          *Config
	VaultCredential *protos.VaultCredential
	Direction       CallDirection
	AssistantID     uint64
	Auth            *types.Authentication
	FromURI         string
	ToURI           string
	ConversationID  uint64
}

func (p SessionEstablishedPipeline) CallID() string { return p.ID }

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
