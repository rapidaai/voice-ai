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

func (assistantApi *assistantDeploymentGrpcApi) GetAllAssistantWhatsappDeployment(
	ctx context.Context,
	req *assistant_api.GetAllAssistantDeploymentRequest,
) (*assistant_api.GetAllAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		assistantApi.logger.Errorf("unauthenticated request for get all assistant whatsapp deployments")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantWhatsappDeploymentResponse]()
	}
	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &assistant_api.Paginate{Page: 1, PageSize: 20}
	}
	cnt, deployments, err := assistantApi.deploymentService.GetAllAssistantWhatsappDeployment(
		ctx,
		iAuth,
		req.GetAssistantId(),
		req.GetCriterias(),
		paginate,
	)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantWhatsappDeploymentResponse]("Unable to get assistant whatsapp deployments.")
	}
	out := []*assistant_api.AssistantWhatsappDeployment{}
	if err = utils.Cast(deployments, &out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant whatsapp deployments %v", err)
	}
	return utils.PaginatedSuccess[assistant_api.GetAllAssistantWhatsappDeploymentResponse, []*assistant_api.AssistantWhatsappDeployment](
		uint32(cnt),
		paginate.GetPage(),
		out,
	)
}
