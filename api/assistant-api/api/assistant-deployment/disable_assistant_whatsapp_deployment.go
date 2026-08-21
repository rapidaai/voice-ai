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

func (deploymentApi *assistantDeploymentGrpcApi) DisableAssistantWhatsappDeployment(ctx context.Context, req *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !hasProjectCapability(iAuth) {
		deploymentApi.logger.Errorf("unauthenticated request for disable assistant whatsapp deployment")
		return exceptions.AuthenticationError[assistant_api.GetAssistantWhatsappDeploymentResponse]()
	}

	deployment, err := deploymentApi.deploymentService.DisableAssistantWhatsappDeployment(ctx, iAuth, req.GetAssistantId())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantWhatsappDeploymentResponse]("Unable to disable assistant whatsapp deployment.")
	}
	var out *assistant_api.AssistantWhatsappDeployment
	if err = utils.Cast(deployment, &out); err != nil {
		deploymentApi.logger.Errorf("unable to cast assistant whatsapp deployment %v", err)
	}
	return utils.Success[assistant_api.GetAssistantWhatsappDeploymentResponse, *assistant_api.AssistantWhatsappDeployment](out)
}
