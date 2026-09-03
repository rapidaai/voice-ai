// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

const (
	PollInterval   = 5 * time.Minute
	OwnershipTTL   = 10 * time.Minute
	OwnerKeyPrefix = "sip:registration:owner:"

	MaxConcurrent       = 10
	MaxTransientRetries = 10

	OptKeyPhone            = "phone"
	OptKeyCredentialID     = "rapida.credential_id"
	OptKeySIPStatus        = "rapida.sip_status"
	OptKeySIPError         = "rapida.sip_error"
	OptKeySIPRetry         = "rapida.sip_retry_count"
	OptKeySIPInbound       = "rapida.sip_inbound"
	OptKeySIPFailureClass  = "rapida.sip_failure_class"
	OptKeySIPFailureReason = "rapida.sip_failure_reason"
	OptKeySIPResponseCode  = "rapida.sip_response_code"
	OptKeySIPResponseText  = "rapida.sip_response_text"
	OptKeySIPLastAttemptAt = "rapida.sip_last_attempt_at"
	OptKeySIPNextRetryAt   = "rapida.sip_next_retry_at"
	OptKeySIPOwnerInstance = "rapida.sip_owner_instance"
	OptKeySIPLastSuccessAt = "rapida.sip_last_success_at"
)

type RegistrationStatus string

const (
	StatusActive      RegistrationStatus = "active"
	StatusFailed      RegistrationStatus = "failed"
	StatusRejected    RegistrationStatus = "rejected"
	StatusConfigError RegistrationStatus = "config_error"
	StatusUnreachable RegistrationStatus = "unreachable"
	StatusDisabled    RegistrationStatus = "disabled"
)

func isTerminalRegistrationStatus(status RegistrationStatus) bool {
	switch status {
	case StatusDisabled, StatusRejected, StatusConfigError, StatusUnreachable:
		return true
	default:
		return false
	}
}

type RegistrationFailureClass = sip_runtime.RegistrationFailureClass

const (
	RegistrationFailureClassConfig     = sip_runtime.RegistrationFailureClassConfig
	RegistrationFailureClassAuth       = sip_runtime.RegistrationFailureClassAuth
	RegistrationFailureClassRejected   = sip_runtime.RegistrationFailureClassRejected
	RegistrationFailureClassTransient  = sip_runtime.RegistrationFailureClassTransient
	RegistrationFailureClassNetwork    = sip_runtime.RegistrationFailureClassNetwork
	RegistrationFailureClassOwnership  = sip_runtime.RegistrationFailureClassOwnership
	RegistrationFailureClassDuplicate  = sip_runtime.RegistrationFailureClassDuplicate
	RegistrationFailureClassRenewal    = sip_runtime.RegistrationFailureClassRenewal
	RegistrationFailureClassUnregister = sip_runtime.RegistrationFailureClassUnregister
)

type RegistrationFailureReason = sip_runtime.RegistrationFailureReason

const (
	RegistrationFailureReasonMissingDID              = sip_runtime.RegistrationFailureReasonMissingDID
	RegistrationFailureReasonMissingCredentialID     = sip_runtime.RegistrationFailureReasonMissingCredentialID
	RegistrationFailureReasonDuplicateDID            = sip_runtime.RegistrationFailureReasonDuplicateDID
	RegistrationFailureReasonAssistantNotFound       = sip_runtime.RegistrationFailureReasonAssistantNotFound
	RegistrationFailureReasonVaultCredentialNotFound = sip_runtime.RegistrationFailureReasonVaultCredentialNotFound
	RegistrationFailureReasonInvalidSIPConfig        = sip_runtime.RegistrationFailureReasonInvalidSIPConfig
	RegistrationFailureReasonMissingSIPServer        = sip_runtime.RegistrationFailureReasonMissingSIPServer
	RegistrationFailureReasonOwnershipClaimFailed    = sip_runtime.RegistrationFailureReasonOwnershipClaimFailed
	RegistrationFailureReasonAuthFailed              = sip_runtime.RegistrationFailureReasonAuthFailed
	RegistrationFailureReasonRegistrarRejected       = sip_runtime.RegistrationFailureReasonRegistrarRejected
	RegistrationFailureReasonRegistrarUnreachable    = sip_runtime.RegistrationFailureReasonRegistrarUnreachable
	RegistrationFailureReasonTransportError          = sip_runtime.RegistrationFailureReasonTransportError
	RegistrationFailureReasonRegisterTimeout         = sip_runtime.RegistrationFailureReasonRegisterTimeout
	RegistrationFailureReasonRenewalFailed           = sip_runtime.RegistrationFailureReasonRenewalFailed
	RegistrationFailureReasonUnregisterFailed        = sip_runtime.RegistrationFailureReasonUnregisterFailed
	RegistrationFailureReasonInvalidContactAddress   = sip_runtime.RegistrationFailureReasonInvalidContactAddress
)

