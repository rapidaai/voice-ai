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
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func (m *manager) RegistrationRenewed(ctx context.Context, event sip_infra.RegistrationEvent) {
	retryCount := 0
	var assistant internal_assistant_entity.Assistant
	if err := m.postgres.DB(ctx).Where("id = ?", event.AssistantID).First(&assistant).Error; err == nil {
		auth := m.serviceAuthentication(assistant.OrganizationId, assistant.ProjectId)
		observer := m.observer(ctx, auth)
		defer observer.Close(context.Background())
		attributes := observability.Attributes{
			"did":            event.DID,
			"assistant_id":   strconv.FormatUint(event.AssistantID, 10),
			"deployment_id":  strconv.FormatUint(event.DeploymentID, 10),
			"owner":          m.instanceID,
			"server":         event.Server,
			"expires_at":     formatRegistrationTime(event.ExpiresAt),
			"granted_expiry": strconv.FormatUint(uint64(event.GrantedExpiry), 10),
			"retry_count":    strconv.Itoa(retryCount),
		}
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: event.AssistantID},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterRenewed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_COMPLETE.String(),
					Description: "SIP registration renewed",
				}},
				Attributes: attributes,
			},
		)
	}
	m.writeRegistrationStatus(ctx, event.DeploymentID, RegistrationStatusUpdate{
		Status:        StatusActive,
		Error:         "",
		RetryCount:    &retryCount,
		OwnerInstance: m.instanceID,
		LastSuccessAt: time.Now().UTC(),
	})
}

func (m *manager) RegistrationRenewalFailed(ctx context.Context, event sip_infra.RegistrationEvent) {
	var assistant internal_assistant_entity.Assistant
	if err := m.postgres.DB(ctx).Where("id = ?", event.AssistantID).First(&assistant).Error; err == nil {
		auth := m.serviceAuthentication(assistant.OrganizationId, assistant.ProjectId)
		observer := m.observer(ctx, auth)
		defer observer.Close(context.Background())
		attributes := observability.Attributes{
			"did":            event.DID,
			"assistant_id":   strconv.FormatUint(event.AssistantID, 10),
			"deployment_id":  strconv.FormatUint(event.DeploymentID, 10),
			"owner":          m.instanceID,
			"server":         event.Server,
			"expires_at":     formatRegistrationTime(event.ExpiresAt),
			"granted_expiry": strconv.FormatUint(uint64(event.GrantedExpiry), 10),
			"retry_count":    strconv.Itoa(event.RetryCount),
			"next_retry_at":  formatRegistrationTime(event.NextRetryAt),
			"failure_class":  string(event.FailureClass),
			"failure_reason": string(event.FailureReason),
			"error":          registrationEventError(event),
		}
		if event.StatusCode > 0 {
			attributes["response_code"] = strconv.Itoa(event.StatusCode)
		}
		if event.StatusText != "" {
			attributes["response_text"] = event.StatusText
		}
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: event.AssistantID},
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration renewal failed",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterRenewalFailed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration renewal failed",
				}},
				Attributes: attributes,
			},
		)
	}
	m.writeRegistrationStatus(ctx, event.DeploymentID, m.registrationStatusUpdateFromEvent(event, StatusActive))
}

func (m *manager) RegistrationExpired(ctx context.Context, event sip_infra.RegistrationEvent) {
	var assistant internal_assistant_entity.Assistant
	if err := m.postgres.DB(ctx).Where("id = ?", event.AssistantID).First(&assistant).Error; err == nil {
		auth := m.serviceAuthentication(assistant.OrganizationId, assistant.ProjectId)
		observer := m.observer(ctx, auth)
		defer observer.Close(context.Background())
		attributes := observability.Attributes{
			"did":            event.DID,
			"assistant_id":   strconv.FormatUint(event.AssistantID, 10),
			"deployment_id":  strconv.FormatUint(event.DeploymentID, 10),
			"owner":          m.instanceID,
			"server":         event.Server,
			"expires_at":     formatRegistrationTime(event.ExpiresAt),
			"granted_expiry": strconv.FormatUint(uint64(event.GrantedExpiry), 10),
			"retry_count":    strconv.Itoa(event.RetryCount),
			"failure_class":  string(event.FailureClass),
			"failure_reason": string(event.FailureReason),
			"error":          registrationEventError(event),
		}
		if event.StatusCode > 0 {
			attributes["response_code"] = strconv.Itoa(event.StatusCode)
		}
		if event.StatusText != "" {
			attributes["response_text"] = event.StatusText
		}
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: event.AssistantID},
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration expired",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPRegisterExpired,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration expired",
				}},
				Attributes: attributes,
			},
		)
	}
	m.writeRegistrationStatus(ctx, event.DeploymentID, m.registrationStatusUpdateFromEvent(event, StatusUnreachable))
	m.releaseOwner(ctx, event.DID)
}

func (m *manager) RegistrationUnregisterFailed(ctx context.Context, event sip_infra.RegistrationEvent) {
	var assistant internal_assistant_entity.Assistant
	if err := m.postgres.DB(ctx).Where("id = ?", event.AssistantID).First(&assistant).Error; err == nil {
		auth := m.serviceAuthentication(assistant.OrganizationId, assistant.ProjectId)
		observer := m.observer(ctx, auth)
		defer observer.Close(context.Background())
		attributes := observability.Attributes{
			"did":            event.DID,
			"assistant_id":   strconv.FormatUint(event.AssistantID, 10),
			"deployment_id":  strconv.FormatUint(event.DeploymentID, 10),
			"owner":          m.instanceID,
			"server":         event.Server,
			"expires_at":     formatRegistrationTime(event.ExpiresAt),
			"granted_expiry": strconv.FormatUint(uint64(event.GrantedExpiry), 10),
			"failure_class":  string(event.FailureClass),
			"failure_reason": string(event.FailureReason),
			"error":          registrationEventError(event),
		}
		if event.StatusCode > 0 {
			attributes["response_code"] = strconv.Itoa(event.StatusCode)
		}
		if event.StatusText != "" {
			attributes["response_text"] = event.StatusText
		}
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: event.AssistantID},
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration unregister failed",
				Attributes: attributes,
			},
			observability.RecordEvent{
				Component:  observability.ComponentSIP,
				Event:      observability.SIPUnregisterFailed,
				Attributes: attributes,
			},
			observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:        observability.MetricSIPRegistrationStatus,
					Value:       type_enums.RECORD_FAILED.String(),
					Description: "SIP registration unregister failed",
				}},
				Attributes: attributes,
			},
		)
	}
	m.writeRegistrationStatus(ctx, event.DeploymentID, m.registrationStatusUpdateFromEvent(event, StatusActive))
}

func (m *manager) registrationStatusUpdateFromEvent(event sip_infra.RegistrationEvent, status RegistrationStatus) RegistrationStatusUpdate {
	return RegistrationStatusUpdate{
		Status:        status,
		Error:         registrationEventError(event),
		FailureClass:  event.FailureClass,
		FailureReason: event.FailureReason,
		ResponseCode:  event.StatusCode,
		ResponseText:  event.StatusText,
		RetryCount:    &event.RetryCount,
		LastAttemptAt: time.Now().UTC(),
		NextRetryAt:   event.NextRetryAt,
		OwnerInstance: m.instanceID,
	}
}

func registrationEventError(event sip_infra.RegistrationEvent) string {
	if event.Error == nil {
		return ""
	}
	return event.Error.Error()
}
