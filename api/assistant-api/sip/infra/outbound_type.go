// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"time"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type OutboundMode string

const (
	OutboundModeTrunkTermination OutboundMode = "trunk_termination"
)

type OutboundLegPurpose string

const (
	OutboundLegPurposePrimary        OutboundLegPurpose = "primary_outbound_call"
	OutboundLegPurposeTransferBridge OutboundLegPurpose = "transfer_bridge_call"
)

type OutboundDialogPhase string

const (
	OutboundDialogPhaseInviting   OutboundDialogPhase = "inviting"
	OutboundDialogPhaseProceeding OutboundDialogPhase = "proceeding"
	OutboundDialogPhaseAnswered   OutboundDialogPhase = "answered"
	OutboundDialogPhaseConfirmed  OutboundDialogPhase = "confirmed"
	OutboundDialogPhaseTerminated OutboundDialogPhase = "terminated"
)

func (p OutboundDialogPhase) IsPreAnswer() bool {
	return p == OutboundDialogPhaseInviting || p == OutboundDialogPhaseProceeding
}

type OutboundCallStatus string

const (
	OutboundCallStatusInitiated OutboundCallStatus = callcontext.CallStatusInitiated
	OutboundCallStatusRinging   OutboundCallStatus = callcontext.CallStatusRinging
	OutboundCallStatusAnswered  OutboundCallStatus = callcontext.CallStatusAnswered
	OutboundCallStatusFailed    OutboundCallStatus = callcontext.CallStatusFailed
	OutboundCallStatusCompleted OutboundCallStatus = callcontext.CallStatusCompleted
	OutboundCallStatusCancelled OutboundCallStatus = callcontext.CallStatusCancelled
)

type SIPAuthConfig struct {
	Username string
	Password string
	Realm    string
}

type MakeCallOptions struct {
	Auth               *types.Authentication
	Assistant          *internal_assistant_entity.Assistant
	ConversationID     uint64
	ContextID          string
	VaultCredential    *protos.VaultCredential
	CallStatusObserver internal_type.ProviderCallStatusReporter
}

func (o MakeCallOptions) toCore() internal_core.MakeCallOptions {
	return internal_core.MakeCallOptions{
		Auth:               o.Auth,
		Assistant:          o.Assistant,
		ConversationID:     o.ConversationID,
		ContextID:          o.ContextID,
		VaultCredential:    o.VaultCredential,
		CallStatusObserver: o.CallStatusObserver,
	}
}

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

func (o TransferBridgeCallOptions) toCore() internal_core.TransferBridgeCallOptions {
	return internal_core.TransferBridgeCallOptions{
		ParentCallID:       o.ParentCallID,
		Attempt:            o.Attempt,
		TotalAttempts:      o.TotalAttempts,
		Auth:               o.Auth,
		Assistant:          o.Assistant,
		ConversationID:     o.ConversationID,
		ContextID:          o.ContextID,
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

func (c OutboundConfig) EffectiveRingingTimeout() time.Duration {
	return c.toCore().EffectiveRingingTimeout()
}

func (c OutboundConfig) EffectiveMaxCallDuration() time.Duration {
	return c.toCore().EffectiveMaxCallDuration()
}

func (c OutboundConfig) toCore() internal_core.OutboundConfig {
	return internal_core.OutboundConfig{
		Mode:      internal_core.OutboundMode(c.Mode),
		Address:   c.Address,
		Port:      c.Port,
		Transport: internal_core.Transport(c.Transport),
		Domain:    c.Domain,
		Auth: internal_core.SIPAuthConfig{
			Username: c.Auth.Username,
			Password: c.Auth.Password,
			Realm:    c.Auth.Realm,
		},
		Headers:             copyJSONCompatibleMap(c.Headers),
		RingingTimeout:      c.RingingTimeout,
		MaxCallDuration:     c.MaxCallDuration,
		MediaTimeoutInitial: c.MediaTimeoutInitial,
		MediaTimeout:        c.MediaTimeout,
	}
}

type OutboundCallIdentity struct {
	ToUser   string
	FromUser string
}

type OutboundInviteRequest struct {
	Config   OutboundConfig
	Identity OutboundCallIdentity
}

func (r OutboundInviteRequest) Validate() error {
	return r.toCore().Validate()
}

func (r OutboundInviteRequest) toCore() internal_core.OutboundInviteRequest {
	return internal_core.OutboundInviteRequest{
		Config: r.Config.toCore(),
		Identity: internal_core.OutboundCallIdentity{
			ToUser:   r.Identity.ToUser,
			FromUser: r.Identity.FromUser,
		},
	}
}

const (
	MetadataOutboundLegPurpose           = "outbound_leg_purpose"
	MetadataOutboundParentCallID         = "outbound_parent_call_id"
	MetadataOutboundParentContextID      = "outbound_parent_context_id"
	MetadataOutboundParentConversationID = "outbound_parent_conversation_id"
	MetadataOutboundTransferTarget       = "outbound_transfer_target"
	MetadataOutboundTransferAttempt      = "outbound_transfer_attempt"
	MetadataOutboundTransferTotal        = "outbound_transfer_total"
)

func (c *Config) ToOutboundConfig() OutboundConfig {
	coreOutbound := c.toCore().ToOutboundConfig()
	return OutboundConfig{
		Mode:      OutboundMode(coreOutbound.Mode),
		Address:   coreOutbound.Address,
		Port:      coreOutbound.Port,
		Transport: Transport(coreOutbound.Transport),
		Domain:    coreOutbound.Domain,
		Auth: SIPAuthConfig{
			Username: coreOutbound.Auth.Username,
			Password: coreOutbound.Auth.Password,
			Realm:    coreOutbound.Auth.Realm,
		},
		Headers:             copyJSONCompatibleMap(coreOutbound.Headers),
		RingingTimeout:      coreOutbound.RingingTimeout,
		MaxCallDuration:     coreOutbound.MaxCallDuration,
		MediaTimeoutInitial: coreOutbound.MediaTimeoutInitial,
		MediaTimeout:        coreOutbound.MediaTimeout,
	}
}

func NewOutboundInviteRequest(cfg *Config, toUser string, fromUser string) (OutboundInviteRequest, error) {
	coreRequest, err := internal_core.NewOutboundInviteRequest(cfg.toCore(), toUser, fromUser)
	if err != nil {
		return OutboundInviteRequest{}, err
	}
	return OutboundInviteRequest{
		Config: OutboundConfig{
			Mode:      OutboundMode(coreRequest.Config.Mode),
			Address:   coreRequest.Config.Address,
			Port:      coreRequest.Config.Port,
			Transport: Transport(coreRequest.Config.Transport),
			Domain:    coreRequest.Config.Domain,
			Auth: SIPAuthConfig{
				Username: coreRequest.Config.Auth.Username,
				Password: coreRequest.Config.Auth.Password,
				Realm:    coreRequest.Config.Auth.Realm,
			},
			Headers:         copyJSONCompatibleMap(coreRequest.Config.Headers),
			RingingTimeout:  coreRequest.Config.RingingTimeout,
			MaxCallDuration: coreRequest.Config.MaxCallDuration,
		},
		Identity: OutboundCallIdentity{
			ToUser:   coreRequest.Identity.ToUser,
			FromUser: coreRequest.Identity.FromUser,
		},
	}, nil
}
