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

func (iApi *indexerApi) IndexKnowledgeDocument(ctx context.Context, cer *knowledge_api.IndexKnowledgeDocumentRequest) (*knowledge_api.IndexKnowledgeDocumentResponse, error) {
	iApi.logger.Debugf("index document request %v, %v", cer, ctx)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	_, projectAuthErr := types.RequireProject(iAuth)
	if !isAuthenticated || projectAuthErr != nil {
		iApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.IndexKnowledgeDocumentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	return iApi.indexerServiceClient.IndexKnowledgeDocument(ctx, iAuth,
		&knowledge_api.IndexKnowledgeDocumentRequest{
			KnowledgeId:         cer.GetKnowledgeId(),
			KnowledgeDocumentId: cer.GetKnowledgeDocumentId(),
		})
}
