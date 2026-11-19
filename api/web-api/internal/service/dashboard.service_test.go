// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service

import (
	"context"
	"testing"
	"time"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/protos"
)

type dashboardServiceContractStub struct {
	query DashboardQuery
}

func (stub *dashboardServiceContractStub) GetProjectDashboard(_ context.Context, _ authentication.Principal, query DashboardQuery) (*protos.ProjectDashboard, error) {
	stub.query = query
	return &protos.ProjectDashboard{}, nil
}

func TestDashboardServiceContractAcceptsResolvedQuery(t *testing.T) {
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	query := DashboardQuery{
		GeneratedAt:        now,
		FromDate:           now.Add(-24 * time.Hour),
		ToDate:             now,
		ComparisonFromDate: now.Add(-48 * time.Hour),
		ComparisonToDate:   now.Add(-24 * time.Hour),
		AssistantLimit:     3,
		RecentFailureLimit: 5,
	}
	stub := &dashboardServiceContractStub{}
	var service DashboardService = stub

	if _, err := service.GetProjectDashboard(context.Background(), authentication.Principal{}, query); err != nil {
		t.Fatal(err)
	}
	if stub.query != query {
		t.Fatalf("query = %+v, want %+v", stub.query, query)
	}
}
