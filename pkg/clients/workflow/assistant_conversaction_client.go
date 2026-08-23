// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package workflow_client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	assistant_api "github.com/rapidaai/protos"
)

type AssistantConversationServiceClient interface {
	GetAllAssistantConversation(c context.Context, auth *types.Authentication, assistantId uint64, criteria []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversation, error)
	GetAllConversationMessage(c context.Context, auth *types.Authentication, assistantId, assistantConversationId uint64, criteria []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversationMessage, error)
}

type assistantConversationServiceClient struct {
	clients.InternalClient
	cfg             *config.AppConfig
	logger          commons.Logger
	assistantClient assistant_api.TalkServiceClient
}

func NewAssistantConversationServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) AssistantConversationServiceClient {
	conn, err := grpc.NewClient(config.Assistant.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (client *assistantConversationServiceClient) GetAllAssistantConversation(c context.Context, auth *types.Authentication, assistantId uint64, criteria []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversation, error) {
	client.logger.Debugf("get all assistant request")
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, nil, err
	}
	res, err := client.assistantClient.GetAllAssistantConversation(authContext,
		&assistant_api.GetAllAssistantConversationRequest{
			AssistantId: assistantId,
			Paginate:    paginate,
			Criterias:   criteria,
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

func (client *assistantConversationServiceClient) GetAllConversationMessage(c context.Context, auth *types.Authentication, assistantId, assistantConversationId uint64, criteria []*assistant_api.Criteria, paginate *assistant_api.Paginate) (*assistant_api.Paginated, []*assistant_api.AssistantConversationMessage, error) {
	client.logger.Debugf("get all assistant request")
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, nil, err
	}
	res, err := client.assistantClient.GetAllConversationMessage(authContext,
		&assistant_api.GetAllConversationMessageRequest{
			AssistantId:             assistantId,
			AssistantConversationId: assistantConversationId,
			Paginate:                paginate,
			Criterias:               criteria,
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
