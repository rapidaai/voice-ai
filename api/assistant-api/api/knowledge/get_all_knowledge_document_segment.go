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

func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeDocumentSegment(ctx context.Context, cer *knowledge_api.GetAllKnowledgeDocumentSegmentRequest) (*knowledge_api.GetAllKnowledgeDocumentSegmentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	knowledge, err := knowledgeApi.knowledgeService.Get(ctx, iAuth, cer.GetKnowledgeId())
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
			err,
			"Unable to get Knowledge, or you do not have access to the knowledge.",
		)
	}

	cnt, _kns, err := knowledgeApi.knowledgeDocumentService.GetAllDocumentSegment(
		ctx,
		iAuth,
		cer.GetKnowledgeId(),
		knowledge.StorageNamespace,
		cer.GetCriterias(),
		cer.GetPaginate(),
	)
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
			err,
			"Unable to get Knowledge Document Segment, please try again later.",
		)
	}
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		_kns)
}
