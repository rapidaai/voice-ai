package web_api

import (
	"context"
	"errors"

	clients "github.com/lexatic/web-backend/pkg/clients"
	endpoint_client "github.com/lexatic/web-backend/pkg/clients/endpoint"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webEndpointApi struct {
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	endpointClient clients.EndpointServiceClient
}

type webEndpointGRPCApi struct {
	webEndpointApi
}

func NewEndpointGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.EndpointReaderServiceServer {
	return &webEndpointGRPCApi{
		webEndpointApi{
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			endpointClient: endpoint_client.NewEndpointServiceClientGRPC(config, logger),
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
	return endpoint.endpointClient.GetEndpoint(c, iRequest.GetId(), iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)
}

func (endpoint *webEndpointGRPCApi) GetAllEndpoint(c context.Context, iRequest *web_api.GetAllEndpointRequest) (*web_api.GetAllEndpointResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	return endpoint.endpointClient.GetAllEndpoint(c, iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iRequest.GetCriterias(), iRequest.GetPaginate())
}

func (endpoint *webEndpointGRPCApi) CreateEndpoint(c context.Context, iRequest *web_api.CreateEndpointRequest) (*web_api.EndpointProviderModelResponse, error) {
	endpoint.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpoint.endpointClient.CreateEndpoint(c, iRequest, iRequest.GetEndpoint().GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iAuth.GetUserInfo().Id)
}
