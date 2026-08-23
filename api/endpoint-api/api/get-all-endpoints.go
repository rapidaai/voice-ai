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

func (endpointGRPCApi *endpointGRPCApi) GetAllEndpoint(ctx context.Context, cepm *endpoint_grpc_api.GetAllEndpointRequest) (*endpoint_grpc_api.GetAllEndpointResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	cnt, endpoints, err := endpointGRPCApi.endpointService.GetAll(ctx, iAuth,
		cepm.GetCriterias(),
		cepm.GetPaginate())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetAllEndpointResponse](
			err,
			"Unable to get all the endpoint.",
		)
	}
	out := []*endpoint_grpc_api.Endpoint{}
	err = utils.Cast(endpoints, &out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}

	for _, e := range out {
		analytics := endpointGRPCApi.endpointLogService.GetAggregatedEndpointAnalytics(ctx, iAuth, e.Id)
		e.EndpointAnalytics = analytics
	}
	return utils.PaginatedSuccess[endpoint_grpc_api.GetAllEndpointResponse](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
