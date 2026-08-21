// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"

	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) DeleteAssistantConfiguration(
	ctx context.Context,
	req *protos.DeleteAssistantConfigurationRequest,
) (*protos.GetAssistantConfigurationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationUnauthenticated.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationUnauthenticated.Error,
				HumanMessage: pkg_errors.AssistantConfigurationUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationUnauthenticated.Error)
	}
	if !hasUserProjectCapability(iAuth) {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationMissingAuthScope.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationMissingAuthScope.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationMissingAuthScope.Error,
				HumanMessage: pkg_errors.AssistantConfigurationMissingAuthScope.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationMissingAuthScope.Error)
	}
	if !validator.NonZero(req.GetAssistantId()) {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationInvalidAssistantID.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationInvalidAssistantID.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationInvalidAssistantID.Error,
				HumanMessage: pkg_errors.AssistantConfigurationInvalidAssistantID.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationInvalidAssistantID.Error)
	}
	if !validator.NonZero(req.GetId()) {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationInvalidID.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationInvalidID.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationInvalidID.Error,
				HumanMessage: pkg_errors.AssistantConfigurationInvalidID.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationInvalidID.Error)
	}
	configuration, err := assistantApi.assistantConfigService.Delete(ctx, iAuth, req.GetId(), req.GetAssistantId())
	if err != nil {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationDelete.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationDelete.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationDelete.Error,
				HumanMessage: pkg_errors.AssistantConfigurationDelete.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationDelete.Error)
	}
	out := &protos.AssistantConfiguration{}
	if err := utils.Cast(configuration, out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant configuration %v", err)
	}
	return utils.Success[protos.GetAssistantConfigurationResponse, *protos.AssistantConfiguration](out)
}
