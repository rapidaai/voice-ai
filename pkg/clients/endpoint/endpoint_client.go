// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package endpoint_client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	endpoint_api "github.com/rapidaai/protos"
)

type EndpointServiceClient interface {
	GetAllEndpoint(c context.Context, auth *types.Authentication, criteria []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.Endpoint, error)
	GetEndpoint(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.GetEndpointRequest) (*endpoint_api.Endpoint, error)
	CreateEndpoint(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointResponse, error)
	GetAllEndpointProviderModel(c context.Context, auth *types.Authentication, endpointId uint64, criteria []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointProviderModel, error)
	UpdateEndpointVersion(c context.Context, auth *types.Authentication, endpointId, endpointProviderModelId uint64) (*endpoint_api.UpdateEndpointVersionResponse, error)
	CreateEndpointProviderModel(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointProviderModelRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error)
	CreateEndpointCacheConfiguration(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointCacheConfigurationRequest) (*endpoint_api.CreateEndpointCacheConfigurationResponse, error)
	CreateEndpointRetryConfiguration(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointRetryConfigurationRequest) (*endpoint_api.CreateEndpointRetryConfigurationResponse, error)
	ForkEndpoint(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.ForkEndpointRequest) (*endpoint_api.BaseResponse, error)
	CreateEndpointTag(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointTagRequest) (*endpoint_api.GetEndpointResponse, error)
	UpdateEndpointDetail(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.UpdateEndpointDetailRequest) (*endpoint_api.GetEndpointResponse, error)

	GetAllEndpointLog(c context.Context, auth *types.Authentication, endpointId uint64, criteria []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointLog, error)
	GetEndpointLog(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.GetEndpointLogRequest) (*endpoint_api.GetEndpointLogResponse, error)
}

type endpointServiceClient struct {
	clients.InternalClient
	cfg            *config.AppConfig
	logger         commons.Logger
	endpointClient endpoint_api.EndpointServiceClient
}

func NewEndpointServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) EndpointServiceClient {
	conn, err := grpc.NewClient(config.Endpoint.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (client *endpointServiceClient) GetAllEndpoint(c context.Context, auth *types.Authentication, criteria []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.Endpoint, error) {
	client.logger.Debugf("get all endpoint request")
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, nil, err
	}
	res, err := client.endpointClient.GetAllEndpoint(authContext, &endpoint_api.GetAllEndpointRequest{
		Paginate:  paginate,
		Criterias: criteria,
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

func (client *endpointServiceClient) GetEndpoint(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.GetEndpointRequest) (*endpoint_api.Endpoint, error) {
	client.logger.Debugf("get endpoint request")
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.GetEndpoint(authContext, endpointRequest)
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

func (client *endpointServiceClient) CreateEndpoint(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointRequest) (*endpoint_api.CreateEndpointResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.CreateEndpoint(authContext, endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) GetAllEndpointProviderModel(c context.Context, auth *types.Authentication, endpointId uint64, criteria []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointProviderModel, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, nil, err
	}
	res, err := client.endpointClient.GetAllEndpointProviderModel(authContext, &endpoint_api.GetAllEndpointProviderModelRequest{
		Criterias:  criteria,
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

func (client *endpointServiceClient) UpdateEndpointVersion(c context.Context, auth *types.Authentication, endpointId, endpointProviderModelId uint64) (*endpoint_api.UpdateEndpointVersionResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.UpdateEndpointVersion(authContext, &endpoint_api.UpdateEndpointVersionRequest{
		EndpointId:              endpointId,
		EndpointProviderModelId: endpointProviderModelId,
	})
	if err != nil {
		client.logger.Errorf("error while calling to UpdateEndpointVersion %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) CreateEndpointProviderModel(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointProviderModelRequest) (*endpoint_api.CreateEndpointProviderModelResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.CreateEndpointProviderModel(authContext, endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to CreateEndpointProviderModel %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) CreateEndpointCacheConfiguration(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointCacheConfigurationRequest) (*endpoint_api.CreateEndpointCacheConfigurationResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.CreateEndpointCacheConfiguration(authContext, endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointCacheConfigurationt %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) CreateEndpointRetryConfiguration(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointRetryConfigurationRequest) (*endpoint_api.CreateEndpointRetryConfigurationResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.CreateEndpointRetryConfiguration(authContext, endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointRetryConfiguration %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) CreateEndpointTag(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.CreateEndpointTagRequest) (*endpoint_api.GetEndpointResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.CreateEndpointTag(authContext, endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointTag %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) UpdateEndpointDetail(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.UpdateEndpointDetailRequest) (*endpoint_api.GetEndpointResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.UpdateEndpointDetail(authContext, endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateEndpointTag %v", err)
		return nil, err
	}
	return res, nil
}
func (client *endpointServiceClient) ForkEndpoint(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.ForkEndpointRequest) (*endpoint_api.BaseResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.ForkEndpoint(authContext, endpointRequest)
	if err != nil {
		client.logger.Errorf("error while calling to ForkEndpoint %v", err)
		return nil, err
	}
	return res, nil
}

func (client *endpointServiceClient) GetAllEndpointLog(c context.Context, auth *types.Authentication, endpointId uint64, criteria []*endpoint_api.Criteria, paginate *endpoint_api.Paginate) (*endpoint_api.Paginated, []*endpoint_api.EndpointLog, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, nil, err
	}
	res, err := client.endpointClient.GetAllEndpointLog(authContext, &endpoint_api.GetAllEndpointLogRequest{
		EndpointId: endpointId,
		Paginate:   paginate,
		Criterias:  criteria,
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

func (client *endpointServiceClient) GetEndpointLog(c context.Context, auth *types.Authentication, endpointRequest *endpoint_api.GetEndpointLogRequest) (*endpoint_api.GetEndpointLogResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.endpointClient.GetEndpointLog(authContext, endpointRequest)
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
