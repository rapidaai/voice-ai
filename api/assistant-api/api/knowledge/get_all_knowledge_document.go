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

// GetAllKnowledgeDocument implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeDocument(ctx context.Context, cer *knowledge_api.GetAllKnowledgeDocumentRequest) (*knowledge_api.GetAllKnowledgeDocumentResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	cnt, _kns, err := knowledgeApi.knowledgeDocumentService.GetAll(ctx, iAuth, cer.GetKnowledgeId(), cer.GetCriterias(), cer.GetPaginate())
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentResponse](
			err,
			"Unable to get Knowledge Document, please try again later.",
		)
	}

	out := []*knowledge_api.KnowledgeDocument{}
	err = utils.Cast(_kns, &out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast knowledge document %v", err)
	}
	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeDocumentResponse](
		totalItem,
		cer.GetPaginate().GetPage(),
		out)
}
