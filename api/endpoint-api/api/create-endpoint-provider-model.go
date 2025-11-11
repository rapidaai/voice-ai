package endpoint_api

import (
	"context"
	"errors"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	endpoint_grpc_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *endpoint_grpc_api.CreateEndpointProviderModelRequest) (*endpoint_grpc_api.CreateEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("create Endpoint provider model request %v %v", iRequest, ctx)

	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[endpoint_grpc_api.CreateEndpointProviderModelResponse](
			errors.New("unauthenticated request for CreateEndpointProviderModel"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)

	}

	ep, err := endpointGRPCApi.endpointService.Get(ctx, iAuth, iRequest.GetEndpointId(), nil, internal_services.NewDefaultGetEndpointOption())
	if err != nil {
		return utils.Error[endpoint_grpc_api.CreateEndpointProviderModelResponse](
			err,
			"Unable to create endpoint version, please try again later",
		)
	}

	epm, err := endpointGRPCApi.createEndpointProviderModel(ctx, iAuth, ep, iRequest.GetEndpointProviderModelAttribute())
	if err != nil {
		return utils.Error[endpoint_grpc_api.CreateEndpointProviderModelResponse](
			err,
			"Unable to create endpoint version, please try again later",
		)
	}
	out := &endpoint_grpc_api.EndpointProviderModel{}
	err = utils.Cast(epm, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast the endpoint provider model to the response object")
	}
	return utils.Success[endpoint_grpc_api.CreateEndpointProviderModelResponse, *endpoint_grpc_api.EndpointProviderModel](out)
}

func (endpointGRPCApi *endpointGRPCApi) createEndpointProviderModel(ctx context.Context,
	iAuth types.SimplePrinciple,
	endpoint *internal_gorm.Endpoint,
	ea *endpoint_grpc_api.EndpointProviderModelAttribute,
) (*internal_gorm.EndpointProviderModel, error) {
	endpointGRPCApi.logger.Debugf("creating endpoint provider model with provider model attributes %+v", ea)
	prompt := protojson.Format(ea.GetChatCompletePrompt())
	epm, err := endpointGRPCApi.endpointService.CreateEndpointProviderModel(ctx,
		iAuth,
		endpoint.Id,
		ea.GetDescription(),
		ea.GetModelProviderId(),
		ea.GetModelProviderName(),
		prompt,
		ea.GetEndpointModelOptions(),
	)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to create endpoint provider model wuth error %+v", err)
		return nil, err
	}
	return epm, nil
}
