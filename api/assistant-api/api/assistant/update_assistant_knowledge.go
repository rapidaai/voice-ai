// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"

	"github.com/rapidaai/pkg/exceptions"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) UpdateAssistantKnowledge(ctx context.Context, cawr *assistant_api.UpdateAssistantKnowledgeRequest) (*assistant_api.GetAssistantKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for UpdateAssistantKnowledge")
		return exceptions.AuthenticationError[assistant_api.GetAssistantKnowledgeResponse]()
	}
	wl, err := assistantApi.assistantKnowledgeService.Update(
		ctx,
		iAuth,
		cawr.GetId(),
		cawr.GetAssistantId(),
		cawr.GetKnowledgeId(),
		gorm_types.RetrievalMethod(cawr.GetRetrievalMethod()),
		cawr.GetRerankerEnable(),
		cawr.GetScoreThreshold(),
		cawr.GetTopK(),
		&cawr.RerankerModelProviderId,
		&cawr.RerankerModelProviderName,
		cawr.GetAssistantKnowledgeRerankerOptions())
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantKnowledgeResponse](err.Error())
	}
	aAnalysis := &assistant_api.AssistantKnowledge{}
	err = utils.Cast(wl, aAnalysis)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant knowledge to the response object")
	}
	return utils.Success[assistant_api.GetAssistantKnowledgeResponse, *assistant_api.AssistantKnowledge](aAnalysis)
}
