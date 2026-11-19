// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"context"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantDeploymentGrpcApi) GetAllAssistantDebuggerDeployment(
	ctx context.Context,
	req *assistant_api.GetAllAssistantDeploymentRequest,
) (*assistant_api.GetAllAssistantDebuggerDeploymentResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &assistant_api.Paginate{Page: 1, PageSize: 20}
	}
	cnt, deployments, err := assistantApi.deploymentService.GetAllAssistantDebuggerDeployment(
		ctx,
		iAuth,
		req.GetAssistantId(),
		req.GetCriterias(),
		paginate,
	)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantDebuggerDeploymentResponse]("Unable to get assistant debugger deployments.")
	}
	out := []*assistant_api.AssistantDebuggerDeployment{}
	if err = utils.Cast(deployments, &out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant debugger deployments %v", err)
	}
	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[assistant_api.GetAllAssistantDebuggerDeploymentResponse, []*assistant_api.AssistantDebuggerDeployment](
		totalItem,
		paginate.GetPage(),
		out,
	)
}
