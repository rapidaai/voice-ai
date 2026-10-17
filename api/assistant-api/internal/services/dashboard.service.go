package internal_services

import (
	"context"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type ProjectDashboardAnalyticsService interface {
	GetProjectDashboardAnalytics(context.Context, types.ProjectContext, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error)
}