// RegistrationStatusUpdate is the single durable write contract for registration visibility.
type RegistrationStatusUpdate struct {
	Status RegistrationStatus // Current deployment-level SIP registration status.
	Error  string             // Human-readable latest registration failure.

	FailureClass  RegistrationFailureClass  // Stable high-level class for filtering and alerts.
	FailureReason RegistrationFailureReason // Stable machine-readable reason for the latest failure.
	ResponseCode  int                       // Registrar SIP response code for the latest attempt.
	ResponseText  string                    // Registrar SIP response text for the latest attempt.

	RetryCount    *int      // Current retry count for transient registration failures.
	LastAttemptAt time.Time // Time of the latest REGISTER attempt.
	NextRetryAt   time.Time // Expected time of the next retry, when retrying.
	OwnerInstance string    // Rapida instance currently owning or attempting the DID.
	LastSuccessAt time.Time // Time of the latest successful REGISTER or renewal.
}

// Record is a single DID-registration work item carried by every Stage. The
// Outcome field is written by handlers (claimed/peer/registered/...) so
// Reconcile can emit a single structured tick-summary log instead of N
// per-record lines.
type Record struct {
	DID            string
	AssistantID    uint64
	ProjectID      uint64
	OrganizationID uint64
	DeploymentID   uint64
	CredentialID   uint64
	Status         string
	Outcome        string
}

// Outcome values written by handlers.
const (
	OutcomePeerOwned     = "peer_owned"
	OutcomeAlreadyActive = "already_active"
	OutcomeRegistered    = "registered"
	OutcomeRejected      = "rejected"
	OutcomeAuthFailed    = "auth_failed"
	OutcomeConfigError   = "config_error"
	OutcomeTransient     = "transient"
	OutcomeClaimError    = "claim_error"
)

// ManagerOptions wires the manager's external dependencies. ApplyOpDefaults overlays
// platform SIP defaults onto the per-DID vault config and is supplied by the
// SIP engine.
type ManagerOptions struct {
	Logger             commons.Logger
	Postgres           connectors.PostgresConnector
	Redis              connectors.RedisConnector
	Vault              web_client.VaultClient
	RegistrationClient *sip_runtime.RegistrationClient
	AssistantConfig    *config.AssistantConfig
	Sip                *config.SIPConfig
	ApplyOpDefaults    func(*sip_runtime.Config)
}

type ManagerOption func(*ManagerOptions)

func WithLogger(logger commons.Logger) ManagerOption {
	return func(options *ManagerOptions) {
		options.Logger = logger
	}
}

func WithPostgres(postgres connectors.PostgresConnector) ManagerOption {
	return func(options *ManagerOptions) {
		options.Postgres = postgres
	}
}

func WithRedis(redis connectors.RedisConnector) ManagerOption {
	return func(options *ManagerOptions) {
		options.Redis = redis
	}
}

func WithVault(vault web_client.VaultClient) ManagerOption {
	return func(options *ManagerOptions) {
		options.Vault = vault
	}
}

func WithRegistrationClient(registrationClient *sip_runtime.RegistrationClient) ManagerOption {
	return func(options *ManagerOptions) {
		options.RegistrationClient = registrationClient
	}
}

func WithAssistantConfig(assistantConfig *config.AssistantConfig) ManagerOption {
	return func(options *ManagerOptions) {
		options.AssistantConfig = assistantConfig
	}
}

func WithSIPConfig(sipConfig *config.SIPConfig) ManagerOption {
	return func(options *ManagerOptions) {
		options.Sip = sipConfig
	}
}

func WithApplyOpDefaults(applyOpDefaults func(*sip_runtime.Config)) ManagerOption {
	return func(options *ManagerOptions) {
		options.ApplyOpDefaults = applyOpDefaults
	}
}
