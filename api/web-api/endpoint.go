package web_api

import (
	"context"
	"errors"
	"strconv"

	endpoint_client "github.com/lexatic/web-backend/pkg/clients/endpoint"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webEndpointApi struct {
	WebApi
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	redis          connectors.RedisConnector
	endpointClient endpoint_client.EndpointServiceClient
}

type webEndpointGRPCApi struct {
	webEndpointApi
}

func NewEndpointGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.EndpointServiceServer {
	return &webEndpointGRPCApi{
		webEndpointApi{
			WebApi:         NewWebApi(config, logger, postgres, redis),
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			endpointClient: endpoint_client.NewEndpointServiceClientGRPC(config, logger, redis),
		},
	}
}

func (endpoint *webEndpointGRPCApi) GetEndpoint(c context.Context, iRequest *web_api.GetEndpointRequest) (*web_api.GetEndpointResponse, error) {
	endpoint.logger.Debugf("GetEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	_endpoint, err := endpoint.endpointClient.GetEndpoint(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[web_api.GetEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	if _endpoint.EndpointProviderModel != nil {
		_endpoint.EndpointProviderModel.CreatedUser = endpoint.GetUser(c, iAuth, _endpoint.EndpointProviderModel.GetCreatedBy())
		_endpoint.EndpointProviderModel.ProviderModel = endpoint.GetProviderModel(c, iAuth, _endpoint.EndpointProviderModel.GetProviderModelId())
	}

	return utils.Success[web_api.GetEndpointResponse, *web_api.Endpoint](_endpoint)

}

/*
 */

func (endpoint *webEndpointGRPCApi) GetAllDeployment(c context.Context, iRequest *web_api.GetAllDeploymentRequest) (*web_api.GetAllDeploymentResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _deployments, err := endpoint.endpointClient.GetAllDeployment(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllDeploymentResponse](
			err,
			"Unable to get deployments, please try again in sometime.")
	}

	for _, _ep := range _deployments {
		orgId, _ := strconv.ParseUint(_ep.GetOrganizationId(), 10, 64)
		_ep.Organization = endpoint.GetOrganization(c, iAuth, orgId)
	}
	return utils.PaginatedSuccess[web_api.GetAllDeploymentResponse, []*web_api.SearchableDeployment](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_deployments)
}

/*
 */
func (endpoint *webEndpointGRPCApi) GetAllEndpoint(c context.Context, iRequest *web_api.GetAllEndpointRequest) (*web_api.GetAllEndpointResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _endpoint, err := endpoint.endpointClient.GetAllEndpoint(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	for _, _ep := range _endpoint {
		if _ep.GetEndpointProviderModel() != nil {
			_ep.EndpointProviderModel.CreatedUser = endpoint.GetUser(c, iAuth, _ep.EndpointProviderModel.GetCreatedBy())
			_ep.EndpointProviderModel.ProviderModel = endpoint.GetProviderModel(c, iAuth, _ep.EndpointProviderModel.GetProviderModelId())
		}
	}
	return utils.PaginatedSuccess[web_api.GetAllEndpointResponse, []*web_api.Endpoint](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoint)
}

func (endpoint *webEndpointGRPCApi) CreateEndpoint(c context.Context, iRequest *web_api.CreateEndpointRequest) (*web_api.CreateEndpointResponse, error) {
	endpoint.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpoint.endpointClient.CreateEndpoint(c, iAuth, iRequest)
}

func (endpointGRPCApi *webEndpointGRPCApi) GetAllEndpointProviderModel(ctx context.Context, iRequest *web_api.GetAllEndpointProviderModelRequest) (*web_api.GetAllEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _endpoints, err := endpointGRPCApi.endpointClient.GetAllEndpointProviderModel(ctx, iAuth, iRequest.GetEndpointId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllEndpointProviderModelResponse](
			err,
			"Unable to get your endpoint provider models, please try again in sometime.")
	}

	for _, _ep := range _endpoints {
		_ep.CreatedUser = endpointGRPCApi.GetUser(ctx, iAuth, _ep.GetCreatedBy())
		_ep.ProviderModel = endpointGRPCApi.GetProviderModel(ctx, iAuth, _ep.GetProviderModelId())
	}
	return utils.PaginatedSuccess[web_api.GetAllEndpointProviderModelResponse, []*web_api.EndpointProviderModel](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoints)
}

func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointVersion(ctx context.Context, iRequest *web_api.UpdateEndpointVersionRequest) (*web_api.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("Update endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointVersion(ctx, iAuth, iRequest.GetEndpointId(), iRequest.GetEndpointProviderModelId())
}

func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *web_api.CreateEndpointProviderModelRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint provider model")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointProviderModel(ctx, iAuth, iRequest)
}

// CreateEndpointCacheConfiguration implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointCacheConfiguration(ctx context.Context, iRequest *web_api.CreateEndpointCacheConfigurationRequest) (*web_api.CreateEndpointCacheConfigurationResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint caching configuration")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointCacheConfiguration(ctx, iAuth, iRequest)
}

// CreateEndpointRetryConfiguration implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointRetryConfiguration(ctx context.Context, iRequest *web_api.CreateEndpointRetryConfigurationRequest) (*web_api.CreateEndpointRetryConfigurationResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint retry configuration")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointRetryConfiguration(ctx, iAuth, iRequest)
}

// CreateEndpointTag implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointTag(ctx context.Context, iRequest *web_api.CreateEndpointTagRequest) (*web_api.GetEndpointResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint tag")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointTag(ctx, iAuth, iRequest)
}
func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointDetail(ctx context.Context, iRequest *web_api.UpdateEndpointDetailRequest) (*web_api.GetEndpointResponse, error) {
	endpointGRPCApi.logger.Debugf("update endpoint details request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint tag")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointDetail(ctx, iAuth, iRequest)
}

// ForkEndpoint implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) ForkEndpoint(ctx context.Context, iRequest *web_api.ForkEndpointRequest) (*web_api.BaseResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to fork endpoint")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.ForkEndpoint(ctx, iAuth, iRequest)
}

func (endpointGRPCApi *webEndpointGRPCApi) Invoke(ctx context.Context, iRequest *web_api.ForkEndpointRequest) (*web_api.BaseResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to fork endpoint")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.ForkEndpoint(ctx, iAuth, iRequest)
}
