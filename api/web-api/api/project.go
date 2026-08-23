package web_api

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"

	"github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	internal_organization_service "github.com/rapidaai/api/web-api/internal/service/organization"
	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer "github.com/rapidaai/pkg/clients/external/emailer"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type webProjectApi struct {
	cfg                 *config.WebAppConfig
	logger              commons.Logger
	redis               connectors.RedisConnector
	postgres            connectors.PostgresConnector
	projectService      internal_service.ProjectService
	emailerClient       external_clients.Emailer
	userService         internal_service.UserService
	organizationService internal_service.OrganizationService
}

type webProjectRPCApi struct {
	webProjectApi
}

type webProjectGRPCApi struct {
	webProjectApi
}

func NewProjectRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) *webProjectRPCApi {
	return &webProjectRPCApi{
		webProjectApi{
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			projectService: internal_project_service.NewProjectService(logger, postgres),
			emailerClient:  external_emailer.NewEmailer(config.EmailerConfig, logger),
		},
	}
}

func NewProjectGRPC(config *config.WebAppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) protos.ProjectServiceServer {
	return &webProjectGRPCApi{
		webProjectApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			redis:               redis,
			projectService:      internal_project_service.NewProjectService(logger, postgres),
			userService:         internal_user_service.NewUserService(logger, postgres),
			emailerClient:       external_emailer.NewEmailer(config.EmailerConfig, logger),
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
		},
	}
}

func (wProjectApi *webProjectGRPCApi) CreateProject(ctx context.Context, irRequest *protos.CreateProjectRequest) (*protos.CreateProjectResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	userContext, err := iAuth.UserContext()
	if err != nil {
		return nil, err
	}
	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		wProjectApi.logger.Errorf("current org is null, you can't create project without an organization.")
		return utils.Error[protos.CreateProjectResponse](
			errors.New("you cannot create a project when you are not part of any organization"),
			"Please create organization before creating a project.")
	}

	prj, err := wProjectApi.projectService.Create(ctx, iAuth, organizationContext.OrganizationID, irRequest.GetProjectName(), irRequest.GetProjectDescription())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Create from grpc with err %v", err)
		return utils.Error[protos.CreateProjectResponse](
			err,
			"Unable to create project for your organization, please try again in sometime")
	}

	_, err = wProjectApi.userService.CreateProjectRole(ctx, iAuth, userContext.UserID, type_enums.PROJECT_ROLE_ADMIN.String(), prj.Id, type_enums.RECORD_ACTIVE)
	if err != nil {
		wProjectApi.logger.Errorf("userService.CreateProjectRole from grpc with err %v", err)
		return utils.Error[protos.CreateProjectResponse](
			err, "Unable to create project role for you, please try again in sometime")
	}
	ot := &protos.Project{}
	err = utils.Cast(prj, ot)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project to proto object %v", err)
	}
	return utils.Success[protos.CreateProjectResponse, *protos.Project](ot)
}

/*
update project request
*/
func (wProjectApi *webProjectGRPCApi) UpdateProject(ctx context.Context, irRequest *protos.UpdateProjectRequest) (*protos.UpdateProjectResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	if _, err := iAuth.OrganizationContext(); err != nil {
		wProjectApi.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return utils.Error[protos.UpdateProjectResponse](
			errors.New("you cannot update a project when you are not part of any organization"),
			"Please create organization before updating a project.")
	}

	prj, err := wProjectApi.projectService.Update(ctx, iAuth, irRequest.GetProjectId(), irRequest.ProjectName, irRequest.ProjectDescription)
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Update from grpc with err %v", err)
		return utils.Error[protos.UpdateProjectResponse](err,
			"Unable to update the project, please try again in sometime.")
	}

	ot := &protos.Project{}
	err = utils.Cast(prj, ot)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project to proto object %v", err)
	}

	return utils.Success[protos.UpdateProjectResponse, *protos.Project](ot)
}
func (wProjectApi *webProjectGRPCApi) GetAllProject(ctx context.Context, irRequest *protos.GetAllProjectRequest) (*protos.GetAllProjectResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		wProjectApi.logger.Errorf("current org is not null, you can't create multiple organization at same time.")
		return utils.Error[protos.GetAllProjectResponse](
			errors.New("you are not part of any active organization"),
			"Please create organization and try again.",
		)
	}

	cnt, prjs, err := wProjectApi.projectService.GetAll(ctx, iAuth,
		organizationContext.OrganizationID, irRequest.GetCriterias(), irRequest.GetPaginate())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.GetAll from grpc with err %v", err)
		return utils.Error[protos.GetAllProjectResponse](
			err,
			"Unable to get the projects, please try again in sometime.",
		)
	}

	out := []*protos.Project{}
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
		for _, upr := range _m {
			prj.Members = append(prj.Members, &protos.User{
				Role:  upr.Role,
				Id:    upr.UserAuthId,
				Name:  upr.Member.Name,
				Email: upr.Member.Email,
			})
		}
	}
	return utils.PaginatedSuccess[protos.GetAllProjectResponse, []*protos.Project](uint32(cnt), irRequest.GetPaginate().GetPage(), out)
}

