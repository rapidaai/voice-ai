package internal_services

import (
	"context"
	"github.com/rapidaai/pkg/authentication"

	"github.com/rapidaai/protos"
)

type ProjectDashboardAnalyticsService interface {
	GetProjectDashboardAnalytics(context.Context, authentication.Tenant, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error)
}
