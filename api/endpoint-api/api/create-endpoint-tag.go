package endpoint_api

import (
	"context"
	"errors"

	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpointTag(ctx context.Context, eRequest *protos.CreateEndpointTagRequest) (*protos.GetEndpointResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[protos.GetEndpointResponse](
			errors.New("unauthenticated request for CreateEndpointProviderModel"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	_, err := endpointGRPCApi.endpointService.CreateOrUpdateEndpointTag(ctx, iAuth, eRequest.GetEndpointId(), eRequest.GetTags())
	if err != nil {
		return utils.Error[protos.GetEndpointResponse](
			err,
			"Unable to create endpoint tags for endpoint",
		)
	}
	// // calling to index the endpoint
	// endpointGRPCApi.endpointService.IndexEndpoint(ctx, iAuth, eRequest.GetEndpointId())
	ep, err := endpointGRPCApi.endpointService.Get(ctx, iAuth, eRequest.GetEndpointId(), nil, internal_services.NewDefaultGetEndpointOption())
	if err != nil {
		return utils.Error[protos.GetEndpointResponse](
			err,
			"Unable to get the endpoint for given endpoint id.",
		)
	}

	out := &protos.Endpoint{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint %v", err)
	}

	return utils.Success[protos.GetEndpointResponse](out)
}
