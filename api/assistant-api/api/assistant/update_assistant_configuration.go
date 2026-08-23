// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) UpdateAssistantConfiguration(
	ctx context.Context,
	req *protos.UpdateAssistantConfigurationRequest,
) (*protos.GetAssistantConfigurationResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
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
	if !validator.NotBlank(req.GetConfigurationType()) {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationMissingType.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationMissingType.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationMissingType.Error,
				HumanMessage: pkg_errors.AssistantConfigurationMissingType.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationMissingType.Error)
	}
	if !validator.OneOf(
		req.GetConfigurationType(),
		string(internal_assistant_entity.AssistantConfigurationTypeAuthentication),
		string(internal_assistant_entity.AssistantConfigurationTypeWebhook),
		string(internal_assistant_entity.AssistantConfigurationTypeAnalysis),
		string(internal_assistant_entity.AssistantConfigurationTypeTelemetry),
		string(internal_assistant_entity.AssistantConfigurationTypeStorage),
	) {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationInvalidType.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationInvalidType.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationInvalidType.Error,
				HumanMessage: pkg_errors.AssistantConfigurationInvalidType.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationInvalidType.Error)
	}
	if !validator.NotBlank(req.GetProvider()) {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationMissingProvider.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationMissingProvider.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationMissingProvider.Error,
				HumanMessage: pkg_errors.AssistantConfigurationMissingProvider.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationMissingProvider.Error)
	}
	for _, option := range req.GetOptions() {
		if option == nil || !validator.NotBlank(option.GetKey()) {
			return &protos.GetAssistantConfigurationResponse{
				Code:    pkg_errors.AssistantConfigurationInvalidOption.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.AssistantConfigurationInvalidOption.Code),
					ErrorMessage: pkg_errors.AssistantConfigurationInvalidOption.Error,
					HumanMessage: pkg_errors.AssistantConfigurationInvalidOption.ErrorMessage,
				},
			}, errors.New(pkg_errors.AssistantConfigurationInvalidOption.Error)
		}
	}
	configuration, err := assistantApi.assistantConfigService.Update(
		ctx,
		iAuth,
		req.GetId(),
		req.GetAssistantId(),
		req.GetConfigurationType(),
		req.GetProvider(),
		req.GetEnabled(),
		req.GetOptions(),
	)
	if err != nil {
		return &protos.GetAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationUpdate.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationUpdate.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationUpdate.Error,
				HumanMessage: pkg_errors.AssistantConfigurationUpdate.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationUpdate.Error)
	}
	out := &protos.AssistantConfiguration{}
	if err := utils.Cast(configuration, out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant configuration %v", err)
	}
	return utils.Success[protos.GetAssistantConfigurationResponse, *protos.AssistantConfiguration](out)
}
