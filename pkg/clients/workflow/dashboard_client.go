package workflow_client

import (
	"context"
	"errors"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProjectDashboardAnalyticsClient interface {
	GetProjectDashboardAnalytics(context.Context, *types.Authentication, *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error)
}

type projectDashboardAnalyticsClient struct {
	clients.InternalClient
	client protos.ProjectDashboardAnalyticsServiceClient
}

func NewProjectDashboardAnalyticsClientGRPC(cfg *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) ProjectDashboardAnalyticsClient {
	connection, err := grpc.NewClient(cfg.Assistant.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Errorf("unable to create dashboard analytics connection: %v", err)
	}
	return &projectDashboardAnalyticsClient{
		InternalClient: clients.NewInternalClient(cfg, logger, redis),
		client:         protos.NewProjectDashboardAnalyticsServiceClient(connection),
	}
}

func (client *projectDashboardAnalyticsClient) GetProjectDashboardAnalytics(ctx context.Context, auth *types.Authentication, request *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
	authContext, err := client.WithAuth(ctx, auth)
	if err != nil {
		return nil, err
	}
	response, err := client.client.GetProjectDashboardAnalytics(authContext, request)
	if err != nil {
		return nil, err
	}
	if !response.GetSuccess() || response.GetData() == nil {
		return nil, errors.New("dashboard analytics response did not include data")
	}
	return response.GetData(), nil
}
