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

func (endpointGRPCApi *endpointGRPCApi) GetEndpointLog(ctx context.Context, cepm *endpoint_grpc_api.GetEndpointLogRequest) (*endpoint_grpc_api.GetEndpointLogResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	ep, err := endpointGRPCApi.endpointLogService.GetEndpointLog(ctx,
		iAuth,
		cepm.GetId(),
		cepm.GetEndpointId())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetEndpointLogResponse](
			err,
			"Unable to get the endpoint log for given id.",
		)
	}

	out := &endpoint_grpc_api.EndpointLog{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}
	return utils.Success[endpoint_grpc_api.GetEndpointLogResponse](out)
}
