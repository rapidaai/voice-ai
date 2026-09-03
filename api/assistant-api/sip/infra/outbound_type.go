// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

type OutboundMode = internal_core.OutboundMode

const OutboundModeTrunkTermination = internal_core.OutboundModeTrunkTermination

type OutboundLegPurpose = internal_core.OutboundLegPurpose

const (
	OutboundLegPurposePrimary        = internal_core.OutboundLegPurposePrimary
	OutboundLegPurposeTransferBridge = internal_core.OutboundLegPurposeTransferBridge
)

type OutboundDialogPhase = internal_core.OutboundDialogPhase

const (
	OutboundDialogPhaseInviting   = internal_core.OutboundDialogPhaseInviting
	OutboundDialogPhaseProceeding = internal_core.OutboundDialogPhaseProceeding
	OutboundDialogPhaseAnswered   = internal_core.OutboundDialogPhaseAnswered
	OutboundDialogPhaseConfirmed  = internal_core.OutboundDialogPhaseConfirmed
	OutboundDialogPhaseTerminated = internal_core.OutboundDialogPhaseTerminated
)

type OutboundCallStatus = internal_core.OutboundCallStatus

const (
	OutboundCallStatusInitiated = internal_core.OutboundCallStatusInitiated
	OutboundCallStatusRinging   = internal_core.OutboundCallStatusRinging
	OutboundCallStatusAnswered  = internal_core.OutboundCallStatusAnswered
	OutboundCallStatusFailed    = internal_core.OutboundCallStatusFailed
	OutboundCallStatusCompleted = internal_core.OutboundCallStatusCompleted
	OutboundCallStatusCancelled = internal_core.OutboundCallStatusCancelled
)

type SIPAuthConfig = internal_core.SIPAuthConfig
type MakeCallOptions = internal_core.MakeCallOptions
type TransferBridgeCallOptions = internal_core.TransferBridgeCallOptions
type OutboundConfig = internal_core.OutboundConfig
type OutboundCallIdentity = internal_core.OutboundCallIdentity
type OutboundInviteRequest = internal_core.OutboundInviteRequest

const (
	MetadataOutboundLegPurpose           = internal_core.MetadataOutboundLegPurpose
	MetadataOutboundParentCallID         = internal_core.MetadataOutboundParentCallID
	MetadataOutboundParentContextID      = internal_core.MetadataOutboundParentContextID
	MetadataOutboundParentConversationID = internal_core.MetadataOutboundParentConversationID
	MetadataOutboundTransferTarget       = internal_core.MetadataOutboundTransferTarget
	MetadataOutboundTransferAttempt      = internal_core.MetadataOutboundTransferAttempt
	MetadataOutboundTransferTotal        = internal_core.MetadataOutboundTransferTotal
)

func NewOutboundInviteRequest(cfg *Config, toUser string, fromUser string) (OutboundInviteRequest, error) {
	return internal_core.NewOutboundInviteRequest(cfg, toUser, fromUser)
}
