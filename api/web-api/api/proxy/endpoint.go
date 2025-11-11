package web_proxy_api

import (
	"context"
	"errors"

	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type webEndpointApi struct {
	web_api.WebApi
	cfg            *config.WebAppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	redis          connectors.RedisConnector
	endpointClient endpoint_client.EndpointServiceClient
}

type webEndpointGRPCApi struct {
	webEndpointApi
}

func NewEndpointGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.EndpointServiceServer {
	return &webEndpointGRPCApi{
		webEndpointApi{
			WebApi:         web_api.NewWebApi(config, logger, postgres, redis),
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			endpointClient: endpoint_client.NewEndpointServiceClientGRPC(&config.AppConfig, logger, redis),
		},
	}
}

func (endpoint *webEndpointGRPCApi) GetEndpoint(c context.Context, iRequest *protos.GetEndpointRequest) (*protos.GetEndpointResponse, error) {
	endpoint.logger.Debugf("GetEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	_endpoint, err := endpoint.endpointClient.GetEndpoint(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[protos.GetEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	return utils.Success[protos.GetEndpointResponse, *protos.Endpoint](_endpoint)

}

/*
 */
func (endpoint *webEndpointGRPCApi) GetAllEndpoint(c context.Context, iRequest *protos.GetAllEndpointRequest) (*protos.GetAllEndpointResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _endpoint, err := endpoint.endpointClient.GetAllEndpoint(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	for _, _ep := range _endpoint {
		if _ep.GetEndpointProviderModel() != nil {
			_ep.EndpointProviderModel.CreatedUser = endpoint.GetUser(c, iAuth, _ep.EndpointProviderModel.GetCreatedBy())
		}
	}
	return utils.PaginatedSuccess[protos.GetAllEndpointResponse, []*protos.Endpoint](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoint)
}

func (endpoint *webEndpointGRPCApi) CreateEndpoint(c context.Context, iRequest *protos.CreateEndpointRequest) (*protos.CreateEndpointResponse, error) {
	endpoint.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpoint.endpointClient.CreateEndpoint(c, iAuth, iRequest)
}

func (endpointGRPCApi *webEndpointGRPCApi) GetAllEndpointProviderModel(ctx context.Context, iRequest *protos.GetAllEndpointProviderModelRequest) (*protos.GetAllEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _endpoints, err := endpointGRPCApi.endpointClient.GetAllEndpointProviderModel(ctx, iAuth, iRequest.GetEndpointId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllEndpointProviderModelResponse](
			err,
			"Unable to get your endpoint provider models, please try again in sometime.")
	}

	for _, _ep := range _endpoints {
		_ep.CreatedUser = endpointGRPCApi.GetUser(ctx, iAuth, _ep.GetCreatedBy())
	}
	return utils.PaginatedSuccess[protos.GetAllEndpointProviderModelResponse, []*protos.EndpointProviderModel](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoints)
}

func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointVersion(ctx context.Context, iRequest *protos.UpdateEndpointVersionRequest) (*protos.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("Update endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointVersion(ctx, iAuth, iRequest.GetEndpointId(), iRequest.GetEndpointProviderModelId())
}

func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *protos.CreateEndpointProviderModelRequest) (*protos.CreateEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint provider model")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointProviderModel(ctx, iAuth, iRequest)
}

// CreateEndpointCacheConfiguration implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointCacheConfiguration(ctx context.Context, iRequest *protos.CreateEndpointCacheConfigurationRequest) (*protos.CreateEndpointCacheConfigurationResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint caching configuration")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointCacheConfiguration(ctx, iAuth, iRequest)
}

// CreateEndpointRetryConfiguration implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointRetryConfiguration(ctx context.Context, iRequest *protos.CreateEndpointRetryConfigurationRequest) (*protos.CreateEndpointRetryConfigurationResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint retry configuration")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointRetryConfiguration(ctx, iAuth, iRequest)
}

// CreateEndpointTag implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointTag(ctx context.Context, iRequest *protos.CreateEndpointTagRequest) (*protos.GetEndpointResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint tag")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointTag(ctx, iAuth, iRequest)
}
func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointDetail(ctx context.Context, iRequest *protos.UpdateEndpointDetailRequest) (*protos.GetEndpointResponse, error) {
	endpointGRPCApi.logger.Debugf("update endpoint details request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint tag")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointDetail(ctx, iAuth, iRequest)
}

// ForkEndpoint implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) ForkEndpoint(ctx context.Context, iRequest *protos.ForkEndpointRequest) (*protos.BaseResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to fork endpoint")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.ForkEndpoint(ctx, iAuth, iRequest)
}

func (endpoint *webEndpointGRPCApi) GetEndpointLog(c context.Context, iRequest *protos.GetEndpointLogRequest) (*protos.GetEndpointLogResponse, error) {
	endpoint.logger.Debugf("GetEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpoint.endpointClient.GetEndpointLog(c, iAuth, iRequest)
}

/*
 */

func (endpoint *webEndpointGRPCApi) GetAllEndpointLog(c context.Context, iRequest *protos.GetAllEndpointLogRequest) (*protos.GetAllEndpointLogResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _deployments, err := endpoint.endpointClient.GetAllEndpointLog(c, iAuth, iRequest.GetEndpointId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllEndpointLogResponse](
			err,
			"Unable to get deployments, please try again in sometime.")
	}
	return utils.PaginatedSuccess[protos.GetAllEndpointLogResponse, []*protos.EndpointLog](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_deployments)
}