func (wProjectApi *webProjectGRPCApi) GetProject(ctx context.Context, irRequest *protos.GetProjectRequest) (*protos.GetProjectResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	if irRequest.GetProjectId() == 0 {
		return utils.Error[protos.GetProjectResponse](
			errors.New("projectid is not getting passed"),
			"Please select the project to see the details.",
		)
	}

	prj, err := wProjectApi.projectService.Get(ctx, iAuth, irRequest.GetProjectId())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Get from grpc with err %v", err)
		return utils.Error[protos.GetProjectResponse](
			err,
			"Please select the project to see the details.",
		)
	}

	ot := &protos.Project{}
	utils.Cast(prj, ot)
	var projectMemebers []*internal_entity.UserProjectRole
	projectMemebers, err = wProjectApi.userService.GetAllActiveProjectMember(ctx, prj.Id)
	if err != nil {
		wProjectApi.logger.Errorf("userService.GetAllProjectMember from grpc with err %v", err)
		return nil, err
	}

	projectMembers := make([]*protos.User, len(projectMemebers))
	for idx, upr := range projectMemebers {
		projectMembers[idx] = &protos.User{
			Role:  upr.Role,
			Id:    upr.UserAuthId,
			Name:  upr.Member.Name,
			Email: upr.Member.Email,
		}
	}

	ot.Members = projectMembers
	return utils.Success[protos.GetProjectResponse, *protos.Project](ot)
}

