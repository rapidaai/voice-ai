// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// CreateKnowledge implements protos.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) CreateKnowledge(ctx context.Context, cer *protos.CreateKnowledgeRequest) (*protos.CreateKnowledgeResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	_kn, err := knowledgeApi.knowledgeService.CreateKnowledge(ctx, iAuth,
		cer.GetName(),
		&cer.Description,
		&cer.Visibility,
		cer.GetEmbeddingModelProviderName(),
		cer.GetKnowledgeEmbeddingModelOptions(),
	)
	if err != nil {
		return utils.Error[protos.CreateKnowledgeResponse](
			err,
			"Unable to create knowledge, please try again later.",
		)
	}

	_, err = knowledgeApi.knowledgeService.CreateOrUpdateKnowledgeTag(ctx, iAuth, _kn.Id, cer.GetTags())
	if err != nil {
		return utils.Error[protos.CreateKnowledgeResponse](
			err,
			"Unable to create knowledge tags, please try again.",
		)
	}

	out := &protos.Knowledge{}
	err = utils.Cast(_kn, out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast the knowledge model to the response object")
	}
	return utils.Success[protos.CreateKnowledgeResponse, *protos.Knowledge](out)

}
