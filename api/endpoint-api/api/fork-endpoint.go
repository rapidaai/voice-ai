// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) ForkEndpoint(ctx context.Context, eRequest *endpoint_grpc_api.ForkEndpointRequest) (*endpoint_grpc_api.BaseResponse, error) {
	_, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.BaseResponse](
			errors.New("unauthenticated request for CreateEndpointProviderModel"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)

	}
	return nil, nil
}
