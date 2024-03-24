package web_api

import (
	"bytes"
	"context"
	"errors"
	"strings"

	config "github.com/lexatic/web-backend/config"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_vault_service "github.com/lexatic/web-backend/internal/services/vault"
	provider_client "github.com/lexatic/web-backend/pkg/clients/provider"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webVaultApi struct {
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	redis          connectors.RedisConnector
	vaultService   internal_services.VaultService
	providerClient provider_client.ProviderServiceClient
}

type webVaultRPCApi struct {
	webVaultApi
}

type webVaultGRPCApi struct {
	webVaultApi
}

func NewVaultRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) *webVaultRPCApi {
	return &webVaultRPCApi{
		webVaultApi{
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			vaultService:   internal_vault_service.NewVaultService(logger, postgres),
			providerClient: provider_client.NewProviderServiceClientGRPC(config, logger, redis),
		},
	}
}

func NewVaultGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.VaultServiceServer {
	return &webVaultGRPCApi{
		webVaultApi{
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			vaultService:   internal_vault_service.NewVaultService(logger, postgres),
			providerClient: provider_client.NewProviderServiceClientGRPC(config, logger, redis),
		},
	}
}

func (wVault *webVaultGRPCApi) CreateProviderCredential(ctx context.Context, irRequest *web_api.CreateProviderCredentialRequest) (*web_api.CreateProviderCredentialResponse, error) {
	wVault.logger.Debugf("CreateProviderCredential from grpc with requestPayload %v, %v", irRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wVault.logger.Errorf("CreateProviderCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}
	vlt, err := wVault.vaultService.Create(ctx, iAuth, iAuth.GetOrganizationRole().OrganizationId, irRequest.GetProviderId(), irRequest.GetKeyName(), irRequest.GetProviderKey())
	if err != nil {
		wVault.logger.Errorf("vaultService.Create from grpc with err %v", err)
		return &web_api.CreateProviderCredentialResponse{
			Code:    400,
			Success: false,
			Error: &web_api.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create provider credential, please try again",
			}}, nil
	}

	out := &web_api.ProviderCredential{}
	err = utils.Cast(vlt, out)
	if err != nil {
		wVault.logger.Errorf("unable to cast the provider credentials to proto %v", err)
	}
	return utils.Success[web_api.CreateProviderCredentialResponse, *web_api.ProviderCredential](out)
}

func (wVault *webVaultGRPCApi) DeleteProviderCredential(c context.Context, irRequest *web_api.DeleteProviderCredentialRequest) (*web_api.DeleteProviderCredentialResponse, error) {
	wVault.logger.Debugf("DeleteProviderCredential from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wVault.logger.Errorf("DeleteProviderCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	_, err := wVault.vaultService.Delete(c, iAuth, irRequest.GetProviderKeyId())
	if err != nil {
		wVault.logger.Errorf("vaultService.Delete from grpc with err %v", err)
		return &web_api.DeleteProviderCredentialResponse{
			Code:    400,
			Success: false,
			Error: &web_api.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to delete provider credential, please try again",
			}}, nil
	}

	return utils.Success[web_api.DeleteProviderCredentialResponse, uint64](irRequest.ProviderKeyId)
}

func (wVault *webVaultGRPCApi) GetAllProviderCredential(c context.Context, irRequest *web_api.GetAllProviderCredentialRequest) (*web_api.GetAllProviderCredentialResponse, error) {
	wVault.logger.Debugf("GetAllProviderCredential from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wVault.logger.Errorf("GetAllProviderCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}
	cnt, vlts, err := wVault.vaultService.GetAll(c, iAuth, iAuth.GetOrganizationRole().OrganizationId, irRequest.GetCriterias(), irRequest.GetPaginate())
	if err != nil {
		wVault.logger.Errorf("vaultService.GetAll from grpc with err %v", err)
		return utils.Error[web_api.GetAllProviderCredentialResponse](
			err,
			"Unable to get provider credentials, please try again",
		)
	}

	out := make([]*web_api.ProviderCredential, len(*vlts))
	err = utils.Cast(vlts, &out)
	if err != nil {
		wVault.logger.Errorf("unable to cast vault object to proto %v", err)
	}

	pmap := make(map[uint64]*web_api.Provider)
	if p, err := wVault.providerClient.GetAllProviders(c); err == nil {
		for _, provider := range p {
			pmap[provider.GetId()] = provider
		}
	}

	for _, c := range out {
		if val, ok := pmap[c.ProviderId]; ok {
			c.Provider = val.Name
			c.Image = val.Image
		}
		if irRequest.GetMask() {
			c.Key = maskCred(c.Key)
		}
	}

	return utils.PaginatedSuccess[web_api.GetAllProviderCredentialResponse, []*web_api.ProviderCredential](
		uint32(cnt),
		irRequest.GetPaginate().GetPage(),
		out)

}

func maskCred(key string) string {
	var buffer bytes.Buffer
	l := len(key)
	first := key[:2]
	buffer.WriteString(first)
	last := key[l-2:]
	if l-4 > 0 {
		buffer.WriteString(strings.Repeat("*", l-4))
	} else {
		buffer.WriteString(strings.Repeat("*", 1))
	}
	buffer.WriteString(last)
	return buffer.String()
}

/*
this is not good idea as these apis are opened to public
*/
func (wVault *webVaultGRPCApi) GetProviderCredential(ctx context.Context, request *web_api.GetProviderCredentialRequest) (*web_api.GetProviderCredentialResponse, error) {
	wVault.logger.Debugf("GetProviderCredential from grpc with requestPayload %v, %v", request, ctx)
	vlt, err := wVault.vaultService.Get(ctx, request.GetOrganizationId(), request.GetProviderId())
	if err != nil {
		return utils.Error[web_api.GetProviderCredentialResponse](
			err,
			"Unable to get provider credential, please try again",
		)
	}
	out := &web_api.ProviderCredential{}
	err = utils.Cast(vlt, out)
	if err != nil {
		wVault.logger.Errorf("unable to cast vault object to proto %v", err)
	}
	return utils.Success[web_api.GetProviderCredentialResponse, *web_api.ProviderCredential](out)
}

func (wVault *webVaultGRPCApi) UpdateVaultCredentials(ctx context.Context, request *web_api.UpdateVaultCredentialsRequest) (*web_api.UpdateVaultCredentialResponse, error) {
	wVault.logger.Debugf("UpdateVaultCredentials from grpc with requestPayload %v, %v", request, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wVault.logger.Errorf("DeleteProviderCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	_credential, err := wVault.vaultService.Update(ctx, iAuth, request.Id, request.ProviderId, request.GetKey(), request.GetName())
	if err != nil {
		wVault.logger.Errorf("vaultService.UpdateVaultCredentials from grpc with err %v", err)
		return utils.Error[web_api.UpdateVaultCredentialResponse](
			err,
			"Unable to update provider credential, please try again",
		)
	}
	out := &web_api.ProviderCredential{}
	err = utils.Cast(_credential, out)
	if err != nil {
		wVault.logger.Errorf("unable to cast the provider credentials to proto %v", err)
	}
	return utils.Success[web_api.UpdateVaultCredentialResponse, *web_api.ProviderCredential](out)

}
