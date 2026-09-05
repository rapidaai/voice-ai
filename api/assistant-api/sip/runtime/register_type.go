// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rapidaai/pkg/validator"
)

// Registration errors
var (
	ErrRegistrationFailed  = errors.New("SIP registration failed")
	ErrRegistrationExpired = errors.New("SIP registration expired")
	ErrDIDNotRegistered    = errors.New("DID is not registered")
	ErrMissingDID          = errors.New("DID is required for registration")
	ErrMissingServer       = errors.New("SIP server is required for registration")
	ErrAuthFailed          = errors.New("SIP authentication failed")
	ErrPermanentFailure    = errors.New("SIP registration permanently rejected")
)

type RegistrationFailureClass string

const (
	RegistrationFailureClassConfig     RegistrationFailureClass = "config"
	RegistrationFailureClassAuth       RegistrationFailureClass = "auth"
	RegistrationFailureClassRejected   RegistrationFailureClass = "rejected"
	RegistrationFailureClassTransient  RegistrationFailureClass = "transient"
	RegistrationFailureClassNetwork    RegistrationFailureClass = "network"
	RegistrationFailureClassOwnership  RegistrationFailureClass = "ownership"
	RegistrationFailureClassDuplicate  RegistrationFailureClass = "duplicate"
	RegistrationFailureClassRenewal    RegistrationFailureClass = "renewal"
	RegistrationFailureClassUnregister RegistrationFailureClass = "unregister"
)

type RegistrationFailureReason string

const (
	RegistrationFailureReasonMissingDID          RegistrationFailureReason = "missing_did"
	RegistrationFailureReasonMissingCredentialID RegistrationFailureReason = "missing_credential_id"
	RegistrationFailureReasonDuplicateDID        RegistrationFailureReason = "duplicate_did"
	RegistrationFailureReasonAssistantNotFound   RegistrationFailureReason = "assistant_not_found"
	// #nosec G101, this is a registration state enum, not a credential.
	RegistrationFailureReasonVaultCredentialNotFound RegistrationFailureReason = "vault_credential_not_found"
	RegistrationFailureReasonInvalidSIPConfig        RegistrationFailureReason = "invalid_sip_config"
	RegistrationFailureReasonMissingSIPServer        RegistrationFailureReason = "missing_sip_server"
	RegistrationFailureReasonOwnershipClaimFailed    RegistrationFailureReason = "ownership_claim_failed"
	RegistrationFailureReasonAuthFailed              RegistrationFailureReason = "auth_failed"
	RegistrationFailureReasonRegistrarRejected       RegistrationFailureReason = "registrar_rejected"
	RegistrationFailureReasonRegistrarUnreachable    RegistrationFailureReason = "registrar_unreachable"
	RegistrationFailureReasonTransportError          RegistrationFailureReason = "transport_error"
	RegistrationFailureReasonRegisterTimeout         RegistrationFailureReason = "register_timeout"
	RegistrationFailureReasonRenewalFailed           RegistrationFailureReason = "renewal_failed"
	RegistrationFailureReasonUnregisterFailed        RegistrationFailureReason = "unregister_failed"
	RegistrationFailureReasonInvalidContactAddress   RegistrationFailureReason = "invalid_contact_address"
)

type RegistrationError struct {
	Class      RegistrationFailureClass
	Reason     RegistrationFailureReason
	StatusCode int
	StatusText string
	Retryable  bool
	Cause      error
}

func (err *RegistrationError) Error() string {
	if !validator.NonNil(err) {
		return ""
	}
	message := fmt.Sprintf("%s: %s", err.Class, err.Reason)
	if err.StatusCode > 0 {
		message = fmt.Sprintf("%s: %d %s", message, err.StatusCode, err.StatusText)
	}
	if err.Cause != nil {
		message = fmt.Sprintf("%s: %v", message, err.Cause)
	}
	return message
}

func (err *RegistrationError) Unwrap() error {
	if !validator.NonNil(err) {
		return nil
	}
	return err.Cause
}

type RegistrationEvent struct {
	DID           string
	DeploymentID  uint64
	AssistantID   uint64
	Server        string
	ExpiresAt     time.Time
	GrantedExpiry uint32
	RetryCount    int
	NextRetryAt   time.Time
	Error         error
	FailureClass  RegistrationFailureClass
	FailureReason RegistrationFailureReason
	StatusCode    int
	StatusText    string
}

type RegistrationObserver interface {
	RegistrationRenewed(ctx context.Context, event RegistrationEvent)
	RegistrationRenewalFailed(ctx context.Context, event RegistrationEvent)
	RegistrationExpired(ctx context.Context, event RegistrationEvent)
	RegistrationUnregisterFailed(ctx context.Context, event RegistrationEvent)
}

type RegistrationSnapshot struct {
	DID               string
	Active            bool
	Healthy           bool
	ExpiresAt         time.Time
	RenewalRetryCount int
	LastRenewalError  error
	FailureClass      RegistrationFailureClass
	FailureReason     RegistrationFailureReason
}
