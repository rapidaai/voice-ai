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

// DeleteKnowledgeDocumentSegment implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) DeleteKnowledgeDocumentSegment(ctx context.Context, dsr *knowledge_api.DeleteKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.BaseResponse](
			errors.New("unauthenticated request for delete knowledge document sgement"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	_, err := knowledgeApi.knowledgeDocumentService.DeleteDocumentSegment(ctx, iAuth, dsr.GetIndex(), dsr.GetDocumentId(), dsr.GetReason())
	if err != nil {
		knowledgeApi.logger.Errorf("unable to delete knowledge segment with error %v", err)
		return utils.Error[knowledge_api.BaseResponse](
			err,
			"Unable to delete document segment, please try again.",
		)
	}
	return utils.JustSuccess()
}
