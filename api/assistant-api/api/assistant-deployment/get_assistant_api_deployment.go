// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAssistantApiDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantApiDeployment(ctx context.Context, getter *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	apiDeployment, err := deploymentApi.deploymentService.GetAssistantApiDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	out := &assistant_api.AssistantApiDeployment{}
	err = utils.Cast(apiDeployment, out)
	if err != nil {
		deploymentApi.logger.Errorf("unable to cast the api deployment model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantApiDeploymentResponse, *assistant_api.AssistantApiDeployment](out)
}
