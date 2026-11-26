package web_api

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rapidaai/api/web-api/config"
	internal_connects "github.com/rapidaai/api/web-api/internal/connect"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_vault_service "github.com/rapidaai/api/web-api/internal/service/vault"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
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
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}
	vlt, err := wVault.vaultService.Create(
		ctx,
		principal,
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
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}
	vlt, err := wVault.vaultService.Delete(c, principal, irRequest.GetVaultId())
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
	criteria := make([]*protos.Criteria, 0, len(irRequest.GetCriterias()))
	if len(irRequest.GetCriterias()) > 2 {
		return providerCredentialListError(pkg_errors.ProviderCredentialInvalidCriteria), nil
	}
	seen := make(map[string]struct{}, len(irRequest.GetCriterias()))
	for _, criterion := range irRequest.GetCriterias() {
		if criterion == nil {
			return providerCredentialListError(pkg_errors.ProviderCredentialInvalidCriteria), nil
		}
		key := criterion.GetKey()
		if _, exists := seen[key]; exists {
			return providerCredentialListError(pkg_errors.ProviderCredentialInvalidCriteria), nil
		}
		value := strings.TrimSpace(criterion.GetValue())
		if strings.ContainsRune(value, '\x00') || !validator.Between(utf8.RuneCountInString(value), 1, 200) {
			return providerCredentialListError(pkg_errors.ProviderCredentialInvalidCriteria), nil
		}
		switch key {
		case "name":
			if criterion.GetLogic() != "contains" {
				return providerCredentialListError(pkg_errors.ProviderCredentialInvalidCriteria), nil
			}
		case "provider":
			if criterion.GetLogic() != "=" {
				return providerCredentialListError(pkg_errors.ProviderCredentialInvalidCriteria), nil
			}
		default:
			return providerCredentialListError(pkg_errors.ProviderCredentialInvalidCriteria), nil
		}
		seen[key] = struct{}{}
		criteria = append(criteria, &protos.Criteria{Key: key, Logic: criterion.GetLogic(), Value: value})
	}
	paginate := &protos.Paginate{Page: 1, PageSize: 25}
	if irRequest.GetPaginate() != nil {
		paginate.Page = irRequest.GetPaginate().GetPage()
		paginate.PageSize = irRequest.GetPaginate().GetPageSize()
		if paginate.Page == 0 || paginate.PageSize == 0 || paginate.PageSize > 100 {
			return providerCredentialListError(pkg_errors.ProviderCredentialInvalidPagination), nil
		}
	}
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}
	cnt, vlts, err := wVault.vaultService.GetAllOrganizationCredential(c, principal, criteria, paginate)
	if err != nil {
		wVault.logger.Errorf("vaultService.GetAll from grpc with err %v", err)
		return utils.Error[protos.GetAllOrganizationCredentialResponse](
			err,
			"Unable to get provider credentials, please try again",
		)
	}

	out := make([]*protos.VaultCredential, 0, len(vlts))
	for _, credential := range vlts {
		if credential == nil {
			continue
		}
		item := &protos.VaultCredential{
			Id: credential.Id, Name: credential.Name, Provider: credential.Provider,
			Status: credential.Status.String(), OrganizationId: credential.OrganizationId,
			CreatedDate: timestamppb.New(time.Time(credential.CreatedDate)),
		}
		if updated := time.Time(credential.UpdatedDate); !updated.IsZero() {
			item.UpdatedDate = timestamppb.New(updated)
		}
		out = append(out, item)
	}
	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		wVault.logger.Errorf("invalid provider credential count %d: %v", cnt, err)
		return utils.Error[protos.GetAllOrganizationCredentialResponse](
			err,
			"Unable to get provider credentials, please try again",
		)
	}
	return utils.PaginatedSuccess[protos.GetAllOrganizationCredentialResponse, []*protos.VaultCredential](
		totalItem,
		paginate.GetPage(),
		out)
}

func providerCredentialListError(platformError pkg_errors.PlatformError) *protos.GetAllOrganizationCredentialResponse {
	return &protos.GetAllOrganizationCredentialResponse{
		Code:  platformError.HTTPStatusCodeInt32(),
		Error: &protos.Error{ErrorCode: uint64(platformError.Code), ErrorMessage: platformError.Error, HumanMessage: platformError.ErrorMessage},
	}
}

func (wVault *webVaultGRPCApi) GetOauth2Credential(ctx context.Context, request *protos.GetCredentialRequest) (*protos.GetCredentialResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}
	vlt, err := wVault.vaultService.Get(
		ctx, tenant, request.GetVaultId())

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
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}
	//
	vlt, err := wVault.vaultService.Get(ctx, tenant, request.GetVaultId())
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
