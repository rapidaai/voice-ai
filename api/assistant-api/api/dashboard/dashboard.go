package dashboard_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_dashboard_service "github.com/rapidaai/api/assistant-api/internal/services/dashboard"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type projectDashboardAnalyticsGRPCApi struct {
	logger                    commons.Logger
	projectDashboardAnalytics internal_services.ProjectDashboardAnalyticsService
}

func NewProjectDashboardAnalyticsGRPC(logger commons.Logger, postgres connectors.PostgresConnector) protos.ProjectDashboardAnalyticsServiceServer {
	return &projectDashboardAnalyticsGRPCApi{
		logger:                    logger,
		projectDashboardAnalytics: internal_dashboard_service.NewProjectDashboardAnalyticsService(postgres),
	}
}

func (dashboardAPI *projectDashboardAnalyticsGRPCApi) GetProjectDashboardAnalytics(ctx context.Context, request *protos.GetProjectDashboardAnalyticsRequest) (*protos.GetProjectDashboardAnalyticsResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	auth, err := types.Authorize(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	auth, err = auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	analytics, err := dashboardAPI.projectDashboardAnalytics.GetProjectDashboardAnalytics(ctx, projectContext, request)
	if err != nil {
		switch {
		case errors.Is(err, internal_dashboard_service.ErrInvalidProjectDashboardAnalyticsRequest):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return nil, status.FromContextError(err).Err()
		default:
			if dashboardAPI.logger != nil {
				dashboardAPI.logger.Errorf("unable to load project dashboard analytics: %v", err)
			}
			return nil, status.Error(codes.Internal, "unable to load project dashboard analytics")
		}
	}
	return &protos.GetProjectDashboardAnalyticsResponse{Code: 200, Success: true, Data: analytics}, nil
}
