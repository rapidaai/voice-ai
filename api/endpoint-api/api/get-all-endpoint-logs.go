// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetAllEndpointLog(ctx context.Context, gaep *endpoint_grpc_api.GetAllEndpointLogRequest) (*endpoint_grpc_api.GetAllEndpointLogResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	cnt, epms, err := endpointGRPCApi.endpointLogService.GetAllEndpointLog(ctx,
		iAuth,
		gaep.GetEndpointId(),
		gaep.GetCriterias(),
		gaep.GetPaginate())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetAllEndpointLogResponse](
			err,
			"Unable to get all the endpoint provider model.",
		)
	}
	out := []*endpoint_grpc_api.EndpointLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}

	return utils.PaginatedSuccess[endpoint_grpc_api.GetAllEndpointLogResponse, []*endpoint_grpc_api.EndpointLog](
		uint32(cnt),
		gaep.GetPaginate().GetPage(),
		out)
}
