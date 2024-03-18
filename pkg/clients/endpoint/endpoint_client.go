package integration

import (
	"context"

	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	endpoint_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type EndpointServiceClient interface {
	GetAllEndpoint(c context.Context, auth types.SimplePrinciple, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.GetAllEndpointResponse, error)
	GetEndpoint(c context.Context, auth types.SimplePrinciple, endpointId uint64) (*endpoint_api.GetEndpointResponse, error)
	CreateEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error)

	GetAllEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointId uint64, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.GetAllEndpointProviderModelResponse, error)
	UpdateEndpointVersion(c context.Context, auth types.SimplePrinciple, endpointId, endpointProviderModelId uint64) (*endpoint_api.UpdateEndpointVersionResponse, error)
	// CreateEndpointFromTestcase(c context.Context, iRequest *endpoint_api.CreateEndpointFromTestcaseRequest, principle *types.PlainAuthPrinciple) (*endpoint_api.CreateEndpointProviderModelResponse, error)
	CreateEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error)
}

type endpointServiceClient struct {
	clients.InternalClient
	cfg            *config.AppConfig
	logger         commons.Logger
	endpointClient endpoint_api.EndpointServiceClient
}

func NewEndpointServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) EndpointServiceClient {
	conn, err := grpc.Dial(config.EndpointHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (client *endpointServiceClient) GetAllEndpoint(c context.Context, auth types.SimplePrinciple, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.GetAllEndpointResponse, error) {
	client.logger.Debugf("get all endpoint request")
	res, err := client.endpointClient.GetAllEndpoint(client.WithAuth(c, auth), &endpoint_api.GetAllEndpointRequest{
		Paginate:  paginate,
		Criterias: criterias,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) GetEndpoint(c context.Context, auth types.SimplePrinciple, endpointId uint64) (*endpoint_api.GetEndpointResponse, error) {
	client.logger.Debugf("get endpoint request")
	res, err := client.endpointClient.GetEndpoint(client.WithAuth(c, auth), &endpoint_api.GetEndpointRequest{
		Id: endpointId,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) CreateEndpoint(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error) {
	res, err := client.endpointClient.CreateEndpoint(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) GetAllEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointId uint64, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.GetAllEndpointProviderModelResponse, error) {

	res, err := client.endpointClient.GetAllEndpointProviderModel(client.WithAuth(c, auth), &endpoint_api.GetAllEndpointProviderModelRequest{
		Criterias:  criterias,
		Paginate:   paginate,
		EndpointId: endpointId,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all provider models %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) UpdateEndpointVersion(c context.Context, auth types.SimplePrinciple, endpointId, endpointProviderModelId uint64) (*endpoint_api.UpdateEndpointVersionResponse, error) {
	res, err := client.endpointClient.UpdateEndpointVersion(client.WithAuth(c, auth), &endpoint_api.UpdateEndpointVersionRequest{
		EndpointId:              endpointId,
		EndpointProviderModelId: endpointProviderModelId,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) CreateEndpointProviderModel(c context.Context, auth types.SimplePrinciple, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error) {
	res, err := client.endpointClient.CreateEndpointProviderModel(client.WithAuth(c, auth), endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) CreateEndpointCacheConfiguration(c context.Context, endpointRequest *endpoint_api.CreateEndpointCacheConfigurationRequest) {

}
func (client *endpointServiceClient) CreateEndpointRetryConfiguration(c context.Context, endpointRequest *endpoint_api.CreateEndpointRetryConfigurationRequest) {
}
func (client *endpointServiceClient) CreateEndpointTag(c context.Context, endpointRequest *endpoint_api.CreateEndpointTagRequest) {
}
func (client *endpointServiceClient) ForkEndpoint(c context.Context, endpointRequest *endpoint_api.ForkEndpointRequest) {
}
