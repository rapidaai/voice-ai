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
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAssistantConversation(ctx context.Context, cepm *protos.GetAssistantConversationRequest) (*protos.GetAssistantConversationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[protos.GetAssistantConversationResponse](
			errors.New("unauthenticated request for get assistant converstaion"),
			"Please provider valid service credentials to perform GetAssistantConversation, read docs @ docs.rapida.ai",
		)
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
