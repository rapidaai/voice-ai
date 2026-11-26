// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_billing_service

import (
	"context"
	"errors"
	"math"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

type billingTestPostgres struct {
	database *gorm.DB
}

func (*billingTestPostgres) Connect(context.Context) error { return nil }
func (*billingTestPostgres) Name() string                  { return "billing-test" }
func (*billingTestPostgres) IsConnected(context.Context) bool {
	return true
}
func (*billingTestPostgres) Disconnect(context.Context) error { return nil }
func (connector *billingTestPostgres) Query(ctx context.Context, query string, destination interface{}) error {
	return connector.DB(ctx).Raw(query).Scan(destination).Error
}
func (connector *billingTestPostgres) DB(ctx context.Context) *gorm.DB {
	if transaction, ok := connectors.PostgresTxFromContext(ctx); ok {
		return transaction.WithContext(ctx)
	}
	return connector.database.WithContext(ctx)
}

func newBillingServiceTest(t *testing.T) (internal_service.BillingService, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE billing_plans (
			id integer primary key,
			name text not null,
			slug text not null,
			description text not null,
			is_default boolean not null,
			is_active boolean not null,
			sort_order integer not null,
			price_monthly integer not null,
			price_yearly integer not null,
			currency text not null,
			stripe_url text not null,
			metadata text not null default '{}',
			created_date datetime not null default current_timestamp,
			updated_date datetime
		)`,
		`CREATE TABLE billing_plan_quotas (
			id integer primary key,
			billing_plan_id integer not null,
			resource_type text not null,
			quota_limit integer not null,
			unit text not null,
			created_date datetime not null default current_timestamp,
			updated_date datetime
		)`,
		`CREATE TABLE billing_subscriptions (
			id integer primary key,
			organization_id integer not null,
			billing_plan_id integer not null,
			billing_interval text not null,
			current_period_start datetime not null default current_timestamp,
			current_period_end datetime,
			metadata text not null default '{}',
			created_date datetime not null default current_timestamp,
			updated_date datetime,
			status text not null,
			created_actor_type text,
			created_actor_id integer,
			updated_actor_type text,
			updated_actor_id integer
		)`,
		`INSERT INTO billing_plans
			(id, name, slug, description, is_default, is_active, sort_order, price_monthly, price_yearly, currency, stripe_url)
		VALUES
			(100, 'Free', 'free', 'Free plan', true, true, 1, 0, 0, 'usd', ''),
			(200, 'Inactive', 'inactive', 'Inactive plan', false, false, 2, 100, 1000, 'usd', '')`,
		`INSERT INTO billing_plan_quotas (id, billing_plan_id, resource_type, quota_limit, unit) VALUES
			(102, 100, 'minutes', 1000, 'minute'),
			(101, 100, 'projects', 10, 'count')`,
		`INSERT INTO billing_subscriptions
			(id, organization_id, billing_plan_id, billing_interval, status, created_actor_type)
		VALUES
			(1000, 10, 100, 'monthly', 'ACTIVE', 'unknown'),
			(1001, 20, 100, 'monthly', 'ARCHIEVE', 'unknown')`,
	}
	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return NewBillingService(&billingTestPostgres{database: database}), database
}

func TestGetSubscriptionsLoadsActivePlansAndOrderedQuotas(t *testing.T) {
	service, _ := newBillingServiceTest(t)

	subscriptions, err := service.GetSubscriptions(context.Background(), []uint64{30, 10, 10, 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("subscriptions = %+v", subscriptions)
	}
	subscription, ok := subscriptions[10]
	if !ok || subscription.OrganizationID != 10 || subscription.Plan.Id != 100 || subscription.Plan.Slug != "free" {
		t.Fatalf("subscription = %+v", subscription)
	}
	if len(subscription.Plan.Quotas) != 2 || subscription.Plan.Quotas[0].Id != 101 || subscription.Plan.Quotas[1].Id != 102 {
		t.Fatalf("quotas = %+v", subscription.Plan.Quotas)
	}
}

func TestGetSubscriptionsAllowsMissingAndEmptySelections(t *testing.T) {
	service, database := newBillingServiceTest(t)

	subscriptions, err := service.GetSubscriptions(context.Background(), []uint64{30})
	if err != nil || len(subscriptions) != 0 {
		t.Fatalf("missing subscriptions = %+v, error = %v", subscriptions, err)
	}
	if err := database.Exec("DROP TABLE billing_subscriptions").Error; err != nil {
		t.Fatal(err)
	}
	subscriptions, err = service.GetSubscriptions(context.Background(), nil)
	if err != nil || len(subscriptions) != 0 {
		t.Fatalf("empty subscriptions = %+v, error = %v", subscriptions, err)
	}
}

func TestGetSubscriptionsRejectsInvalidOrganizationIDs(t *testing.T) {
	service, database := newBillingServiceTest(t)
	if err := database.Exec("DROP TABLE billing_subscriptions").Error; err != nil {
		t.Fatal(err)
	}

	for _, organizationIDs := range [][]uint64{{0}, {uint64(math.MaxInt64) + 1}} {
		_, err := service.GetSubscriptions(context.Background(), organizationIDs)
		var invalid pkg_errors.BillingOrganizationSelectionInvalidError
		if !errors.As(err, &invalid) {
			t.Fatalf("GetSubscriptions(%v) error = %v", organizationIDs, err)
		}
	}
}

func TestGetSubscriptionsRejectsMissingOrInactivePlans(t *testing.T) {
	tests := []struct {
		name   string
		planID uint64
	}{
		{name: "missing", planID: 999},
		{name: "inactive", planID: 200},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, database := newBillingServiceTest(t)
			if err := database.Exec("UPDATE billing_subscriptions SET billing_plan_id = ? WHERE organization_id = 10", test.planID).Error; err != nil {
				t.Fatal(err)
			}
			_, err := service.GetSubscriptions(context.Background(), []uint64{10})
			var invalid pkg_errors.BillingSubscriptionInvalidError
			if !errors.As(err, &invalid) {
				t.Fatalf("GetSubscriptions() error = %v", err)
			}
		})
	}
}

func TestGetSubscriptionsPreservesDatabaseFailure(t *testing.T) {
	service, database := newBillingServiceTest(t)
	if err := database.Exec("DROP TABLE billing_subscriptions").Error; err != nil {
		t.Fatal(err)
	}

	_, err := service.GetSubscriptions(context.Background(), []uint64{10})
	if err == nil || errors.As(err, new(pkg_errors.BillingSubscriptionInvalidError)) {
		t.Fatalf("GetSubscriptions() error = %v", err)
	}
}
