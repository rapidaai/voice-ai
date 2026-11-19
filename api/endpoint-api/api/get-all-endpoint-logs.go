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

func (endpointGRPCApi *endpointGRPCApi) GetAllEndpointLog(ctx context.Context, gaep *endpoint_grpc_api.GetAllEndpointLogRequest) (*endpoint_grpc_api.GetAllEndpointLogResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
	}
	cnt, epms, err := endpointGRPCApi.endpointLogService.GetAllEndpointLog(ctx,
		iAuth,
		gaep.GetEndpointId(),
		gaep.GetCriterias(),
		gaep.GetPaginate())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetAllEndpointLogResponse](
			err,
			"Unable to get all the endpoint provider model.",
		)
	}
	out := []*endpoint_grpc_api.EndpointLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}

	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		return nil, err
	}
	return utils.PaginatedSuccess[endpoint_grpc_api.GetAllEndpointLogResponse, []*endpoint_grpc_api.EndpointLog](
		totalItem,
		gaep.GetPaginate().GetPage(),
		out)
}
