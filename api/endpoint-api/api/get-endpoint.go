// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_api

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetEndpoint(ctx context.Context, cepm *endpoint_grpc_api.GetEndpointRequest) (*endpoint_grpc_api.GetEndpointResponse, error) {
	start := time.Now()
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	ep, err := endpointGRPCApi.endpointService.Get(ctx, iAuth, cepm.GetId(), cepm.EndpointProviderModelId, internal_services.NewDefaultGetEndpointOption())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetEndpointResponse](
			err,
			"Unable to get the endpoint for given endpoint id.",
		)
	}

	endpointGRPCApi.logger.Benchmark("endpointGRPCApi.GetEndpoint", time.Since(start))
	out := &endpoint_grpc_api.Endpoint{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}
	endpointGRPCApi.logger.Benchmark("endpointGRPCApi.GetEndpoint.EndpointAnalytics", time.Since(start))
	return utils.Success[endpoint_grpc_api.GetEndpointResponse](out)
}