func (wProjectApi *webProjectGRPCApi) AddUserToProjects(ctx context.Context, irRequest *protos.AddUserToProjectsRequest) (*protos.AddUserToProjectsResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	userContext, err := iAuth.UserContext()
	if err != nil {
		return nil, err
	}
	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		return nil, err
	}
	currentOrgRole, err := wProjectApi.userService.GetActiveOrInvitedOrganizationRoleForOrganization(ctx, userContext.UserID, organizationContext.OrganizationID)
	if err != nil || currentOrgRole == nil {
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsMissingOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsMissingOrganization.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsMissingOrganization.Error,
				HumanMessage: pkg_errors.AddUserToProjectsMissingOrganization.ErrorMessage,
			},
		}, nil
	}
	if !validator.OneOf(currentOrgRole.Role, type_enums.ORGANIZATION_ROLE_OWNER.String(), type_enums.ORGANIZATION_ROLE_ADMIN.String()) {
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsUnauthorized.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsUnauthorized.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsUnauthorized.Error,
				HumanMessage: pkg_errors.AddUserToProjectsUnauthorized.ErrorMessage,
			},
		}, nil
	}
	if !validator.NonZero(irRequest.GetUserId()) {
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsInvalidUser.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsInvalidUser.Error,
				HumanMessage: pkg_errors.AddUserToProjectsInvalidUser.ErrorMessage,
			},
		}, nil
	}
	if !validator.NotEmpty(irRequest.GetProjectRoles()) {
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsMissingProjectRoles.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsMissingProjectRoles.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsMissingProjectRoles.Error,
				HumanMessage: pkg_errors.AddUserToProjectsMissingProjectRoles.ErrorMessage,
			},
		}, nil
	}

	eUser, err := wProjectApi.userService.GetUser(ctx, irRequest.GetUserId())
	if err != nil {
		wProjectApi.logger.Errorf("unable to get user for project assignment err %v", err)
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsInvalidUser.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsInvalidUser.Error,
				HumanMessage: pkg_errors.AddUserToProjectsInvalidUser.ErrorMessage,
			},
		}, nil
	}
	org, err := wProjectApi.userService.GetActiveOrInvitedOrganizationRole(ctx, eUser.GetId())
	if err != nil || org.GetOrganizationId() != currentOrgRole.OrganizationId {
		if err != nil {
			wProjectApi.logger.Errorf("unable to get organization role for project assignment err %v", err)
		}
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsUserNotInOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsUserNotInOrganization.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsUserNotInOrganization.Error,
				HumanMessage: pkg_errors.AddUserToProjectsUserNotInOrganization.ErrorMessage,
			},
		}, nil
	}

	projectIds := make([]uint64, 0, len(irRequest.GetProjectRoles()))
	projectRoles := map[uint64]string{}
	for _, projectRole := range irRequest.GetProjectRoles() {
		if !validator.NonZero(projectRole.GetProjectId()) {
			return &protos.AddUserToProjectsResponse{
				Code:    pkg_errors.AddUserToProjectsInvalidProjects.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.AddUserToProjectsInvalidProjects.Code),
					ErrorMessage: pkg_errors.AddUserToProjectsInvalidProjects.Error,
					HumanMessage: pkg_errors.AddUserToProjectsInvalidProjects.ErrorMessage,
				},
			}, nil
		}
		if _, ok := projectRoles[projectRole.GetProjectId()]; ok {
			return &protos.AddUserToProjectsResponse{
				Code:    pkg_errors.AddUserToProjectsDuplicateProject.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.AddUserToProjectsDuplicateProject.Code),
					ErrorMessage: pkg_errors.AddUserToProjectsDuplicateProject.Error,
					HumanMessage: pkg_errors.AddUserToProjectsDuplicateProject.ErrorMessage,
				},
			}, nil
		}
		if !validator.OneOf(projectRole.GetProjectRole(), type_enums.PROJECT_ROLE_SUPER_ADMIN.String(), type_enums.PROJECT_ROLE_ADMIN.String(), type_enums.PROJECT_ROLE_WRITER.String(), type_enums.PROJECT_ROLE_READER.String()) {
			return &protos.AddUserToProjectsResponse{
				Code:    pkg_errors.AddUserToProjectsInvalidProjectRole.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.AddUserToProjectsInvalidProjectRole.Code),
					ErrorMessage: pkg_errors.AddUserToProjectsInvalidProjectRole.Error,
					HumanMessage: pkg_errors.AddUserToProjectsInvalidProjectRole.ErrorMessage,
				},
			}, nil
		}
		projectIds = append(projectIds, projectRole.GetProjectId())
		projectRoles[projectRole.GetProjectId()] = projectRole.GetProjectRole()
	}

	projects, err := wProjectApi.projectService.GetAllByOrganization(ctx, iAuth, currentOrgRole.OrganizationId, projectIds)
	if err != nil {
		wProjectApi.logger.Errorf("projectService.GetAllByOrganization from grpc with err %v", err)
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsInvalidProjects.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsInvalidProjects.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsInvalidProjects.Error,
				HumanMessage: pkg_errors.AddUserToProjectsInvalidProjects.ErrorMessage,
			},
		}, nil
	}
	existingProjectRoles, err := wProjectApi.userService.GetProjectRolesForUsers(ctx, projectIds, []uint64{eUser.Id})
	if err != nil {
		wProjectApi.logger.Errorf("unable to get existing project roles for assignment err %v", err)
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsCreateProjectRoles.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsCreateProjectRoles.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsCreateProjectRoles.Error,
				HumanMessage: pkg_errors.AddUserToProjectsCreateProjectRoles.ErrorMessage,
			},
		}, nil
	}
	if len(existingProjectRoles) > 0 {
		return &protos.AddUserToProjectsResponse{
			Code:    pkg_errors.AddUserToProjectsUserAlreadyInProject.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.AddUserToProjectsUserAlreadyInProject.Code),
				ErrorMessage: pkg_errors.AddUserToProjectsUserAlreadyInProject.Error,
				HumanMessage: pkg_errors.AddUserToProjectsUserAlreadyInProject.ErrorMessage,
			},
		}, nil
	}

	for _, projectRole := range irRequest.GetProjectRoles() {
		_, err = wProjectApi.userService.CreateProjectRole(ctx, iAuth, eUser.Id, projectRole.GetProjectRole(), projectRole.GetProjectId(), org.Status)
		if err != nil {
			wProjectApi.logger.Errorf("unable to create project role err %v", err)
			return &protos.AddUserToProjectsResponse{
				Code:    pkg_errors.AddUserToProjectsCreateProjectRoles.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.AddUserToProjectsCreateProjectRoles.Code),
					ErrorMessage: pkg_errors.AddUserToProjectsCreateProjectRoles.Error,
					HumanMessage: pkg_errors.AddUserToProjectsCreateProjectRoles.ErrorMessage,
				},
			}, nil
		}
	}

	out := make([]*protos.Project, 0, len(projects))
	for _, project := range projects {
		out = append(out, &protos.Project{
			Id:          project.Id,
			Name:        project.Name,
			Description: project.Description,
			Status:      project.Status.String(),
			CreatedDate: timestamppb.New(time.Time(project.CreatedDate)),
		})
	}
	return &protos.AddUserToProjectsResponse{
		Code:    200,
		Success: true,
		Data:    out,
	}, nil
}

