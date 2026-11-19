// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_billing_service

import (
	"context"
	"fmt"
	"math"
	"sort"

	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/validator"
)

// NewBillingService constructs the Web API billing persistence service.
func NewBillingService(postgres connectors.PostgresConnector) internal_service.BillingService {
	return &billingService{postgres: postgres}
}

type billingService struct {
	postgres connectors.PostgresConnector
}

// GetSubscriptions loads active subscriptions and their active plans for the requested Organizations.
func (service *billingService) GetSubscriptions(ctx context.Context, organizationIDs []uint64) (map[uint64]internal_entity.BillingSubscription, error) {
	subscriptionsByOrganization := make(map[uint64]internal_entity.BillingSubscription)
	if len(organizationIDs) == 0 {
		return subscriptionsByOrganization, nil
	}

	requestedOrganizations := make(map[uint64]struct{}, len(organizationIDs))
	uniqueOrganizationIDs := make([]uint64, 0, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		if !validator.Between(organizationID, uint64(1), uint64(math.MaxInt64)) {
			return nil, pkg_errors.BillingOrganizationSelectionInvalidError{}
		}
		if _, exists := requestedOrganizations[organizationID]; exists {
			continue
		}
		requestedOrganizations[organizationID] = struct{}{}
		uniqueOrganizationIDs = append(uniqueOrganizationIDs, organizationID)
	}
	sort.Slice(uniqueOrganizationIDs, func(left, right int) bool {
		return uniqueOrganizationIDs[left] < uniqueOrganizationIDs[right]
	})

	subscriptions := make([]internal_entity.BillingSubscription, 0, len(uniqueOrganizationIDs))
	result := service.postgres.DB(ctx).
		Preload("Plan", "is_active = ?", true).
		Preload("Plan.Quotas", func(query *gorm.DB) *gorm.DB {
			return query.Order("billing_plan_quotas.id ASC")
		}).
		Where("organization_id IN ? AND status = ?", uniqueOrganizationIDs, type_enums.RECORD_ACTIVE).
		Order("organization_id ASC, id ASC").
		Find(&subscriptions)
	if result.Error != nil {
		return nil, fmt.Errorf("load billing subscriptions: %w", result.Error)
	}

	for _, subscription := range subscriptions {
		_, requested := requestedOrganizations[subscription.OrganizationID]
		_, duplicate := subscriptionsByOrganization[subscription.OrganizationID]
		if !requested || duplicate || !validator.Between(subscription.Id, uint64(1), uint64(math.MaxInt64)) ||
			!validator.Between(subscription.BillingPlanID, uint64(1), uint64(math.MaxInt64)) ||
			subscription.Plan == nil || subscription.Plan.Id != subscription.BillingPlanID || !subscription.Plan.IsActive {
			return nil, pkg_errors.BillingSubscriptionInvalidError{}
		}
		subscriptionsByOrganization[subscription.OrganizationID] = subscription
	}
	return subscriptionsByOrganization, nil
}
