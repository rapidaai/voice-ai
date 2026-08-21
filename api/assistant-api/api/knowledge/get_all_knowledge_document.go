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

// GetAllKnowledgeDocument implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeDocument(ctx context.Context, cer *knowledge_api.GetAllKnowledgeDocumentRequest) (*knowledge_api.GetAllKnowledgeDocumentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
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
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeDocumentResponse](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}
