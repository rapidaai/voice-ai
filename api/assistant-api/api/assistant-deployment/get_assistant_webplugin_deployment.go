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

	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

// GetAssistantWebpluginDeployment implements protos.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantWebpluginDeployment(ctx context.Context, getter *protos.GetAssistantDeploymentRequest) (*protos.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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
