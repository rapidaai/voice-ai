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
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAssistantConversation(ctx context.Context, cepm *protos.GetAssistantConversationRequest) (*protos.GetAssistantConversationResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	ep, err := assistantApi.conversactionService.Get(ctx,
		iAuth, cepm.
			GetAssistantId(),
		cepm.
			GetId(),
		internal_services.
			NewDefaultGetConversationOption().
			WithFieldSelector(
				cepm.
					GetSelectors(),
			))
	if err != nil {
		return utils.Error[protos.
			GetAssistantConversationResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}
	out := &protos.AssistantConversation{}
	err = utils.Cast(ep, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant %v", err)
	}
	return &protos.GetAssistantConversationResponse{
		Data:    out,
		Success: true,
		Code:    200,
	}, nil
}
