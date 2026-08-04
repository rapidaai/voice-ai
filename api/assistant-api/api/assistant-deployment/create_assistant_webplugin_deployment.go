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

// CreateAssistantWebpluginDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) CreateAssistantWebpluginDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantWebpluginDeploymentResponse](
			errors.New("unauthenticated request for create assistant web plugin deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	if deployment.GetPlugin() == nil {
		return utils.Error[assistant_api.GetAssistantWebpluginDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for webplugin.",
		)
	}
	if deployment.GetPlugin().UnclearInputTimeout != nil &&
		!validator.Between(*deployment.GetPlugin().UnclearInputTimeout, 0.5, 5) {
		return &assistant_api.GetAssistantWebpluginDeploymentResponse{
			Code:    pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.Error)
	}
	if !validator.Between(int(deployment.GetPlugin().GetIdealTimeout()), 5, 120) {
		return &assistant_api.GetAssistantWebpluginDeploymentResponse{
			Code:    pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.Error)
	}

	wpDeployment, err := deploymentApi.deploymentService.CreateWebPluginDeployment(ctx,
		iAuth, deployment.GetPlugin().GetAssistantId(),
		deployment.GetPlugin().Greeting,
		deployment.GetPlugin().Mistake,
		deployment.GetPlugin().UnclearInputTimeout,
		deployment.GetPlugin().UnclearInputMessage,
		deployment.GetPlugin().GreetingInterruptible,
		&deployment.GetPlugin().IdealTimeout,
		&deployment.GetPlugin().IdealTimeoutBackoff,
		&deployment.GetPlugin().IdealTimeoutMessage,
		&deployment.GetPlugin().MaxSessionDuration,
		deployment.GetPlugin().GetSuggestion(),
		deployment.GetPlugin().GetInputAudio(),
		deployment.GetPlugin().GetOutputAudio(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantWebpluginDeploymentResponse](
			errors.New("unauthenticated request for create assistant webplugin deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantWebpluginDeploymentResponse](wpDeployment)
}
