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

// CreateAssistantApiDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) CreateAssistantApiDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantApiDeploymentResponse, error) {
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
	if deployment.GetApi().UnclearInputTimeout != nil &&
		!validator.Between(*deployment.GetApi().UnclearInputTimeout, 2, 10) {
		return &assistant_api.GetAssistantApiDeploymentResponse{
			Code:    pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.Error)
	}
	if !validator.Between(int(deployment.GetApi().GetIdealTimeout()), 5, 120) {
		return &assistant_api.GetAssistantApiDeploymentResponse{
			Code:    pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.Error)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreateApiDeployment(ctx,
		iAuth, deployment.GetApi().GetAssistantId(),
		deployment.GetApi().Greeting,
		deployment.GetApi().Mistake,
		deployment.GetApi().UnclearInputTimeout,
		deployment.GetApi().UnclearInputMessage,
		deployment.GetApi().GreetingInterruptible,
		&deployment.GetApi().IdealTimeout,
		&deployment.GetApi().IdealTimeoutBackoff,
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
