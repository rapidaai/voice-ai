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

func (endpoint *webEndpointGRPCApi) CreateEndpoint(c context.Context, iRequest *web_api.CreateEndpointRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpoint.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpoint.endpointClient.CreateEndpoint(c, iRequest, iRequest.GetEndpoint().GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iAuth.GetUserInfo().Id)
}

func (endpoint *webEndpointGRPCApi) CreateEndpointFromTestcase(c context.Context, iRequest *web_api.CreateEndpointFromTestcaseRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpoint.logger.Debugf("Create endpoint from test case grpc with requestPayload %v, %v", iRequest, c)

	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for creating endpoint")
		return nil, errors.New("unauthenticated request")
	}
	principle := iAuth.PlainAuthPrinciple()

	return endpoint.endpointClient.CreateEndpointFromTestcase(c, iRequest, &principle)
}

func (endpointGRPCApi *webEndpointGRPCApi) GetAllEndpointProviderModel(ctx context.Context, iRequest *web_api.GetAllEndpointProviderModelRequest) (*web_api.GetAllEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	return endpointGRPCApi.endpointClient.GetAllEndpointProviderModel(ctx, iRequest.GetEndpointId(), iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iRequest.GetCriterias(), iRequest.GetPaginate())
}

func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointVersion(ctx context.Context, iRequest *web_api.UpdateEndpointVersionRequest) (*web_api.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("Update endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointVersion(ctx, iRequest.GetEndpointId(), iRequest.GetEndpointProviderModelId(), iAuth.GetUserInfo().Id, iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)
}

func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *web_api.CreateEndpointRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint provider model")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointProviderModel(ctx, iRequest, iRequest.GetEndpoint().GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iAuth.GetUserInfo().Id)
}
