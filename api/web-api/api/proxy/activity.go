package web_proxy_api

import (
	"context"

	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	web_api "github.com/rapidaai/api/web-api/api"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_vault_service "github.com/rapidaai/api/web-api/internal/service/vault"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	protos "github.com/rapidaai/protos"

	config "github.com/rapidaai/api/web-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
)

type webActivityApi struct {
	web_api.WebApi
	cfg          *config.WebAppConfig
	logger       commons.Logger
	postgres     connectors.PostgresConnector
	redis        connectors.RedisConnector
	auditClient  integration_client.AuditServiceClient
	vaultService internal_service.VaultService
}

type webActivityGRPCApi struct {
	webActivityApi
}

func NewActivityGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector, auditClient integration_client.AuditServiceClient) protos.AuditLoggingServiceServer {
	return &webActivityGRPCApi{
		webActivityApi{
			WebApi:       web_api.NewWebApi(config, logger, postgres, redis),
			cfg:          config,
			logger:       logger,
			postgres:     postgres,
			redis:        redis,
			auditClient:  auditClient,
			vaultService: internal_vault_service.NewVaultService(logger, postgres),
		},
	}
}

func (wActivity *webActivityGRPCApi) GetAuditLog(c context.Context, irRequest *protos.GetAuditLogRequest) (*protos.GetAuditLogResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	return wActivity.auditClient.GetAuditLog(c, principal, irRequest.GetId())
}

func (wActivity *webActivityGRPCApi) GetAllAuditLog(c context.Context, irRequest *protos.GetAllAuditLogRequest) (*protos.GetAllAuditLogResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	// check if he is already part of current organization
	return wActivity.auditClient.GetAllAuditLog(c, principal, irRequest)
}

func (wActivity *webActivityGRPCApi) CreateMetadata(c context.Context, irRequest *protos.CreateMetadataRequest) (*protos.CreateMetadataResponse, error) {
	return nil, fmt.Errorf("unimplimented method")
}
