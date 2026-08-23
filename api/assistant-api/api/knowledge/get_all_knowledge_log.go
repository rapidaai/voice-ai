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

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeLog(ctx context.Context, gaar *knowledge_api.GetAllKnowledgeLogRequest) (*knowledge_api.GetAllKnowledgeLogResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	cnt, epms, err := knowledgeApi.knowledgeService.GetAllLog(ctx,
		iAuth,
		gaar.GetProjectId(),
		gaar.GetCriterias(),
		gaar.GetPaginate(),
		gaar.GetOrder())
	if err != nil {
		return exceptions.BadRequestError[knowledge_api.GetAllKnowledgeLogResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*knowledge_api.KnowledgeLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast assistant webhook logs %v", err)
	}

	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeLogResponse, []*knowledge_api.KnowledgeLog](
		uint32(cnt),
		gaar.GetPaginate().GetPage(),
		out)
}
