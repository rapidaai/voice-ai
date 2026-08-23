// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAssistantDebuggerDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantDebuggerDeployment(ctx context.Context, getter *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantDebuggerDeploymentResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	debuggerDeployment, err := deploymentApi.deploymentService.GetAssistantDebuggerDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	var out *assistant_api.AssistantDebuggerDeployment
	err = utils.Cast(debuggerDeployment, &out)
	if err != nil {
		deploymentApi.logger.Errorf("unable to cast the api deployment model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantDebuggerDeploymentResponse, *assistant_api.AssistantDebuggerDeployment](out)
}
