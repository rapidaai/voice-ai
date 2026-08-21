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
	"github.com/rapidaai/protos"
)

// CreateKnowledge implements protos.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) CreateKnowledge(ctx context.Context, cer *protos.CreateKnowledgeRequest) (*protos.CreateKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !hasProjectCapability(iAuth) {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[protos.CreateKnowledgeResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
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
