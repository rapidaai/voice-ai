// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_dashboard_service

import (
	"context"
	"errors"
	"time"

	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_productusage_service "github.com/rapidaai/api/web-api/internal/service/productusage"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	workflow_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type dashboardService struct {
	logger    commons.Logger
	analytics workflow_client.ProjectDashboardAnalyticsClient
	usage     internal_productusage_service.Service
}

// NewDashboardService constructs the internal project Dashboard service.
func NewDashboardService(appConfig *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) internal_service.DashboardService {
	return &dashboardService{
		logger:    logger,
		analytics: workflow_client.NewProjectDashboardAnalyticsClientGRPC(appConfig, logger, redis),
		usage:     internal_productusage_service.NewProductUsageService(postgres, logger),
	}
}

func (service *dashboardService) GetProjectDashboard(ctx context.Context, principal authentication.Principal, query internal_service.DashboardQuery) (*protos.ProjectDashboard, error) {
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
		data, err := service.analytics.GetProjectDashboardAnalytics(ctx, principal, &protos.GetProjectDashboardAnalyticsRequest{
			FromDate:           timestamppb.New(query.FromDate),
			ToDate:             timestamppb.New(query.ToDate),
			ComparisonFromDate: timestamppb.New(query.ComparisonFromDate),
			ComparisonToDate:   timestamppb.New(query.ComparisonToDate),
			AssistantLimit:     query.AssistantLimit,
			RecentFailureLimit: query.RecentFailureLimit,
		})
		analyticsResults <- analyticsResult{data: data, err: err, duration: time.Since(startedAt)}
	}()
	go func() {
		startedAt := time.Now()
		data, err := service.usage.GetDashboardUsage(ctx, principal, query.FromDate, query.ToDate)
		usageResults <- usageResult{data: data, err: err, duration: time.Since(startedAt)}
	}()

	analytics := <-analyticsResults
	usage := <-usageResults
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if errors.Is(analytics.err, context.Canceled) || status.Code(analytics.err) == codes.Canceled {
		return nil, context.Canceled
	}
	if errors.Is(analytics.err, context.DeadlineExceeded) || status.Code(analytics.err) == codes.DeadlineExceeded {
		return nil, context.DeadlineExceeded
	}
	if errors.Is(usage.err, context.Canceled) || status.Code(usage.err) == codes.Canceled {
		return nil, context.Canceled
	}
	if errors.Is(usage.err, context.DeadlineExceeded) || status.Code(usage.err) == codes.DeadlineExceeded {
		return nil, context.DeadlineExceeded
	}
	if status.Code(analytics.err) == codes.InvalidArgument {
		return nil, pkg_errors.DashboardAnalyticsRequestInvalidError{
			Message: status.Convert(analytics.err).Message(),
			Cause:   analytics.err,
		}
	}

	if service.logger != nil {
		if analytics.err != nil {
			service.logger.Warnf("project dashboard analytics section unavailable after %s: %v", analytics.duration, analytics.err)
		} else {
			service.logger.Debugf("project dashboard analytics section loaded in %s", analytics.duration)
		}
		if usage.err != nil {
			service.logger.Warnf("project dashboard usage section unavailable after %s: %v", usage.duration, usage.err)
		} else {
			service.logger.Debugf("project dashboard usage section loaded in %s", usage.duration)
		}
	}
	if analytics.err != nil && usage.err != nil {
		if service.logger != nil {
			service.logger.Errorf("unable to load project dashboard: analytics: %v; usage: %v", analytics.err, usage.err)
		}
		return nil, pkg_errors.DashboardUnavailableError{Cause: errors.Join(analytics.err, usage.err)}
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

	return &protos.ProjectDashboard{
		GeneratedAt: timestamppb.New(query.GeneratedAt),
		Period: &protos.DashboardPeriod{
			FromDate:           timestamppb.New(query.FromDate),
			ToDate:             timestamppb.New(query.ToDate),
			ComparisonFromDate: timestamppb.New(query.ComparisonFromDate),
			ComparisonToDate:   timestamppb.New(query.ComparisonToDate),
		},
		Analytics: analyticsSection,
		Usage:     usageSection,
		Partial:   analytics.err != nil || usage.err != nil,
	}, nil
}
