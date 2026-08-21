package web_api

import (
	"context"

	"github.com/rapidaai/api/web-api/config"
	internal_connects "github.com/rapidaai/api/web-api/internal/connect"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_vault_service "github.com/rapidaai/api/web-api/internal/service/vault"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type webVaultApi struct {
	cfg               *config.WebAppConfig
	logger            commons.Logger
	postgres          connectors.PostgresConnector
	redis             connectors.RedisConnector
	vaultService      internal_service.VaultService
	integrationClient integration_client.IntegrationServiceClient
	hubspotConnect    internal_connects.HubspotConnect
}

type webVaultRPCApi struct {
	webVaultApi
}

type webVaultGRPCApi struct {
	webVaultApi
}

func NewVaultRPC(config *config.WebAppConfig, oauthCfg *config.OAuth2Config, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) *webVaultRPCApi {
	return &webVaultRPCApi{
		webVaultApi{
			cfg:               config,
			logger:            logger,
			postgres:          postgres,
			vaultService:      internal_vault_service.NewVaultService(logger, postgres),
			integrationClient: integration_client.NewIntegrationServiceClientGRPC(&config.AppConfig, logger, redis),
			hubspotConnect:    internal_connects.NewHubspotConnect(config, oauthCfg, logger, postgres),
		},
	}
}

func NewVaultGRPC(config *config.WebAppConfig, oauthCfg *config.OAuth2Config, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.VaultServiceServer {
	return &webVaultGRPCApi{
		webVaultApi{
			cfg:               config,
			logger:            logger,
			postgres:          postgres,
			redis:             redis,
			vaultService:      internal_vault_service.NewVaultService(logger, postgres),
			integrationClient: integration_client.NewIntegrationServiceClientGRPC(&config.AppConfig, logger, redis),
			hubspotConnect:    internal_connects.NewHubspotConnect(config, oauthCfg, logger, postgres),
		},
	}
}

func (wVault *webVaultGRPCApi) CreateProviderCredential(ctx context.Context, irRequest *protos.CreateProviderCredentialRequest) (*protos.GetCredentialResponse, error) {
	iAuth, _ := types.GetAuthPrincipleGPRC(ctx)
	userID, err := types.RequireUser(iAuth)
	if err != nil {
		wVault.logger.Errorf("CreateProviderCredential from grpc with unauthenticated request")
		return utils.AuthenticateError[protos.GetCredentialResponse]()
	}
	projectContext, err := types.RequireProject(iAuth)
	if err != nil {
		wVault.logger.Errorf("CreateProviderCredential from grpc without project context: %v", err)
		return utils.AuthenticateError[protos.GetCredentialResponse]()
	}

	vlt, err := wVault.vaultService.Create(
		ctx,
		userID,
		projectContext,
		irRequest.GetProvider(),
		irRequest.GetName(), irRequest.GetCredential().AsMap())
	if err != nil {
		wVault.logger.Errorf("vaultService.Create from grpc with err %v", err)
		return utils.Error[protos.GetCredentialResponse](
			err,
			"Unable to create provider credential, please try again")
	}

	out := &protos.VaultCredential{}
	err = utils.Cast(vlt, out)
	if err != nil {
		wVault.logger.Errorf("unable to cast the provider credentials to proto %v", err)
	}
	return utils.Success[protos.GetCredentialResponse](out)
}

func (wVault *webVaultGRPCApi) DeleteCredential(c context.Context, irRequest *protos.DeleteCredentialRequest) (*protos.GetCredentialResponse, error) {
	iAuth, _ := types.GetAuthPrincipleGPRC(c)
	userID, err := types.RequireUser(iAuth)
	if err != nil {
		wVault.logger.Errorf("DeleteProviderCredential from grpc with unauthenticated request")
		return utils.AuthenticateError[protos.GetCredentialResponse]()
	}
	projectContext, err := types.RequireProject(iAuth)
	if err != nil {
		wVault.logger.Errorf("DeleteProviderCredential from grpc without project context: %v", err)
		return utils.AuthenticateError[protos.GetCredentialResponse]()
	}

	vlt, err := wVault.vaultService.Delete(c, userID, projectContext, irRequest.GetVaultId())
	if err != nil {
		wVault.logger.Errorf("vaultService.Delete from grpc with err %v", err)
		return &protos.GetCredentialResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to delete provider credential, please try again",
			}}, nil
	}
	out := &protos.VaultCredential{}
	err = utils.Cast(vlt, out)
	if err != nil {
		wVault.logger.Errorf("unable to cast the provider credentials to proto %v", err)
	}
	return utils.Success[protos.GetCredentialResponse](out)
}

