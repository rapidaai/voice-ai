package assistant_talk_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
)

/*
GetAllAssistantConversation retrieves all assistant conversations based on the provided request.

Parameters:
- ctx: A context.Context object that carries deadlines, cancellation signals, and other request-scoped values across API boundaries.
- cer: A pointer to lexatic_backend.GetAllAssistantConversationRequest, containing the necessary parameters for retrieving conversations.

Returns:
- A pointer to lexatic_backend.GetAllAssistantConversationResponse, containing the retrieved conversations and any error that occurred during the process.
- An error object, which will be nil if no error occurred.
*/
func (cApi *ConversationGrpcApi) GetAllAssistantConversation(ctx context.Context, cer *lexatic_backend.GetAllAssistantConversationRequest) (*lexatic_backend.GetAllAssistantConversationResponse, error) {
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
		return utils.Error[lexatic_backend.GetAllAssistantConversationResponse](
			err,
			"Unable to get all the assistant for the conversaction.",
		)
	}
	out := []*lexatic_backend.AssistantConversation{}
	err = utils.Cast(acs, &out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}
	return utils.PaginatedSuccess[lexatic_backend.GetAllAssistantConversationResponse, []*lexatic_backend.AssistantConversation](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}

func (cApi *ConversationGrpcApi) GetAllConversationMessage(ctx context.Context, cer *lexatic_backend.GetAllConversationMessageRequest) (*lexatic_backend.GetAllConversationMessageResponse, error) {
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
		return utils.Error[lexatic_backend.GetAllConversationMessageResponse](
			err,
			"Unable to get all the assistant.",
		)
	}
	out := []*lexatic_backend.AssistantConversationMessage{}
	err = utils.Cast(acs, &out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[lexatic_backend.GetAllConversationMessageResponse, []*lexatic_backend.AssistantConversationMessage](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}
