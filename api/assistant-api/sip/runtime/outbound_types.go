// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type OutboundMode string

type OutboundLegPurpose string

type OutboundDialogPhase string

func (p OutboundDialogPhase) IsPreAnswer() bool {
	return p == OutboundDialogPhaseInviting || p == OutboundDialogPhaseProceeding
}

type OutboundCallStatus string

type SIPAuthConfig struct {
	Username string
	Password string
	Realm    string
}

// MakeCallOptions carries application context into a primary outbound call leg.
type MakeCallOptions struct {
	Auth               *types.Authentication
	Assistant          *internal_assistant_entity.Assistant
	ConversationID     uint64
	ContextID          string
	VaultCredential    *protos.VaultCredential
	CallStatusObserver internal_type.ProviderCallStatusReporter
}

// TransferBridgeCallOptions carries parent call context into a transfer B-leg.
type TransferBridgeCallOptions struct {
	// ParentCallID identifies the inbound leg that requested the transfer.
	ParentCallID string
	// Attempt is the one-based transfer target attempt number.
	Attempt int
	// TotalAttempts is the number of transfer targets available for this request.
	TotalAttempts int
	// Auth is the authenticated principal associated with the parent call.
	Auth *types.Authentication
	// Assistant is the assistant resolved for the parent call.
	Assistant *internal_assistant_entity.Assistant
	// ConversationID is the active parent assistant conversation.
	ConversationID uint64
	// ContextID is the durable call context ID associated with the parent call.
	ContextID string
	// VaultCredential is the SIP credential used for the transfer B-leg.
	VaultCredential *protos.VaultCredential
	// CallStatusObserver receives provider-neutral outbound status updates.
	CallStatusObserver internal_type.ProviderCallStatusReporter
}

func (o TransferBridgeCallOptions) makeCallOptions() MakeCallOptions {
	return MakeCallOptions{
		Auth:               o.Auth,
		Assistant:          o.Assistant,
		VaultCredential:    o.VaultCredential,
		CallStatusObserver: o.CallStatusObserver,
	}
}

type OutboundConfig struct {
	Mode                OutboundMode
	Address             string
	Port                int
	Transport           Transport
	Domain              string
	Auth                SIPAuthConfig
	Headers             map[string]string
	RingingTimeout      time.Duration
	MaxCallDuration     time.Duration
	MediaTimeoutInitial time.Duration
	MediaTimeout        time.Duration
}

type OutboundCallIdentity struct {
	ToUser   string
	FromUser string
}

type OutboundInviteRequest struct {
	Config   OutboundConfig
	Identity OutboundCallIdentity
}

func (c OutboundConfig) EffectiveRingingTimeout() time.Duration {
	if c.RingingTimeout > 0 {
		return c.RingingTimeout
	}
	return defaultOutboundRingingTimeout
}

func (c OutboundConfig) EffectiveMaxCallDuration() time.Duration {
	if c.MaxCallDuration > 0 {
		return c.MaxCallDuration
	}
	return 0
}
