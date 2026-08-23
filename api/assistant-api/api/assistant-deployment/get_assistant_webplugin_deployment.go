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
	protos "github.com/rapidaai/protos"
)

// GetAssistantWebpluginDeployment implements protos.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantWebpluginDeployment(ctx context.Context, getter *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	webpluginDeployment, err := deploymentApi.deploymentService.GetAssistantWebpluginDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[protos.GetAssistantWebpluginDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	var out *protos.AssistantWebpluginDeployment
	err = utils.Cast(webpluginDeployment, &out)
	if err != nil {
		deploymentApi.logger.Warnf("unable to cast the web plugin deployment model to the response object")
	}
	return &protos.GetAssistantWebpluginDeploymentResponse{
		Data:    out,
		Success: true,
		Code:    200,
	}, nil
}
