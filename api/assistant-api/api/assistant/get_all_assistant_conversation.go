// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAllAssistantConversation implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantConversation(ctx context.Context, cepm *assistant_api.GetAllAssistantConversationRequest) (*assistant_api.GetAllAssistantConversationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[assistant_api.GetAllAssistantConversationResponse](
			errors.New("unauthenticated request for get all assistant skills"),
			"Please provider valid service credentials to get all assistant skills, read docs @ docs.rapida.ai",
		)
	}
	cnt, conversations, err := assistantApi.conversactionService.GetAll(ctx, iAuth,
		cepm.GetAssistantId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(), internal_services.NewDefaultGetConversationOption())
	if err != nil {
		return utils.Error[assistant_api.GetAllAssistantConversationResponse](
			err,
			"Unable to get all the assistant conversation request.",
		)
	}

	out := []*assistant_api.AssistantConversation{}
	err = utils.Cast(conversations, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant conversation %v", err)
	}
	return utils.PaginatedSuccess[assistant_api.GetAllAssistantConversationResponse, []*assistant_api.AssistantConversation](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
