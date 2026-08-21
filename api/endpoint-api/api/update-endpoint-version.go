// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (endpointGRPCApi *endpointGRPCApi) UpdateEndpointVersion(ctx context.Context, cer *endpoint_grpc_api.UpdateEndpointVersionRequest) (*endpoint_grpc_api.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("update endpoint version request %v, %v", cer, ctx)
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	ep, err := endpointGRPCApi.endpointService.UpdateEndpointVersion(ctx,
		iAuth,
		cer.GetEndpointId(), cer.GetEndpointProviderModelId())
	if err != nil {
		return utils.Error[endpoint_grpc_api.UpdateEndpointVersionResponse](
			errors.New("unauthenticated request for updateendpointversion"),
			"Unable to update endpoint for given endpoint id.",
		)
	}
	out := &endpoint_grpc_api.Endpoint{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}

	return utils.Success[endpoint_grpc_api.UpdateEndpointVersionResponse, *endpoint_grpc_api.Endpoint](out)
}
