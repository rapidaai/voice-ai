// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"strconv"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

// handleMarkActive implements the "Update status" pipeline step on the
// success path: clears any prior error and resets the transient retry
// counter. Skips DB writes if the DID was already locally active (renewal
// loop carries the binding) — avoids one upsert tuple per tick per DID at
// scale. Always terminal; returns nil.
func (m *manager) handleMarkActive(ctx context.Context, p MarkActivePipeline) Pipeline {
	rec := p.Record
	if rec.Outcome == OutcomeAlreadyActive {
		return nil
	}
	retryCount := 0
	now := time.Now().UTC()
	m.writeRegistrationStatus(ctx, rec.DeploymentID, RegistrationStatusUpdate{
		Status:        StatusActive,
		Error:         "",
		RetryCount:    &retryCount,
		OwnerInstance: m.instanceID,
		LastSuccessAt: now,
	})
	return nil
}

func (m *manager) writeRegistrationStatus(ctx context.Context, deploymentID uint64, update RegistrationStatusUpdate) {
	if validator.NotBlank(string(update.Status)) {
		m.upsertOption(ctx, deploymentID, OptKeySIPStatus, string(update.Status))
	}
	if validator.NotBlank(update.Error) || update.Status == StatusActive {
		m.upsertOption(ctx, deploymentID, OptKeySIPError, update.Error)
	}
	if validator.NotBlank(string(update.FailureClass)) || update.Status == StatusActive {
		m.upsertOption(ctx, deploymentID, OptKeySIPFailureClass, string(update.FailureClass))
	}
	if validator.NotBlank(string(update.FailureReason)) || update.Status == StatusActive {
		m.upsertOption(ctx, deploymentID, OptKeySIPFailureReason, string(update.FailureReason))
	}
	if update.ResponseCode > 0 || update.Status == StatusActive {
		responseCode := ""
		if update.ResponseCode > 0 {
			responseCode = strconv.Itoa(update.ResponseCode)
		}
		m.upsertOption(ctx, deploymentID, OptKeySIPResponseCode, responseCode)
	}
	if validator.NotBlank(update.ResponseText) || update.Status == StatusActive {
		m.upsertOption(ctx, deploymentID, OptKeySIPResponseText, update.ResponseText)
	}
	if validator.NonNil(update.RetryCount) {
		m.upsertOption(ctx, deploymentID, OptKeySIPRetry, strconv.Itoa(*update.RetryCount))
	}
	if !update.LastAttemptAt.IsZero() {
		m.upsertOption(ctx, deploymentID, OptKeySIPLastAttemptAt, formatRegistrationTime(update.LastAttemptAt))
	}
	if !update.NextRetryAt.IsZero() || update.Status == StatusActive {
		m.upsertOption(ctx, deploymentID, OptKeySIPNextRetryAt, formatRegistrationTime(update.NextRetryAt))
	}
	if validator.NotBlank(update.OwnerInstance) {
		m.upsertOption(ctx, deploymentID, OptKeySIPOwnerInstance, update.OwnerInstance)
	}
	if !update.LastSuccessAt.IsZero() {
		m.upsertOption(ctx, deploymentID, OptKeySIPLastSuccessAt, formatRegistrationTime(update.LastSuccessAt))
	}
}

// upsertOption mirrors the upsert pattern used by CreatePhoneDeployment so
// existing rows are updated in place rather than duplicated.
func (m *manager) upsertOption(ctx context.Context, deploymentID uint64, key, value string) {
	db := m.postgres.DB(ctx)
	opt := &internal_assistant_entity.AssistantDeploymentTelephonyOption{
		AssistantDeploymentTelephonyId: deploymentID,
		Mutable: gorm_models.Mutable{
			CreatedActorType: types.ActorTypeService.String(),
			CreatedActorID:   m.assistantConfig.ServiceID,
		},
		Metadata: gorm_models.Metadata{
			Key:   key,
			Value: value,
		},
	}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_telephony_id"}, {Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"value":              clause.Expr{SQL: "excluded.value"},
			"updated_date":       time.Now(),
			"updated_actor_type": types.ActorTypeService.String(),
			"updated_actor_id":   m.assistantConfig.ServiceID,
		}),
	}).Create(opt).Error; err != nil {
		m.logger.Warnw("Failed to upsert deployment option",
			"deployment_id", deploymentID, "key", key, "error", err)
	}
}

func formatRegistrationTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// handleTransient bumps the retry counter for transport / 5xx style errors.
// After MaxTransientRetries the deployment is marked unreachable so subsequent
// reconciles short-circuit it via the terminal-status filter in loadRecords.
func (m *manager) handleTransient(ctx context.Context, rec *Record, err error) {
	db := m.postgres.DB(ctx)
	var opt internal_assistant_entity.AssistantDeploymentTelephonyOption
	retry := 0
	if dbErr := db.Where("assistant_deployment_telephony_id = ? AND key = ?",
		rec.DeploymentID, OptKeySIPRetry).First(&opt).Error; dbErr == nil {
		retry, _ = strconv.Atoi(opt.Value)
	}
	retry++
	statusUpdate := m.registrationStatusUpdateFromError(err)
	statusUpdate.RetryCount = &retry
	statusUpdate.LastAttemptAt = time.Now().UTC()
	statusUpdate.NextRetryAt = statusUpdate.LastAttemptAt.Add(PollInterval)
	statusUpdate.OwnerInstance = m.instanceID

	if retry >= MaxTransientRetries {
		m.logger.Errorw("SIP registration unreachable after max retries — will not retry",
			"did", rec.DID, "assistant_id", rec.AssistantID, "retries", retry, "error", err)
		statusUpdate.Status = StatusUnreachable
		auth := m.serviceAuthentication(rec.OrganizationID, rec.ProjectID)
		observer := m.observer(ctx, auth)
		defer observer.Close(context.Background())
		attributes := observability.Attributes{
			"did":            rec.DID,
			"assistant_id":   strconv.FormatUint(rec.AssistantID, 10),
			"deployment_id":  strconv.FormatUint(rec.DeploymentID, 10),
			"credential_id":  strconv.FormatUint(rec.CredentialID, 10),
			"owner":          m.instanceID,
			"failure_class":  string(statusUpdate.FailureClass),
			"failure_reason": string(statusUpdate.FailureReason),
			"retry_count":    strconv.Itoa(retry),
			"error":          err.Error(),
		}
		if statusUpdate.ResponseCode > 0 {
			attributes["response_code"] = strconv.Itoa(statusUpdate.ResponseCode)
		}
		if statusUpdate.ResponseText != "" {
			attributes["response_text"] = statusUpdate.ResponseText
		}
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: rec.AssistantID},
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration unreachable after max retries",
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
					Description: "SIP registration unreachable after max retries",
				}},
				Attributes: attributes,
			},
		)
		m.writeRegistrationStatus(ctx, rec.DeploymentID, statusUpdate)
		return
	}

	m.logger.Warnw("SIP registration failed (will retry)",
		"did", rec.DID, "assistant_id", rec.AssistantID, "retry", retry, "error", err)
	auth := m.serviceAuthentication(rec.OrganizationID, rec.ProjectID)
	observer := m.observer(ctx, auth)
	defer observer.Close(context.Background())
	attributes := observability.Attributes{
		"did":            rec.DID,
		"assistant_id":   strconv.FormatUint(rec.AssistantID, 10),
		"deployment_id":  strconv.FormatUint(rec.DeploymentID, 10),
		"credential_id":  strconv.FormatUint(rec.CredentialID, 10),
		"owner":          m.instanceID,
		"failure_class":  string(statusUpdate.FailureClass),
		"failure_reason": string(statusUpdate.FailureReason),
		"retry_count":    strconv.Itoa(retry),
		"error":          err.Error(),
	}
	if statusUpdate.ResponseCode > 0 {
		attributes["response_code"] = strconv.Itoa(statusUpdate.ResponseCode)
	}
	if statusUpdate.ResponseText != "" {
		attributes["response_text"] = statusUpdate.ResponseText
	}
	_ = observer.Record(ctx, observability.AssistantScope{AssistantID: rec.AssistantID},
		observability.RecordLog{
			Level:      observability.LevelError,
			Message:    "SIP registration failed and will retry",
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
				Description: "SIP registration failed and will retry",
			}},
			Attributes: attributes,
		},
	)
	m.writeRegistrationStatus(ctx, rec.DeploymentID, statusUpdate)
}
