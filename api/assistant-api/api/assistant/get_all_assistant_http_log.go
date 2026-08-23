// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) GetAllAssistantHTTPLog(ctx context.Context, req *protos.GetAllAssistantHTTPLogRequest) (*protos.GetAllAssistantHTTPLogResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	paginate := req.GetPaginate()
	if paginate == nil {
		paginate = &protos.Paginate{
			Page:     1,
			PageSize: 50,
		}
	}

	cnt, logs, err := assistantApi.assistantHTTPLogService.GetAllLog(
		ctx,
		iAuth,
		req.GetProjectId(),
		req.GetCriterias(),
		paginate,
		req.GetOrder(),
	)
	if err != nil {
		assistantApi.logger.Errorf("failed to get assistant HTTP logs: %v", err)
		return exceptions.BadRequestError[protos.GetAllAssistantHTTPLogResponse]("Unable to get assistant HTTP logs.")
	}

	out := []*protos.AssistantHTTPLog{}
	if err := utils.Cast(logs, &out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant http logs %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantHTTPLogResponse, []*protos.AssistantHTTPLog](
		uint32(cnt),
		paginate.GetPage(),
		out,
	)
}
