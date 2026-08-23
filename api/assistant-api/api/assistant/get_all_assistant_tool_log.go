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

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAllAssistantToolLog(ctx context.Context, gaar *protos.GetAllAssistantToolLogRequest) (*protos.GetAllAssistantToolLogResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	cnt, epms, err := assistantApi.assistantToolService.GetAllLog(ctx,
		iAuth,
		gaar.GetProjectId(),
		gaar.GetCriterias(),
		gaar.GetPaginate(),
		gaar.GetOrder())
	if err != nil {
		return exceptions.BadRequestError[protos.GetAllAssistantToolLogResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*protos.AssistantToolLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant webhook logs %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantToolLogResponse, []*protos.AssistantToolLog](
		uint32(cnt),
		gaar.GetPaginate().GetPage(),
		out)
}
