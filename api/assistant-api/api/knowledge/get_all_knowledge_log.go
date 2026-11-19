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

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeLog(ctx context.Context, gaar *knowledge_api.GetAllKnowledgeLogRequest) (*knowledge_api.GetAllKnowledgeLogResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
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

	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeLogResponse, []*knowledge_api.KnowledgeLog](
		totalItem,
		gaar.GetPaginate().GetPage(),
		out)
}
