// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package workflow_client

import (
	"context"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	assistant_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AssistantConversationServiceClient interface {
	GetAllAssistantConversation(c context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversation, error)
	GetAllConversationMessage(c context.Context, auth types.SimplePrinciple, assistantId, assistantConversationId uint64, criterias []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversationMessage, error)
}

type assistantConversationServiceClient struct {
	clients.InternalClient
	cfg             *config.AppConfig
	logger          commons.Logger
	assistantClient assistant_api.TalkServiceClient
}

func NewAssistantConversationServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) AssistantConversationServiceClient {
	logger.Debugf("conntecting to assistant conversaction client with %s", config.AssistantHost)
	conn, err := grpc.NewClient(config.AssistantHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Errorf("Unable to create connection %v", err)
	}
	return &assistantConversationServiceClient{
		InternalClient:  clients.NewInternalClient(config, logger, redis),
		cfg:             config,
		logger:          logger,
		assistantClient: assistant_api.NewTalkServiceClient(conn),
	}
}

func (client *assistantConversationServiceClient) GetAllAssistantConversation(c context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversation, error) {
	client.logger.Debugf("get all assistant request")
	res, err := client.assistantClient.GetAllAssistantConversation(client.WithAuth(c, auth),
		&assistant_api.GetAllAssistantConversationRequest{
			AssistantId: assistantId,
			Paginate:    paginate,
			Criterias:   criterias,
		})
	if err != nil {
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	return res.GetPaginated(), res.GetData(), nil
}

func (client *assistantConversationServiceClient) GetAllConversationMessage(c context.Context, auth types.SimplePrinciple, assistantId, assistantConversationId uint64, criterias []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversationMessage, error) {
	client.logger.Debugf("get all assistant request")
	res, err := client.assistantClient.GetAllConversationMessage(client.WithAuth(c, auth),
		&assistant_api.GetAllConversationMessageRequest{
			AssistantId:             assistantId,
			AssistantConversationId: assistantConversationId,
			Paginate:                paginate,
			Criterias:               criterias,
		})
	if err != nil {
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all assistant %v", err)
		return nil, nil, err
	}

	return res.GetPaginated(), res.GetData(), nil
}
