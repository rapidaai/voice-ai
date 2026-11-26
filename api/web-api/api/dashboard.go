// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"context"
	"errors"
	"time"

	web_config "github.com/rapidaai/api/web-api/config"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_dashboard_service "github.com/rapidaai/api/web-api/internal/service/dashboard"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type webDashboardGRPCApi struct {
	logger           commons.Logger
	dashboardService internal_service.DashboardService
	now              func() time.Time
}

// NewDashboardGRPC constructs the Web API Dashboard controller.
func NewDashboardGRPC(cfg *web_config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.DashboardServiceServer {
	return &webDashboardGRPCApi{
		logger:           logger,
		dashboardService: internal_dashboard_service.NewDashboardService(&cfg.AppConfig, logger, postgres, redis),
		now:              time.Now,
	}
}

func (dashboardAPI *webDashboardGRPCApi) GetProjectDashboard(ctx context.Context, request *protos.GetProjectDashboardRequest) (*protos.GetProjectDashboardResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if !validator.NonZero(principal.Tenant().ProjectID) {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}

	now := dashboardAPI.now().UTC()
	toDate := now
	if request.GetToDate() != nil {
		if err := request.GetToDate().CheckValid(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "toDate: %v", err)
		}
		toDate = request.GetToDate().AsTime().UTC()
	}
	fromDate := toDate.Add(-30 * 24 * time.Hour)
	if request.GetFromDate() != nil {
		if err := request.GetFromDate().CheckValid(); err != nil {
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
	if !timestamppb.New(fromDate).IsValid() || !timestamppb.New(toDate).IsValid() ||
		!timestamppb.New(comparisonFromDate).IsValid() || !timestamppb.New(comparisonToDate).IsValid() {
		return nil, status.Error(codes.InvalidArgument, "resolved date range is outside supported timestamp bounds")
	}

	assistantLimit := uint32(3)
	if request.AssistantLimit != nil {
		assistantLimit = request.GetAssistantLimit()
		if !validator.Between(assistantLimit, uint32(1), uint32(10)) {
			return nil, status.Error(codes.InvalidArgument, "assistantLimit must be from 1 through 10")
		}
	}
	recentFailureLimit := uint32(5)
	if request.RecentFailureLimit != nil {
		recentFailureLimit = request.GetRecentFailureLimit()
		if !validator.Between(recentFailureLimit, uint32(1), uint32(20)) {
			return nil, status.Error(codes.InvalidArgument, "recentFailureLimit must be from 1 through 20")
		}
	}

	dashboard, err := dashboardAPI.dashboardService.GetProjectDashboard(ctx, principal, internal_service.DashboardQuery{
		GeneratedAt:        now,
		FromDate:           fromDate,
		ToDate:             toDate,
		ComparisonFromDate: comparisonFromDate,
		ComparisonToDate:   comparisonToDate,
		AssistantLimit:     assistantLimit,
		RecentFailureLimit: recentFailureLimit,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, status.Error(codes.Canceled, context.Canceled.Error())
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
		}
		var invalidRequest pkg_errors.DashboardAnalyticsRequestInvalidError
		if errors.As(err, &invalidRequest) {
			return nil, status.Error(codes.InvalidArgument, invalidRequest.Error())
		}
		var unavailable pkg_errors.DashboardUnavailableError
		if errors.As(err, &unavailable) {
			return nil, status.Error(codes.Internal, "unable to load project dashboard")
		}
		if dashboardAPI.logger != nil {
			dashboardAPI.logger.Errorf("unable to load project dashboard: %v", err)
		}
		return nil, status.Error(codes.Internal, "unable to load project dashboard")
	}

	return &protos.GetProjectDashboardResponse{Code: 200, Success: true, Data: dashboard}, nil
}
