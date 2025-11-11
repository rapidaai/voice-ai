package endpoint_api

import (
	"context"
	"errors"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

func (endpointGRPCApi *endpointGRPCApi) CreateEndpoint(ctx context.Context, cer *protos.CreateEndpointRequest) (*protos.CreateEndpointResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint request %v, %v", cer, ctx)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[protos.CreateEndpointResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	endpoint, err := endpointGRPCApi.endpointService.CreateEndpoint(
		ctx,
		iAuth,
		cer.GetEndpointAttribute().GetName(),
		&cer.GetEndpointAttribute().Description,
		&cer.GetEndpointAttribute().Visibility,
		&cer.GetEndpointAttribute().Source,
		&cer.GetEndpointAttribute().SourceIdentifier)
	if err != nil {
		return utils.Error[protos.CreateEndpointResponse](
			err,
			"Unable to create endpoint, please try again later",
		)
	}
	epModel, err := endpointGRPCApi.createEndpointProviderModel(ctx, iAuth, endpoint, cer.GetEndpointProviderModelAttribute())
	if err != nil {
		return utils.Error[protos.CreateEndpointResponse](
			err,
			"Unable to create endpoint provider model, please try again later",
		)
	}

	_, err = endpointGRPCApi.endpointService.AttachProviderModelToEndpoint(ctx, iAuth, epModel.Id, endpoint.Id)
	if err != nil {
		return utils.Error[protos.CreateEndpointResponse](
			err,
			"Unable to attach endpoint provider model, please try again later",
		)
	}

	if cer.GetRetryConfiguration() != nil {
		_, err = endpointGRPCApi.endpointService.ConfigureEndpointRetry(ctx,
			iAuth,
			endpoint.Id,
			internal_gorm.Retry(cer.GetRetryConfiguration().GetRetryType()),
			cer.GetRetryConfiguration().GetMaxAttempts(),
			cer.GetRetryConfiguration().GetDelaySeconds(),
			cer.GetRetryConfiguration().GetExponentialBackoff(),
			cer.GetRetryConfiguration().GetRetryables(),
		)
		if err != nil {
			return utils.Error[protos.CreateEndpointResponse](
				errors.New("unauthenticated request for CreateEndpointProviderModel"),
				"Unable to configure endpoint retry, please try again later",
			)

		}
	}
	if cer.GetCacheConfiguration() != nil {
		_, err = endpointGRPCApi.endpointService.ConfigureEndpointCaching(ctx,
			iAuth,
			endpoint.Id,
			internal_gorm.Cache(cer.GetCacheConfiguration().GetCacheType()),
			cer.GetCacheConfiguration().GetExpiryInterval(),
			cer.GetCacheConfiguration().GetMatchThreshold())
		if err != nil {
			return utils.Error[protos.CreateEndpointResponse](
				errors.New("unauthenticated request for CreateEndpointProviderModel"),
				"Unable to configure endpoint retry, please try again later",
			)

		}
	}
	_, err = endpointGRPCApi.endpointService.CreateOrUpdateEndpointTag(ctx, iAuth, endpoint.Id, cer.GetTags())
	if err != nil {
		return utils.Error[protos.CreateEndpointResponse](
			err,
			"Unable to create endpoint tags, please try again.",
		)
	}

	endpoint.EndpointProviderModel = epModel
	out := &protos.Endpoint{}
	err = utils.Cast(endpoint, out)
	if err != nil {
		endpointGRPCApi.logger.Errorf("unable to cast the endpoint provider model to the response object")
	}

	// calling to index the endpoint
	// endpointGRPCApi.endpointService.IndexEndpoint(ctx, iAuth, endpoint.Id)
	return utils.Success[protos.CreateEndpointResponse, *protos.Endpoint](out)
}
