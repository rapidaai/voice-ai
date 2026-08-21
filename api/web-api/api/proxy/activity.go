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
	"github.com/rapidaai/pkg/types"
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

func NewActivityGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.AuditLoggingServiceServer {
	return &webActivityGRPCApi{
		webActivityApi{
			WebApi:       web_api.NewWebApi(config, logger, postgres, redis),
			cfg:          config,
			logger:       logger,
			postgres:     postgres,
			redis:        redis,
			auditClient:  integration_client.NewAuditServiceClient(&config.AppConfig, logger, redis),
			vaultService: internal_vault_service.NewVaultService(logger, postgres),
		},
	}
}

func (wActivity *webActivityGRPCApi) GetAuditLog(c context.Context, irRequest *protos.GetAuditLogRequest) (*protos.GetAuditLogResponse, error) {
	wActivity.logger.Debugf("GetActivities from grpc with requestPayload %v, %v", irRequest, c)
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	return wActivity.auditClient.GetAuditLog(c, iAuth, irRequest.GetId())
}

func (wActivity *webActivityGRPCApi) GetAllAuditLog(c context.Context, irRequest *protos.GetAllAuditLogRequest) (*protos.GetAllAuditLogResponse, error) {
	wActivity.logger.Debugf("GetActivities from grpc with requestPayload %v, %v", irRequest, c)
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	// check if he is already part of current organization
	return wActivity.auditClient.GetAllAuditLog(c, iAuth, irRequest)
}

func (wActivity *webActivityGRPCApi) CreateMetadata(c context.Context, irRequest *protos.CreateMetadataRequest) (*protos.CreateMetadataResponse, error) {
	return nil, fmt.Errorf("unimplimented method")
}
