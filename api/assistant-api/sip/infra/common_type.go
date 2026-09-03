// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/protos"
)

var (
	ErrInvalidConfig              = internal_core.ErrInvalidConfig
	ErrSessionNotFound            = internal_core.ErrSessionNotFound
	ErrSessionClosed              = internal_core.ErrSessionClosed
	ErrRTPNotInitialized          = internal_core.ErrRTPNotInitialized
	ErrRTPHandlerStopped          = internal_core.ErrRTPHandlerStopped
	ErrRTPOutputQueueFull         = internal_core.ErrRTPOutputQueueFull
	ErrRTPPortRangeExhausted      = internal_core.ErrRTPPortRangeExhausted
	ErrSDPParseFailed             = internal_core.ErrSDPParseFailed
	ErrCodecNotSupported          = internal_core.ErrCodecNotSupported
	ErrConnectionFailed           = internal_core.ErrConnectionFailed
	ErrAuthRequired               = internal_core.ErrAuthRequired
	ErrOutboundFromUserRequired   = internal_core.ErrOutboundFromUserRequired
	ErrInboundACKTimeout          = internal_core.ErrInboundACKTimeout
	ErrInboundInviteCancelled     = internal_core.ErrInboundInviteCancelled
	ErrInboundAnswerPolicyTimeout = internal_core.ErrInboundAnswerPolicyTimeout
	ErrBridgeLifecycleRejected    = internal_core.ErrBridgeLifecycleRejected
)

type SIPError = internal_core.SIPError

func NewSIPError(op, callID, message string, err error) *SIPError {
	return internal_core.NewSIPError(op, callID, message, err)
}

type Transport = internal_core.Transport

const (
	TransportUDP = internal_core.TransportUDP
	TransportTCP = internal_core.TransportTCP
	TransportTLS = internal_core.TransportTLS
)

type Config = internal_core.Config

type CallState = internal_core.CallState

const (
	CallStateInitializing    = internal_core.CallStateInitializing
	CallStateRinging         = internal_core.CallStateRinging
	CallStateConnected       = internal_core.CallStateConnected
	CallStateOnHold          = internal_core.CallStateOnHold
	CallStateTransferring    = internal_core.CallStateTransferring
	CallStateBridgeConnected = internal_core.CallStateBridgeConnected
	CallStateEnding          = internal_core.CallStateEnding
	CallStateEnded           = internal_core.CallStateEnded
	CallStateFailed          = internal_core.CallStateFailed
	CallStateCancelled       = internal_core.CallStateCancelled
)

type CallDirection = internal_core.CallDirection

const (
	CallDirectionInbound  = internal_core.CallDirectionInbound
	CallDirectionOutbound = internal_core.CallDirectionOutbound
)

type InboundSetupPhase = internal_core.InboundSetupPhase

const (
	InboundSetupPhaseInviteReceived   = internal_core.InboundSetupPhaseInviteReceived
	InboundSetupPhaseTryingSent       = internal_core.InboundSetupPhaseTryingSent
	InboundSetupPhaseRingingSent      = internal_core.InboundSetupPhaseRingingSent
	InboundSetupPhaseAuthenticated    = internal_core.InboundSetupPhaseAuthenticated
	InboundSetupPhaseRouted           = internal_core.InboundSetupPhaseRouted
	InboundSetupPhaseMediaAllocated   = internal_core.InboundSetupPhaseMediaAllocated
	InboundSetupPhaseApplicationReady = internal_core.InboundSetupPhaseApplicationReady
	InboundSetupPhaseAnswerReady      = internal_core.InboundSetupPhaseAnswerReady
	InboundSetupPhaseAnswered         = internal_core.InboundSetupPhaseAnswered
	InboundSetupPhaseACKConfirmed     = internal_core.InboundSetupPhaseACKConfirmed
	InboundSetupPhaseMediaFlowing     = internal_core.InboundSetupPhaseMediaFlowing
)

type InboundAnswerMode = internal_core.InboundAnswerMode

const (
	InboundAnswerModeImmediate            = internal_core.InboundAnswerModeImmediate
	InboundAnswerModeAfterMinRingDuration = internal_core.InboundAnswerModeAfterMinRingDuration
)

type InboundSetupTimings = internal_core.InboundSetupTimings
type SessionInfo = internal_core.SessionInfo
type EventType = internal_core.EventType

const (
	EventTypeInvite     = internal_core.EventTypeInvite
	EventTypeRinging    = internal_core.EventTypeRinging
	EventTypeConnected  = internal_core.EventTypeConnected
	EventTypeBye        = internal_core.EventTypeBye
	EventTypeCancel     = internal_core.EventTypeCancel
	EventTypeDTMF       = internal_core.EventTypeDTMF
	EventTypeError      = internal_core.EventTypeError
	EventTypeRTPStarted = internal_core.EventTypeRTPStarted
	EventTypeRTPStopped = internal_core.EventTypeRTPStopped
)

const (
	BridgeCallTimeout                    = internal_core.BridgeCallTimeout
	BridgeSafetyTimeout                  = internal_core.BridgeSafetyTimeout
	MetadataBridgeTransferTarget         = internal_core.MetadataBridgeTransferTarget
	MetadataBridgeTransferStatus         = internal_core.MetadataBridgeTransferStatus
	MetadataBridgeTransferDuration       = internal_core.MetadataBridgeTransferDuration
	MetadataBridgeTransferOutboundCallID = internal_core.MetadataBridgeTransferOutboundCallID
	MetadataDisconnectReason             = internal_core.MetadataDisconnectReason
	MetadataDisconnectRawReason          = internal_core.MetadataDisconnectRawReason
	PostTransferActionEndCall            = internal_core.PostTransferActionEndCall
	PostTransferActionResumeAI           = internal_core.PostTransferActionResumeAI
)

type Event = internal_core.Event

func NewEvent(eventType EventType, callID string, data map[string]interface{}) Event {
	return internal_core.NewEvent(eventType, callID, data)
}

const (
	DisconnectReasonRemoteHangup   = internal_core.DisconnectReasonRemoteHangup
	DisconnectReasonNormalClearing = internal_core.DisconnectReasonNormalClearing
	DisconnectReasonBusy           = internal_core.DisconnectReasonBusy
	DisconnectReasonNoAnswer       = internal_core.DisconnectReasonNoAnswer
	DisconnectReasonRejected       = internal_core.DisconnectReasonRejected
	DisconnectReasonCancelled      = internal_core.DisconnectReasonCancelled
	DisconnectReasonNetworkFailure = internal_core.DisconnectReasonNetworkFailure
	DisconnectReasonRemoteError    = internal_core.DisconnectReasonRemoteError
)

type DisconnectMetadata = internal_core.DisconnectMetadata
type DTMFEvent = internal_core.DTMFEvent
type RTPStats = internal_core.RTPStats

func ParseConfigFromVault(vaultCredential *protos.VaultCredential) (*Config, error) {
	return internal_core.ParseConfigFromVault(vaultCredential)
}
