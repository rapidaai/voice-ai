package integration

import (
	"context"

	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	"github.com/lexatic/web-backend/pkg/commons"
	webhook_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type webhookServiceClient struct {
	cfg           *config.AppConfig
	logger        commons.Logger
	webhookClient webhook_api.WebhookManagerServiceClient
}

func NewWebhookServiceClientGRPC(config *config.AppConfig, logger commons.Logger) clients.WebhookServiceClient {

	logger.Debugf("conntecting to webhook client with %s", config.WebhookHost)
	conn, err := grpc.Dial(config.WebhookHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return &webhookServiceClient{
		cfg:           config,
		logger:        logger,
		webhookClient: webhook_api.NewWebhookManagerServiceClient(conn),
	}
}

func (client *webhookServiceClient) CreateWebhook(c context.Context,
	url, description string, eventType []string, maxRetryCount uint32,
	userId, projectId, organizationId uint64,
) (*webhook_api.CreateWebhookResponse, error) {
	res, err := client.webhookClient.CreateWebhook(c, &webhook_api.CreateWebhookRequest{
		Url: url, Description: description, EventType: eventType,
		MaxRetryCount: maxRetryCount,
		UserId:        userId,
		ProjectId:     projectId, OrganizationId: organizationId,
	})
	if err != nil {
		client.logger.Errorf("unable to create webhook error %v", err)
		return nil, err
	}
	return res, nil
}
func (client *webhookServiceClient) DisableWebhook(ctx context.Context, id, projectId, organizationId uint64) (*webhook_api.DisableWebhookResponse, error) {
	client.logger.Debugf("getting webhook for sepcific id %v", id)
	res, err := client.webhookClient.DisableWebhook(ctx, &webhook_api.DisableWebhookRequest{
		WebhookId: id, ProjectId: projectId, OrganizationId: organizationId,
	})
	if err != nil {
		client.logger.Errorf("unable to disable webhook error %v", err)
		return nil, err
	}
	return res, nil
}
func (client *webhookServiceClient) DeleteWebhook(ctx context.Context, id, projectId, organizationId uint64) (*webhook_api.DeleteWebhookResponse, error) {
	client.logger.Debugf("getting webhook for sepcific id %v", id)
	res, err := client.webhookClient.DeleteWebhook(ctx, &webhook_api.DeleteWehbookRequest{
		WebhookId: id, ProjectId: projectId, OrganizationId: organizationId,
	})
	if err != nil {
		client.logger.Errorf("unable to delete webhook error %v", err)
		return nil, err
	}
	return res, nil
}
func (client *webhookServiceClient) GetWebhook(c context.Context, id, projectId, organizationId uint64) (*webhook_api.GetWebhookResponse, error) {
	client.logger.Debugf("getting webhook for sepcific id %v", id)
	res, err := client.webhookClient.GetWebhook(c, &webhook_api.GetWebhookRequest{
		Id: id, ProjectId: projectId, OrganizationId: organizationId,
	})
	if err != nil {
		client.logger.Errorf("unable to get webhook error %v", err)
		return nil, err
	}
	return res, nil

}

func (client *webhookServiceClient) GetAllWebhook(c context.Context, projectId, organizationId uint64, page, pageSize uint32) (*webhook_api.GetAllWebhookResponse, error) {
	res, err := client.webhookClient.GetAllWebhook(c, &webhook_api.GetAllWebhookRequest{
		ProjectId: projectId, OrganizationId: organizationId, Page: page, PageSize: pageSize,
	})
	if err != nil {
		client.logger.Errorf("unable to get all the webhook error %v", err)
		return nil, err
	}
	return res, nil

}
