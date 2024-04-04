package web_api

import (
	"context"
	"errors"

	config "github.com/lexatic/web-backend/config"
	webhook_client "github.com/lexatic/web-backend/pkg/clients/webhook"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webWebhookApi struct {
	cfg           *config.AppConfig
	logger        commons.Logger
	redis         connectors.RedisConnector
	postgres      connectors.PostgresConnector
	webhookClient webhook_client.WebhookServiceClient
}

type webWebhookGRPCApi struct {
	webWebhookApi
}

// CreateWebhook implements lexatic_backend.WebhookManagerServiceServer.
func (webhookGrpc *webWebhookGRPCApi) CreateWebhook(ctx context.Context, iRequest *web_api.CreateWebhookRequest) (*web_api.CreateWebhookResponse, error) {
	webhookGrpc.logger.Debugf("CreateWebhook from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		webhookGrpc.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return webhookGrpc.webhookClient.CreateWebhook(ctx,
		iRequest.GetUrl(), iRequest.GetDescription(), iRequest.GetEventType(), iRequest.GetMaxRetryCount(),
		iAuth.GetUserInfo().Id,
		iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)
}

// DeleteWebhook implements lexatic_backend.WebhookManagerServiceServer.
func (webhookGrpc *webWebhookGRPCApi) DeleteWebhook(ctx context.Context, iRequest *web_api.DeleteWehbookRequest) (*web_api.DeleteWebhookResponse, error) {
	webhookGrpc.logger.Debugf("DeleteWebhook from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		webhookGrpc.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return webhookGrpc.webhookClient.DeleteWebhook(ctx, iRequest.GetWebhookId(), iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)

}

// DisableWebhook implements lexatic_backend.WebhookManagerServiceServer.
func (webhookGrpc *webWebhookGRPCApi) DisableWebhook(ctx context.Context, iRequest *web_api.DisableWebhookRequest) (*web_api.DisableWebhookResponse, error) {
	webhookGrpc.logger.Debugf("DisableWebhook from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		webhookGrpc.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return webhookGrpc.webhookClient.DisableWebhook(ctx, iRequest.GetWebhookId(), iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)

}

// GetAllWebhook implements lexatic_backend.WebhookManagerServiceServer.
func (webhookGrpc *webWebhookGRPCApi) GetAllWebhook(ctx context.Context, iRequest *web_api.GetAllWebhookRequest) (*web_api.GetAllWebhookResponse, error) {
	webhookGrpc.logger.Debugf("GetAllWebhook from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		webhookGrpc.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return webhookGrpc.webhookClient.GetAllWebhook(ctx, iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iRequest.GetCriterias(), iRequest.GetPaginate())
}

// GetWebhook implements lexatic_backend.WebhookManagerServiceServer.
func (webhookGrpc *webWebhookGRPCApi) GetWebhook(ctx context.Context, iRequest *web_api.GetWebhookRequest) (*web_api.GetWebhookResponse, error) {
	webhookGrpc.logger.Debugf("GetWebhook from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		webhookGrpc.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	return webhookGrpc.webhookClient.GetWebhook(ctx, iRequest.GetId(), iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)
}

func NewWebhookGRPC(config *config.AppConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
) web_api.WebhookManagerServiceServer {
	return &webWebhookGRPCApi{
		webWebhookApi{
			cfg:           config,
			logger:        logger,
			postgres:      postgres,
			redis:         redis,
			webhookClient: webhook_client.NewWebhookServiceClientGRPC(config, logger),
		},
	}
}
