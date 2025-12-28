// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// GetAllAssistantWebhook implements protos.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantWebhook(ctx context.Context, cawr *protos.GetAllAssistantWebhookRequest) (*protos.GetAllAssistantWebhookResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAllAssistantWebhookResponse]()
	}
	cnt, epms, err := assistantApi.assistantWebhookService.GetAll(ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetCriterias(),
		cawr.GetPaginate())
	if err != nil {
		return exceptions.BadRequestError[protos.GetAllAssistantWebhookResponse]("Unable to get the assistant webhooks.")
	}
	out := []*protos.AssistantWebhook{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant provider model %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantWebhookResponse, []*protos.AssistantWebhook](
		uint32(cnt),
		cawr.GetPaginate().GetPage(),
		out)
}
