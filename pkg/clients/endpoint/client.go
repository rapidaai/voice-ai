package integration

import (
	"context"

	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	"github.com/lexatic/web-backend/pkg/commons"
	endpoint_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type endpointServiceClient struct {
	cfg            *config.AppConfig
	logger         commons.Logger
	endpointClient endpoint_api.EndpointReaderServiceClient
}

func NewEndpointServiceClientGRPC(config *config.AppConfig, logger commons.Logger) clients.EndpointServiceClient {
	logger.Debugf("conntecting to endpoint client with %s", config.EndpointHost)
	conn, err := grpc.Dial(config.EndpointHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return &endpointServiceClient{
		cfg:            config,
		logger:         logger,
		endpointClient: endpoint_api.NewEndpointReaderServiceClient(conn),
	}
}

func (client *endpointServiceClient) GetAllEndpoint(c context.Context, projectId, organizationId uint64, criterias []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.GetAllEndpointResponse, error) {
	res, err := client.endpointClient.GetAllEndpoint(c, &endpoint_api.GetAllEndpointRequest{
		ProjectId:      projectId,
		OrganizationId: organizationId,
		Paginate:       paginate,
		Criterias:      criterias,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	client.logger.Debugf("got response for get all endpoint %+v", res)
	return res, nil
}

func (client *endpointServiceClient) GetEndpoint(c context.Context, endpointId uint64, projectId, organizationId uint64) (*endpoint_api.GetEndpointResponse, error) {
	res, err := client.endpointClient.GetEndpoint(c, &endpoint_api.GetEndpointRequest{
		// should be endpoint id
		Id:             endpointId,
		ProjectId:      projectId,
		OrganizationId: organizationId,
	})
	if err != nil {
		client.logger.Debugf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	client.logger.Debugf("got response for get endpoint %+v", res)
	return res, nil
}

func (client *endpointServiceClient) CreateEndpoint(c context.Context, endpointRequest *endpoint_api.CreateEndpointRequest, projectId, organizationId, userId uint64) (*endpoint_api.EndpointProviderModelResponse, error) {
	endpointRequest.GetEndpoint().OrganizationId = organizationId
	endpointRequest.GetEndpoint().ProjectId = projectId
	endpointRequest.CreatedBy = userId
	res, err := client.endpointClient.CreateEndpoint(c, endpointRequest)
	if err != nil {
		client.logger.Debugf("error while calling to get all endpoint %v", err)
		return nil, err
	}
	client.logger.Debugf("got response for get endpoint %+v", res)
	return res, nil
}
