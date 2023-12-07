package web_api

import (
	"context"
	"errors"

	internal_project_service "github.com/lexatic/web-backend/internal/services/project"

	"github.com/gin-gonic/gin"
	config "github.com/lexatic/web-backend/config"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_organization_service "github.com/lexatic/web-backend/internal/services/organization"
	internal_user_service "github.com/lexatic/web-backend/internal/services/user"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webOrganizationApi struct {
	cfg                 *config.AppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	organizationService internal_services.OrganizationService
	userService         internal_services.UserService
	projectService      internal_services.ProjectService
}

type webOrganizationRPCApi struct {
	webOrganizationApi
}

type webOrganizationGRPCApi struct {
	webOrganizationApi
}

func NewOrganizationRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) *webOrganizationRPCApi {
	return &webOrganizationRPCApi{
		webOrganizationApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
			userService:         internal_user_service.NewUserService(logger, postgres),
		},
	}
}

func NewOrganizationGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.OrganizationServiceServer {
	return &webOrganizationGRPCApi{
		webOrganizationApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
			userService:         internal_user_service.NewUserService(logger, postgres),
			projectService:      internal_project_service.NewProjectService(logger, postgres),
		},
	}
}

func (orgR *webOrganizationRPCApi) CreateOrganization(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(401, "illegal request.")
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

	aOrg, err := orgR.organizationService.Create(c, auth, irRequest.OrganizationName, irRequest.OrganizationSize, irRequest.OrganizationIndustry)
	if err != nil {
		c.JSON(500, commons.Response{
			Code:    500,
			Success: false,
			Data:    commons.ErrorMessage{Code: 100, Message: err},
		})
		return
	}

	oRole, err := orgR.userService.CreateOrganizationRole(c, auth, "owner", auth.GetUserInfo().Id, aOrg.Id, "active")
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

func (orgG *webOrganizationGRPCApi) CreateOrganization(c context.Context, irRequest *web_api.CreateOrganizationRequest) (*web_api.CreateOrganizationResponse, error) {
	orgG.logger.Debugf("CreateOrganization from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	aOrg, err := orgG.organizationService.Create(c, iAuth, irRequest.OrganizationName, irRequest.OrganizationSize, irRequest.OrganizationIndustry)
	if err != nil {
		orgG.logger.Errorf("CreateOrganization from grpc with erro %v", err)
		return nil, err
	}

	aRole, err := orgG.userService.CreateOrganizationRole(c, iAuth, "owner", iAuth.GetUserInfo().Id, aOrg.Id, "active")
	if err != nil {
		orgG.logger.Errorf("CreateOrganizationRole from grpc with erro %v", err)
		return nil, err
	}

	var org *web_api.Organization
	var orgRole *web_api.OrganizationRole
	types.Cast(aOrg, org)
	types.Cast(aRole, orgRole)

	return &web_api.CreateOrganizationResponse{
		Code:    int32(200),
		Success: true,
		Data:    org,
	}, nil

}

func (orgG *webOrganizationGRPCApi) UpdateOrganization(c context.Context, irRequest *web_api.UpdateOrganizationRequest) (*web_api.BaseResponse, error) {
	orgG.logger.Debugf("UpdateOrganization from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		orgG.logger.Errorf("UpdateOrganization from grpc not authenticated")
		return nil, errors.New("unauthenticated request")
	}
	_, err := orgG.organizationService.Update(c, iAuth, irRequest.OrganizationId, irRequest.OrganizationName, irRequest.OrganizationIndustry, irRequest.OrganizationContact)
	if err != nil {
		orgG.logger.Errorf("UpdateOrganization from grpc with erro %v", err)
		return nil, err
	}
	return &web_api.BaseResponse{
		Code:    int32(200),
		Success: true,
	}, nil
}

func (orgG *webOrganizationGRPCApi) GetOrganization(c context.Context, irRequest *web_api.GetOrganizationRequest) (*web_api.GetOrganizationResponse, error) {
	orgG.logger.Debugf("GetOrganization from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}

	aRole, err := orgG.userService.GetOrganizationRole(c, iAuth.GetUserInfo().Id)
	if err != nil {
		orgG.logger.Errorf("userService.GetOrganizationRole from grpc with erro %v", err)
		return nil, err
	}

	aOrg, err := orgG.organizationService.Get(c, aRole.OrganizationId)
	if err != nil {
		orgG.logger.Errorf("organizationService.Get from grpc with erro %v", err)
		return nil, err
	}

	org := &web_api.Organization{}
	orgRole := &web_api.OrganizationRole{}
	_ = types.Cast(&aOrg, org)
	_ = types.Cast(aRole, orgRole)
	return &web_api.GetOrganizationResponse{
		Code:    int32(200),
		Success: true,
		Data:    org,
		Role:    orgRole,
	}, nil
}

func (orgG *webOrganizationGRPCApi) UpdateBillingInformation(c context.Context, irRequest *web_api.UpdateBillingInformationRequest) (*web_api.BaseResponse, error) {
	orgG.logger.Debugf("UpdateBillingInformation from grpc with requestPayload %v, %v", irRequest, c)
	return nil, nil
}
