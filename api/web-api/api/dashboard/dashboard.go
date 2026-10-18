package dashboard_api

import (
	"context"
	"errors"
	"time"

	web_config "github.com/rapidaai/api/web-api/config"
	internal_productusage_service "github.com/rapidaai/api/web-api/internal/service/productusage"
	workflow_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type dashboardUsageService interface {
	GetDashboardUsage(context.Context, *types.Authentication, time.Time, time.Time) (*protos.DashboardUsageSection, error)
}

type webDashboardGRPCApi struct {
	logger    commons.Logger
	analytics workflow_client.ProjectDashboardAnalyticsClient
	usage     dashboardUsageService
	now       func() time.Time
}

func NewDashboardGRPC(cfg *web_config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.DashboardServiceServer {
	return &webDashboardGRPCApi{
		logger:    logger,
		analytics: workflow_client.NewProjectDashboardAnalyticsClientGRPC(&cfg.AppConfig, logger, redis),
		usage:     internal_productusage_service.NewProductUsageService(postgres, logger),
		now:       time.Now,
	}
}

func (dashboardAPI *webDashboardGRPCApi) GetProjectDashboard(ctx context.Context, request *protos.GetProjectDashboardRequest) (*protos.GetProjectDashboardResponse, error) {
	auth, err := types.Authorize(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	auth, err = auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if _, err = auth.ProjectContext(); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if request == nil {
		request = &protos.GetProjectDashboardRequest{}
	}
	now := dashboardAPI.now().UTC()
	toDate := now
	if request.GetToDate() != nil {
		if err = request.GetToDate().CheckValid(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "toDate: %v", err)
		}
		toDate = request.GetToDate().AsTime().UTC()
	}
	fromDate := toDate.Add(-30 * 24 * time.Hour)
	if request.GetFromDate() != nil {
		if err = request.GetFromDate().CheckValid(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "fromDate: %v", err)
		}
		fromDate = request.GetFromDate().AsTime().UTC()
	}
	if !fromDate.Before(toDate) {
		return nil, status.Error(codes.InvalidArgument, "fromDate must be before toDate")
	}
	if toDate.After(fromDate.Add(90 * 24 * time.Hour)) {
		return nil, status.Error(codes.InvalidArgument, "date range cannot exceed 90 days")
	}
	duration := toDate.Sub(fromDate)
	comparisonFromDate := fromDate.Add(-duration)
	comparisonToDate := fromDate
	analyticsFromDate := timestamppb.New(fromDate)
	analyticsToDate := timestamppb.New(toDate)
	analyticsComparisonFromDate := timestamppb.New(comparisonFromDate)
	analyticsComparisonToDate := timestamppb.New(comparisonToDate)
	if !analyticsFromDate.IsValid() || !analyticsToDate.IsValid() || !analyticsComparisonFromDate.IsValid() || !analyticsComparisonToDate.IsValid() {
		return nil, status.Error(codes.InvalidArgument, "resolved date range is outside supported timestamp bounds")
	}

	assistantLimit := uint32(3)
	if request.AssistantLimit != nil {
		assistantLimit = request.GetAssistantLimit()
		if assistantLimit < 1 || assistantLimit > 10 {
			return nil, status.Error(codes.InvalidArgument, "assistantLimit must be from 1 through 10")
		}
	}
	recentFailureLimit := uint32(5)
	if request.RecentFailureLimit != nil {
		recentFailureLimit = request.GetRecentFailureLimit()
		if recentFailureLimit < 1 || recentFailureLimit > 20 {
			return nil, status.Error(codes.InvalidArgument, "recentFailureLimit must be from 1 through 20")
		}
	}

	type analyticsResult struct {
		data     *protos.ProjectDashboardAnalytics
		err      error
		duration time.Duration
	}
	type usageResult struct {
		data     *protos.DashboardUsageSection
		err      error
		duration time.Duration
	}
	analyticsResults := make(chan analyticsResult, 1)
	usageResults := make(chan usageResult, 1)
	go func() {
		startedAt := time.Now()
		data, callErr := dashboardAPI.analytics.GetProjectDashboardAnalytics(ctx, auth, &protos.GetProjectDashboardAnalyticsRequest{
			FromDate:           analyticsFromDate,
			ToDate:             analyticsToDate,
			ComparisonFromDate: analyticsComparisonFromDate,
			ComparisonToDate:   analyticsComparisonToDate,
			AssistantLimit:     assistantLimit,
			RecentFailureLimit: recentFailureLimit,
		})
		analyticsResults <- analyticsResult{data: data, err: callErr, duration: time.Since(startedAt)}
	}()
	go func() {
		startedAt := time.Now()
		data, callErr := dashboardAPI.usage.GetDashboardUsage(ctx, auth, fromDate, toDate)
		usageResults <- usageResult{data: data, err: callErr, duration: time.Since(startedAt)}
	}()

	analytics := <-analyticsResults
	usage := <-usageResults
	if err = ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	for _, sectionErr := range []error{analytics.err, usage.err} {
		switch {
		case errors.Is(sectionErr, context.Canceled), status.Code(sectionErr) == codes.Canceled:
			return nil, status.Error(codes.Canceled, context.Canceled.Error())
		case errors.Is(sectionErr, context.DeadlineExceeded), status.Code(sectionErr) == codes.DeadlineExceeded:
			return nil, status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
		}
	}
	if status.Code(analytics.err) == codes.InvalidArgument {
		return nil, status.Error(codes.InvalidArgument, status.Convert(analytics.err).Message())
	}
	if dashboardAPI.logger != nil {
		if analytics.err != nil {
			dashboardAPI.logger.Warnf("project dashboard analytics section unavailable after %s: %v", analytics.duration, analytics.err)
		} else {
			dashboardAPI.logger.Debugf("project dashboard analytics section loaded in %s", analytics.duration)
		}
		if usage.err != nil {
			dashboardAPI.logger.Warnf("project dashboard usage section unavailable after %s: %v", usage.duration, usage.err)
		} else {
			dashboardAPI.logger.Debugf("project dashboard usage section loaded in %s", usage.duration)
		}
	}
	if analytics.err != nil && usage.err != nil {
		if dashboardAPI.logger != nil {
			dashboardAPI.logger.Errorf("unable to load project dashboard: analytics: %v; usage: %v", analytics.err, usage.err)
		}
		return nil, status.Error(codes.Internal, "unable to load project dashboard")
	}

	analyticsSection := &protos.DashboardAnalyticsSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE}
	if analytics.err == nil && analytics.data != nil {
		analyticsSection.State = protos.DashboardSectionState_DASHBOARD_SECTION_STATE_READY
		if analytics.data.GetProjectState() == nil || !analytics.data.GetProjectState().GetHasActivity() {
			analyticsSection.State = protos.DashboardSectionState_DASHBOARD_SECTION_STATE_EMPTY
		} else {
			analyticsSection.Data = analytics.data
		}
	}
	usageSection := usage.data
	if usage.err != nil || usageSection == nil {
		usageSection = &protos.DashboardUsageSection{State: protos.DashboardSectionState_DASHBOARD_SECTION_STATE_UNAVAILABLE}
	}

	return &protos.GetProjectDashboardResponse{
		Code:    200,
		Success: true,
		Data: &protos.ProjectDashboard{
			GeneratedAt: timestamppb.New(now),
			Period: &protos.DashboardPeriod{
				FromDate:           timestamppb.New(fromDate),
				ToDate:             timestamppb.New(toDate),
				ComparisonFromDate: timestamppb.New(comparisonFromDate),
				ComparisonToDate:   timestamppb.New(comparisonToDate),
			},
			Analytics: analyticsSection,
			Usage:     usageSection,
			Partial:   analytics.err != nil || usage.err != nil,
		},
	}, nil
}
