package endpoint_client

import (
	"context"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	endpoint_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type EndpointServiceClient interface {
	GetAllEndpoint(c context.Context, auth types.SimplePrinciple, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.Endpoint, error)
	GetEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.GetEndpointRequest) (*endpoint_api.Endpoint, error)
	CreateEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointResponse, error)
	GetAllEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointId uint64, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointProviderModel, error)
	UpdateEndpointVersion(c context.Context, auth types.SimplePrinciple, endpointId, endpointProviderModelId uint64) (*endpoint_api.UpdateEndpointVersionResponse, error)
	CreateEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointProviderModelRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error)
	CreateEndpointCacheConfiguration(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointCacheConfigurationRequest) (*endpoint_api.CreateEndpointCacheConfigurationResponse, error)
	CreateEndpointRetryConfiguration(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRetryConfigurationRequest) (*endpoint_api.CreateEndpointRetryConfigurationResponse, error)
	ForkEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.ForkEndpointRequest) (*endpoint_api.BaseResponse, error)
	CreateEndpointTag(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointTagRequest) (*endpoint_api.GetEndpointResponse, error)
	UpdateEndpointDetail(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.UpdateEndpointDetailRequest) (*endpoint_api.GetEndpointResponse, error)

	GetAllEndpointLog(c context.Context, auth types.SimplePrinciple, endpointId uint64, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointLog, error)
	GetEndpointLog(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.GetEndpointLogRequest) (*endpoint_api.GetEndpointLogResponse, error)
}

type endpointServiceClient struct {
	clients.InternalClient
	cfg            *config.AppConfig
	logger         commons.Logger
	endpointClient endpoint_api.EndpointServiceClient
}

func NewEndpointServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) EndpointServiceClient {
	conn, err := grpc.NewClient(config.EndpointHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Errorf("Unable to create connection %v", err)
	}
	return &endpointServiceClient{
		InternalClient: clients.NewInternalClient(config, logger, redis),
		cfg:            config,
		logger:         logger,
		endpointClient: endpoint_api.NewEndpointServiceClient(conn),
	}
}

func (client *endpointServiceClient) GetAllEndpoint(c context.Context, auth types.SimplePrinciple, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.Endpoint, error) {
	client.logger.Debugf("get all endpoint request")
	res, err := client.endpointClient.GetAllEndpoint(client.WithAuth(c, auth), &endpoint_api.GetAllEndpointRequest{
		Paginate:  paginate,
		Criterias: criterias,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, nil, err
	}

	return res.GetPaginated(), res.GetData(), nil
}

func (client *endpointServiceClient) GetEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.GetEndpointRequest) (*endpoint_api.Endpoint, error) {
	client.logger.Debugf("get endpoint request")
	res, err := client.endpointClient.GetEndpoint(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to get endpoint %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get endpoint %v", err)
		return nil, err
	}

	return res.GetData(), nil
}

func (client *endpointServiceClient) CreateEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointResponse, error) {
	res, err := client.endpointClient.CreateEndpoint(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) GetAllEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointId uint64, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointProviderModel, error) {
	res, err := client.endpointClient.GetAllEndpointProviderModel(client.WithAuth(c, auth), &endpoint_api.GetAllEndpointProviderModelRequest{
		Criterias:  criterias,
		Paginate:   paginate,
		EndpointId: endpointId,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint provider models %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all endpoint provider models %v", err)
		return nil, nil, err
	}

	return res.GetPaginated(), res.GetData(), nil
}

func (client *endpointServiceClient) UpdateEndpointVersion(c context.Context, auth types.SimplePrinciple, endpointId, endpointProviderModelId uint64) (*endpoint_api.UpdateEndpointVersionResponse, error) {
	res, err := client.endpointClient.UpdateEndpointVersion(client.WithAuth(c, auth), &endpoint_api.UpdateEndpointVersionRequest{
		EndpointId:              endpointId,
		EndpointProviderModelId: endpointProviderModelId,
	})
	if err != nil {
		client.logger.Errorf("error while calling to UpdateEndpointVersion %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) CreateEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointProviderModelRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error) {
	res, err := client.endpointClient.CreateEndpointProviderModel(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to CreateEndpointProviderModel %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) CreateEndpointCacheConfiguration(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointCacheConfigurationRequest) (*endpoint_api.CreateEndpointCacheConfigurationResponse, error) {
	res, err := client.endpointClient.CreateEndpointCacheConfiguration(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointCacheConfigurationt %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) CreateEndpointRetryConfiguration(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRetryConfigurationRequest) (*endpoint_api.CreateEndpointRetryConfigurationResponse, error) {
	res, err := client.endpointClient.CreateEndpointRetryConfiguration(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointRetryConfiguration %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) CreateEndpointTag(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointTagRequest) (*endpoint_api.GetEndpointResponse, error) {
	res, err := client.endpointClient.CreateEndpointTag(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointTag %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) UpdateEndpointDetail(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.UpdateEndpointDetailRequest) (*endpoint_api.GetEndpointResponse, error) {
	res, err := client.endpointClient.UpdateEndpointDetail(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointTag %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) ForkEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.ForkEndpointRequest) (*endpoint_api.BaseResponse, error) {
	res, err := client.endpointClient.ForkEndpoint(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to ForkEndpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) GetAllEndpointLog(c context.Context, auth types.SimplePrinciple, endpointId uint64, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointLog, error) {
	res, err := client.endpointClient.GetAllEndpointLog(client.WithAuth(c, auth), &endpoint_api.GetAllEndpointLogRequest{
		EndpointId: endpointId,
		Paginate:   paginate,
		Criterias:  criterias,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint log %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all endpoint log %v", err)
		return nil, nil, err
	}

	return res.GetPaginated(), res.GetData(), nil
}

func (client *endpointServiceClient) GetEndpointLog(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.GetEndpointLogRequest) (*endpoint_api.GetEndpointLogResponse, error) {
	res, err := client.endpointClient.GetEndpointLog(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to get endpoint %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get endpoint %v", err)
		return nil, err
	}
	return res, nil
}
