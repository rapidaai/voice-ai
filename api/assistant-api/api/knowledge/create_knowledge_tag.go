// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

// CreateKnowledgeTag implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) CreateKnowledgeTag(ctx context.Context, eRequest *knowledge_api.CreateKnowledgeTagRequest) (*knowledge_api.GetKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			errors.New("unauthenticated request for CreateAssistantProviderModel"),
			"Please provider valid service credentials to create assistant tag, read docs @ docs.rapida.ai",
		)
	}
	tag, err := knowledgeApi.knowledgeService.CreateOrUpdateKnowledgeTag(ctx, iAuth, eRequest.GetKnowledgeId(), eRequest.GetTags())
	if err != nil {
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			err,
			"Unable to create knowledge tags for knowledge",
		)
	}

	_kn, err := knowledgeApi.knowledgeService.Get(ctx, iAuth, tag.KnowledgeId)
	if err != nil {
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			err,
			"Unable to get knowledge, please try again later.",
		)
	}
	out := &knowledge_api.Knowledge{}
	err = utils.Cast(_kn, out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast the knowledge model to the response object")
	}
	return utils.Success[knowledge_api.GetKnowledgeResponse, *knowledge_api.Knowledge](out)
}
