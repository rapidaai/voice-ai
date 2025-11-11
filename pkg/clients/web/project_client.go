package web_client

import (
	"context"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	project_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProjectClient interface {
	GetProject(c context.Context, auth types.SimplePrinciple, projectId uint64) (*project_api.GetProjectResponse, error)
}
type projectServiceClient struct {
	clients.InternalClient
	cfg           *config.AppConfig
	logger        commons.Logger
	projectClient project_api.ProjectServiceClient
}

func NewProjectServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) ProjectClient {
	conn, err := grpc.NewClient(config.WebHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	providerClient := project_api.NewProjectServiceClient(conn)
	return &projectServiceClient{
		InternalClient: clients.NewInternalClient(config, logger, redis),
		cfg:            config,
		logger:         logger,
		projectClient:  providerClient,
	}
}

func (pClient projectServiceClient) GetProject(c context.Context, auth types.SimplePrinciple, projectId uint64) (*project_api.GetProjectResponse, error) {
	pr, err := pClient.projectClient.GetProject(pClient.WithAuth(c, auth), &project_api.GetProjectRequest{ProjectId: projectId})
	if err != nil {
		pClient.logger.Errorf("Unable to get the project %+v", err)
		return nil, err
	}
	return pr, nil
}
