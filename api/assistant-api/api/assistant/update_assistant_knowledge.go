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

	"github.com/rapidaai/pkg/exceptions"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) UpdateAssistantKnowledge(ctx context.Context, cawr *assistant_api.UpdateAssistantKnowledgeRequest) (*assistant_api.GetAssistantKnowledgeResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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
