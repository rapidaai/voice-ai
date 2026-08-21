// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	assistant_api "github.com/rapidaai/protos"
)

// CreateAssistantPhoneDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) CreateAssistantPhoneDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantPhoneDeploymentResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	// name, role, tone, expertise, greeting, mistake, ending string,
	//
	if deployment.GetPhone() == nil {
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for phone.",
		)
	}
	if deployment.GetPhone().UnclearInputTimeout != nil &&
		!validator.Between(*deployment.GetPhone().UnclearInputTimeout, 2, 10) {
		return &assistant_api.GetAssistantPhoneDeploymentResponse{
			Code:    pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.Error)
	}
	if !validator.Between(int(deployment.GetPhone().GetIdealTimeout()), 5, 120) {
		return &assistant_api.GetAssistantPhoneDeploymentResponse{
			Code:    pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.Code),
				ErrorMessage: pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.Error,
				HumanMessage: pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.Error)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreatePhoneDeployment(ctx,
		iAuth, deployment.GetPhone().GetAssistantId(),
		deployment.GetPhone().Greeting,
		deployment.GetPhone().Mistake,
		deployment.GetPhone().UnclearInputTimeout,
		deployment.GetPhone().UnclearInputMessage,
		deployment.GetPhone().GreetingInterruptible,
		&deployment.GetPhone().IdealTimeout,
		&deployment.GetPhone().IdealTimeoutBackoff,
		&deployment.GetPhone().IdealTimeoutMessage,
		&deployment.GetPhone().MaxSessionDuration,
		deployment.GetPhone().GetPhoneProviderName(),
		deployment.GetPhone().GetInputAudio(),
		deployment.GetPhone().GetOutputAudio(),
		deployment.GetPhone().GetPhoneOptions(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("illegal request for create assistant phone deployment"),
			"Please provider valid a valid request to create assistant phone deployment.",
		)
	}
	return utils.Success[assistant_api.GetAssistantPhoneDeploymentResponse](wpDeployment)
}
