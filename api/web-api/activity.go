package web_api

import (
	"context"
	"errors"
	"fmt"

	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_vault_service "github.com/lexatic/web-backend/internal/services/vault"
	integration_client "github.com/lexatic/web-backend/pkg/clients/integration"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webActivityApi struct {
	cfg          *config.AppConfig
	logger       commons.Logger
	postgres     connectors.PostgresConnector
	redis        connectors.RedisConnector
	auditClient  integration_client.AuditServiceClient
	vaultService internal_services.VaultService
}

type webActivityGRPCApi struct {
	webActivityApi
}

func NewActivityGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.AuditLoggingServiceServer {
	return &webActivityGRPCApi{
		webActivityApi{
			cfg:          config,
			logger:       logger,
			postgres:     postgres,
			redis:        redis,
			auditClient:  integration_client.NewAuditServiceClient(config, logger, redis),
			vaultService: internal_vault_service.NewVaultService(logger, postgres),
		},
	}
}

func (wActivity *webActivityGRPCApi) GetAuditLog(c context.Context, irRequest *web_api.GetAuditLogRequest) (*web_api.GetAuditLogResponse, error) {
	wActivity.logger.Debugf("GetActivities from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wActivity.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	// check if he is already part of current organization
	return wActivity.auditClient.GetAuditLog(c, iAuth, irRequest.GetId())
}

func (wActivity *webActivityGRPCApi) GetAllAuditLog(c context.Context, irRequest *web_api.GetAllAuditLogRequest) (*web_api.GetAllAuditLogResponse, error) {
	wActivity.logger.Debugf("GetActivities from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wActivity.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	// check if he is already part of current organization
	return wActivity.auditClient.GetAllAuditLog(c, iAuth, irRequest)
}

func (wActivity *webActivityGRPCApi) CreateMetadata(c context.Context, irRequest *web_api.CreateMetadataRequest) (*web_api.CreateMetadataResponse, error) {
	return nil, fmt.Errorf("unimplimented method")
}
