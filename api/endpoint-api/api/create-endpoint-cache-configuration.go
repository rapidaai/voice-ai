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

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpointCacheConfiguration(ctx context.Context, eRequest *endpoint_grpc_api.CreateEndpointCacheConfigurationRequest) (*endpoint_grpc_api.CreateEndpointCacheConfigurationResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
	}

	cec, err := endpointGRPCApi.endpointService.ConfigureEndpointCaching(ctx,
		iAuth,
		eRequest.GetEndpointId(),
		internal_gorm.Cache(eRequest.GetData().GetCacheType()),
		eRequest.GetData().GetExpiryInterval(),
		eRequest.GetData().GetMatchThreshold())
	if err != nil {
		return utils.Error[endpoint_grpc_api.CreateEndpointCacheConfigurationResponse](
			err,
			"Unable to configure endpoint caching, please try again later",
		)
	}

	out := &endpoint_grpc_api.EndpointCacheConfiguration{}
	err = utils.Cast(cec, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast the endpoint cache configuration to the response object")
	}
	return utils.Success[endpoint_grpc_api.CreateEndpointCacheConfigurationResponse, *endpoint_grpc_api.EndpointCacheConfiguration](out)
}
