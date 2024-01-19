package web_api

import (
	"context"
	"errors"

	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_vault_service "github.com/lexatic/web-backend/internal/services/vault"
	clients "github.com/lexatic/web-backend/pkg/clients"
	integration_client "github.com/lexatic/web-backend/pkg/clients/integration"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webActivityApi struct {
	cfg               *config.AppConfig
	logger            commons.Logger
	postgres          connectors.PostgresConnector
	integrationClient clients.IntegrationServiceClient
	vaultService      internal_services.VaultService
}

type webActivityGRPCApi struct {
	webActivityApi
}

func NewActivityGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.ActivityServiceServer {
	return &webActivityGRPCApi{
		webActivityApi{
			cfg:               config,
			logger:            logger,
			postgres:          postgres,
			integrationClient: integration_client.NewIntegrationServiceClientGRPC(config, logger),
			vaultService:      internal_vault_service.NewVaultService(logger, postgres),
		},
	}
}

func (wActivity *webActivityGRPCApi) GetActivities(c context.Context, irRequest *web_api.GetActivityRequest) (*web_api.GetActivityResponse, error) {
	wActivity.logger.Debugf("GetActivities from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wActivity.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	// check if he is already part of current organization
	adt, err := wActivity.integrationClient.GetAuditLog(c, iAuth.GetOrganizationRole().OrganizationId, irRequest.GetProjectId(), irRequest.GetPage(), irRequest.GetPageSize())
	if err != nil {
		return &web_api.GetActivityResponse{
			Code:    500,
			Success: false,
		}, nil
	}

	if adt.GetSuccess() {
		// later will inject all the params from db
		wActivity.logger.Debugf("response from %v", adt.GetData())
		var out []*web_api.Activity
		err := types.Cast(adt.GetData(), &out)
		if err != nil {
			wActivity.logger.Debugf("unable to cast the object with error %v", err)
		}
		return &web_api.GetActivityResponse{
			Code:    200,
			Success: true,
			Data:    out,
		}, nil
	}
	wActivity.logger.Debugf("Got response from integration service %+v", adt)
	return &web_api.GetActivityResponse{
		Code:    200,
		Success: false,
	}, nil
}