func (wVault *webVaultGRPCApi) GetAllOrganizationCredential(c context.Context, irRequest *protos.GetAllOrganizationCredentialRequest) (*protos.GetAllOrganizationCredentialResponse, error) {
	iAuth, _ := types.GetSimplePrincipleGRPC(c)
	projectContext, err := types.RequireProject(iAuth)
	if err != nil {
		wVault.logger.Errorf("GetAllOrganizationCredential from grpc with unauthenticated request")
		return utils.AuthenticateError[protos.GetAllOrganizationCredentialResponse]()
	}
	cnt, vlts, err := wVault.vaultService.GetAllOrganizationCredential(c, projectContext, irRequest.GetCriterias(), irRequest.GetPaginate())
	if err != nil {
		wVault.logger.Errorf("vaultService.GetAll from grpc with err %v", err)
		return utils.Error[protos.GetAllOrganizationCredentialResponse](
			err,
			"Unable to get provider credentials, please try again",
		)
	}

	out := make([]*protos.VaultCredential, len(vlts))
	err = utils.Cast(vlts, &out)
	if err != nil {
		wVault.logger.Errorf("unable to cast vault object to proto %v", err)
	}

	for _, c := range out {
		c.Value = nil
	}
	return utils.PaginatedSuccess[protos.GetAllOrganizationCredentialResponse, []*protos.VaultCredential](
		uint32(cnt),
		irRequest.GetPaginate().GetPage(),
		out)
}

func (wVault *webVaultGRPCApi) GetOauth2Credential(ctx context.Context, request *protos.GetCredentialRequest) (*protos.GetCredentialResponse, error) {
	iAuth, _ := types.GetSimplePrincipleGRPC(ctx)
	projectContext, err := types.RequireProject(iAuth)
	if err != nil {
		wVault.logger.Errorf("GetAllProviderCredential from grpc with unauthenticated request")
		return utils.AuthenticateError[protos.GetCredentialResponse]()
	}
	vlt, err := wVault.vaultService.Get(
		ctx, projectContext, request.GetVaultId())

	if err != nil {
		wVault.logger.Errorf("unable to get tool credentials %v", err)
		return utils.Error[protos.GetCredentialResponse](err, "Unable to get tool credential to get list of files.")
	}
	token, _, err := wVault.hubspotConnect.ToToken(vlt.Value)
	if err != nil {
		wVault.logger.Errorf("unable to get tool credentials %v", err)
		return utils.Error[protos.GetCredentialResponse](err, "Unable to get tool credential to get list of files.")
	}
	newToken, err := wVault.hubspotConnect.RefreshToken(ctx, token)

	vlt.Value = newToken.Map()
	var out protos.VaultCredential
	err = utils.Cast(vlt, &out)
	if err != nil {
		wVault.logger.Errorf("unable to cast vault object to proto %v", err)
	}
	return utils.Success[protos.GetCredentialResponse, *protos.VaultCredential](&out)
}

func (wVault *webVaultGRPCApi) GetCredential(ctx context.Context, request *protos.GetCredentialRequest) (*protos.GetCredentialResponse, error) {
	iAuth, _ := types.GetSimplePrincipleGRPC(ctx)
	projectContext, err := types.RequireProject(iAuth)
	if err != nil {
		wVault.logger.Errorf("GetCredentialRequest from grpc with unauthenticated request")
		return utils.AuthenticateError[protos.GetCredentialResponse]()
	}
	//
	vlt, err := wVault.vaultService.Get(ctx, projectContext, request.GetVaultId())
	if err != nil {
		return utils.Error[protos.GetCredentialResponse](
			err,
			"Unable to get vault credential, please try again",
		)
	}
	var out protos.VaultCredential
	err = utils.Cast(vlt, &out)
	if err != nil {
		wVault.logger.Errorf("unable to cast vault object to proto %v", err)
	}
	return utils.Success[protos.GetCredentialResponse, *protos.VaultCredential](&out)
}
