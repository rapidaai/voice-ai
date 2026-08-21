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

// GetAllKnowledge implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledge(ctx context.Context, cer *knowledge_api.GetAllKnowledgeRequest) (*knowledge_api.GetAllKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !hasProjectCapability(iAuth) {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetAllKnowledgeResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	cnt, _kns, err := knowledgeApi.knowledgeService.GetAll(ctx, iAuth, cer.GetCriterias(), cer.GetPaginate())
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeResponse](
			err,
			"Unable to get knowledge, please try again later.",
		)
	}

	out := []*knowledge_api.Knowledge{}
	err = utils.Cast(_kns, &out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast knowledge provider model %v", err)
	}

	for _, kn := range out {
		documentCount, wordCount, tokenCount := knowledgeApi.knowledgeDocumentService.GetCounts(ctx, iAuth, kn.Id)
		kn.DocumentCount = documentCount
		kn.WordCount = wordCount
		kn.TokenCount = tokenCount
	}

	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeResponse](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}
