package dashboard_api

import (
	"context"
	"errors"
	"testing"

	internal_dashboard_service "github.com/rapidaai/api/assistant-api/internal/services/dashboard"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type projectDashboardAnalyticsServiceStub struct {
	data *protos.ProjectDashboardAnalytics
	err  error
}

func (stub projectDashboardAnalyticsServiceStub) GetProjectDashboardAnalytics(context.Context, types.ProjectContext, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
	return stub.data, stub.err
}

func projectDashboardAnalyticsContext() context.Context {
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 7, ProjectID: 11},
	}
	return context.WithValue(context.Background(), types.CTX_, auth)
}

func TestProjectDashboardAnalyticsAPIRejectsNilRequest(t *testing.T) {
	dashboardAPI := &projectDashboardAnalyticsGRPCApi{projectDashboardAnalytics: projectDashboardAnalyticsServiceStub{}}
	if _, err := dashboardAPI.GetProjectDashboardAnalytics(projectDashboardAnalyticsContext(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error = %v", err)
	}
}

func TestProjectDashboardAnalyticsAPIRequiresAuthentication(t *testing.T) {
	dashboardAPI := &projectDashboardAnalyticsGRPCApi{projectDashboardAnalytics: projectDashboardAnalyticsServiceStub{}}
	if _, err := dashboardAPI.GetProjectDashboardAnalytics(context.Background(), &protos.GetProjectDashboardAnalyticsRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("error = %v", err)
	}
}

func TestProjectDashboardAnalyticsAPIRequiresProjectContext(t *testing.T) {
	tests := []*types.Authentication{
		{
			AuthType:          types.AuthTypeOrg,
			ActorValue:        &types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 7},
			OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		},
		{
			AuthType:          types.AuthTypeUser,
			ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: 3},
			UserValue:         &types.UserContext{UserID: 3},
			OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		},
	}
	for _, auth := range tests {
		ctx := context.WithValue(context.Background(), types.CTX_, auth)
		dashboardAPI := &projectDashboardAnalyticsGRPCApi{projectDashboardAnalytics: projectDashboardAnalyticsServiceStub{}}
		if _, err := dashboardAPI.GetProjectDashboardAnalytics(ctx, &protos.GetProjectDashboardAnalyticsRequest{}); status.Code(err) != codes.PermissionDenied {
			t.Fatalf("auth type %v error = %v", auth.AuthType, err)
		}
	}
}

func TestProjectDashboardAnalyticsAPIReturnsSuccess(t *testing.T) {
	want := &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{HasActivity: true}}
	dashboardAPI := &projectDashboardAnalyticsGRPCApi{projectDashboardAnalytics: projectDashboardAnalyticsServiceStub{data: want}}
	response, err := dashboardAPI.GetProjectDashboardAnalytics(projectDashboardAnalyticsContext(), &protos.GetProjectDashboardAnalyticsRequest{})
	if err != nil || !response.GetSuccess() || response.GetData() != want {
		t.Fatalf("response = %+v, error = %v", response, err)
	}
}

func TestProjectDashboardAnalyticsAPIMapsErrors(t *testing.T) {
	tests := []struct {
		err  error
		code codes.Code
	}{
		{err: internal_dashboard_service.ErrInvalidProjectDashboardAnalyticsRequest, code: codes.InvalidArgument},
		{err: context.Canceled, code: codes.Canceled},
		{err: errors.New("database unavailable"), code: codes.Internal},
	}
	for _, test := range tests {
		dashboardAPI := &projectDashboardAnalyticsGRPCApi{projectDashboardAnalytics: projectDashboardAnalyticsServiceStub{err: test.err}}
		_, err := dashboardAPI.GetProjectDashboardAnalytics(projectDashboardAnalyticsContext(), &protos.GetProjectDashboardAnalyticsRequest{})
		if status.Code(err) != test.code {
			t.Fatalf("error = %v, want %v", err, test.code)
		}
	}
}
