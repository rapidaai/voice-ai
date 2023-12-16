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

/*
For creation of organization and
*/
func (orgG *webOrganizationGRPCApi) CreateOrganization(c context.Context, irRequest *web_api.CreateOrganizationRequest) (*web_api.CreateOrganizationResponse, error) {
	orgG.logger.Debugf("CreateOrganization from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		orgG.logger.Errorf("unauthenticated request for create organization")
		return nil, errors.New("unauthenticated request")
	}

	// check if he is already part of current organization
	currentOrg := iAuth.GetOrganizationRole()
	if currentOrg != nil {
		orgG.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return &web_api.CreateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &web_api.OrganizationError{
				ErrorCode:    400,
				ErrorMessage: "Alerady part of another organization, you can't be part of multiple organization.",
				HumanMessage: "You are already part of an active organization.",
			}}, nil
	}

	// Creation of organization
	aOrg, err := orgG.organizationService.Create(c, iAuth, irRequest.OrganizationName, irRequest.OrganizationSize, irRequest.OrganizationIndustry)
	if err != nil {
		orgG.logger.Errorf("CreateOrganization from grpc with erro %v", err)
		return &web_api.CreateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &web_api.OrganizationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create organization, please try again.",
			}}, nil
	}

	// creation of organization role
	aRole, err := orgG.userService.CreateOrganizationRole(c, iAuth, "owner", iAuth.GetUserInfo().Id, aOrg.Id, "active")
	if err != nil {
		orgG.logger.Errorf("CreateOrganizationRole from grpc with erro %v", err)
		return &web_api.CreateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &web_api.OrganizationError{
				ErrorCode:    401,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to assign role for your organization.",
			}}, nil
	}

	org := &web_api.Organization{}
	orgRole := &web_api.OrganizationRole{}
	err = types.Cast(aOrg, org)
	if err != nil {
		orgG.logger.Errorf("unable to cast organization to proto org err %v", err)
	}
	err = types.Cast(aRole, orgRole)
	if err != nil {
		orgG.logger.Errorf("unable to cast organization to proto org err %v", err)
	}
	return &web_api.CreateOrganizationResponse{
		Code:    200,
		Success: true,
		Data:    org,
		Role:    orgRole,
	}, nil

}

func (orgG *webOrganizationGRPCApi) UpdateOrganization(c context.Context, irRequest *web_api.UpdateOrganizationRequest) (*web_api.UpdateOrganizationResponse, error) {
	orgG.logger.Debugf("UpdateOrganization from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		orgG.logger.Errorf("UpdateOrganization from grpc not authenticated")
		return nil, errors.New("unauthenticated request")
	}

	// updating organization
	_, err := orgG.organizationService.Update(c, iAuth, irRequest.OrganizationId, irRequest.OrganizationName, irRequest.OrganizationIndustry, irRequest.OrganizationContact)
	if err != nil {
		orgG.logger.Errorf("UpdateOrganization from grpc with erro %v", err)
		return &web_api.UpdateOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &web_api.OrganizationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to update the organization, please try again in sometime.",
			}}, nil
	}
	// response
	return &web_api.UpdateOrganizationResponse{
		Code:    200,
		Success: true,
	}, nil

}

// getting all the organization
func (orgG *webOrganizationGRPCApi) GetOrganization(c context.Context, irRequest *web_api.GetOrganizationRequest) (*web_api.GetOrganizationResponse, error) {
	orgG.logger.Debugf("GetOrganization from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		orgG.logger.Errorf("GetOrganization from grpc not authenticated")
		return nil, errors.New("unauthenticated request")
	}

	aRole, err := orgG.userService.GetOrganizationRole(c, iAuth.GetUserInfo().Id)
	if err != nil {
		orgG.logger.Errorf("userService.GetOrganizationRole from grpc with erro %v", err)
		return &web_api.GetOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &web_api.OrganizationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to find role your organization, please try again later.",
			}}, nil
	}

	aOrg, err := orgG.organizationService.Get(c, aRole.OrganizationId)
	if err != nil {
		orgG.logger.Errorf("organizationService.Get from grpc with erro %v", err)
		return &web_api.GetOrganizationResponse{
			Code:    400,
			Success: false,
			Error: &web_api.OrganizationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to find your organization, please try again later.",
			}}, nil
	}

	org := &web_api.Organization{}
	orgRole := &web_api.OrganizationRole{}
	err = types.Cast(aOrg, org)
	if err != nil {
		orgG.logger.Errorf("unable to cast organization to proto org err %v", err)
	}
	err = types.Cast(aRole, orgRole)
	if err != nil {
		orgG.logger.Errorf("unable to cast organization role to proto org role err %v", err)
	}
	return &web_api.GetOrganizationResponse{
		Code:    200,
		Success: true,
		Data:    org,
		Role:    orgRole,
	}, nil
}

func (orgG *webOrganizationGRPCApi) UpdateBillingInformation(c context.Context, irRequest *web_api.UpdateBillingInformationRequest) (*web_api.BaseResponse, error) {
	orgG.logger.Debugf("UpdateBillingInformation from grpc with requestPayload %v, %v", irRequest, c)
	return nil, nil
}
