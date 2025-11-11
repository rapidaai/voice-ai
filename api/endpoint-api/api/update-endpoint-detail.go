package endpoint_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) UpdateEndpointDetail(ctx context.Context, eRequest *endpoint_grpc_api.UpdateEndpointDetailRequest) (*endpoint_grpc_api.GetEndpointResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.GetEndpointResponse](
			errors.New("unauthenticated request for UpdateEndpointDetail"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	_, err := endpointGRPCApi.endpointService.UpdateEndpointDetail(ctx,
		iAuth, eRequest.GetEndpointId(), eRequest.GetName(), &eRequest.Description)
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetEndpointResponse](
			err,
			"Unable to create endpoint tags for endpoint",
		)
	}
	// this is for indexing
	// endpointGRPCApi.endpointService.IndexEndpoint(ctx, iAuth, eRequest.GetEndpointId())
	ep, err := endpointGRPCApi.endpointService.Get(ctx, iAuth, eRequest.GetEndpointId(), nil, internal_services.NewDefaultGetEndpointOption())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetEndpointResponse](
			err,
			"Unable to get the endpoint for given endpoint id.",
		)
	}
	out := &endpoint_grpc_api.Endpoint{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint %v", err)
	}
	return utils.Success[endpoint_grpc_api.GetEndpointResponse](out)
}
