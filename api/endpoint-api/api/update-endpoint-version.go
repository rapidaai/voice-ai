package endpoint_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) UpdateEndpointVersion(ctx context.Context, cer *endpoint_grpc_api.UpdateEndpointVersionRequest) (*endpoint_grpc_api.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("update endpoint version request %v, %v", cer, ctx)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for UpdateEndpointVersion")
		return utils.Error[endpoint_grpc_api.UpdateEndpointVersionResponse](
			errors.New("unauthenticated request for updateendpointversion"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	ep, err := endpointGRPCApi.endpointService.UpdateEndpointVersion(ctx,
		iAuth,
		cer.GetEndpointId(), cer.GetEndpointProviderModelId())
	if err != nil {
		return utils.Error[endpoint_grpc_api.UpdateEndpointVersionResponse](
			errors.New("unauthenticated request for updateendpointversion"),
			"Unable to update endpoint for given endpoint id.",
		)
	}
	out := &endpoint_grpc_api.Endpoint{}
	err = utils.Cast(ep, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast endpoint provider model %v", err)
	}

	return utils.Success[endpoint_grpc_api.UpdateEndpointVersionResponse, *endpoint_grpc_api.Endpoint](out)
}
