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

// GetAllConversationMessage implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllConversationMessage(ctx context.Context, cepm *assistant_api.GetAllConversationMessageRequest) (*assistant_api.GetAllConversationMessageResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for GetAllassistant")
		return utils.Error[assistant_api.GetAllConversationMessageResponse](
			errors.New("unauthenticated request for get all assistant skills"),
			"Please provider valid service credentials to get all assistant skills, read docs @ docs.rapida.ai",
		)
	}
	cnt, messages, err := assistantApi.conversactionService.GetAllConversationMessage(ctx, iAuth,
		cepm.GetAssistantConversationId(),
		cepm.GetCriterias(),
		cepm.GetPaginate(),
		cepm.GetOrder(),
		internal_services.NewDefaultGetMessageOption())
	if err != nil {
		return utils.Error[assistant_api.GetAllConversationMessageResponse](
			err,
			"Unable to get all the conversation messages.",
		)
	}
	out := []*assistant_api.AssistantConversationMessage{}
	err = utils.Cast(messages, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant skill %v", err)
	}

	return utils.PaginatedSuccess[assistant_api.GetAllConversationMessageResponse, []*assistant_api.AssistantConversationMessage](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
