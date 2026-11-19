// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

// GetAllKnowledge implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledge(ctx context.Context, cer *knowledge_api.GetAllKnowledgeRequest) (*knowledge_api.GetAllKnowledgeResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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

	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeResponse](
		totalItem,
		cer.GetPaginate().GetPage(),
		out)
}