func (wProjectApi *webProjectGRPCApi) DeleteUserFromProject(ctx context.Context, irRequest *protos.DeleteUserFromProjectRequest) (*protos.DeleteUserFromProjectResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	userContext, err := iAuth.UserContext()
	if err != nil {
		return nil, err
	}
	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		return nil, err
	}
	currentOrgRole, err := wProjectApi.userService.GetActiveOrInvitedOrganizationRoleForOrganization(ctx, userContext.UserID, organizationContext.OrganizationID)
	if err != nil || currentOrgRole == nil {
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectMissingOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectMissingOrganization.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectMissingOrganization.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectMissingOrganization.ErrorMessage,
			},
		}, nil
	}
	if !validator.OneOf(currentOrgRole.Role, type_enums.ORGANIZATION_ROLE_OWNER.String(), type_enums.ORGANIZATION_ROLE_ADMIN.String()) {
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectUnauthorized.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectUnauthorized.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectUnauthorized.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectUnauthorized.ErrorMessage,
			},
		}, nil
	}
	if !validator.NonZero(irRequest.GetUserId()) {
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectInvalidUser.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectInvalidUser.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectInvalidUser.ErrorMessage,
			},
		}, nil
	}
	if !validator.NonZero(irRequest.GetProjectId()) {
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectInvalidProject.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectInvalidProject.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectInvalidProject.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectInvalidProject.ErrorMessage,
			},
		}, nil
	}

	eUser, err := wProjectApi.userService.GetUser(ctx, irRequest.GetUserId())
	if err != nil {
		wProjectApi.logger.Errorf("unable to get user for project delete err %v", err)
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectInvalidUser.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectInvalidUser.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectInvalidUser.ErrorMessage,
			},
		}, nil
	}
	org, err := wProjectApi.userService.GetAnyOrganizationRole(ctx, eUser.GetId())
	if err != nil || org.GetOrganizationId() != currentOrgRole.OrganizationId || !validator.OneOf(org.Status.String(), type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_INVITED.String()) {
		if err != nil {
			wProjectApi.logger.Errorf("unable to get organization role for project delete err %v", err)
		}
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectUserNotInOrg.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectUserNotInOrg.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectUserNotInOrg.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectUserNotInOrg.ErrorMessage,
			},
		}, nil
	}
	if _, err = wProjectApi.projectService.GetAllByOrganization(ctx, iAuth, currentOrgRole.OrganizationId, []uint64{irRequest.GetProjectId()}); err != nil {
		wProjectApi.logger.Errorf("projectService.GetAllByOrganization from grpc with err %v", err)
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectInvalidProject.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectInvalidProject.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectInvalidProject.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectInvalidProject.ErrorMessage,
			},
		}, nil
	}
	projectRole, err := wProjectApi.userService.GetActiveOrInvitedProjectRole(ctx, eUser.GetId(), irRequest.GetProjectId())
	if err != nil || !validator.OneOf(projectRole.Status.String(), type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_INVITED.String()) {
		if err != nil {
			wProjectApi.logger.Errorf("unable to get project role for project delete err %v", err)
		}
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectUserNotInProject.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectUserNotInProject.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectUserNotInProject.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectUserNotInProject.ErrorMessage,
			},
		}, nil
	}

	if err = wProjectApi.userService.ArchiveUserFromProject(ctx, iAuth, eUser.GetId(), irRequest.GetProjectId()); err != nil {
		wProjectApi.logger.Errorf("unable to archive user from project err %v", err)
		return &protos.DeleteUserFromProjectResponse{
			Code:    pkg_errors.DeleteUserFromProjectArchiveRole.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromProjectArchiveRole.Code),
				ErrorMessage: pkg_errors.DeleteUserFromProjectArchiveRole.Error,
				HumanMessage: pkg_errors.DeleteUserFromProjectArchiveRole.ErrorMessage,
			},
		}, nil
	}

	return &protos.DeleteUserFromProjectResponse{
		Code:    200,
		Success: true,
		Id:      eUser.GetId(),
	}, nil
}

