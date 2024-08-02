package web_api

import (
	"context"
	"errors"
	"strings"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"

	internal_organization_service "github.com/lexatic/web-backend/internal/services/organization"
	internal_user_service "github.com/lexatic/web-backend/internal/services/user"
	integration_client "github.com/lexatic/web-backend/pkg/clients/integration"
	"github.com/lexatic/web-backend/pkg/utils"

	config "github.com/lexatic/web-backend/config"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_project_service "github.com/lexatic/web-backend/internal/services/project"
	"github.com/lexatic/web-backend/pkg/ciphers"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type webProjectApi struct {
	cfg                 *config.AppConfig
	logger              commons.Logger
	redis               connectors.RedisConnector
	postgres            connectors.PostgresConnector
	projectService      internal_services.ProjectService
	sendgridClient      integration_client.SendgridServiceClient
	userService         internal_services.UserService
	organizationService internal_services.OrganizationService
}

type webProjectRPCApi struct {
	webProjectApi
}

type webProjectGRPCApi struct {
	webProjectApi
}

func NewProjectRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) *webProjectRPCApi {
	return &webProjectRPCApi{
		webProjectApi{
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			projectService: internal_project_service.NewProjectService(logger, postgres),
			sendgridClient: integration_client.NewSendgridServiceClientGRPC(config, logger),
		},
	}
}

func NewProjectGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.ProjectServiceServer {
	return &webProjectGRPCApi{
		webProjectApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			redis:               redis,
			projectService:      internal_project_service.NewProjectService(logger, postgres),
			userService:         internal_user_service.NewUserService(logger, postgres),
			sendgridClient:      integration_client.NewSendgridServiceClientGRPC(config, logger),
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
		return utils.Error[web_api.CreateProjectResponse](
			errors.New("you cannot create a project when you are not part of any organization"),
			"Please create organization before creating a project.")
	}

	prj, err := wProjectApi.projectService.Create(ctx, iAuth, iAuth.GetOrganizationRole().OrganizationId, irRequest.GetProjectName(), irRequest.GetProjectDescription())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Create from grpc with err %v", err)
		return utils.Error[web_api.CreateProjectResponse](
			err,
			"Unable to create project for your organization, please try again in sometime")
	}

	_, err = wProjectApi.userService.CreateProjectRole(ctx, iAuth, iAuth.GetUserInfo().Id, "admin", prj.Id, "active")
	if err != nil {
		wProjectApi.logger.Errorf("userService.CreateProjectRole from grpc with err %v", err)
		return utils.Error[web_api.CreateProjectResponse](
			err, "Unable to create project role for you, please try again in sometime")
	}
	ot := &web_api.Project{}
	err = utils.Cast(prj, ot)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project to proto object %v", err)
	}
	return utils.Success[web_api.CreateProjectResponse, *web_api.Project](ot)
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
		return utils.Error[web_api.UpdateProjectResponse](
			errors.New("you cannot update a project when you are not part of any organization"),
			"Please create organization before updating a project.")

	}

	prj, err := wProjectApi.projectService.Update(ctx, iAuth, irRequest.GetProjectId(), irRequest.ProjectName, irRequest.ProjectDescription)
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Update from grpc with err %v", err)
		return utils.Error[web_api.UpdateProjectResponse](err,
			"Unable to update the project, please try again in sometime.")
	}

	ot := &web_api.Project{}
	err = utils.Cast(prj, ot)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project to proto object %v", err)
	}

	return utils.Success[web_api.UpdateProjectResponse, *web_api.Project](ot)

}
func (wProjectApi *webProjectGRPCApi) GetAllProject(ctx context.Context, irRequest *web_api.GetAllProjectRequest) (*web_api.GetAllProjectResponse, error) {
	wProjectApi.logger.Debugf("GetAllProject from grpc with requestPayload %v, %v", irRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("GetAllProject from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	currentOrgRole := iAuth.GetOrganizationRole()
	if currentOrgRole == nil {
		wProjectApi.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return utils.Error[web_api.GetAllProjectResponse](
			errors.New("you are not part of any active organization"),
			"Please create organization and try again.",
		)
	}

	cnt, prjs, err := wProjectApi.projectService.GetAll(ctx, iAuth,
		currentOrgRole.OrganizationId, irRequest.GetCriterias(), irRequest.GetPaginate())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.GetAll from grpc with err %v", err)
		return utils.Error[web_api.GetAllProjectResponse](
			err,
			"Unable to get the projects, please try again in sometime.",
		)

	}

	out := []*web_api.Project{}
	err = utils.Cast(prjs, &out)
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
			prj.Members = append(prj.Members, &web_api.User{
				Role:  upr.Role,
				Id:    upr.UserAuthId,
				Name:  upr.Member.Name,
				Email: upr.Member.Email,
			})
		}

	}

	return utils.PaginatedSuccess[web_api.GetAllProjectResponse, []*web_api.Project](uint32(cnt), irRequest.GetPaginate().GetPage(), out)
}

