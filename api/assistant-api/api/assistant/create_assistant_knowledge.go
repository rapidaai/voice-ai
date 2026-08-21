// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/exceptions"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// CreateAssistantKnowledgeConfiguration implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantKnowledge(ctx context.Context, cepm *assistant_api.CreateAssistantKnowledgeRequest) (*assistant_api.GetAssistantKnowledgeResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	aK, err := assistantApi.assistantKnowledgeService.Create(
		ctx,
		iAuth,
		cepm.GetAssistantId(),
		cepm.GetKnowledgeId(),
		gorm_types.RetrievalMethod(cepm.GetRetrievalMethod()),
		cepm.GetRerankerEnable(),
		cepm.GetScoreThreshold(),
		cepm.GetTopK(),
		&cepm.RerankerModelProviderId,
		&cepm.RerankerModelProviderName,
		cepm.GetAssistantKnowledgeRerankerOptions(),
	)
	if err != nil {
		return exceptions.BadRequestError[assistant_api.GetAssistantKnowledgeResponse](err.Error())
	}

	out := &assistant_api.AssistantKnowledge{}
	err = utils.Cast(aK, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast assistant knowledge %v", err)
	}
	return utils.Success[assistant_api.GetAssistantKnowledgeResponse, *assistant_api.AssistantKnowledge](out)
}

func (assistantApi *assistantGrpcApi) createAssistantKnowledge(
	ctx context.Context,
	iAuth *types.Authentication,
	assistantId uint64, cepm *assistant_api.CreateAssistantKnowledgeRequest) (*internal_assistant_entity.AssistantKnowledge, error) {
	return assistantApi.assistantKnowledgeService.Create(
		ctx,
		iAuth,
		assistantId,
		cepm.GetKnowledgeId(),
		gorm_types.RetrievalMethod(cepm.GetRetrievalMethod()),
		cepm.GetRerankerEnable(),
		cepm.GetScoreThreshold(),
		cepm.GetTopK(),
		&cepm.RerankerModelProviderId,
		&cepm.RerankerModelProviderName,
		cepm.GetAssistantKnowledgeRerankerOptions(),
	)

}
