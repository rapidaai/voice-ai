package endpoint_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) GetAllEndpointProviderModel(ctx context.Context, gaep *endpoint_grpc_api.GetAllEndpointProviderModelRequest) (*endpoint_grpc_api.GetAllEndpointProviderModelResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.GetAllEndpointProviderModelResponse](
			errors.New("unauthenticated request for getallendpointprovidermodel"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	cnt, epms, err := endpointGRPCApi.endpointService.GetAllEndpointProviderModel(ctx,
		iAuth,
		gaep.GetEndpointId(),
		gaep.GetCriterias(),
		gaep.GetPaginate())
	if err != nil {
		return utils.Error[endpoint_grpc_api.GetAllEndpointProviderModelResponse](
			err,
			"Unable to get all the endpoint provider model.",
		)
	}
	out := []*endpoint_grpc_api.EndpointProviderModel{}
	err = utils.Cast(epms, &out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}

	return utils.PaginatedSuccess[endpoint_grpc_api.GetAllEndpointProviderModelResponse, []*endpoint_grpc_api.EndpointProviderModel](
		uint32(cnt),
		gaep.GetPaginate().GetPage(),
		out)
}
