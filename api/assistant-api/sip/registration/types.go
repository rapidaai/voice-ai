// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
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
	OptKeyExtension        = "extension"
	OptKeyCallerID         = "caller_id"
	OptKeyCredentialID     = "rapida.credential_id"
	OptKeySIPConfigHash    = "rapida.sip_config_hash"
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

type RegistrationFailureClass = sip_infra.RegistrationFailureClass

const (
	RegistrationFailureClassConfig     = sip_infra.RegistrationFailureClassConfig
	RegistrationFailureClassAuth       = sip_infra.RegistrationFailureClassAuth
	RegistrationFailureClassRejected   = sip_infra.RegistrationFailureClassRejected
	RegistrationFailureClassTransient  = sip_infra.RegistrationFailureClassTransient
	RegistrationFailureClassNetwork    = sip_infra.RegistrationFailureClassNetwork
	RegistrationFailureClassOwnership  = sip_infra.RegistrationFailureClassOwnership
	RegistrationFailureClassDuplicate  = sip_infra.RegistrationFailureClassDuplicate
	RegistrationFailureClassRenewal    = sip_infra.RegistrationFailureClassRenewal
	RegistrationFailureClassUnregister = sip_infra.RegistrationFailureClassUnregister
)

type RegistrationFailureReason = sip_infra.RegistrationFailureReason

const (
	RegistrationFailureReasonMissingDID              = sip_infra.RegistrationFailureReasonMissingDID
	RegistrationFailureReasonMissingCredentialID     = sip_infra.RegistrationFailureReasonMissingCredentialID
	RegistrationFailureReasonDuplicateDID            = sip_infra.RegistrationFailureReasonDuplicateDID
	RegistrationFailureReasonAssistantNotFound       = sip_infra.RegistrationFailureReasonAssistantNotFound
	RegistrationFailureReasonVaultCredentialNotFound = sip_infra.RegistrationFailureReasonVaultCredentialNotFound
	RegistrationFailureReasonInvalidSIPConfig        = sip_infra.RegistrationFailureReasonInvalidSIPConfig
	RegistrationFailureReasonMissingSIPServer        = sip_infra.RegistrationFailureReasonMissingSIPServer
	RegistrationFailureReasonOwnershipClaimFailed    = sip_infra.RegistrationFailureReasonOwnershipClaimFailed
	RegistrationFailureReasonAuthFailed              = sip_infra.RegistrationFailureReasonAuthFailed
	RegistrationFailureReasonRegistrarRejected       = sip_infra.RegistrationFailureReasonRegistrarRejected
	RegistrationFailureReasonRegistrarUnreachable    = sip_infra.RegistrationFailureReasonRegistrarUnreachable
	RegistrationFailureReasonTransportError          = sip_infra.RegistrationFailureReasonTransportError
	RegistrationFailureReasonRegisterTimeout         = sip_infra.RegistrationFailureReasonRegisterTimeout
	RegistrationFailureReasonRenewalFailed           = sip_infra.RegistrationFailureReasonRenewalFailed
	RegistrationFailureReasonUnregisterFailed        = sip_infra.RegistrationFailureReasonUnregisterFailed
	RegistrationFailureReasonInvalidContactAddress   = sip_infra.RegistrationFailureReasonInvalidContactAddress
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
	ConfigHash    string    // Fingerprint of the config this verdict was reached with.
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
	// ConfigHash fingerprints the registration-relevant options so a terminal
	// failure can be retried automatically once the config actually changes.
	ConfigHash string
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
	RegistrationClient *sip_infra.RegistrationClient
	AssistantConfig    *config.AssistantConfig
	Sip                *config.SIPConfig
	ApplyOpDefaults    func(*sip_infra.Config)
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

func WithRegistrationClient(registrationClient *sip_infra.RegistrationClient) ManagerOption {
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

func WithApplyOpDefaults(applyOpDefaults func(*sip_infra.Config)) ManagerOption {
	return func(options *ManagerOptions) {
		options.ApplyOpDefaults = applyOpDefaults
	}
}
