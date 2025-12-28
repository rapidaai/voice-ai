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

func (assistantApi *assistantGrpcApi) UpdateAssistantAnalysis(ctx context.Context, cawr *protos.UpdateAssistantAnalysisRequest) (*protos.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantAnalysis")
		return exceptions.AuthenticationError[protos.GetAssistantAnalysisResponse]()
	}
	wl, err := assistantApi.assistantAnalysisService.Update(
		ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetId(),
		cawr.GetName(),
		cawr.GetEndpointId(),
		cawr.GetEndpointVersion(),
		cawr.GetEndpointParameters(),
		cawr.GetExecutionPriority(),
		&cawr.Description)
	if err != nil {
		return exceptions.BadRequestError[protos.GetAssistantAnalysisResponse]("Unable to create assistant webhook.")
	}
	aAnalysis := &protos.AssistantAnalysis{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[protos.GetAssistantAnalysisResponse, *protos.AssistantAnalysis](aAnalysis)
}
