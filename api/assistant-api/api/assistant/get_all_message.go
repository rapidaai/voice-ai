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

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAllMessage(ctx context.Context, cepm *assistant_api.GetAllMessageRequest) (*assistant_api.GetAllMessageResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	cnt, epms, err := assistantApi.conversactionService.GetAllMessage(ctx,
		iAuth,
		cepm.GetCriterias(),
		cepm.GetPaginate(), cepm.GetOrder(),
		internal_services.NewGetMessageOption().WithFieldSelector(cepm.GetSelectors()))
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllMessageResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*assistant_api.AssistantConversationMessage{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllMessageResponse, []*assistant_api.AssistantConversationMessage](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
