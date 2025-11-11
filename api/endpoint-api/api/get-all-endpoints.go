package endpoint_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetAllEndpoint(ctx context.Context, cepm *endpoint_grpc_api.GetAllEndpointRequest) (*endpoint_grpc_api.GetAllEndpointResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		endpointGRPCApi.logger.Errorf("unauthenticated request for GetAllEndpoint")
		return utils.Error[endpoint_grpc_api.GetAllEndpointResponse](
			errors.New("unauthenticated request for get allendpoint"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
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
	return utils.PaginatedSuccess[endpoint_grpc_api.GetAllEndpointResponse](
		uint32(cnt),
		cepm.GetPaginate().GetPage(),
		out)
}
