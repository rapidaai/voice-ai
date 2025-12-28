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

// CreateAssistantDebuggerDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantDebuggerDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](
			errors.New("unauthenticated request for create assistant debugger deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	if deployment.GetDebugger() == nil {
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for debugger.",
		)
	}

	wpDeployment, err := deploymentApi.deploymentService.CreateDebuggerDeployment(ctx,
		iAuth, deployment.GetDebugger().GetAssistantId(),
		deployment.GetDebugger().Greeting,
		deployment.GetDebugger().Mistake,
		&deployment.GetDebugger().IdealTimeout,
		&deployment.GetDebugger().IdealTimeoutMessage,
		&deployment.GetDebugger().MaxSessionDuration,
		deployment.GetDebugger().GetInputAudio(),
		deployment.GetDebugger().GetOutputAudio(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](
			errors.New("unauthenticated request for create assistant debugger deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantDebuggerDeploymentResponse](wpDeployment)

}
