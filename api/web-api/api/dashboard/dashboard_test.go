package dashboard_api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type analyticsStub struct {
	data    *protos.ProjectDashboardAnalytics
	err     error
	request *protos.GetProjectDashboardAnalyticsRequest
}

func (stub *analyticsStub) GetProjectDashboardAnalytics(_ context.Context, _ *types.Authentication, request *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
	stub.request = request
	return stub.data, stub.err
}

type usageStub struct {
	data   *protos.DashboardUsageSection
	err    error
	called *bool
}

func (stub usageStub) GetDashboardUsage(context.Context, *types.Authentication, time.Time, time.Time) (*protos.DashboardUsageSection, error) {
	if stub.called != nil {
		*stub.called = true
	}
	return stub.data, stub.err
}

func dashboardAPIContext() context.Context {
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 7, ProjectID: 11},
	}
	return context.WithValue(context.Background(), types.CTX_, auth)
}

func TestDashboardAPIRequiresProjectAuthentication(t *testing.T) {
	api := &webDashboardGRPCApi{analytics: &analyticsStub{}, usage: usageStub{}, now: time.Now}
	if _, err := api.GetProjectDashboard(context.Background(), &protos.GetProjectDashboardRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("error = %v", err)
	}
}

func TestDashboardAPIResolvesRangeAndReturnsSuccess(t *testing.T) {
	now := time.Date(2026, 10, 17, 8, 30, 0, 0, time.UTC)
	analytics := &analyticsStub{data: &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{HasActivity: true}}}
	api := &webDashboardGRPCApi{
		analytics: analytics,
		usage:     usageStub{data: &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY}},
		now:       func() time.Time { return now },
	}

	response, err := api.GetProjectDashboard(dashboardAPIContext(), &protos.GetProjectDashboardRequest{})
	if err != nil {
		t.Fatal(err)
	}
	dashboard := response.GetData()
	if !response.GetSuccess() || dashboard.GetPartial() || dashboard.GetAnalytics().GetState() != protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY {
		t.Fatalf("response = %+v", response)
	}
	if got := dashboard.GetPeriod().GetToDate().AsTime(); !got.Equal(now) {
		t.Fatalf("toDate = %v, want %v", got, now)
	}
	if got := dashboard.GetPeriod().GetFromDate().AsTime(); !got.Equal(now.Add(-30 * 24 * time.Hour)) {
		t.Fatalf("fromDate = %v", got)
	}
	if analytics.request.GetAssistantLimit() != 3 || analytics.request.GetRecentFailureLimit() != 5 {
		t.Fatalf("limits = %d, %d", analytics.request.GetAssistantLimit(), analytics.request.GetRecentFailureLimit())
	}
}

func TestDashboardAPIComposesSectionStates(t *testing.T) {
	sectionErr := errors.New("section unavailable")
	tests := []struct {
		name          string
		analytics     *analyticsStub
		usage         usageStub
		wantAnalytics protos.DashboardSectionState
		wantUsage     protos.DashboardSectionState
		wantPartial   bool
	}{
		{
			name: "full",
			analytics: &analyticsStub{data: &protos.ProjectDashboardAnalytics{
				ProjectState: &protos.DashboardProjectState{HasActivity: true},
			}},
			usage:         usageStub{data: &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY}},
			wantAnalytics: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY,
			wantUsage:     protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY,
		},
		{
			name:          "empty",
			analytics:     &analyticsStub{data: &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{}}},
			usage:         usageStub{data: &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_EMPTY}},
			wantAnalytics: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_EMPTY,
			wantUsage:     protos.DashboardSectionState_DASHBOARD_SECTION_STATE_EMPTY,
		},
		{
			name:          "analytics unavailable",
			analytics:     &analyticsStub{err: sectionErr},
			usage:         usageStub{data: &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY}},
			wantAnalytics: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE,
			wantUsage:     protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY,
			wantPartial:   true,
		},
		{
			name: "usage unavailable",
			analytics: &analyticsStub{data: &protos.ProjectDashboardAnalytics{
				ProjectState: &protos.DashboardProjectState{HasActivity: true},
			}},
			usage:         usageStub{err: sectionErr},
			wantAnalytics: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY,
			wantUsage:     protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE,
			wantPartial:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &webDashboardGRPCApi{analytics: test.analytics, usage: test.usage, now: time.Now}
			response, err := api.GetProjectDashboard(dashboardAPIContext(), &protos.GetProjectDashboardRequest{})
			if err != nil {
				t.Fatal(err)
			}
			dashboard := response.GetData()
			if dashboard.GetPartial() != test.wantPartial || dashboard.GetAnalytics().GetState() != test.wantAnalytics || dashboard.GetUsage().GetState() != test.wantUsage {
				t.Fatalf("dashboard = %+v", dashboard)
			}
		})
	}
}

func TestDashboardAPIMapsDownstreamErrors(t *testing.T) {
	tests := []struct {
		name      string
		analytics error
		usage     error
		code      codes.Code
	}{
		{name: "both unavailable", analytics: errors.New("analytics unavailable"), usage: errors.New("usage unavailable"), code: codes.Internal},
		{name: "canceled", analytics: status.Error(codes.Canceled, "downstream canceled"), code: codes.Canceled},
		{name: "deadline", analytics: status.Error(codes.DeadlineExceeded, "downstream deadline"), code: codes.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &webDashboardGRPCApi{
				analytics: &analyticsStub{err: test.analytics},
				usage:     usageStub{err: test.usage, data: &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY}},
				now:       time.Now,
			}
			_, err := api.GetProjectDashboard(dashboardAPIContext(), &protos.GetProjectDashboardRequest{})
			if status.Code(err) != test.code {
				t.Fatalf("error = %v, want %v", err, test.code)
			}
		})
	}
}

func TestDashboardAPIRejectsInvalidInputsBeforeLoadingSections(t *testing.T) {
	now := time.Now().UTC()
	zero := uint32(0)
	eleven := uint32(11)
	twentyOne := uint32(21)
	tests := []*protos.GetProjectDashboardRequest{
		{FromDate: &timestamppb.Timestamp{Seconds: 253402300800}},
		{ToDate: timestamppb.New(time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC))},
		{FromDate: timestamppb.New(now), ToDate: timestamppb.New(now)},
		{FromDate: timestamppb.New(now.Add(-91 * 24 * time.Hour)), ToDate: timestamppb.New(now)},
		{AssistantLimit: &zero},
		{AssistantLimit: &eleven},
		{RecentFailureLimit: &zero},
		{RecentFailureLimit: &twentyOne},
	}
	for _, request := range tests {
		analytics := &analyticsStub{err: errors.New("analytics unavailable")}
		usageCalled := false
		api := &webDashboardGRPCApi{
			analytics: analytics,
			usage:     usageStub{called: &usageCalled},
			now:       func() time.Time { return now },
		}
		if _, err := api.GetProjectDashboard(dashboardAPIContext(), request); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("request %+v error = %v", request, err)
		}
		if analytics.request != nil || usageCalled {
			t.Fatalf("request %+v loaded analytics=%v usage=%v", request, analytics.request != nil, usageCalled)
		}
	}
}
