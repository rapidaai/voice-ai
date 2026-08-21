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

func (deploymentApi *assistantDeploymentGrpcApi) DisableAssistantDebuggerDeployment(ctx context.Context, req *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		deploymentApi.logger.Errorf("unauthenticated request for disable assistant debugger deployment")
		return exceptions.AuthenticationError[assistant_api.GetAssistantDebuggerDeploymentResponse]()
	}

	deployment, err := deploymentApi.deploymentService.DisableAssistantDebuggerDeployment(ctx, iAuth, req.GetAssistantId())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantDebuggerDeploymentResponse]("Unable to disable assistant debugger deployment.")
	}
	var out *assistant_api.AssistantDebuggerDeployment
	if err = utils.Cast(deployment, &out); err != nil {
		deploymentApi.logger.Errorf("unable to cast assistant debugger deployment %v", err)
	}
	return utils.Success[assistant_api.GetAssistantDebuggerDeploymentResponse, *assistant_api.AssistantDebuggerDeployment](out)
}
