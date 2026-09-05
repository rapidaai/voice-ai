// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package web_client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	project_api "github.com/rapidaai/protos"
)

type ProjectClient interface {
	GetProject(c context.Context, auth *types.Authentication, projectId uint64) (*project_api.GetProjectResponse, error)
}
type projectServiceClient struct {
	clients.InternalClient
	cfg           *config.AppConfig
	logger        commons.Logger
	projectClient project_api.ProjectServiceClient
}

func NewProjectServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) ProjectClient {
	conn, err := grpc.NewClient(config.Web.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return NewProjectClientWithClient(config, logger, redis, project_api.NewProjectServiceClient(conn))
}

// NewProjectClientWithClient creates a project client using the provided gRPC client.
func NewProjectClientWithClient(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector, projectClient project_api.ProjectServiceClient) ProjectClient {
	return &projectServiceClient{
		InternalClient: clients.NewInternalClient(config, logger, redis),
		cfg:            config,
		logger:         logger,
		projectClient:  projectClient,
	}
}

func (pClient projectServiceClient) GetProject(c context.Context, auth *types.Authentication, projectId uint64) (*project_api.GetProjectResponse, error) {
	authContext, err := pClient.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	pr, err := pClient.projectClient.GetProject(authContext, &project_api.GetProjectRequest{ProjectId: projectId})
	if err != nil {
		pClient.logger.Errorf("Unable to get the project %+v", err)
		return nil, err
	}
	return pr, nil
}
