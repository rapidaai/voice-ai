package web_api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"

	"github.com/gin-gonic/gin"

	config "github.com/rapidaai/api/web-api/config"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_organization_service "github.com/rapidaai/api/web-api/internal/service/organization"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	internal_vault_service "github.com/rapidaai/api/web-api/internal/service/vault"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer "github.com/rapidaai/pkg/clients/external/emailer"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type webOrganizationApi struct {
	cfg                 *config.WebAppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	redis               connectors.RedisConnector
	organizationService internal_service.OrganizationService
	userService         internal_service.UserService
	vaultService        internal_service.VaultService
	projectService      internal_service.ProjectService
	emailerClient       external_clients.Emailer
}

type webOrganizationRPCApi struct {
	webOrganizationApi
}

type webOrganizationGRPCApi struct {
	webOrganizationApi
}

func NewOrganizationRPC(config *config.WebAppConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
) *webOrganizationRPCApi {
	return &webOrganizationRPCApi{
		webOrganizationApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			redis:               redis,
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
			userService:         internal_user_service.NewUserService(logger, postgres),
			emailerClient:       external_emailer.NewEmailer(config.EmailerConfig, logger),
		},
	}
}

func NewOrganizationGRPC(config *config.WebAppConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector) protos.OrganizationServiceServer {
	return &webOrganizationGRPCApi{
		webOrganizationApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			redis:               redis,
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
			userService:         internal_user_service.NewUserService(logger, postgres),
			projectService:      internal_project_service.NewProjectService(logger, postgres),
			vaultService:        internal_vault_service.NewVaultService(logger, postgres),
			emailerClient:       external_emailer.NewEmailer(config.EmailerConfig, logger),
		},
	}
}

func (orgR *webOrganizationRPCApi) CreateOrganization(c *gin.Context) {
	principal, ok := middleware_authentication.Principal(c.Request.Context())
	if !ok {
		_ = c.Error(status.Error(codes.Unauthenticated, "valid authentication is required"))
		c.AbortWithStatusJSON(401, "illegal request.")
		return
	}

	orgR.logger.Debugf("CreateOrganization from rpc with gin context %v", c)
	var irRequest struct {
		OrganizationName     string `json:"organization_name"`
		OrganizationSize     string `json:"organization_size"`
		OrganizationIndustry string `json:"organization_industry"`
	}

	err := c.Bind(&irRequest)
	if err != nil {
		c.JSON(500, "unable to parse the request, some of the required field missing.")
		return
	}

	aOrg, err := orgR.organizationService.Create(c, principal, irRequest.OrganizationName, irRequest.OrganizationSize, irRequest.OrganizationIndustry)
	if err != nil {
		c.JSON(500, commons.Response{
			Code:    500,
			Success: false,
			Data:    commons.ErrorMessage{Code: 100, Message: err},
		})
		return
	}

	oRole, err := orgR.userService.CreateOrganizationRole(c, principal, internal_service.OrganizationOwnerRoleDefinitionID, principal.Subject().ID, aOrg.Id, type_enums.RECORD_ACTIVE)
	if err != nil {
		c.JSON(500, commons.Response{
			Code:    500,
			Success: false,
			Data:    commons.ErrorMessage{Code: 100, Message: err},
		})
		return
	}
	c.JSON(200, commons.Response{
		Code:    200,
		Success: true,
		Data:    map[string]interface{}{"Organization": aOrg, "Role": oRole},
	})
}

/*
For creation of organization and
*/
func (orgG *webOrganizationGRPCApi) CreateOrganization(c context.Context, irRequest *protos.CreateOrganizationRequest) (*protos.CreateOrganizationResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	userID := principal.Subject().ID
	if principal.Tenant().OrganizationID != 0 {
		orgG.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return &protos.CreateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: "Alerady part of another organization, you can't be part of multiple organization.",
				HumanMessage: "You are already part of an active organization.",
			}}, nil
	}

	// Creation of organization
	aOrg, err := orgG.organizationService.Create(c, principal, irRequest.OrganizationName, irRequest.OrganizationSize, irRequest.OrganizationIndustry)
	if err != nil {
		orgG.logger.Errorf("CreateOrganization from grpc with erro %v", err)
		return &protos.CreateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create organization, please try again.",
			}}, nil
	}

	// creation of organization role
	_, err = orgG.userService.CreateOrganizationRole(c, principal, internal_service.OrganizationOwnerRoleDefinitionID, userID, aOrg.Id, type_enums.RECORD_ACTIVE)
	if err != nil {
		orgG.logger.Errorf("CreateOrganizationRole from grpc with erro %v", err)
		return &protos.CreateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    401,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to assign role for your organization.",
			}}, nil
	}

	org := &protos.Organization{}
	err = utils.Cast(aOrg, org)
	if err != nil {
		orgG.logger.Errorf("unable to cast organization to proto org err %v", err)
	}
	return &protos.CreateOrganizationResponse{
		Code:    200,
		Success: true,
		Data:    org,
	}, nil
}

func (orgG *webOrganizationGRPCApi) UpdateOrganization(c context.Context, irRequest *protos.UpdateOrganizationRequest) (*protos.UpdateOrganizationResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if irRequest.GetOrganizationId() != principal.Tenant().OrganizationID {
		return nil, status.Error(codes.PermissionDenied, "organization context does not match request")
	}

	// updating organization
	_, err := orgG.organizationService.Update(c, principal, irRequest.OrganizationId, irRequest.OrganizationName, irRequest.OrganizationIndustry, irRequest.OrganizationContact)
	if err != nil {
		orgG.logger.Errorf("UpdateOrganization from grpc with erro %v", err)
		return &protos.UpdateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to update the organization, please try again in sometime.",
			}}, nil
	}
	// response
	return &protos.UpdateOrganizationResponse{
		Code:    200,
		Success: true,
	}, nil
}

// getting all the organization
func (orgG *webOrganizationGRPCApi) GetOrganization(c context.Context, irRequest *protos.GetOrganizationRequest) (*protos.GetOrganizationResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	tenant := principal.Tenant()
	aOrg, err := orgG.organizationService.Get(c, tenant.OrganizationID)
	if err != nil {
		orgG.logger.Errorf("organizationService.Get from grpc with erro %v", err)
		return &protos.GetOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to find your organization, please try again later.",
			}}, nil
	}

	org := &protos.Organization{}
	err = utils.Cast(aOrg, org)
	if err != nil {
		orgG.logger.Errorf("unable to cast organization to proto org err %v", err)
	}
	return &protos.GetOrganizationResponse{
		Code:    200,
		Success: true,
		Data:    org,
	}, nil
}

func (orgG *webOrganizationGRPCApi) UpdateBillingInformation(c context.Context, irRequest *protos.UpdateBillingInformationRequest) (*protos.BaseResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if principal.Tenant().OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	return nil, nil
}
