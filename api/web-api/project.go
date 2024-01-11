package web_api

import (
	"context"
	"errors"
	"strings"
	"sync"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"

	internal_clients "github.com/lexatic/web-backend/internal/clients"
	integration_client "github.com/lexatic/web-backend/internal/clients/integration"
	internal_organization_service "github.com/lexatic/web-backend/internal/services/organization"
	internal_user_service "github.com/lexatic/web-backend/internal/services/user"

	config "github.com/lexatic/web-backend/config"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_project_service "github.com/lexatic/web-backend/internal/services/project"
	"github.com/lexatic/web-backend/pkg/ciphers"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type webProjectApi struct {
	cfg                 *config.AppConfig
	logger              commons.Logger
	projectService      internal_services.ProjectService
	integrationClient   internal_clients.IntegrationServiceClient
	userService         internal_services.UserService
	organizationService internal_services.OrganizationService
}

type webProjectRPCApi struct {
	webProjectApi
}

type webProjectGRPCApi struct {
	webProjectApi
}

func NewProjectRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) *webProjectRPCApi {
	return &webProjectRPCApi{
		webProjectApi{
			cfg:               config,
			logger:            logger,
			projectService:    internal_project_service.NewProjectService(logger, postgres),
			integrationClient: integration_client.NewIntegrationServiceClientGRPC(config, logger),
		},
	}
}

func NewProjectGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.ProjectServiceServer {
	return &webProjectGRPCApi{
		webProjectApi{
			cfg:                 config,
			logger:              logger,
			projectService:      internal_project_service.NewProjectService(logger, postgres),
			userService:         internal_user_service.NewUserService(logger, postgres),
			integrationClient:   integration_client.NewIntegrationServiceClientGRPC(config, logger),
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
		},
	}
}

func (wProjectApi *webProjectGRPCApi) CreateProject(ctx context.Context, irRequest *web_api.CreateProjectRequest) (*web_api.CreateProjectResponse, error) {
	wProjectApi.logger.Debugf("CreateProject from grpc with requestPayload %v, %v", irRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("CreateProject from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}
	currentOrgRole := iAuth.GetOrganizationRole()
	if currentOrgRole == nil {
		wProjectApi.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return &web_api.CreateProjectResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: "You cannot create a project when you are not part of any organization.",
				HumanMessage: "Please create organization before creating a project.",
			}}, nil
	}

	prj, err := wProjectApi.projectService.Create(ctx, iAuth, iAuth.GetOrganizationRole().OrganizationId, irRequest.GetProjectName(), irRequest.GetProjectDescription())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Create from grpc with err %v", err)
		return &web_api.CreateProjectResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create project for your organization, please try again in sometime.",
			}}, nil
	}

	_, err = wProjectApi.userService.CreateProjectRole(ctx, iAuth, iAuth.GetUserInfo().Id, "admin", prj.Id, "active")
	if err != nil {
		wProjectApi.logger.Errorf("userService.CreateProjectRole from grpc with err %v", err)
		return &web_api.CreateProjectResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create project role for you, please try again in sometime.",
			}}, nil
	}
	ot := web_api.Project{}
	err = types.Cast(prj, &ot)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project to proto object %v", err)
	}
	return &web_api.CreateProjectResponse{
		Success: true,
		Code:    200,
		Data:    &ot,
	}, nil
}

