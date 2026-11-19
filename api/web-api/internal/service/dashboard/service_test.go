// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_dashboard_service

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_productusage_service "github.com/rapidaai/api/web-api/internal/service/productusage"
	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type dashboardAnalyticsStub struct {
	call func(context.Context, authentication.Principal, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error)
}

func (stub dashboardAnalyticsStub) GetProjectDashboardAnalytics(ctx context.Context, principal authentication.Principal, request *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
	return stub.call(ctx, principal, request)
}

type dashboardUsageStub struct {
	internal_productusage_service.Service
	call func(context.Context, authentication.Principal, time.Time, time.Time) (*protos.DashboardUsageSection, error)
}

func (stub dashboardUsageStub) GetDashboardUsage(ctx context.Context, principal authentication.Principal, fromDate, toDate time.Time) (*protos.DashboardUsageSection, error) {
	return stub.call(ctx, principal, fromDate, toDate)
}

func dashboardPrincipal(t *testing.T) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 17},
		authentication.Tenant{OrganizationID: 7, ProjectID: 11},
		authentication.Proof{
			Method:    authentication.AuthenticationProjectCredential,
			Audience:  authentication.AudienceManagement,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "17"},
		},
		[]authentication.PermissionKey{"project.dashboard.read"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func dashboardQuery() internal_service.DashboardQuery {
	toDate := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	fromDate := toDate.Add(-24 * time.Hour)
	return internal_service.DashboardQuery{
		GeneratedAt:        toDate,
		FromDate:           fromDate,
		ToDate:             toDate,
		ComparisonFromDate: fromDate.Add(-24 * time.Hour),
		ComparisonToDate:   fromDate,
		AssistantLimit:     3,
		RecentFailureLimit: 5,
	}
}

func TestDashboardServiceForwardsPrincipalAndComposesSections(t *testing.T) {
	principal := dashboardPrincipal(t)
	query := dashboardQuery()
	analyticsData := &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{HasActivity: true}}
	usageData := &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY}
	var analyticsPrincipal authentication.Principal
	var usagePrincipal authentication.Principal
	var analyticsRequest *protos.GetProjectDashboardAnalyticsRequest
	var usageFromDate time.Time
	var usageToDate time.Time
	service := &dashboardService{
		analytics: dashboardAnalyticsStub{call: func(_ context.Context, received authentication.Principal, request *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
			analyticsPrincipal = received
			analyticsRequest = request
			return analyticsData, nil
		}},
		usage: dashboardUsageStub{call: func(_ context.Context, received authentication.Principal, fromDate, toDate time.Time) (*protos.DashboardUsageSection, error) {
			usagePrincipal = received
			usageFromDate = fromDate
			usageToDate = toDate
			return usageData, nil
		}},
	}

	dashboard, err := service.GetProjectDashboard(context.Background(), principal, query)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(analyticsPrincipal, principal) || !reflect.DeepEqual(usagePrincipal, principal) {
		t.Fatalf("principal was not forwarded unchanged: analytics=%+v usage=%+v", analyticsPrincipal, usagePrincipal)
	}
	if !analyticsRequest.GetFromDate().AsTime().Equal(query.FromDate) || !analyticsRequest.GetComparisonToDate().AsTime().Equal(query.ComparisonToDate) || analyticsRequest.GetAssistantLimit() != query.AssistantLimit || analyticsRequest.GetRecentFailureLimit() != query.RecentFailureLimit {
		t.Fatalf("analytics request = %+v", analyticsRequest)
	}
	if usageFromDate != query.FromDate || usageToDate != query.ToDate {
		t.Fatalf("usage range = %v to %v", usageFromDate, usageToDate)
	}
	if dashboard.GetPartial() || dashboard.GetAnalytics().GetData() != analyticsData || dashboard.GetUsage() != usageData {
		t.Fatalf("dashboard = %+v", dashboard)
	}
	if !dashboard.GetGeneratedAt().AsTime().Equal(query.GeneratedAt) || !dashboard.GetPeriod().GetComparisonFromDate().AsTime().Equal(query.ComparisonFromDate) {
		t.Fatalf("dashboard period = %+v", dashboard.GetPeriod())
	}
}

func TestDashboardServiceComposesEmptyAndPartialSections(t *testing.T) {
	sectionErr := errors.New("section unavailable")
	tests := []struct {
		name          string
		analyticsData *protos.ProjectDashboardAnalytics
		analyticsErr  error
		usageData     *protos.DashboardUsageSection
		usageErr      error
		analyticsWant protos.DashboardSectionState
		usageWant     protos.DashboardSectionState
		partialWant   bool
	}{
		{name: "empty", analyticsData: &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{}}, usageData: &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_EMPTY}, analyticsWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_EMPTY, usageWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_EMPTY},
		{name: "analytics unavailable", analyticsErr: sectionErr, usageData: &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY}, analyticsWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE, usageWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY, partialWant: true},
		{name: "usage unavailable", analyticsData: &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{HasActivity: true}}, usageErr: sectionErr, analyticsWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY, usageWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE, partialWant: true},
		{name: "nil sections preserve non-partial state", analyticsWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE, usageWant: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &dashboardService{
				analytics: dashboardAnalyticsStub{call: func(context.Context, authentication.Principal, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
					return test.analyticsData, test.analyticsErr
				}},
				usage: dashboardUsageStub{call: func(context.Context, authentication.Principal, time.Time, time.Time) (*protos.DashboardUsageSection, error) {
					return test.usageData, test.usageErr
				}},
			}
			dashboard, err := service.GetProjectDashboard(context.Background(), dashboardPrincipal(t), dashboardQuery())
			if err != nil {
				t.Fatal(err)
			}
			if dashboard.GetPartial() != test.partialWant || dashboard.GetAnalytics().GetState() != test.analyticsWant || dashboard.GetUsage().GetState() != test.usageWant {
				t.Fatalf("dashboard = %+v", dashboard)
			}
		})
	}
}

