package endpoint_api

import (
	"context"
	"errors"
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetEndpointLog(ctx context.Context, cepm *endpoint_grpc_api.GetEndpointLogRequest) (*endpoint_grpc_api.GetEndpointLogResponse, error) {
	start := time.Now()
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.GetEndpointLogResponse](
			errors.New("unauthenticated request for get endpoint"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	ep, err := endpointGRPCApi.endpointLogService.GetEndpointLog(ctx,
		iAuth,
		cepm.GetId(),
		cepm.GetEndpointId())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetEndpointLogResponse](
			err,
			"Unable to get the endpoint log for given id.",
		)
	}

	endpointGRPCApi.logger.Benchmark("endpointGRPCApi.GetEndpoint", time.Since(start))
	out := &endpoint_grpc_api.EndpointLog{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}
	endpointGRPCApi.logger.Benchmark("endpointGRPCApi.GetEndpoint.EndpointAnalytics", time.Since(start))
	return utils.Success[endpoint_grpc_api.GetEndpointLogResponse](out)
}
