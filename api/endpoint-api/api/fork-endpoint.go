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
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) ForkEndpoint(ctx context.Context, eRequest *endpoint_grpc_api.ForkEndpointRequest) (*endpoint_grpc_api.BaseResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	_ = iAuth
	return nil, status.Error(codes.Unimplemented, "ForkEndpoint is not implemented")
}
