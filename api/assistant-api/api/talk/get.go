// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	protos "github.com/rapidaai/protos"
)

/*
GetAllAssistantConversation retrieves all assistant conversations based on the provided request.

Parameters:
- ctx: A context.Context object that carries deadlines, cancellation signals, and other request-scoped values across API boundaries.
- cer: A pointer to protos.GetAllAssistantConversationRequest, containing the necessary parameters for retrieving conversations.

Returns:
- A pointer to protos.GetAllAssistantConversationResponse, containing the retrieved conversations and any error that occurred during the process.
- An error object, which will be nil if no error occurred.
*/
func (cApi *ConversationGrpcApi) GetAllAssistantConversation(ctx context.Context, cer *protos.GetAllAssistantConversationRequest) (*protos.GetAllAssistantConversationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return utils.Error[assistant_api.GetAllAssistantConversationResponse](
			errors.New("unauthenticated request for GetAllAssistantConversation"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	cnt, acs, err := cApi.assistantConversationService.GetAll(ctx,
		iAuth,
		cer.GetAssistantId(),
		cer.GetCriterias(),
		cer.GetPaginate(), internal_services.
			NewDefaultGetConversationOption())

	if err != nil {
		return utils.Error[protos.GetAllAssistantConversationResponse](
			err,
			"Unable to get all the assistant for the conversaction.",
		)
	}
	out := []*protos.AssistantConversation{}
	err = utils.Cast(acs, &out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.PaginatedSuccess[protos.GetAllAssistantConversationResponse, []*protos.AssistantConversation](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}

func (cApi *ConversationGrpcApi) GetAllConversationMessage(ctx context.Context, cer *protos.GetAllConversationMessageRequest) (*protos.GetAllConversationMessageResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return utils.Error[assistant_api.GetAllConversationMessageResponse](
			errors.New("unauthenticated request for GetAllConversationMessageResponse"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	cnt, acs, err := cApi.assistantConversationService.GetAllConversationMessage(
		ctx,
		iAuth,
		cer.GetAssistantConversationId(),
		cer.GetCriterias(),
		cer.GetPaginate(),
		cer.GetOrder(), internal_services.NewDefaultGetMessageOption())
	if err != nil {
		return utils.Error[protos.GetAllConversationMessageResponse](
			err,
			"Unable to get all the assistant.",
		)
	}
	out := []*protos.AssistantConversationMessage{}
	err = utils.Cast(acs, &out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllConversationMessageResponse, []*protos.AssistantConversationMessage](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}
