// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAllAssistant implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantTool(ctx context.Context, cepm *assistant_api.GetAllAssistantToolRequest) (*assistant_api.GetAllAssistantToolResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	cnt, assistants, err := assistantApi.assistantToolService.GetAll(ctx, iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAllAssistantToolResponse](
			err,
			"Unable to get all the skill request.",
		)
	}
	out := []*assistant_api.AssistantTool{}
	err = utils.Cast(assistants, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant skill %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantToolResponse, []*assistant_api.AssistantTool](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
