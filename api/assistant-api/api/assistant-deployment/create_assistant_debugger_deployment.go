// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"context"
	"errors"

	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	assistant_api "github.com/rapidaai/protos"
)

// CreateAssistantDebuggerDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) CreateAssistantDebuggerDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantDebuggerDeploymentResponse, error) {
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
	if !validator.Between(int(deployment.GetDebugger().GetIdealTimeout()), 5, 120) {
		return &assistant_api.GetAssistantDebuggerDeploymentResponse{
			Code:    pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.Error)
	}
	if !validator.Between(int(deployment.GetDebugger().GetIdealTimeoutBackoff()), 0, 5) {
		return &assistant_api.GetAssistantDebuggerDeploymentResponse{
			Code:    pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.Code),
				ErrorMessage: pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.Error,
				HumanMessage: pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.Error)
	}
	if !validator.Between(int(deployment.GetDebugger().GetMaxSessionDuration()), 180, 600) {
		return &assistant_api.GetAssistantDebuggerDeploymentResponse{
			Code:    pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.Code),
				ErrorMessage: pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.Error,
				HumanMessage: pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.Error)
	}

	wpDeployment, err := deploymentApi.deploymentService.CreateDebuggerDeployment(ctx,
		iAuth, deployment.GetDebugger().GetAssistantId(),
		deployment.GetDebugger().Greeting,
		deployment.GetDebugger().Mistake,
		deployment.GetDebugger().UnclearInputTimeout,
		deployment.GetDebugger().UnclearInputMessage,
		deployment.GetDebugger().GreetingInterruptible,
		&deployment.GetDebugger().IdealTimeout,
		&deployment.GetDebugger().IdealTimeoutBackoff,
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
