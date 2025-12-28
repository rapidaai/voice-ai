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

// CreateAssistantApiDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantApiDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("unauthenticated request for create assistant api deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	if deployment.GetApi() == nil {
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for api.",
		)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreateApiDeployment(ctx,
		iAuth, deployment.GetApi().GetAssistantId(),
		deployment.GetApi().Greeting,
		deployment.GetApi().Mistake,
		&deployment.GetApi().IdealTimeout,
		&deployment.GetApi().IdealTimeoutMessage,
		&deployment.GetApi().MaxSessionDuration,
		deployment.GetApi().GetInputAudio(),
		deployment.GetApi().GetOutputAudio(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("unauthenticated request for create assistant api deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantApiDeploymentResponse](wpDeployment)
}
