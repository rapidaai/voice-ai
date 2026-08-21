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

func (deploymentApi *assistantDeploymentGrpcApi) DisableAssistantApiDeployment(ctx context.Context, req *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		deploymentApi.logger.Errorf("unauthenticated request for disable assistant api deployment")
		return exceptions.AuthenticationError[assistant_api.GetAssistantApiDeploymentResponse]()
	}

	deployment, err := deploymentApi.deploymentService.DisableAssistantApiDeployment(ctx, iAuth, req.GetAssistantId())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantApiDeploymentResponse]("Unable to disable assistant api deployment.")
	}
	var out *assistant_api.AssistantApiDeployment
	if err = utils.Cast(deployment, &out); err != nil {
		deploymentApi.logger.Errorf("unable to cast assistant api deployment %v", err)
	}
	return utils.Success[assistant_api.GetAssistantApiDeploymentResponse, *assistant_api.AssistantApiDeployment](out)
}
