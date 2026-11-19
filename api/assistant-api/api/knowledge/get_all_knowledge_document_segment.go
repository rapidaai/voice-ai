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

func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeDocumentSegment(ctx context.Context, cer *knowledge_api.GetAllKnowledgeDocumentSegmentRequest) (*knowledge_api.GetAllKnowledgeDocumentSegmentResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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
	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
		totalItem,
		cer.GetPaginate().GetPage(),
		_kns)
}
