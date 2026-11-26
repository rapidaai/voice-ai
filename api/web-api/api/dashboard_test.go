// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type dashboardTransportServer struct {
	protos.UnimplementedDashboardServiceServer
	request *protos.GetProjectDashboardRequest
}

func (server *dashboardTransportServer) GetProjectDashboard(_ context.Context, request *protos.GetProjectDashboardRequest) (*protos.GetProjectDashboardResponse, error) {
	server.request = request
	return &protos.GetProjectDashboardResponse{}, nil
}

type dashboardServiceStub struct {
	data      *protos.ProjectDashboard
	err       error
	called    bool
	principal authentication.Principal
	query     internal_service.DashboardQuery
}

func (stub *dashboardServiceStub) GetProjectDashboard(_ context.Context, principal authentication.Principal, query internal_service.DashboardQuery) (*protos.ProjectDashboard, error) {
	stub.called = true
	stub.principal = principal
	stub.query = query
	return stub.data, stub.err
}

func dashboardAPIContext(t *testing.T, projectID uint64) context.Context {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 13},
		authentication.Tenant{OrganizationID: 7, ProjectID: projectID},
		authentication.Proof{
			Method:    authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "dashboard-test"},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestDashboardGRPCTransportOwnsRequestAllocation(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	service := &dashboardTransportServer{}
	protos.RegisterDashboardServiceServer(grpcServer, service)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
		<-serveErr
	})

	connection, err := grpc.NewClient(
		"passthrough:///dashboard",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	if _, err := protos.NewDashboardServiceClient(connection).GetProjectDashboard(context.Background(), &protos.GetProjectDashboardRequest{}); err != nil {
		t.Fatal(err)
	}
	if service.request == nil {
		t.Fatal("generated gRPC transport passed a nil request")
	}
}

func TestDashboardAPIRequiresProjectAuthentication(t *testing.T) {
	service := &dashboardServiceStub{}
	api := &webDashboardGRPCApi{dashboardService: service, now: time.Now}

	if _, err := api.GetProjectDashboard(context.Background(), &protos.GetProjectDashboardRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing principal error = %v", err)
	}
	if _, err := api.GetProjectDashboard(dashboardAPIContext(t, 0), &protos.GetProjectDashboardRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("missing project error = %v", err)
	}
	if service.called {
		t.Fatal("service called without project authentication")
	}
}

func TestDashboardAPIResolvesValidatedQuery(t *testing.T) {
	now := time.Date(2026, 10, 17, 8, 30, 0, 0, time.UTC)
	dashboard := &protos.ProjectDashboard{}
	service := &dashboardServiceStub{data: dashboard}
	api := &webDashboardGRPCApi{dashboardService: service, now: func() time.Time { return now }}

	response, err := api.GetProjectDashboard(dashboardAPIContext(t, 11), &protos.GetProjectDashboardRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !response.GetSuccess() || response.GetData() != dashboard {
		t.Fatalf("response = %+v", response)
	}
	if service.principal.Tenant().ProjectID != 11 || service.query.GeneratedAt != now || service.query.ToDate != now || service.query.FromDate != now.Add(-30*24*time.Hour) {
		t.Fatalf("principal = %+v query = %+v", service.principal, service.query)
	}
	if service.query.AssistantLimit != 3 || service.query.RecentFailureLimit != 5 || service.query.ComparisonToDate != service.query.FromDate {
		t.Fatalf("query defaults = %+v", service.query)
	}
}

func TestDashboardAPIRejectsInvalidInputBeforeServiceCall(t *testing.T) {
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
		service := &dashboardServiceStub{}
		api := &webDashboardGRPCApi{dashboardService: service, now: func() time.Time { return now }}
		if _, err := api.GetProjectDashboard(dashboardAPIContext(t, 11), request); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("request %+v error = %v", request, err)
		}
		if service.called {
			t.Fatalf("request %+v reached service", request)
		}
	}
}

func TestDashboardAPIMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		code    codes.Code
		message string
	}{
		{name: "canceled", err: context.Canceled, code: codes.Canceled, message: context.Canceled.Error()},
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded, message: context.DeadlineExceeded.Error()},
		{name: "invalid analytics request", err: pkg_errors.DashboardAnalyticsRequestInvalidError{Message: "invalid analytics range"}, code: codes.InvalidArgument, message: "invalid analytics range"},
		{name: "empty invalid analytics request", err: pkg_errors.DashboardAnalyticsRequestInvalidError{}, code: codes.InvalidArgument},
		{name: "dashboard unavailable", err: pkg_errors.DashboardUnavailableError{}, code: codes.Internal, message: "unable to load project dashboard"},
		{name: "unknown", err: errors.New("unexpected"), code: codes.Internal, message: "unable to load project dashboard"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &webDashboardGRPCApi{dashboardService: &dashboardServiceStub{err: test.err}, now: time.Now}
			_, err := api.GetProjectDashboard(dashboardAPIContext(t, 11), &protos.GetProjectDashboardRequest{})
			if status.Code(err) != test.code || status.Convert(err).Message() != test.message {
				t.Fatalf("error = %v, want %v %q", err, test.code, test.message)
			}
		})
	}
}
