// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAssistantDashboard implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAssistantDashboard(ctx context.Context, dashboardRequest *assistant_api.GetAssistantDashboardRequest) (*assistant_api.GetAssistantDashboardResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	assistantDashboard, err := assistantApi.assistantService.GetAssistantDashboard(
		ctx,
		iAuth,
		dashboardRequest.GetAssistantId(),
		dashboardRequest.GetFromDate(),
		dashboardRequest.GetToDate(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAssistantDashboardResponse](
			err,
			"Unable to get assistant dashboard.",
		)
	}

	return utils.Success[assistant_api.GetAssistantDashboardResponse, *assistant_api.AssistantDashboard](assistantDashboard)
}
