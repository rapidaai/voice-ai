package internal_services_test

import (
	"testing"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_dashboard_service "github.com/rapidaai/api/assistant-api/internal/services/dashboard"
)

func TestProjectDashboardAnalyticsServiceImplementation(t *testing.T) {
	var dashboardService internal_services.ProjectDashboardAnalyticsService = internal_dashboard_service.NewProjectDashboardAnalyticsService(nil)
	if dashboardService == nil {
		t.Fatal("project dashboard analytics service is nil")
	}
}
