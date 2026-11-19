package web_proxy_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"

	web_api "github.com/rapidaai/api/web-api/api"
	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
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
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	_endpoint, err := endpoint.endpointClient.GetEndpoint(c, principal, iRequest)
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
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _endpoint, err := endpoint.endpointClient.GetAllEndpoint(c, principal, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	return utils.PaginatedSuccess[protos.GetAllEndpointResponse, []*protos.Endpoint](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoint)
}

func (endpoint *webEndpointGRPCApi) CreateEndpoint(c context.Context, iRequest *protos.CreateEndpointRequest) (*protos.CreateEndpointResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpoint.endpointClient.CreateEndpoint(c, principal, iRequest)
}

func (endpointGRPCApi *webEndpointGRPCApi) GetAllEndpointProviderModel(ctx context.Context, iRequest *protos.GetAllEndpointProviderModelRequest) (*protos.GetAllEndpointProviderModelResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	_page, _endpoints, err := endpointGRPCApi.endpointClient.GetAllEndpointProviderModel(ctx, principal, iRequest.GetEndpointId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllEndpointProviderModelResponse](
			err,
			"Unable to get your endpoint provider models, please try again in sometime.")
	}

	return utils.PaginatedSuccess[protos.GetAllEndpointProviderModelResponse, []*protos.EndpointProviderModel](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoints)
}

func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointVersion(ctx context.Context, iRequest *protos.UpdateEndpointVersionRequest) (*protos.UpdateEndpointVersionResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointVersion(ctx, principal, iRequest.GetEndpointId(), iRequest.GetEndpointProviderModelId())
}

func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *protos.CreateEndpointProviderModelRequest) (*protos.CreateEndpointProviderModelResponse, error) {

	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointProviderModel(ctx, principal, iRequest)
}

// CreateEndpointCacheConfiguration implements protos.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointCacheConfiguration(ctx context.Context, iRequest *protos.CreateEndpointCacheConfigurationRequest) (*protos.CreateEndpointCacheConfigurationResponse, error) {

	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointCacheConfiguration(ctx, principal, iRequest)
}

// CreateEndpointRetryConfiguration implements protos.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointRetryConfiguration(ctx context.Context, iRequest *protos.CreateEndpointRetryConfigurationRequest) (*protos.CreateEndpointRetryConfigurationResponse, error) {

	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointRetryConfiguration(ctx, principal, iRequest)
}

// CreateEndpointTag implements protos.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointTag(ctx context.Context, iRequest *protos.CreateEndpointTagRequest) (*protos.GetEndpointResponse, error) {

	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointTag(ctx, principal, iRequest)
}
func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointDetail(ctx context.Context, iRequest *protos.UpdateEndpointDetailRequest) (*protos.GetEndpointResponse, error) {

	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointDetail(ctx, principal, iRequest)
}

// ForkEndpoint implements protos.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) ForkEndpoint(ctx context.Context, iRequest *protos.ForkEndpointRequest) (*protos.BaseResponse, error) {
	//
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpointGRPCApi.endpointClient.ForkEndpoint(ctx, principal, iRequest)
}

func (endpoint *webEndpointGRPCApi) GetEndpointLog(c context.Context, iRequest *protos.GetEndpointLogRequest) (*protos.GetEndpointLogResponse, error) {
	// endpoint.logger.Debugf("GetEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return endpoint.endpointClient.GetEndpointLog(c, principal, iRequest)
}

/*
 */

func (endpoint *webEndpointGRPCApi) GetAllEndpointLog(c context.Context, iRequest *protos.GetAllEndpointLogRequest) (*protos.GetAllEndpointLogResponse, error) {
	// endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	_page, _deployments, err := endpoint.endpointClient.GetAllEndpointLog(c, principal, iRequest.GetEndpointId(), iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllEndpointLogResponse](
			err,
			"Unable to get deployments, please try again in sometime.")
	}
	return utils.PaginatedSuccess[protos.GetAllEndpointLogResponse, []*protos.EndpointLog](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_deployments)
}
