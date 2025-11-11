package endpoint_api

import (
	"context"
	"errors"
	"time"

	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetEndpoint(ctx context.Context, cepm *endpoint_grpc_api.GetEndpointRequest) (*endpoint_grpc_api.GetEndpointResponse, error) {
	start := time.Now()
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.GetEndpointResponse](
			errors.New("unauthenticated request for get endpoint"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
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
