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

	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpointTag(ctx context.Context, eRequest *protos.CreateEndpointTagRequest) (*protos.GetEndpointResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	_, err := endpointGRPCApi.endpointService.CreateOrUpdateEndpointTag(ctx, iAuth, eRequest.GetEndpointId(), eRequest.GetTags())
	if err != nil {
		return utils.Error[protos.GetEndpointResponse](
			err,
			"Unable to create endpoint tags for endpoint",
		)
	}
	// // calling to index the endpoint
	// endpointGRPCApi.endpointService.IndexEndpoint(ctx, iAuth, eRequest.GetEndpointId())
	ep, err := endpointGRPCApi.endpointService.Get(ctx, iAuth, eRequest.GetEndpointId(), nil, internal_services.NewDefaultGetEndpointOption())
	if err != nil {
		return utils.Error[protos.GetEndpointResponse](
			err,
			"Unable to get the endpoint for given endpoint id.",
		)
	}

	out := &protos.Endpoint{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint %v", err)
	}

	return utils.Success[protos.GetEndpointResponse](out)
}
