package workflow_client

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/types"
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
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
		client:         dashboardAnalyticsClientStub{},
	}
	_, err := client.GetProjectDashboardAnalytics(context.Background(), &types.Authentication{}, &protos.GetProjectDashboardAnalyticsRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("GetProjectDashboardAnalytics() error = %v", err)
	}
}

func TestProjectDashboardAnalyticsClientReturnsData(t *testing.T) {
	internalClient := dashboardInternalClient()
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 7, ProjectID: 11},
	}
	want := &protos.ProjectDashboardAnalytics{ProjectState: &protos.DashboardProjectState{HasActivity: true}}
	client := &projectDashboardAnalyticsClient{
		InternalClient: internalClient,
		client:         dashboardAnalyticsClientStub{response: &protos.GetProjectDashboardAnalyticsResponse{Code: 200, Success: true, Data: want}},
	}

	got, err := client.GetProjectDashboardAnalytics(context.Background(), auth, &protos.GetProjectDashboardAnalyticsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("GetProjectDashboardAnalytics() = %p, want %p", got, want)
	}
}

func TestProjectDashboardAnalyticsClientRejectsEmptySuccess(t *testing.T) {
	internalClient := dashboardInternalClient()
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 7, ProjectID: 11},
	}
	client := &projectDashboardAnalyticsClient{
		InternalClient: internalClient,
		client:         dashboardAnalyticsClientStub{response: &protos.GetProjectDashboardAnalyticsResponse{Code: 200, Success: true}},
	}
	if _, err := client.GetProjectDashboardAnalytics(context.Background(), auth, &protos.GetProjectDashboardAnalyticsRequest{}); err == nil {
		t.Fatal("GetProjectDashboardAnalytics() error = nil")
	}
}

func dashboardInternalClient() clients.InternalClient {
	return clients.NewInternalClient(&config.AppConfig{
		Name:      "web-api",
		ServiceID: 41,
		Secret:    "secret",
	}, nil, nil)
}
