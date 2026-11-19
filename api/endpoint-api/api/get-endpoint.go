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
	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetEndpoint(ctx context.Context, cepm *endpoint_grpc_api.GetEndpointRequest) (*endpoint_grpc_api.GetEndpointResponse, error) {
	start := time.Now()
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
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
