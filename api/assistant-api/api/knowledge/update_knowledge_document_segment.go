// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

// UpdateKnowledgeDocumentSegment implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) UpdateKnowledgeDocumentSegment(ctx context.Context, dsr *knowledge_api.UpdateKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	_, err := knowledgeApi.knowledgeDocumentService.UpdateDocumentSegment(ctx,
		iAuth,
		dsr.GetIndex(),
		dsr.GetDocumentId(),
		dsr.GetDocumentName(),
		dsr.GetOrganizations(),
		dsr.GetDates(),
		dsr.GetProducts(),
		dsr.GetEvents(),
		dsr.GetPeople(),
		dsr.GetTimes(),
		dsr.GetQuantities(),
		dsr.GetLocations(),
		dsr.GetIndustries(),
	)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to delete knowledge segment with error %v", err)
		return utils.Error[knowledge_api.BaseResponse](
			err,
			"Unable to update document segment, please try again.",
		)
	}
	return utils.JustSuccess()
}
