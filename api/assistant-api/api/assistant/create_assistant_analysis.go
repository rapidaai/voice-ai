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
	assistant_api "github.com/rapidaai/protos"
)

// CreateAssistantWebhook implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantAnalysis(ctx context.Context, cawr *assistant_api.CreateAssistantAnalysisRequest) (*assistant_api.GetAssistantAnalysisResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[assistant_api.GetAssistantAnalysisResponse]()
	}
	wl, err := assistantApi.assistantAnalysisService.Create(
		ctx,
		iAuth,
		cawr.GetAssistantId(),
		cawr.GetName(),
		cawr.GetEndpointId(),
		cawr.GetEndpointVersion(),
		cawr.GetEndpointParameters(),
		cawr.GetExecutionPriority(),
		&cawr.Description)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantAnalysisResponse]("Unable to create assistant analysis.")
	}
	aAnalysis := &assistant_api.AssistantAnalysis{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant analysis to the response object")
	}
	return utils.Success[assistant_api.GetAssistantAnalysisResponse, *assistant_api.AssistantAnalysis](aAnalysis)
}
