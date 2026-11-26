// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

// BillingPlan is one platform billing tier.
type BillingPlan struct {
	gorm_model.Audited

	Name         string              `json:"name" gorm:"column:name;type:varchar(100);not null"`
	Slug         string              `json:"slug" gorm:"column:slug;type:varchar(50);not null"`
	Description  string              `json:"description" gorm:"column:description;type:text;not null;default:''"`
	IsDefault    bool                `json:"isDefault" gorm:"column:is_default;type:boolean;not null;default:false"`
	IsActive     bool                `json:"isActive" gorm:"column:is_active;type:boolean;not null;default:true"`
	SortOrder    int32               `json:"sortOrder" gorm:"column:sort_order;type:integer;not null;default:0"`
	PriceMonthly int64               `json:"priceMonthly" gorm:"column:price_monthly;type:bigint;not null;default:0"`
	PriceYearly  int64               `json:"priceYearly" gorm:"column:price_yearly;type:bigint;not null;default:0"`
	Currency     string              `json:"currency" gorm:"column:currency;type:varchar(10);not null;default:'usd'"`
	StripeURL    string              `json:"stripeUrl" gorm:"column:stripe_url;type:text;not null;default:''"`
	Quotas       []*BillingPlanQuota `json:"quotas,omitempty" gorm:"foreignKey:BillingPlanID;references:Id"`
}

// TableName returns the billing plan persistence table.
func (BillingPlan) TableName() string {
	return "billing_plans"
}

// BillingPlanQuota is one resource allowance owned by a BillingPlan.
type BillingPlanQuota struct {
	gorm_model.Audited

	BillingPlanID uint64 `json:"billingPlanId" gorm:"column:billing_plan_id;type:bigint;not null"`
	ResourceType  string `json:"resourceType" gorm:"column:resource_type;type:varchar(100);not null"`
	QuotaLimit    int64  `json:"quotaLimit" gorm:"column:quota_limit;type:bigint;not null"`
	Unit          string `json:"unit" gorm:"column:unit;type:varchar(32);not null"`
}

// TableName returns the billing plan quota persistence table.
func (BillingPlanQuota) TableName() string {
	return "billing_plan_quotas"
}

// BillingSubscription binds one Organization to its current BillingPlan.
type BillingSubscription struct {
	gorm_model.Audited
	gorm_model.Mutable

	OrganizationID     uint64                  `json:"organizationId" gorm:"column:organization_id;type:bigint;not null"`
	BillingPlanID      uint64                  `json:"billingPlanId" gorm:"column:billing_plan_id;type:bigint;not null"`
	BillingInterval    string                  `json:"billingInterval" gorm:"column:billing_interval;type:varchar(20);not null;default:'monthly'"`
	CurrentPeriodStart gorm_model.TimeWrapper  `json:"currentPeriodStart" gorm:"column:current_period_start;type:timestamp;not null"`
	CurrentPeriodEnd   *gorm_model.TimeWrapper `json:"currentPeriodEnd" gorm:"column:current_period_end;type:timestamp"`
	Metadata           gorm_types.InterfaceMap `json:"metadata" gorm:"column:metadata;type:jsonb;not null;default:'{}'"`
	Plan               *BillingPlan            `json:"plan,omitempty" gorm:"foreignKey:BillingPlanID;references:Id"`
}

// TableName returns the Organization billing subscription persistence table.
func (BillingSubscription) TableName() string {
	return "billing_subscriptions"
}
