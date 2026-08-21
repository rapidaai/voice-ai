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

func (assistantApi *assistantGrpcApi) GetAllAssistantConfiguration(
	ctx context.Context,
	req *protos.GetAllAssistantConfigurationRequest,
) (*protos.GetAllAssistantConfigurationResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	if !validator.NonZero(req.GetAssistantId()) {
		return &protos.GetAllAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationInvalidAssistantID.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationInvalidAssistantID.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationInvalidAssistantID.Error,
				HumanMessage: pkg_errors.AssistantConfigurationInvalidAssistantID.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationInvalidAssistantID.Error)
	}
	if validator.NotBlank(req.GetConfigurationType()) && !validator.OneOf(
		req.GetConfigurationType(),
		string(internal_assistant_entity.AssistantConfigurationTypeAuthentication),
		string(internal_assistant_entity.AssistantConfigurationTypeWebhook),
		string(internal_assistant_entity.AssistantConfigurationTypeAnalysis),
		string(internal_assistant_entity.AssistantConfigurationTypeTelemetry),
		string(internal_assistant_entity.AssistantConfigurationTypeStorage),
	) {
		return &protos.GetAllAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationInvalidType.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationInvalidType.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationInvalidType.Error,
				HumanMessage: pkg_errors.AssistantConfigurationInvalidType.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationInvalidType.Error)
	}
	cnt, configurations, err := assistantApi.assistantConfigService.GetAll(
		ctx,
		iAuth,
		req.GetAssistantId(),
		req.GetConfigurationType(),
		req.GetProvider(),
		req.GetCriterias(),
		req.GetPaginate(),
	)
	if err != nil {
		return &protos.GetAllAssistantConfigurationResponse{
			Code:    pkg_errors.AssistantConfigurationGetAll.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AssistantConfigurationGetAll.Code),
				ErrorMessage: pkg_errors.AssistantConfigurationGetAll.Error,
				HumanMessage: pkg_errors.AssistantConfigurationGetAll.ErrorMessage,
			},
		}, errors.New(pkg_errors.AssistantConfigurationGetAll.Error)
	}
	out := []*protos.AssistantConfiguration{}
	if err := utils.Cast(configurations, &out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant configurations %v", err)
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantConfigurationResponse, []*protos.AssistantConfiguration](
		uint32(cnt),
		req.GetPaginate().GetPage(),
		out,
	)
}