/*
update project request
*/
func (wProjectApi *webProjectGRPCApi) UpdateProject(ctx context.Context, irRequest *web_api.UpdateProjectRequest) (*web_api.UpdateProjectResponse, error) {
	wProjectApi.logger.Debugf("UpdateProject from grpc with requestPayload %v, %v", irRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("UpdateProject from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	currentOrgRole := iAuth.GetOrganizationRole()
	if currentOrgRole == nil {
		wProjectApi.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return &web_api.UpdateProjectResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: "You cannot update a project when you are not part of any organization.",
				HumanMessage: "Please create organization before updating a project.",
			}}, nil
	}

	prj, err := wProjectApi.projectService.Update(ctx, iAuth, irRequest.GetProjectId(), irRequest.ProjectName, irRequest.ProjectDescription)
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Update from grpc with err %v", err)
		return &web_api.UpdateProjectResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to update the project, please try again in sometime.",
			}}, nil
	}

	ot := web_api.Project{}
	err = types.Cast(prj, &ot)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project to proto object %v", err)
	}
	return &web_api.UpdateProjectResponse{
		Success: true,
		Code:    200,
		Data:    &ot,
	}, nil
}
func (wProjectApi *webProjectGRPCApi) GetAllProject(ctx context.Context, irRequest *emptypb.Empty) (*web_api.GetAllProjectResponse, error) {
	wProjectApi.logger.Debugf("GetAllProject from grpc with requestPayload %v, %v", irRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("GetAllProject from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	currentOrgRole := iAuth.GetOrganizationRole()
	if currentOrgRole == nil {
		wProjectApi.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return &web_api.GetAllProjectResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: "You are not part of any active organization.",
				HumanMessage: "Please create organization and try again.",
			}}, nil
	}

	prjs, err := wProjectApi.projectService.GetAll(ctx, iAuth, currentOrgRole.OrganizationId)
	if err != nil {
		wProjectApi.logger.Errorf("projectService.GetAll from grpc with err %v", err)
		return &web_api.GetAllProjectResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to get the projects, please try again in sometime.",
			}}, nil
	}

	out := []*web_api.Project{}
	err = types.Cast(prjs, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project to proto object %v", err)
	}

	for _, prj := range out {
		_m, err := wProjectApi.userService.GetAllActiveProjectMember(ctx, prj.Id)
		if err != nil {
			wProjectApi.logger.Errorf("no member in the project %v with err %v", prj.Id, err)
			continue
		}
		for _, upr := range *_m {
			prj.Members = append(prj.Members, &web_api.ProjectMember{
				Role:  upr.Role,
				Id:    upr.UserAuthId,
				Name:  upr.Member.Name,
				Email: upr.Member.Email,
			})
		}

	}
	return &web_api.GetAllProjectResponse{
		Success: true,
		Code:    200,
		Data:    out,
	}, nil
}

func (wProjectApi *webProjectGRPCApi) GetProject(ctx context.Context, irRequest *web_api.GetProjectRequest) (*web_api.GetProjectResponse, error) {
	wProjectApi.logger.Debugf("GetProject from grpc with requestPayload %v, %v", irRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("GetProject from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	prj, err := wProjectApi.projectService.Get(ctx, iAuth, irRequest.GetProjectId())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Get from grpc with err %v", err)
		return nil, err
	}

	ot := web_api.Project{}
	types.Cast(prj, &ot)
	var projectMemebers *[]internal_gorm.UserProjectRole
	if irRequest.GetGetInActive() {
		projectMemebers, err = wProjectApi.userService.GetAllProjectMembers(ctx, prj.Id)
	} else {
		projectMemebers, err = wProjectApi.userService.GetAllActiveProjectMember(ctx, prj.Id)
	}
	if err != nil {
		wProjectApi.logger.Errorf("userService.GetAllProjectMember from grpc with err %v", err)
		return nil, err
	}

	projectMembers := make([]*web_api.ProjectMember, len(*projectMemebers))
	for idx, upr := range *projectMemebers {
		projectMembers[idx] = &web_api.ProjectMember{
			Role:  upr.Role,
			Id:    upr.UserAuthId,
			Name:  upr.Member.Name,
			Email: upr.Member.Email,
		}
	}

	ot.Members = projectMembers
	return &web_api.GetProjectResponse{
		Success: true,
		Code:    200,
		Data:    &ot,
	}, nil
}

func (wProjectApi *webProjectGRPCApi) AddUserToProject(ctx context.Context, auth types.Principle, email string, userId uint64, status, role string, projectIds []uint64) (*web_api.AddUsersToProjectResponse, error) {
	projectNames := make([]string, len(projectIds))
	projectOut := make([]*web_api.Project, len(projectIds))

	wg := sync.WaitGroup{}
	for idx, pId := range projectIds {
		wg.Add(1)
		go func(auth types.Principle, projectId, userId uint64, status, role string, index int) {
			defer wg.Done()
			p, err := wProjectApi.GetProject(ctx, &web_api.GetProjectRequest{ProjectId: projectId})
			if err != nil {
				return
			}

			projectOut[index] = p.GetData()
			projectNames = append(projectNames, p.GetData().Name)
		}(auth, pId, userId, status, role, idx)
		wProjectApi.userService.CreateProjectRole(ctx, auth, userId, role, pId, status)
	}
	wg.Wait()

	// sending email
	_, err := wProjectApi.integrationClient.InviteMemberEmail(ctx, auth.GetUserInfo().Id, "", email, auth.GetOrganizationRole().OrganizationName, strings.Join(projectNames[:], ","), auth.GetUserInfo().Name)
	if err != nil {
		wProjectApi.logger.Errorf("error while sending invite email %v", err)
	}
	return &web_api.AddUsersToProjectResponse{
		Code:    200,
		Success: true,
		Data:    projectOut,
	}, nil
}

