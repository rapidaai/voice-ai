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

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetEndpointLog(ctx context.Context, cepm *endpoint_grpc_api.GetEndpointLogRequest) (*endpoint_grpc_api.GetEndpointLogResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
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
