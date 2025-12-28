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

// CreateAssistantWhatsappDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantWhatsappDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantWhatsappDeploymentResponse, error) {
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
	wpDeployment, err := deploymentApi.deploymentService.CreateWhatsappDeployment(ctx,
		iAuth, deployment.GetWhatsapp().GetAssistantId(),
		deployment.GetWhatsapp().Greeting,
		deployment.GetWhatsapp().Mistake,
		&deployment.GetWhatsapp().IdealTimeout,
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
