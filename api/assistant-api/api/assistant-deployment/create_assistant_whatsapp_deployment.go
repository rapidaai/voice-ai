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

// CreateAssistantWhatsappDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) CreateAssistantWhatsappDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	if deployment.GetWhatsapp() == nil {
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for whatsapp.",
		)
	}
	if deployment.GetWhatsapp().UnclearInputTimeout != nil &&
		!validator.Between(*deployment.GetWhatsapp().UnclearInputTimeout, 2, 10) {
		return &assistant_api.GetAssistantWhatsappDeploymentResponse{
			Code:    pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.Error)
	}
	if !validator.Between(int(deployment.GetWhatsapp().GetIdealTimeout()), 5, 120) {
		return &assistant_api.GetAssistantWhatsappDeploymentResponse{
			Code:    pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.Error)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreateWhatsappDeployment(ctx,
		iAuth, deployment.GetWhatsapp().GetAssistantId(),
		deployment.GetWhatsapp().Greeting,
		deployment.GetWhatsapp().Mistake,
		deployment.GetWhatsapp().UnclearInputTimeout,
		deployment.GetWhatsapp().UnclearInputMessage,
		deployment.GetWhatsapp().GreetingInterruptible,
		&deployment.GetWhatsapp().IdealTimeout,
		&deployment.GetWhatsapp().IdealTimeoutBackoff,
		&deployment.GetWhatsapp().IdealTimeoutMessage,
		&deployment.GetWhatsapp().MaxSessionDuration,
		deployment.GetWhatsapp().GetWhatsappProviderName(),
		deployment.GetWhatsapp().GetWhatsappOptions(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](
			errors.New("unauthenticated request for create assistant debugger deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantWhatsappDeploymentResponse](wpDeployment)
}
