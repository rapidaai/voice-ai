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

func (assistantApi *assistantGrpcApi) GetAllAssistantWebhookLog(ctx context.Context, gaar *protos.GetAllAssistantWebhookLogRequest) (*protos.GetAllAssistantWebhookLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAllAssistantWebhookLogResponse]()
	}
	cnt, epms, err := assistantApi.assistantWebhookService.GetAllLog(ctx,
		iAuth,
		gaar.GetProjectId(),
		gaar.GetCriterias(),
		gaar.GetPaginate(),
		gaar.GetOrder())
	if err != nil {
		return exceptions.BadRequestError[protos.GetAllAssistantWebhookLogResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*protos.AssistantWebhookLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant webhook logs %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantWebhookLogResponse, []*protos.AssistantWebhookLog](
		uint32(cnt),
		gaar.GetPaginate().GetPage(),
		out)
}
