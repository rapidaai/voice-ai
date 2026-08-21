// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"context"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantDeploymentGrpcApi) GetAllAssistantWebpluginDeployment(
	ctx context.Context,
	req *assistant_api.GetAllAssistantDeploymentRequest,
) (*assistant_api.GetAllAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		assistantApi.logger.Errorf("unauthenticated request for get all assistant webplugin deployments")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantWebpluginDeploymentResponse]()
	}
	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &assistant_api.Paginate{Page: 1, PageSize: 20}
	}
	cnt, deployments, err := assistantApi.deploymentService.GetAllAssistantWebpluginDeployment(
		ctx,
		iAuth,
		req.GetAssistantId(),
		req.GetCriterias(),
		paginate,
	)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantWebpluginDeploymentResponse]("Unable to get assistant webplugin deployments.")
	}
	out := []*assistant_api.AssistantWebpluginDeployment{}
	if err = utils.Cast(deployments, &out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant webplugin deployments %v", err)
	}
	return utils.PaginatedSuccess[assistant_api.GetAllAssistantWebpluginDeploymentResponse, []*assistant_api.AssistantWebpluginDeployment](
		uint32(cnt),
		paginate.GetPage(),
		out,
	)
}
