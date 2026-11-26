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

func (endpointGRPCApi *endpointGRPCApi) GetAllEndpoint(ctx context.Context, cepm *endpoint_grpc_api.GetAllEndpointRequest) (*endpoint_grpc_api.GetAllEndpointResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
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
	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[endpoint_grpc_api.GetAllEndpointResponse](
		totalItem,
		cepm.GetPaginate().GetPage(),
		out)
}
