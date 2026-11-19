// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	endpoint_grpc_api "github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (endpointGRPCApi *endpointGRPCApi) UpdateEndpointVersion(ctx context.Context, cer *endpoint_grpc_api.UpdateEndpointVersionRequest) (*endpoint_grpc_api.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("update endpoint version request %v, %v", cer, ctx)
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
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
