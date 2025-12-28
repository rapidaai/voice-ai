// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAllAssistantMessage implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantMessage(ctx context.Context, cepm *assistant_api.GetAllAssistantMessageRequest) (*assistant_api.GetAllAssistantMessageResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAllAssistantMessageResponse]()
	}
	cnt, epms, err := assistantApi.conversactionService.GetAllAssistantMessage(ctx,
		iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(), cepm.GetOrder(),
		internal_services.NewGetMessageOption().WithFieldSelector(cepm.GetSelectors()))
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAllAssistantMessageResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*assistant_api.AssistantConversationMessage{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllAssistantMessageResponse, []*assistant_api.AssistantConversationMessage](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
