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

func (deploymentApi *assistantDeploymentGrpcApi) DisableAssistantWebpluginDeployment(ctx context.Context, req *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		deploymentApi.logger.Errorf("unauthenticated request for disable assistant webplugin deployment")
		return exceptions.AuthenticationError[assistant_api.GetAssistantWebpluginDeploymentResponse]()
	}

	deployment, err := deploymentApi.deploymentService.DisableAssistantWebpluginDeployment(ctx, iAuth, req.GetAssistantId())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantWebpluginDeploymentResponse]("Unable to disable assistant webplugin deployment.")
	}
	var out *assistant_api.AssistantWebpluginDeployment
	if err = utils.Cast(deployment, &out); err != nil {
		deploymentApi.logger.Errorf("unable to cast assistant webplugin deployment %v", err)
	}
	return utils.Success[assistant_api.GetAssistantWebpluginDeploymentResponse, *assistant_api.AssistantWebpluginDeployment](out)
}
