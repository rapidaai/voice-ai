// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service

import (
	"context"
	"time"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/protos"
)

// DashboardQuery contains controller-validated Dashboard inputs.
type DashboardQuery struct {
	GeneratedAt        time.Time
	FromDate           time.Time
	ToDate             time.Time
	ComparisonFromDate time.Time
	ComparisonToDate   time.Time
	AssistantLimit     uint32
	RecentFailureLimit uint32
}

// DashboardService loads and composes project Dashboard sections.
type DashboardService interface {
	GetProjectDashboard(context.Context, authentication.Principal, DashboardQuery) (*protos.ProjectDashboard, error)
}
