package endpoint_api

import (
	"context"
	"errors"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpointRetryConfiguration(ctx context.Context, eRequest *endpoint_grpc_api.CreateEndpointRetryConfigurationRequest) (*endpoint_grpc_api.CreateEndpointRetryConfigurationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.CreateEndpointRetryConfigurationResponse](
			errors.New("unauthenticated request for CreateEndpointProviderModel"),
			"Please provide valid service credentials to perform invoke, read docs @ docs.rapida.ai",
		)
	}

	erc, err := endpointGRPCApi.endpointService.ConfigureEndpointRetry(ctx,
		iAuth,
		eRequest.GetEndpointId(),
		internal_gorm.Retry(eRequest.GetData().GetRetryType()),
		eRequest.GetData().GetMaxAttempts(),
		eRequest.GetData().GetDelaySeconds(),
		eRequest.GetData().GetExponentialBackoff(),
		eRequest.GetData().GetRetryables(),
	)
	if err != nil {
		return utils.Error[endpoint_grpc_api.CreateEndpointRetryConfigurationResponse](
			err,
			"Unable to configure endpoint retry, please try again later",
		)
	}

	out := &endpoint_grpc_api.EndpointRetryConfiguration{}
	err = utils.Cast(erc, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast the endpoint retry configuration to the response object")
	}

	return utils.Success[endpoint_grpc_api.CreateEndpointRetryConfigurationResponse, *endpoint_grpc_api.EndpointRetryConfiguration](out)
}
