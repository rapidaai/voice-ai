// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// GetAllConversationMessage implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllConversationMessage(ctx context.Context, cepm *assistant_api.GetAllConversationMessageRequest) (*assistant_api.GetAllConversationMessageResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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

	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[assistant_api.GetAllConversationMessageResponse, []*assistant_api.AssistantConversationMessage](
		totalItem,
		cepm.GetPaginate().GetPage(),
		out)
}