func TestDashboardServiceAppliesFailurePrecedence(t *testing.T) {
	genericErr := errors.New("unavailable")
	tests := []struct {
		name         string
		analyticsErr error
		usageErr     error
		want         error
		wantCode     codes.Code
	}{
		{name: "analytics cancellation before usage deadline", analyticsErr: status.Error(codes.Canceled, "analytics canceled"), usageErr: status.Error(codes.DeadlineExceeded, "usage deadline"), want: context.Canceled},
		{name: "analytics deadline before usage cancellation", analyticsErr: status.Error(codes.DeadlineExceeded, "analytics deadline"), usageErr: status.Error(codes.Canceled, "usage canceled"), want: context.DeadlineExceeded},
		{name: "usage cancellation", analyticsErr: genericErr, usageErr: status.Error(codes.Canceled, "usage canceled"), want: context.Canceled},
		{name: "usage deadline", analyticsErr: genericErr, usageErr: status.Error(codes.DeadlineExceeded, "usage deadline"), want: context.DeadlineExceeded},
		{name: "analytics invalid argument before generic usage failure", analyticsErr: status.Error(codes.InvalidArgument, "invalid analytics range"), usageErr: genericErr, wantCode: codes.InvalidArgument},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &dashboardService{
				analytics: dashboardAnalyticsStub{call: func(context.Context, authentication.Principal, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
					return nil, test.analyticsErr
				}},
				usage: dashboardUsageStub{call: func(context.Context, authentication.Principal, time.Time, time.Time) (*protos.DashboardUsageSection, error) {
					return nil, test.usageErr
				}},
			}
			_, err := service.GetProjectDashboard(context.Background(), dashboardPrincipal(t), dashboardQuery())
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if test.wantCode == codes.InvalidArgument {
				var invalid pkg_errors.DashboardAnalyticsRequestInvalidError
				if !errors.As(err, &invalid) || invalid.Error() != "invalid analytics range" {
					t.Fatalf("error = %T %v", err, err)
				}
			}
		})
	}
}

func TestDashboardServiceReturnsUnavailableWhenBothSectionsFail(t *testing.T) {
	analyticsErr := errors.New("analytics unavailable")
	usageErr := errors.New("usage unavailable")
	service := &dashboardService{
		analytics: dashboardAnalyticsStub{call: func(context.Context, authentication.Principal, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
			return nil, analyticsErr
		}},
		usage: dashboardUsageStub{call: func(context.Context, authentication.Principal, time.Time, time.Time) (*protos.DashboardUsageSection, error) {
			return nil, usageErr
		}},
	}

	_, err := service.GetProjectDashboard(context.Background(), dashboardPrincipal(t), dashboardQuery())
	var unavailable pkg_errors.DashboardUnavailableError
	if !errors.As(err, &unavailable) || !errors.Is(err, analyticsErr) || !errors.Is(err, usageErr) {
		t.Fatalf("error = %T %v", err, err)
	}
}

func TestDashboardServiceStartsBothWorkersAndWaitsForCompletion(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	var completed atomic.Int32
	service := &dashboardService{
		analytics: dashboardAnalyticsStub{call: func(context.Context, authentication.Principal, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
			started <- "analytics"
			<-release
			completed.Add(1)
			return &protos.ProjectDashboardAnalytics{}, nil
		}},
		usage: dashboardUsageStub{call: func(context.Context, authentication.Principal, time.Time, time.Time) (*protos.DashboardUsageSection, error) {
			started <- "usage"
			<-release
			completed.Add(1)
			return &protos.DashboardUsageSection{}, nil
		}},
	}
	principal := dashboardPrincipal(t)
	result := make(chan error, 1)
	go func() {
		_, err := service.GetProjectDashboard(context.Background(), principal, dashboardQuery())
		result <- err
	}()

	first, second := <-started, <-started
	if first == second {
		t.Fatalf("started workers = %q, %q", first, second)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if completed.Load() != 2 {
		t.Fatalf("completed workers = %d, want 2", completed.Load())
	}
}

func TestDashboardServicePropagatesCallerCancellationToBothWorkers(t *testing.T) {
	started := make(chan struct{}, 2)
	observed := make(chan error, 2)
	call := func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		observed <- ctx.Err()
		return ctx.Err()
	}
	service := &dashboardService{
		analytics: dashboardAnalyticsStub{call: func(ctx context.Context, _ authentication.Principal, _ *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
			return nil, call(ctx)
		}},
		usage: dashboardUsageStub{call: func(ctx context.Context, _ authentication.Principal, _ time.Time, _ time.Time) (*protos.DashboardUsageSection, error) {
			return nil, call(ctx)
		}},
	}
	ctx, cancel := context.WithCancel(context.Background())
	principal := dashboardPrincipal(t)
	result := make(chan error, 1)
	go func() {
		_, err := service.GetProjectDashboard(ctx, principal, dashboardQuery())
		result <- err
	}()
	<-started
	<-started
	cancel()

	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	for range 2 {
		if err := <-observed; !errors.Is(err, context.Canceled) {
			t.Fatalf("worker context error = %v", err)
		}
	}
}
