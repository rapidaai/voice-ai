// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"errors"
	"strconv"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

// handleRegister implements the "Register" pipeline step. Skips if the DID is
// already registered by this instance (renewal loop is healthy) and falls
// through to MarkActivePipeline. On terminal failure (rejected, auth, config)
// the handler writes the matching status itself and returns nil so
// MarkActivePipeline does not overwrite it. Transient failures bump the retry
// counter via handleTransient and also halt the chain.
func (m *manager) handleRegister(ctx context.Context, s RegisterPipeline) Pipeline {
	rec := s.Record

	snapshot := m.regClient.Snapshot(rec.DID)
	if snapshot.Active && snapshot.Healthy {
		rec.Outcome = OutcomeAlreadyActive
		m.logger.Debugw("SIP DID already registered — renewal loop active",
			"did", rec.DID, "assistant_id", rec.AssistantID)
		return nil
	}
	if snapshot.Active && !snapshot.Healthy {
		rec.Outcome = OutcomeAlreadyActive
		m.logger.Warnw("SIP DID registered with renewal failure — preserving failure visibility",
			"did", rec.DID,
			"assistant_id", rec.AssistantID,
			"retry_count", snapshot.RenewalRetryCount,
			"failure_class", snapshot.FailureClass,
			"failure_reason", snapshot.FailureReason)
		return nil
	}

	db := m.postgres.DB(ctx)
	var assistant internal_assistant_entity.Assistant
	if err := db.Where("id = ?", rec.AssistantID).First(&assistant).Error; err != nil {
		rec.Outcome = OutcomeConfigError
		m.logger.Warnw("Failed to load assistant for registration",
			"assistant_id", rec.AssistantID, "did", rec.DID, "error", err)
		m.writeRecordStatus(ctx, rec, RegistrationStatusUpdate{
			Status:        StatusConfigError,
			Error:         "assistant not found",
			FailureClass:  RegistrationFailureClassConfig,
			FailureReason: RegistrationFailureReasonAssistantNotFound,
			OwnerInstance: m.instanceID,
		})
		return nil
	}
	rec.ProjectID = assistant.ProjectId
	rec.OrganizationID = assistant.OrganizationId

	auth := &types.ProjectScope{
		ProjectId:      &assistant.ProjectId,
		OrganizationId: &assistant.OrganizationId,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}
	observer := m.observer(ctx, auth)
	defer observer.Close(context.Background())
	scope := observability.AssistantScope{AssistantID: rec.AssistantID}
	attributes := observability.Attributes{
		"did":           rec.DID,
		"assistant_id":  strconv.FormatUint(rec.AssistantID, 10),
		"deployment_id": strconv.FormatUint(rec.DeploymentID, 10),
		"credential_id": strconv.FormatUint(rec.CredentialID, 10),
		"owner":         m.instanceID,
	}

	vaultCred, err := m.vault.GetCredential(ctx, auth, rec.CredentialID)
	if err != nil {
		rec.Outcome = OutcomeConfigError
		m.logger.Warnw("Failed to fetch vault credential for registration",
			"assistant_id", rec.AssistantID, "did", rec.DID,
			"credential_id", rec.CredentialID, "error", err)
		attributes["failure_class"] = string(RegistrationFailureClassConfig)
		attributes["failure_reason"] = string(RegistrationFailureReasonVaultCredentialNotFound)
		attributes["error"] = err.Error()
		_ = observer.Record(ctx, scope,
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "Failed to fetch SIP registration credential",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterFailed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration credential fetch failed",
				}},
				Attributes: attributes,
			},
		)
		m.writeRecordStatus(ctx, rec, RegistrationStatusUpdate{
			Status:        StatusConfigError,
			Error:         "vault credential not found",
			FailureClass:  RegistrationFailureClassConfig,
			FailureReason: RegistrationFailureReasonVaultCredentialNotFound,
			OwnerInstance: m.instanceID,
		})
		return nil
	}

	sipConfig, err := sip_infra.ParseConfigFromVault(vaultCred)
	if err != nil {
		rec.Outcome = OutcomeConfigError
		m.logger.Warnw("Failed to parse SIP config for registration",
			"assistant_id", rec.AssistantID, "did", rec.DID, "error", err)
		attributes["failure_class"] = string(RegistrationFailureClassConfig)
		attributes["failure_reason"] = string(RegistrationFailureReasonInvalidSIPConfig)
		attributes["error"] = err.Error()
		_ = observer.Record(ctx, scope,
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "Failed to parse SIP registration config",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterFailed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration config parse failed",
				}},
				Attributes: attributes,
			},
		)
		m.writeRecordStatus(ctx, rec, RegistrationStatusUpdate{
			Status:        StatusConfigError,
			Error:         "invalid SIP config: " + err.Error(),
			FailureClass:  RegistrationFailureClassConfig,
			FailureReason: RegistrationFailureReasonInvalidSIPConfig,
			OwnerInstance: m.instanceID,
		})
		return nil
	}
	if m.opDefaults != nil {
		m.opDefaults(sipConfig)
	}
	attributes["server"] = sipConfig.Server
	attributes["domain"] = sipConfig.Domain
	attributes["transport"] = string(sipConfig.Transport)
	attributes["port"] = strconv.Itoa(sipConfig.Port)
	attributes["username"] = sipConfig.Username

	m.logger.Debugw("Registering SIP DID with provider",
		"did", rec.DID,
		"assistant_id", rec.AssistantID,
		"deployment_id", rec.DeploymentID,
		"credential_id", rec.CredentialID,
		"server", sipConfig.Server,
		"domain", sipConfig.Domain,
		"transport", sipConfig.Transport,
		"port", sipConfig.Port,
		"username", sipConfig.Username,
		"owner", m.instanceID)
	_ = observer.Record(ctx, scope,
		observability.RecordEvent{
			Component:  observability.ComponentSIP,
			Event:      observability.SIPRegisterStarted,
			Attributes: attributes,
		},
		observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricSIPRegistrationStatus,
				Value:       type_enums.RECORD_IN_PROGRESS.String(),
				Description: "SIP registration started",
			}},
			Attributes: attributes,
		},
	)

	regErr := m.regClient.Register(ctx, &sip_infra.Registration{
		DID:          rec.DID,
		Config:       sipConfig,
		DeploymentID: rec.DeploymentID,
		AssistantID:  rec.AssistantID,
	})
	if regErr == nil {
		rec.Outcome = OutcomeRegistered
		m.logger.Infow("SIP DID registered",
			"did", rec.DID,
			"assistant_id", rec.AssistantID,
			"server", sipConfig.Server,
			"owner", m.instanceID)
		_ = observer.Record(ctx, scope,
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterActive,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_COMPLETE.String(),
					Description: "SIP registration active",
				}},
				Attributes: attributes,
			},
		)
		return MarkActivePipeline{Record: rec}
	}

	statusUpdate := m.registrationStatusUpdateFromError(regErr)
	attributes["failure_class"] = string(statusUpdate.FailureClass)
	attributes["failure_reason"] = string(statusUpdate.FailureReason)
	attributes["error"] = regErr.Error()
	if statusUpdate.ResponseCode > 0 {
		attributes["response_code"] = strconv.Itoa(statusUpdate.ResponseCode)
	}
	if statusUpdate.ResponseText != "" {
		attributes["response_text"] = statusUpdate.ResponseText
	}
	switch statusUpdate.FailureClass {
	case RegistrationFailureClassRejected:
		rec.Outcome = OutcomeRejected
		m.logger.Errorw("SIP registration permanently rejected — will not retry",
			"did", rec.DID, "assistant_id", rec.AssistantID, "error", regErr)
		_ = observer.Record(ctx, scope,
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration permanently rejected",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterFailed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration permanently rejected",
				}},
				Attributes: attributes,
			},
		)
		m.writeRecordStatus(ctx, rec, statusUpdate)
	case RegistrationFailureClassAuth:
		rec.Outcome = OutcomeAuthFailed
		m.logger.Errorw("SIP registration auth failed — marking deployment as failed",
			"did", rec.DID, "assistant_id", rec.AssistantID, "error", regErr)
		_ = observer.Record(ctx, scope,
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration authentication failed",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterFailed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration authentication failed",
				}},
				Attributes: attributes,
			},
		)
		m.writeRecordStatus(ctx, rec, statusUpdate)
	case RegistrationFailureClassConfig:
		rec.Outcome = OutcomeConfigError
		m.logger.Errorw("SIP registration config failed — will not retry",
			"did", rec.DID, "assistant_id", rec.AssistantID, "error", regErr)
		_ = observer.Record(ctx, scope,
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration config failed",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterFailed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration config failed",
				}},
				Attributes: attributes,
			},
		)
		m.writeRecordStatus(ctx, rec, statusUpdate)
	default:
		rec.Outcome = OutcomeTransient
		m.handleTransient(ctx, rec, regErr)
	}
	return nil
}

