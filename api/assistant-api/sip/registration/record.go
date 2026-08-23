// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// loadRecords implements the "GetRecordToRegister" pipeline entry point.
// Returns the desired-state Records — only the latest active phone deployment
// per assistant (older versions are archived by CreatePhoneDeployment) that
// has SIP inbound enabled and has not entered a terminal status. Records
// whose DID collides with another assistant's are dropped with a WARN — the
// schema does not enforce phone-value uniqueness, so the misconfiguration
// fails loudly (DID unreachable) rather than routing non-deterministically.
func (m *manager) loadRecords(ctx context.Context) ([]Record, error) {
	var deployments []internal_assistant_entity.AssistantPhoneDeployment
	if err := m.postgres.DB(ctx).
		Preload("TelephonyOption").
		Where("telephony_provider = ? AND status = ?", "sip", type_enums.RECORD_ACTIVE).
		Find(&deployments).Error; err != nil {
		return nil, err
	}

	assistantIDs := make([]uint64, 0, len(deployments))
	for _, dep := range deployments {
		assistantIDs = append(assistantIDs, dep.AssistantId)
	}
	var assistants []internal_assistant_entity.Assistant
	assistantByID := make(map[uint64]internal_assistant_entity.Assistant, len(assistantIDs))
	if len(assistantIDs) > 0 {
		if err := m.postgres.DB(ctx).Where("id IN ?", assistantIDs).Find(&assistants).Error; err == nil {
			for _, assistant := range assistants {
				assistantByID[assistant.Id] = assistant
			}
		}
	}

	type candidate struct {
		record     Record
		deployment uint64
		assistant  uint64
		sipStatus  string
	}

	grouped := make(map[string][]candidate, len(deployments))
	for _, dep := range deployments {
		opts := dep.GetOptions()

		sipStatus, _ := opts.GetString(OptKeySIPStatus)
		if isTerminalRegistrationStatus(RegistrationStatus(sipStatus)) {
			continue
		}

		sipInbound, _ := opts.GetString(OptKeySIPInbound)
		if !validator.NotBlank(sipInbound) || sipInbound != "true" {
			continue
		}

		did, _ := opts.GetString(OptKeyPhone)
		if !validator.NotBlank(did) {
			if assistant, ok := assistantByID[dep.AssistantId]; ok {
				auth := m.serviceAuthentication(assistant.OrganizationId, assistant.ProjectId)
				observer := m.observer(ctx, auth)
				attributes := observability.Attributes{
					"assistant_id":   strconv.FormatUint(dep.AssistantId, 10),
					"deployment_id":  strconv.FormatUint(dep.Id, 10),
					"owner":          m.instanceID,
					"failure_class":  string(RegistrationFailureClassConfig),
					"failure_reason": string(RegistrationFailureReasonMissingDID),
					"error":          "phone is required for SIP registration",
				}
				_ = observer.Record(ctx, observability.AssistantScope{AssistantID: dep.AssistantId},
					observability.RecordLog{
						Level:      observability.LevelError,
						Message:    "SIP registration phone is missing",
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
							Description: "SIP registration phone is missing",
						}},
						Attributes: attributes,
					},
				)
				_ = observer.Close(context.Background())
			}
			m.writeRegistrationStatus(ctx, dep.Id, RegistrationStatusUpdate{
				Status:        StatusConfigError,
				Error:         "phone is required for SIP registration",
				FailureClass:  RegistrationFailureClassConfig,
				FailureReason: RegistrationFailureReasonMissingDID,
				OwnerInstance: m.instanceID,
			})
			continue
		}
		credentialID, err := opts.GetUint64(OptKeyCredentialID)
		if err != nil {
			if assistant, ok := assistantByID[dep.AssistantId]; ok {
				auth := m.serviceAuthentication(assistant.OrganizationId, assistant.ProjectId)
				observer := m.observer(ctx, auth)
				attributes := observability.Attributes{
					"did":            did,
					"assistant_id":   strconv.FormatUint(dep.AssistantId, 10),
					"deployment_id":  strconv.FormatUint(dep.Id, 10),
					"owner":          m.instanceID,
					"failure_class":  string(RegistrationFailureClassConfig),
					"failure_reason": string(RegistrationFailureReasonMissingCredentialID),
					"error":          "credential_id is required for SIP registration",
				}
				_ = observer.Record(ctx, observability.AssistantScope{AssistantID: dep.AssistantId},
					observability.RecordLog{
						Level:      observability.LevelError,
						Message:    "SIP registration credential is missing",
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
							Description: "SIP registration credential is missing",
						}},
						Attributes: attributes,
					},
				)
				_ = observer.Close(context.Background())
			}
			m.writeRegistrationStatus(ctx, dep.Id, RegistrationStatusUpdate{
				Status:        StatusConfigError,
				Error:         "credential_id is required for SIP registration",
				FailureClass:  RegistrationFailureClassConfig,
				FailureReason: RegistrationFailureReasonMissingCredentialID,
				OwnerInstance: m.instanceID,
			})
			continue
		}

		rec := Record{
			DID:          did,
			AssistantID:  dep.AssistantId,
			DeploymentID: dep.Id,
			CredentialID: credentialID,
			Status:       sipStatus,
		}
		if assistant, ok := assistantByID[dep.AssistantId]; ok {
			rec.ProjectID = assistant.ProjectId
			rec.OrganizationID = assistant.OrganizationId
		}
		key := normalizeDIDForCollision(did)
		grouped[key] = append(grouped[key], candidate{
			record:     rec,
			deployment: dep.Id,
			assistant:  dep.AssistantId,
			sipStatus:  sipStatus,
		})
	}

	selected := make([]Record, 0, len(grouped))
	for didKey, list := range grouped {
		if len(list) == 1 {
			selected = append(selected, list[0].record)
			continue
		}

		// Deterministic winner:
		// 1) keep already active deployment to avoid flapping
		// 2) otherwise latest deployment id
		// 3) finally highest assistant id as stable tie-break
		sort.Slice(list, func(i, j int) bool {
			iActive := RegistrationStatus(list[i].sipStatus) == StatusActive
			jActive := RegistrationStatus(list[j].sipStatus) == StatusActive
			if iActive != jActive {
				return iActive
			}
			if list[i].deployment != list[j].deployment {
				return list[i].deployment > list[j].deployment
			}
			return list[i].assistant > list[j].assistant
		})

		winner := list[0]
		selected = append(selected, winner.record)

		m.logger.Warnw("Duplicate SIP DID detected; keeping one deployment and dropping others",
			"did", didKey,
			"winner_assistant_id", winner.assistant,
			"winner_deployment_id", winner.deployment,
			"dropped_count", len(list)-1)

		for _, loser := range list[1:] {
			reason := fmt.Sprintf(
				"Duplicate DID %s. Inbound registration skipped: kept assistant=%d deployment=%d",
				didKey, winner.assistant, winner.deployment,
			)
			if loser.record.ProjectID != 0 && loser.record.OrganizationID != 0 {
				auth := m.serviceAuthentication(loser.record.OrganizationID, loser.record.ProjectID)
				observer := m.observer(ctx, auth)
				attributes := observability.Attributes{
					"did":                  didKey,
					"assistant_id":         strconv.FormatUint(loser.record.AssistantID, 10),
					"deployment_id":        strconv.FormatUint(loser.record.DeploymentID, 10),
					"credential_id":        strconv.FormatUint(loser.record.CredentialID, 10),
					"winner_assistant_id":  strconv.FormatUint(winner.assistant, 10),
					"winner_deployment_id": strconv.FormatUint(winner.deployment, 10),
					"owner":                m.instanceID,
					"failure_class":        string(RegistrationFailureClassDuplicate),
					"failure_reason":       string(RegistrationFailureReasonDuplicateDID),
					"error":                reason,
				}
				_ = observer.Record(ctx, observability.AssistantScope{AssistantID: loser.record.AssistantID},
					observability.RecordLog{
						Level:      observability.LevelError,
						Message:    "Duplicate SIP DID detected",
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
							Description: "Duplicate SIP DID detected",
						}},
						Attributes: attributes,
					},
				)
				_ = observer.Close(context.Background())
			}
			retryCount := 0
			m.writeRegistrationStatus(ctx, loser.deployment, RegistrationStatusUpdate{
				Status:        StatusConfigError,
				Error:         reason,
				FailureClass:  RegistrationFailureClassDuplicate,
				FailureReason: RegistrationFailureReasonDuplicateDID,
				RetryCount:    &retryCount,
				OwnerInstance: m.instanceID,
			})
		}
	}

	return selected, nil
}

func normalizeDIDForCollision(did string) string {
	v := strings.TrimSpace(did)
	if !validator.NotBlank(v) {
		return ""
	}
	if strings.HasPrefix(v, "+") {
		return v
	}
	// Keep short internal extensions unchanged; normalize phone-like values.
	if len(v) > 5 {
		return "+" + v
	}
	return v
}
