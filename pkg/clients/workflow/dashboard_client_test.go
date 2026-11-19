package workflow_client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	"github.com/rapidaai/pkg/clients"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

type dashboardAnalyticsClientStub struct {
	response *protos.GetProjectDashboardAnalyticsResponse
	err      error
}

func (stub dashboardAnalyticsClientStub) GetProjectDashboardAnalytics(context.Context, *protos.GetProjectDashboardAnalyticsRequest, ...grpc.CallOption) (*protos.GetProjectDashboardAnalyticsResponse, error) {
	return stub.response, stub.err
}

func TestProjectDashboardAnalyticsClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &projectDashboardAnalyticsClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
		client:         dashboardAnalyticsClientStub{},
	}
	_, err := client.GetProjectDashboardAnalytics(context.Background(), authentication.Principal{}, &protos.GetProjectDashboardAnalyticsRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("GetProjectDashboardAnalytics() error = %v", err)
	}
}

func TestProjectDashboardAnalyticsClientReturnsData(t *testing.T) {
	internalClient := dashboardInternalClient(t)
	principal := dashboardPrincipal(t)
	want := &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{HasActivity: true}}
	client := &projectDashboardAnalyticsClient{
		InternalClient: internalClient,
		client:         dashboardAnalyticsClientStub{response: &protos.GetProjectDashboardAnalyticsResponse{Code: 200, Success: true, Data: want}},
	}

	got, err := client.GetProjectDashboardAnalytics(context.Background(), principal, &protos.GetProjectDashboardAnalyticsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("GetProjectDashboardAnalytics() = %p, want %p", got, want)
	}
}

func TestProjectDashboardAnalyticsClientRejectsEmptySuccess(t *testing.T) {
	internalClient := dashboardInternalClient(t)
	principal := dashboardPrincipal(t)
	client := &projectDashboardAnalyticsClient{
		InternalClient: internalClient,
		client:         dashboardAnalyticsClientStub{response: &protos.GetProjectDashboardAnalyticsResponse{Code: 200, Success: true}},
	}
	if _, err := client.GetProjectDashboardAnalytics(context.Background(), principal, &protos.GetProjectDashboardAnalyticsRequest{}); err == nil {
		t.Fatal("GetProjectDashboardAnalytics() error = nil")
	}
}

func dashboardInternalClient(t *testing.T) clients.InternalClient {
	t.Helper()
	issuer, err := authentication_jwt.NewIssuer("workflow-api", bytes.Repeat([]byte{1}, sha256.Size))
	if err != nil {
		t.Fatal(err)
	}
	return clients.NewInternalClientWithIssuer(issuer, nil, nil)
}

func dashboardPrincipal(t *testing.T) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 11},
		authentication.Tenant{OrganizationID: 7, ProjectID: 11},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "11"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
