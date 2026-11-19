package dashboard_api

import (
	"context"
	"errors"
	"strconv"
	"testing"

	internal_dashboard_service "github.com/rapidaai/api/assistant-api/internal/services/dashboard"
	"github.com/rapidaai/pkg/authentication"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type projectDashboardAnalyticsServiceStub struct {
	data *protos.ProjectDashboardAnalytics
	err  error
}

func (stub projectDashboardAnalyticsServiceStub) GetProjectDashboardAnalytics(context.Context, authentication.Tenant, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
	return stub.data, stub.err
}

func projectDashboardAnalyticsContext() context.Context {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 11},
		authentication.Tenant{OrganizationID: 7, ProjectID: 11},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "11"}},
		nil,
	)
	if err != nil {
		panic(err)
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		panic(err)
	}
	return ctx
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
	tests := []struct {
		name      string
		principal authentication.Principal
	}{
		{
			name: "organization",
			principal: mustDashboardPrincipal(authentication.NewDirectPrincipal(
				authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 7},
				authentication.Tenant{OrganizationID: 7},
				authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "7"}}, nil,
			)),
		},
		{
			name: "user",
			principal: mustDashboardPrincipal(authentication.NewDirectPrincipal(
				authentication.Identity{Type: authentication.IdentityUser, ID: 3},
				authentication.Tenant{OrganizationID: 7},
				authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: strconv.FormatUint(3, 10)}}, nil,
			)),
		},
	}
	for _, test := range tests {
		ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), test.principal)
		if err != nil {
			t.Fatal(err)
		}
		dashboardAPI := &projectDashboardAnalyticsGRPCApi{projectDashboardAnalytics: projectDashboardAnalyticsServiceStub{}}
		if _, err := dashboardAPI.GetProjectDashboardAnalytics(ctx, &protos.GetProjectDashboardAnalyticsRequest{}); status.Code(err) != codes.PermissionDenied {
			t.Fatalf("%s principal error = %v", test.name, err)
		}
	}
}

func mustDashboardPrincipal(principal authentication.Principal, err error) authentication.Principal {
	if err != nil {
		panic(err)
	}
	return principal
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