/*
This api will be for future
if you are reading one of the example that you waste time writing code
*/
func (wProjectApi *webProjectGRPCApi) ArchiveProject(c context.Context, irRequest *protos.ArchiveProjectRequest) (*protos.ArchiveProjectResponse, error) {
	wProjectApi.logger.Debugf("ArchiveProjectRequest from grpc with requestPayload %v, %v", irRequest, c)
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	if _, err := wProjectApi.projectService.Archive(c, iAuth, irRequest.Id); err != nil {
		wProjectApi.logger.Errorf("DeleteProviderCredential while archieving project")
		return nil, err
	}
	return utils.Success[protos.ArchiveProjectResponse, uint64](irRequest.Id)
}

func (wProjectApi *webProjectGRPCApi) CreateProjectCredential(c context.Context, irRequest *protos.CreateProjectCredentialRequest) (*protos.CreateProjectCredentialResponse, error) {
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	// name, key string, projectId, organizationId uint64
	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		return nil, err
	}
	pc, err := wProjectApi.projectService.CreateCredential(c, iAuth, irRequest.GetName(), irRequest.GetProjectId(), organizationContext.OrganizationID)
	if err != nil {
		return utils.Error[protos.CreateProjectCredentialResponse](
			err,
			"Unable to create the project credential, please try again in sometime.",
		)
	}
	out := &protos.ProjectCredential{}
	err = utils.Cast(pc, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project credential to proto object %v", err)
	}
	return utils.Success[protos.CreateProjectCredentialResponse, *protos.ProjectCredential](out)
}

func (wProjectApi *webProjectGRPCApi) GetAllProjectCredential(c context.Context, irRequest *protos.GetAllProjectCredentialRequest) (*protos.GetAllProjectCredentialResponse, error) {
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	// name, key string, projectId, organizationId uint64
	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		return nil, err
	}
	cnt, allProjectCredential, err := wProjectApi.projectService.
		GetAllCredential(
			c, iAuth,
			irRequest.GetProjectId(),
			organizationContext.OrganizationID,
			irRequest.GetCriterias(), irRequest.GetPaginate())
	if err != nil {
		return utils.Error[protos.GetAllProjectCredentialResponse](
			err,
			"Unable to get all the project credentials, please try again in sometime.",
		)
	}

	out := []*protos.ProjectCredential{}
	err = utils.Cast(allProjectCredential, &out)
	if err != nil {
		wProjectApi.logger.Errorf("unable to cast project credential to proto object %v", err)
	}

	return utils.PaginatedSuccess[protos.GetAllProjectCredentialResponse, []*protos.ProjectCredential](
		uint32(cnt),
		irRequest.GetPaginate().GetPage(),
		out)
}
