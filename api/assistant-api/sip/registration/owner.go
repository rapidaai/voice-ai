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

	"github.com/rapidaai/api/assistant-api/internal/observability"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/redis/go-redis/v9"
)

// handleClaimOwnership implements "Check if owner is not there -> create
// owner". On a fresh claim or self-owned refresh returns RegisterPipeline; on
// peer-owned or claim error returns nil to stop the chain. Mirrors the
// per-type handler signature of sip/pipeline/registration.go.
func (m *manager) handleClaimOwnership(ctx context.Context, s ClaimOwnershipPipeline) Pipeline {
	rec := s.Record
	key := OwnerKeyPrefix + rec.DID

	claimed, err := m.redis.SetNX(ctx, key, m.instanceID, OwnershipTTL).Result()
	if err != nil {
		rec.Outcome = OutcomeClaimError
		m.logger.Warnw("Ownership claim failed", "did", rec.DID, "error", err)
		auth := projectAuthentication(rec.OrganizationID, rec.ProjectID)
		observer := m.observer(ctx, auth)
		defer observer.Close(context.Background())
		attributes := observability.Attributes{
			"did":            rec.DID,
			"assistant_id":   strconv.FormatUint(rec.AssistantID, 10),
			"deployment_id":  strconv.FormatUint(rec.DeploymentID, 10),
			"credential_id":  strconv.FormatUint(rec.CredentialID, 10),
			"owner":          m.instanceID,
			"failure_class":  string(RegistrationFailureClassOwnership),
			"failure_reason": string(RegistrationFailureReasonOwnershipClaimFailed),
			"error":          err.Error(),
		}
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: rec.AssistantID},
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration ownership claim failed",
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
					Description: "SIP registration ownership claim failed",
				}},
				Attributes: attributes,
			},
		)
		m.writeRegistrationStatus(ctx, rec.DeploymentID, RegistrationStatusUpdate{
			Error:         err.Error(),
			FailureClass:  RegistrationFailureClassOwnership,
			FailureReason: RegistrationFailureReasonOwnershipClaimFailed,
			OwnerInstance: m.instanceID,
		})
		return nil
	}
	if claimed {
		m.logger.Debugw("DID ownership claimed",
			"did", rec.DID, "owner", m.instanceID, "ttl", OwnershipTTL)
		return RegisterPipeline{Record: rec}
	}

	cur, err := m.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// Race: key expired between SETNX and GET. One more attempt.
		again, _ := m.redis.SetNX(ctx, key, m.instanceID, OwnershipTTL).Result()
		if again {
			m.logger.Debugw("DID ownership claimed (post-race)",
				"did", rec.DID, "owner", m.instanceID)
			return RegisterPipeline{Record: rec}
		}
		rec.Outcome = OutcomePeerOwned
		return nil
	}
	if err != nil {
		rec.Outcome = OutcomeClaimError
		m.logger.Warnw("Ownership claim failed", "did", rec.DID, "error", err)
		auth := projectAuthentication(rec.OrganizationID, rec.ProjectID)
		observer := m.observer(ctx, auth)
		defer observer.Close(context.Background())
		attributes := observability.Attributes{
			"did":            rec.DID,
			"assistant_id":   strconv.FormatUint(rec.AssistantID, 10),
			"deployment_id":  strconv.FormatUint(rec.DeploymentID, 10),
			"credential_id":  strconv.FormatUint(rec.CredentialID, 10),
			"owner":          m.instanceID,
			"failure_class":  string(RegistrationFailureClassOwnership),
			"failure_reason": string(RegistrationFailureReasonOwnershipClaimFailed),
			"error":          err.Error(),
		}
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: rec.AssistantID},
			observability.RecordLog{
				Level:      observability.LevelError,
				Message:    "SIP registration ownership claim failed",
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
					Description: "SIP registration ownership claim failed",
				}},
				Attributes: attributes,
			},
		)
		m.writeRegistrationStatus(ctx, rec.DeploymentID, RegistrationStatusUpdate{
			Error:         err.Error(),
			FailureClass:  RegistrationFailureClassOwnership,
			FailureReason: RegistrationFailureReasonOwnershipClaimFailed,
			OwnerInstance: m.instanceID,
		})
		return nil
	}
	if cur == m.instanceID {
		// Already ours — extend the lease.
		m.redis.Expire(ctx, key, OwnershipTTL)
		m.logger.Debugw("DID ownership refreshed",
			"did", rec.DID, "owner", m.instanceID, "ttl", OwnershipTTL)
		return RegisterPipeline{Record: rec}
	}
	rec.Outcome = OutcomePeerOwned
	m.logger.Debugw("DID owned by peer instance — skipping",
		"did", rec.DID, "owner", cur, "self", m.instanceID)
	return nil
}

// releaseOwner deletes the ownership key if (and only if) we still own it.
// Used by the reconcile cleanup branch and by ReleaseAll on graceful
// shutdown so a peer can claim immediately rather than waiting for the TTL.
func (m *manager) releaseOwner(ctx context.Context, did string) {
	if m.redis == nil {
		return
	}
	key := OwnerKeyPrefix + did
	cur, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		return
	}
	if cur != m.instanceID {
		return
	}
	m.redis.Del(ctx, key)
	m.logger.Debugw("DID ownership released", "did", did, "owner", m.instanceID)
}
