// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

var (
	ErrMissingDID          = internal_core.ErrMissingDID
	ErrMissingServer       = internal_core.ErrMissingServer
	ErrRegistrationFailed  = internal_core.ErrRegistrationFailed
	ErrRegistrationExpired = internal_core.ErrRegistrationExpired
	ErrDIDNotRegistered    = internal_core.ErrDIDNotRegistered
	ErrAuthFailed          = internal_core.ErrAuthFailed
	ErrPermanentFailure    = internal_core.ErrPermanentFailure
)

type RegistrationFailureClass = internal_core.RegistrationFailureClass

const (
	RegistrationFailureClassConfig     = internal_core.RegistrationFailureClassConfig
	RegistrationFailureClassAuth       = internal_core.RegistrationFailureClassAuth
	RegistrationFailureClassRejected   = internal_core.RegistrationFailureClassRejected
	RegistrationFailureClassTransient  = internal_core.RegistrationFailureClassTransient
	RegistrationFailureClassNetwork    = internal_core.RegistrationFailureClassNetwork
	RegistrationFailureClassOwnership  = internal_core.RegistrationFailureClassOwnership
	RegistrationFailureClassDuplicate  = internal_core.RegistrationFailureClassDuplicate
	RegistrationFailureClassRenewal    = internal_core.RegistrationFailureClassRenewal
	RegistrationFailureClassUnregister = internal_core.RegistrationFailureClassUnregister
)

type RegistrationFailureReason = internal_core.RegistrationFailureReason

const (
	RegistrationFailureReasonMissingDID              = internal_core.RegistrationFailureReasonMissingDID
	RegistrationFailureReasonMissingCredentialID     = internal_core.RegistrationFailureReasonMissingCredentialID
	RegistrationFailureReasonDuplicateDID            = internal_core.RegistrationFailureReasonDuplicateDID
	RegistrationFailureReasonAssistantNotFound       = internal_core.RegistrationFailureReasonAssistantNotFound
	RegistrationFailureReasonVaultCredentialNotFound = internal_core.RegistrationFailureReasonVaultCredentialNotFound
	RegistrationFailureReasonInvalidSIPConfig        = internal_core.RegistrationFailureReasonInvalidSIPConfig
	RegistrationFailureReasonMissingSIPServer        = internal_core.RegistrationFailureReasonMissingSIPServer
	RegistrationFailureReasonOwnershipClaimFailed    = internal_core.RegistrationFailureReasonOwnershipClaimFailed
	RegistrationFailureReasonAuthFailed              = internal_core.RegistrationFailureReasonAuthFailed
	RegistrationFailureReasonRegistrarRejected       = internal_core.RegistrationFailureReasonRegistrarRejected
	RegistrationFailureReasonRegistrarUnreachable    = internal_core.RegistrationFailureReasonRegistrarUnreachable
	RegistrationFailureReasonTransportError          = internal_core.RegistrationFailureReasonTransportError
	RegistrationFailureReasonRegisterTimeout         = internal_core.RegistrationFailureReasonRegisterTimeout
	RegistrationFailureReasonRenewalFailed           = internal_core.RegistrationFailureReasonRenewalFailed
	RegistrationFailureReasonUnregisterFailed        = internal_core.RegistrationFailureReasonUnregisterFailed
	RegistrationFailureReasonInvalidContactAddress   = internal_core.RegistrationFailureReasonInvalidContactAddress
)

type RegistrationError = internal_core.RegistrationError
type RegistrationEvent = internal_core.RegistrationEvent
type RegistrationObserver = internal_core.RegistrationObserver
type RegistrationSnapshot = internal_core.RegistrationSnapshot

type Registration = internal_core.Registration

type RegistrationClient struct {
	inner *internal_core.RegistrationClient
}