func (wProjectApi *webProjectGRPCApi) GetProject(ctx context.Context, irRequest *web_api.GetProjectRequest) (*web_api.GetProjectResponse, error) {
	wProjectApi.logger.Debugf("GetProject from grpc with requestPayload %v, %v", irRequest, ctx)
	iAuth, isAuthenticated := types.GetClaimPrincipleGRPC[*types.ServiceScope](ctx)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("GetProject from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	if irRequest.GetProjectId() == 0 {
		return utils.Error[web_api.GetProjectResponse](
			errors.New("projectid is not getting passed"),
			"Please select the project to see the details.",
		)

	}

	prj, err := wProjectApi.projectService.Get(ctx, iAuth, irRequest.GetProjectId())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Get from grpc with err %v", err)
		return utils.Error[web_api.GetProjectResponse](
			err,
			"Please select the project to see the details.",
		)
	}

	ot := &web_api.Project{}
	utils.Cast(prj, ot)
	var projectMemebers *[]internal_gorm.UserProjectRole
	projectMemebers, err = wProjectApi.userService.GetAllActiveProjectMember(ctx, prj.Id)
	if err != nil {
		wProjectApi.logger.Errorf("userService.GetAllProjectMember from grpc with err %v", err)
		return nil, err
	}

	projectMembers := make([]*web_api.User, len(*projectMemebers))
	for idx, upr := range *projectMemebers {
		projectMembers[idx] = &web_api.User{
			Role:  upr.Role,
			Id:    upr.UserAuthId,
			Name:  upr.Member.Name,
			Email: upr.Member.Email,
		}
	}

	ot.Members = projectMembers
	return utils.Success[web_api.GetProjectResponse, *web_api.Project](ot)

}

func (wProjectApi *webProjectGRPCApi) AddUserToProject(ctx context.Context, auth types.Principle, email string, userId uint64, status, role string, projectIds []uint64) (*web_api.AddUsersToProjectResponse, error) {
	projectNames := make([]string, len(projectIds))
	projectOut := make([]*internal_gorm.Project, len(projectIds))

	for _, projectId := range projectIds {
		p, err := wProjectApi.projectService.Get(ctx, auth, projectId)
		if err != nil {
			wProjectApi.logger.Debugf("inviting a user without having  a project %v", err)
			continue
		}
		wProjectApi.userService.CreateProjectRole(ctx, auth, userId, role, projectId, status)
		projectOut = append(projectOut, p)
		projectNames = append(projectNames, p.Name)
	}

	// sending email
	_, err := wProjectApi.sendgridClient.InviteMemberEmail(ctx, *auth.GetUserId(), "", email, auth.GetOrganizationRole().OrganizationName, strings.Join(projectNames[:], ","), auth.GetUserInfo().Name)
	if err != nil {
		wProjectApi.logger.Errorf("error while sending invite email %v", err)
	}

	out := []*web_api.Project{}
	err = utils.Cast(projectOut, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project credential to proto object %v", err)
	}
	return utils.Success[web_api.AddUsersToProjectResponse, []*web_api.Project](out)

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
			return utils.Error[web_api.AddUsersToProjectResponse](
				err,
				"Unable to create user for invite err.",
			)
		}
		// , role string, userId uint64, orgnizationId uint64, status string
		_, err = wProjectApi.userService.CreateOrganizationRole(ctx, auth, irRequest.Role, eUser.GetUserInfo().Id, auth.GetOrganizationRole().OrganizationId, "invited")
		if err != nil {
			wProjectApi.logger.Errorf("unable to create organization role err %v", err)
			return utils.Error[web_api.AddUsersToProjectResponse](
				err,
				"Unable to create organization role user for invite err.",
			)
		}
		return wProjectApi.AddUserToProject(ctx, auth, eUser.GetUserInfo().Email, eUser.GetUserInfo().Id, "invited", irRequest.Role, irRequest.ProjectIds)
	} else {
		org, err := wProjectApi.userService.GetOrganizationRole(ctx, eUser.Id)
		if err == nil {
			if org.GetOrganizationId() != auth.GetOrganizationRole().OrganizationId {
				return utils.Error[web_api.AddUsersToProjectResponse](
					err,
					"User is already part of the another organizations, please contact us.",
				)
			}
			return wProjectApi.AddUserToProject(ctx, auth, eUser.Email, eUser.Id, eUser.Status, irRequest.Role, irRequest.ProjectIds)
		}
		_, err = wProjectApi.userService.CreateOrganizationRole(ctx, auth, irRequest.GetRole(), eUser.Id, auth.GetOrganizationRole().OrganizationId, eUser.Status)
		if err != nil {
			wProjectApi.logger.Errorf("unable to create organization role err %v", err)
			return utils.Error[web_api.AddUsersToProjectResponse](
				err,
				"Unable to create organization role user for invite err.",
			)
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

	return utils.Success[web_api.ArchiveProjectResponse, uint64](irRequest.Id)
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
		return utils.Error[web_api.CreateProjectCredentialResponse](
			err,
			"Unable to create the project credential, please try again in sometime.",
		)

	}

	out := &web_api.ProjectCredential{}
	err = utils.Cast(pc, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project credential to proto object %v", err)
	}

	return utils.Success[web_api.CreateProjectCredentialResponse, *web_api.ProjectCredential](out)

}

func (wProjectApi *webProjectGRPCApi) GetAllProjectCredential(c context.Context, irRequest *web_api.GetAllProjectCredentialRequest) (*web_api.GetAllProjectCredentialResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		wProjectApi.logger.Errorf("CreateProjectCredential from grpc with unauthenticated request")
		return nil, errors.New("unauthenticated request")
	}

	// name, key string, projectId, organizationId uint64
	cnt, allProjectCredential, err := wProjectApi.projectService.
		GetAllCredential(
			c, auth,
			irRequest.GetProjectId(),
			auth.GetOrganizationRole().OrganizationId,
			irRequest.GetCriterias(), irRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllProjectCredentialResponse](
			err,
			"Unable to get all the project credentials, please try again in sometime.",
		)

	}

	out := []*web_api.ProjectCredential{}
	err = utils.Cast(allProjectCredential, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project credential to proto object %v", err)
	}

	return utils.PaginatedSuccess[web_api.GetAllProjectCredentialResponse, []*web_api.ProjectCredential](
		uint32(cnt),
		irRequest.GetPaginate().GetPage(),
		out)

}
