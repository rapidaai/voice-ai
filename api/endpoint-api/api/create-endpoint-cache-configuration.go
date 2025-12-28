// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_api

import (
	"context"
	"errors"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpointCacheConfiguration(ctx context.Context, eRequest *endpoint_grpc_api.CreateEndpointCacheConfigurationRequest) (*endpoint_grpc_api.CreateEndpointCacheConfigurationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.CreateEndpointCacheConfigurationResponse](
			errors.New("unauthenticated request for CreateEndpointProviderModel"),
			"Please provide valid service credentials to perform invoke, read docs @ docs.rapida.ai",
		)
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
