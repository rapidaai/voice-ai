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

// DeleteKnowledgeDocumentSegment implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) DeleteKnowledgeDocumentSegment(ctx context.Context, dsr *knowledge_api.DeleteKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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
