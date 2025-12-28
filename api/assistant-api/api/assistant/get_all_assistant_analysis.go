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

func (assistantApi *assistantGrpcApi) GetAllAssistantAnalysis(ctx context.Context, cawr *protos.GetAllAssistantAnalysisRequest) (*protos.GetAllAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAllAssistantAnalysisResponse]()
	}
	cnt, epms, err := assistantApi.assistantAnalysisService.GetAll(ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetCriterias(),
		cawr.GetPaginate())
	if err != nil {
		return exceptions.BadRequestError[protos.GetAllAssistantAnalysisResponse]("Unable to get the assistant webhooks.")
	}
	out := []*protos.AssistantAnalysis{}
	err = utils.Cast(epms, &out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant analysis %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantAnalysisResponse, []*protos.AssistantAnalysis](
		uint32(cnt),
		cawr.GetPaginate().GetPage(),
		out)
}
