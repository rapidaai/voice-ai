// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_entity

import (
	"reflect"
	"strings"
	"testing"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

func TestBillingTableNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "plan", got: (BillingPlan{}).TableName(), want: "billing_plans"},
		{name: "quota", got: (BillingPlanQuota{}).TableName(), want: "billing_plan_quotas"},
		{name: "subscription", got: (BillingSubscription{}).TableName(), want: "billing_subscriptions"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("TableName() = %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestBillingAuditOwnership(t *testing.T) {
	for _, entity := range []any{BillingPlan{}, BillingPlanQuota{}, BillingSubscription{}} {
		entityType := reflect.TypeOf(entity)
		audited, ok := entityType.FieldByName("Audited")
		if !ok || !audited.Anonymous || audited.Type != reflect.TypeOf(gorm_model.Audited{}) {
			t.Fatalf("%s must embed gorm_model.Audited", entityType.Name())
		}
		mutable, hasMutable := entityType.FieldByName("Mutable")
		if entityType.Name() == "BillingSubscription" {
			if !hasMutable || !mutable.Anonymous || mutable.Type != reflect.TypeOf(gorm_model.Mutable{}) {
				t.Fatal("BillingSubscription must embed gorm_model.Mutable")
			}
		} else if hasMutable {
			t.Fatalf("%s must not embed gorm_model.Mutable", entityType.Name())
		}
	}
}

func TestBillingPersistedColumnsAndAssociations(t *testing.T) {
	tests := []struct {
		entity any
		fields map[string]string
	}{
		{
			entity: BillingPlan{},
			fields: map[string]string{
				"Name": "name", "Slug": "slug", "Description": "description", "IsDefault": "is_default",
				"IsActive": "is_active", "SortOrder": "sort_order", "PriceMonthly": "price_monthly",
				"PriceYearly": "price_yearly", "Currency": "currency", "StripeURL": "stripe_url",
			},
		},
		{
			entity: BillingPlanQuota{},
			fields: map[string]string{
				"BillingPlanID": "billing_plan_id", "ResourceType": "resource_type", "QuotaLimit": "quota_limit", "Unit": "unit",
			},
		},
		{
			entity: BillingSubscription{},
			fields: map[string]string{
				"OrganizationID": "organization_id", "BillingPlanID": "billing_plan_id", "BillingInterval": "billing_interval",
				"CurrentPeriodStart": "current_period_start", "CurrentPeriodEnd": "current_period_end", "Metadata": "metadata",
			},
		},
	}

	for _, test := range tests {
		entityType := reflect.TypeOf(test.entity)
		for fieldName, column := range test.fields {
			field, ok := entityType.FieldByName(fieldName)
			if !ok || !strings.Contains(field.Tag.Get("gorm"), "column:"+column) {
				t.Fatalf("%s.%s gorm tag = %q, want column:%s", entityType.Name(), fieldName, field.Tag.Get("gorm"), column)
			}
		}
	}

	quotas, ok := reflect.TypeOf(BillingPlan{}).FieldByName("Quotas")
	if !ok || quotas.Type != reflect.TypeOf([]*BillingPlanQuota(nil)) || !strings.Contains(quotas.Tag.Get("gorm"), "foreignKey:BillingPlanID") {
		t.Fatalf("BillingPlan.Quotas = %v %q", quotas.Type, quotas.Tag.Get("gorm"))
	}
	plan, ok := reflect.TypeOf(BillingSubscription{}).FieldByName("Plan")
	if !ok || plan.Type != reflect.TypeOf((*BillingPlan)(nil)) || !strings.Contains(plan.Tag.Get("gorm"), "foreignKey:BillingPlanID") {
		t.Fatalf("BillingSubscription.Plan = %v %q", plan.Type, plan.Tag.Get("gorm"))
	}
}