func (wProjectApi *webProjectGRPCApi) AddUsersToProject(ctx context.Context, irRequest *web_api.AddUsersToProjectRequest) (*web_api.AddUsersToProjectResponse, error) {
	wProjectApi.logger.Debugf("AddUsersToProject from grpc with requestPayload %v, %v", irRequest, ctx)
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	// get only last project ids
	//
	eUser, err := wProjectApi.userService.Get(ctx, irRequest.Email)
	if err != nil {
		// create a user
		source := "invited-by-other"
		eUser, err := wProjectApi.userService.Create(ctx, "awaited active", irRequest.Email, ciphers.RandomHash("rpd_"), "invited", &source)
		if err != nil {
			wProjectApi.logger.Errorf("unable to create user for invite err %v", err)
			return nil, err
		}
		// , role string, userId uint64, orgnizationId uint64, status string
		_, err = wProjectApi.userService.CreateOrganizationRole(ctx, auth, irRequest.Role, eUser.GetUserInfo().Id, auth.GetOrganizationRole().OrganizationId, "invited")
		if err != nil {
			wProjectApi.logger.Errorf("unable to create organization role err %v", err)
			return nil, err
		}
		return wProjectApi.AddUserToProject(ctx, auth, eUser.GetUserInfo().Email, eUser.GetUserInfo().Id, "invited", irRequest.Role, irRequest.ProjectIds)
	} else {
		org, err := wProjectApi.userService.GetOrganizationRole(ctx, eUser.Id)
		if err == nil {
			if org.GetOrganizationId() != auth.GetOrganizationRole().OrganizationId {
				return nil, errors.New("user is already part of the another organizations")
			}
			return wProjectApi.AddUserToProject(ctx, auth, eUser.Email, eUser.Id, eUser.Status, irRequest.Role, irRequest.ProjectIds)
		}
		_, err = wProjectApi.userService.CreateOrganizationRole(ctx, auth, irRequest.Role, eUser.Id, auth.GetOrganizationRole().OrganizationId, eUser.Status)
		if err != nil {
			wProjectApi.logger.Errorf("unable to create organization role err %v", err)
			return nil, err
		}
		return wProjectApi.AddUserToProject(ctx, auth, eUser.Email, eUser.Id, eUser.Status, irRequest.Role, irRequest.ProjectIds)
	}
}

/*
This api will be for future
if you are reading one of the example that you waste time writing code
*/
func (wProjectApi *webProjectGRPCApi) ArchiveProject(c context.Context, irRequest *web_api.ArchiveProjectRequest) (*web_api.ArchiveProjectResponse, error) {
	wProjectApi.logger.Debugf("ArchiveProjectRequest from grpc with requestPayload %v, %v", irRequest, c)
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("DeleteProviderCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	if _, err := wProjectApi.projectService.Archive(c, auth, irRequest.Id); err != nil {
		wProjectApi.logger.Errorf("DeleteProviderCredential while archieving project")
		return nil, err
	}
	return &web_api.ArchiveProjectResponse{
		Success: true,
		Code:    200,
		Id:      irRequest.Id,
	}, nil
}

func (wProjectApi *webProjectGRPCApi) CreateProjectCredential(c context.Context, irRequest *web_api.CreateProjectCredentialRequest) (*web_api.CreateProjectCredentialResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("CreateProjectCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	// name, key string, projectId, organizationId uint64
	pc, err := wProjectApi.projectService.CreateCredential(c, auth, irRequest.GetName(), irRequest.GetProjectId(), auth.GetOrganizationRole().OrganizationId)
	if err != nil {
		return &web_api.CreateProjectCredentialResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create the project credential, please try again in sometime.",
			}}, nil
	}

	out := &web_api.ProjectCredential{}
	err = types.Cast(pc, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project credential to proto object %v", err)
	}

	return &web_api.CreateProjectCredentialResponse{
		Code:    200,
		Success: true,
		Data:    out,
	}, nil

}

func (wProjectApi *webProjectGRPCApi) GetAllProjectCredential(c context.Context, irRequest *web_api.GetAllProjectCredentialRequest) (*web_api.GetAllProjectCredentialResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("CreateProjectCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	// name, key string, projectId, organizationId uint64
	allProjectCredential, err := wProjectApi.projectService.GetAllCredential(c, auth, irRequest.GetProjectId(), auth.GetOrganizationRole().OrganizationId)
	if err != nil {
		return &web_api.GetAllProjectCredentialResponse{
			Code:    400,
			Success: false,
			Error: &web_api.ProjectError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to get all the project credentials, please try again in sometime.",
			}}, nil
	}

	out := []*web_api.ProjectCredential{}
	err = types.Cast(allProjectCredential, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project credential to proto object %v", err)
	}

	return &web_api.GetAllProjectCredentialResponse{
		Code:    200,
		Success: true,
		Data:    out,
	}, nil

}
