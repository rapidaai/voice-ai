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

func (deploymentApi *assistantDeploymentGrpcApi) DisableAssistantPhoneDeployment(ctx context.Context, req *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	deployment, err := deploymentApi.deploymentService.DisableAssistantPhoneDeployment(ctx, iAuth, req.GetAssistantId())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantPhoneDeploymentResponse]("Unable to disable assistant phone deployment.")
	}
	var out *assistant_api.AssistantPhoneDeployment
	if err = utils.Cast(deployment, &out); err != nil {
		deploymentApi.logger.Errorf("unable to cast assistant phone deployment %v", err)
	}
	return utils.Success[assistant_api.GetAssistantPhoneDeploymentResponse, *assistant_api.AssistantPhoneDeployment](out)
}