func (m *manager) registrationStatusUpdateFromError(err error) RegistrationStatusUpdate {
	now := time.Now().UTC()
	statusUpdate := RegistrationStatusUpdate{
		Error:         err.Error(),
		FailureClass:  RegistrationFailureClassTransient,
		FailureReason: RegistrationFailureReasonRegistrarUnreachable,
		LastAttemptAt: now,
		OwnerInstance: m.instanceID,
	}

	var registrationError *sip_infra.RegistrationError
	if errors.As(err, &registrationError) {
		statusUpdate.FailureClass = registrationError.Class
		statusUpdate.FailureReason = registrationError.Reason
		statusUpdate.ResponseCode = registrationError.StatusCode
		statusUpdate.ResponseText = registrationError.StatusText
	}

	switch {
	case statusUpdate.FailureClass == RegistrationFailureClassConfig:
		statusUpdate.Status = StatusConfigError
	case statusUpdate.FailureClass == RegistrationFailureClassAuth || errors.Is(err, sip_infra.ErrAuthFailed):
		statusUpdate.Status = StatusFailed
		statusUpdate.FailureClass = RegistrationFailureClassAuth
		statusUpdate.FailureReason = RegistrationFailureReasonAuthFailed
	case statusUpdate.FailureClass == RegistrationFailureClassRejected || errors.Is(err, sip_infra.ErrPermanentFailure):
		statusUpdate.Status = StatusRejected
		statusUpdate.FailureClass = RegistrationFailureClassRejected
		statusUpdate.FailureReason = RegistrationFailureReasonRegistrarRejected
	}

	return statusUpdate
}
