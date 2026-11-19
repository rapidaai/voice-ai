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
	"google.golang.org/protobuf/encoding/protojson"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *endpoint_grpc_api.CreateEndpointProviderModelRequest) (*endpoint_grpc_api.CreateEndpointProviderModelResponse, error) {
	iAuth, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
	}
	if !validator.OneOf(iAuth.Subject().Type, authentication.IdentityUser, authentication.IdentityProjectCredential, authentication.IdentityService) {
		return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
	}

	ep, err := endpointGRPCApi.endpointService.Get(ctx, iAuth, iRequest.GetEndpointId(), nil, internal_services.NewDefaultGetEndpointOption())
	if err != nil {
		return utils.Error[endpoint_grpc_api.CreateEndpointProviderModelResponse](
			err,
			"Unable to create endpoint version, please try again later",
		)
	}

	epm, err := endpointGRPCApi.createEndpointProviderModel(ctx, iAuth, ep, iRequest.GetEndpointProviderModelAttribute())
	if err != nil {
		return utils.Error[endpoint_grpc_api.CreateEndpointProviderModelResponse](
			err,
			"Unable to create endpoint version, please try again later",
		)
	}
	out := &endpoint_grpc_api.EndpointProviderModel{}
	err = utils.Cast(epm, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast the endpoint provider model to the response object")
	}
	return utils.Success[endpoint_grpc_api.CreateEndpointProviderModelResponse, *endpoint_grpc_api.EndpointProviderModel](out)
}

func (endpointGRPCApi *endpointGRPCApi) createEndpointProviderModel(ctx context.Context,
	iAuth authentication.Principal,
	endpoint *internal_gorm.Endpoint,
	ea *endpoint_grpc_api.EndpointProviderModelAttribute,
) (*internal_gorm.EndpointProviderModel, error) {
	endpointGRPCApi.logger.Debugf("creating endpoint provider model with provider model attributes %+v", ea)
	prompt := protojson.Format(ea.GetChatCompletePrompt())
	epm, err := endpointGRPCApi.endpointService.CreateEndpointProviderModel(ctx,
		iAuth,
		endpoint.Id,
		ea.GetDescription(),
		ea.GetModelProviderName(),
		prompt,
		ea.GetEndpointModelOptions(),
	)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to create endpoint provider model wuth error %+v", err)
		return nil, err
	}
	return epm, nil
}
