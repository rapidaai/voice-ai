// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package web_api

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func projectMemberProjection(assignments []*internal_entity.UserProjectRole) []*protos.User {
	members := make([]*protos.User, 0)
	seen := map[uint64]struct{}{}
	for _, assignment := range assignments {
		if _, exists := seen[assignment.UserAuthId]; exists {
			continue
		}
		seen[assignment.UserAuthId] = struct{}{}
		members = append(members, &protos.User{Id: assignment.UserAuthId, Name: assignment.Member.Name, Email: assignment.Member.Email})
	}
	return members
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
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	userID := principal.Subject().ID
	tenant := principal.Tenant()

	prj, err := wProjectApi.projectService.Create(ctx, principal, tenant.OrganizationID, irRequest.GetProjectName(), irRequest.GetProjectDescription())
	if err != nil {
		wProjectApi.logger.Errorf("projectService.Create from grpc with err %v", err)
		return utils.Error[protos.CreateProjectResponse](
			err,
			"Unable to create project for your organization, please try again in sometime")
	}

	_, err = wProjectApi.userService.CreateProjectRole(ctx, principal, userID, internal_service.ProjectAdminRoleDefinitionID, prj.Id, type_enums.RECORD_ACTIVE)
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
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	tenant := principal.Tenant()
	if irRequest.GetProjectId() != tenant.ProjectID {
		return nil, status.Error(codes.PermissionDenied, "project context does not match request")
	}

	prj, err := wProjectApi.projectService.Update(ctx, principal, irRequest.GetProjectId(), irRequest.ProjectName, irRequest.ProjectDescription)
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
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	tenant := principal.Tenant()

	cnt, prjs, err := wProjectApi.projectService.GetAll(ctx, principal,
		tenant.OrganizationID, irRequest.GetCriterias(), irRequest.GetPaginate())
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
		prj.Members = projectMemberProjection(_m)
	}
	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		wProjectApi.logger.Errorf("invalid project count %d: %v", cnt, err)
		return utils.Error[protos.GetAllProjectResponse](
			err,
			"Unable to get the projects, please try again in sometime.",
		)
	}
	return utils.PaginatedSuccess[protos.GetAllProjectResponse, []*protos.Project](totalItem, irRequest.GetPaginate().GetPage(), out)
}

func (wProjectApi *webProjectGRPCApi) GetProject(ctx context.Context, irRequest *protos.GetProjectRequest) (*protos.GetProjectResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	if irRequest.GetProjectId() == 0 {
		return utils.Error[protos.GetProjectResponse](
			errors.New("projectid is not getting passed"),
			"Please select the project to see the details.",
		)
	}
	if irRequest.GetProjectId() != principal.Tenant().ProjectID {
		return nil, status.Error(codes.PermissionDenied, "project context does not match request")
	}

	prj, err := wProjectApi.projectService.Get(ctx, principal, irRequest.GetProjectId())
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

	ot.Members = projectMemberProjection(projectMemebers)
	return utils.Success[protos.GetProjectResponse, *protos.Project](ot)
}

func (wProjectApi *webProjectGRPCApi) AddUserToProjects(ctx context.Context, irRequest *protos.AddUserToProjectsRequest) (*protos.AddUserToProjectsResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
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

	projectIDs := make(map[uint64]struct{}, len(irRequest.GetProjectRoles()))
	selections := make([]internal_service.ProjectRoleSelection, 0, len(irRequest.GetProjectRoles()))
	for _, projectRole := range irRequest.GetProjectRoles() {
		if projectRole == nil || !validator.NonZero(projectRole.GetProjectId()) {
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
		if _, ok := projectIDs[projectRole.GetProjectId()]; ok {
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
		roleDefinitionIDs := append([]uint64(nil), projectRole.GetRoleDefinitionIds()...)
		if !validator.NotEmpty(roleDefinitionIDs) {
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
		roleDefinitionIDSet := make(map[uint64]struct{}, len(roleDefinitionIDs))
		for _, roleDefinitionID := range roleDefinitionIDs {
			_, duplicate := roleDefinitionIDSet[roleDefinitionID]
			if !validator.NonZero(roleDefinitionID) || duplicate {
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
			roleDefinitionIDSet[roleDefinitionID] = struct{}{}
		}
		projectIDs[projectRole.GetProjectId()] = struct{}{}
		selections = append(selections, internal_service.ProjectRoleSelection{ProjectID: projectRole.GetProjectId(), RoleDefinitionIDs: roleDefinitionIDs})
	}
	projects, err := wProjectApi.userService.AddUserToProjects(ctx, principal, irRequest.GetUserId(), tenant.OrganizationID, selections)
	if err != nil {
		var conflict pkg_errors.ProjectMembershipConflictError
		var selectionError pkg_errors.RoleAssignmentSelectionUnavailableError
		switch {
		case errors.As(err, &conflict):
			return &protos.AddUserToProjectsResponse{Code: pkg_errors.AddUserToProjectsUserAlreadyInProject.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AddUserToProjectsUserAlreadyInProject.Code), ErrorMessage: pkg_errors.AddUserToProjectsUserAlreadyInProject.Error, HumanMessage: pkg_errors.AddUserToProjectsUserAlreadyInProject.ErrorMessage}}, nil
		case errors.As(err, &selectionError) && selectionError.Selection == pkg_errors.RoleAssignmentMembership:
			return &protos.AddUserToProjectsResponse{Code: pkg_errors.AddUserToProjectsUserNotInOrganization.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AddUserToProjectsUserNotInOrganization.Code), ErrorMessage: pkg_errors.AddUserToProjectsUserNotInOrganization.Error, HumanMessage: pkg_errors.AddUserToProjectsUserNotInOrganization.ErrorMessage}}, nil
		case errors.As(err, &selectionError) && selectionError.Selection == pkg_errors.RoleAssignmentProject:
			return &protos.AddUserToProjectsResponse{Code: pkg_errors.AddUserToProjectsInvalidProjects.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AddUserToProjectsInvalidProjects.Code), ErrorMessage: pkg_errors.AddUserToProjectsInvalidProjects.Error, HumanMessage: pkg_errors.AddUserToProjectsInvalidProjects.ErrorMessage}}, nil
		case errors.As(err, &selectionError):
			return &protos.AddUserToProjectsResponse{Code: pkg_errors.AddUserToProjectsInvalidProjectRole.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AddUserToProjectsInvalidProjectRole.Code), ErrorMessage: pkg_errors.AddUserToProjectsInvalidProjectRole.Error, HumanMessage: pkg_errors.AddUserToProjectsInvalidProjectRole.ErrorMessage}}, nil
		default:
			wProjectApi.logger.Errorf("unable to create project roles: %v", err)
			return &protos.AddUserToProjectsResponse{Code: pkg_errors.AddUserToProjectsCreateProjectRoles.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AddUserToProjectsCreateProjectRoles.Code), ErrorMessage: pkg_errors.AddUserToProjectsCreateProjectRoles.Error, HumanMessage: pkg_errors.AddUserToProjectsCreateProjectRoles.ErrorMessage}}, nil
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
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
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
	if irRequest.GetProjectId() != tenant.ProjectID {
		return nil, status.Error(codes.PermissionDenied, "project context does not match request")
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
	organizationRoles, err := wProjectApi.userService.GetActiveOrInvitedOrganizationRolesForOrganization(ctx, eUser.GetId(), tenant.OrganizationID)
	if err != nil || len(organizationRoles) == 0 {
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
	if _, err = wProjectApi.projectService.GetAllByOrganization(ctx, principal, tenant.OrganizationID, []uint64{irRequest.GetProjectId()}); err != nil {
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
	projectRoles, err := wProjectApi.userService.GetActiveOrInvitedProjectRoles(ctx, eUser.GetId(), irRequest.GetProjectId())
	if err != nil || len(projectRoles) == 0 {
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

	if err = wProjectApi.userService.ArchiveUserFromProject(ctx, principal, eUser.GetId(), irRequest.GetProjectId()); err != nil {
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
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if irRequest.GetId() != principal.Tenant().ProjectID {
		return nil, status.Error(codes.PermissionDenied, "project context does not match request")
	}
	if _, err := wProjectApi.projectService.Archive(c, principal, irRequest.Id); err != nil {
		wProjectApi.logger.Errorf("DeleteProviderCredential while archieving project")
		return nil, err
	}
	return utils.Success[protos.ArchiveProjectResponse, uint64](irRequest.Id)
}

func (wProjectApi *webProjectGRPCApi) CreateProjectCredential(c context.Context, irRequest *protos.CreateProjectCredentialRequest) (*protos.CreateProjectCredentialResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if !validator.Between(irRequest.GetProjectId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.CreateProjectCredentialResponse{
			Code: pkg_errors.CreateProjectCredentialInvalidProject.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreateProjectCredentialInvalidProject.Code),
				ErrorMessage: pkg_errors.CreateProjectCredentialInvalidProject.Error,
				HumanMessage: pkg_errors.CreateProjectCredentialInvalidProject.ErrorMessage,
			},
		}, nil
	}
	tenant := principal.Tenant()
	if irRequest.GetProjectId() != tenant.ProjectID {
		return nil, status.Error(codes.PermissionDenied, "project context does not match request")
	}
	name := strings.TrimSpace(irRequest.GetName())
	if !validator.NotBlank(name) {
		return &protos.CreateProjectCredentialResponse{
			Code: pkg_errors.CreateProjectCredentialMissingName.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreateProjectCredentialMissingName.Code),
				ErrorMessage: pkg_errors.CreateProjectCredentialMissingName.Error,
				HumanMessage: pkg_errors.CreateProjectCredentialMissingName.ErrorMessage,
			},
		}, nil
	}
	if !validator.Between(irRequest.GetRoleDefinitionId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.CreateProjectCredentialResponse{
			Code: pkg_errors.CreateProjectCredentialInvalidRole.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreateProjectCredentialInvalidRole.Code),
				ErrorMessage: pkg_errors.CreateProjectCredentialInvalidRole.Error,
				HumanMessage: pkg_errors.CreateProjectCredentialInvalidRole.ErrorMessage,
			},
		}, nil
	}
	pc, err := wProjectApi.projectService.CreateCredential(
		c, principal, name, irRequest.GetProjectId(), tenant.OrganizationID, irRequest.GetRoleDefinitionId(),
	)
	if err != nil {
		var selectionError pkg_errors.CredentialSelectionNotFoundError
		var organizationError pkg_errors.CredentialOrganizationInvalidError
		switch {
		case errors.As(err, &selectionError):
			return &protos.CreateProjectCredentialResponse{
				Code: pkg_errors.CreateProjectCredentialSelectionNotFound.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateProjectCredentialSelectionNotFound.Code),
					ErrorMessage: pkg_errors.CreateProjectCredentialSelectionNotFound.Error,
					HumanMessage: pkg_errors.CreateProjectCredentialSelectionNotFound.ErrorMessage,
				},
			}, nil
		case errors.As(err, &organizationError):
			return nil, status.Error(codes.Internal, "project credential creation failed")
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		default:
			return nil, status.Error(codes.Internal, "project credential creation failed")
		}
	}
	out := &protos.ProjectCredential{
		Id: pc.Id, ProjectId: pc.ProjectId, OrganizationId: pc.OrganizationId,
		Name: pc.Name, Key: pc.Key, Status: pc.Status.String(),
		RoleDefinitionId: pc.RoleDefinitionID,
		CreatedDate:      timestamppb.New(time.Time(pc.CreatedDate)),
	}
	if pc.RoleDefinition != nil {
		out.RoleName = pc.RoleDefinition.Name
	}
	if updatedDate := time.Time(pc.UpdatedDate); !updatedDate.IsZero() {
		out.UpdatedDate = timestamppb.New(updatedDate)
	}
	if pc.CreatedActorType != "" && pc.CreatedActorID != 0 {
		out.CreatedActor = &protos.AuditActor{Type: pc.CreatedActorType, Id: pc.CreatedActorID}
	}
	if pc.UpdatedActorType != "" && pc.UpdatedActorID != 0 {
		out.UpdatedActor = &protos.AuditActor{Type: pc.UpdatedActorType, Id: pc.UpdatedActorID}
	}
	return &protos.CreateProjectCredentialResponse{Code: 200, Success: true, Data: out}, nil
}

func (wProjectApi *webProjectGRPCApi) GetAllProjectCredential(c context.Context, irRequest *protos.GetAllProjectCredentialRequest) (*protos.GetAllProjectCredentialResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if !validator.Between(tenant.OrganizationID, uint64(1), uint64(math.MaxInt64)) {
		return nil, status.Error(codes.PermissionDenied, "project API key listing permission denied")
	}
	if !validator.Between(irRequest.GetProjectId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.GetAllProjectCredentialResponse{
			Code: pkg_errors.ListProjectAPIKeyInvalidProject.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode: uint64(pkg_errors.ListProjectAPIKeyInvalidProject.Code), ErrorMessage: pkg_errors.ListProjectAPIKeyInvalidProject.Error,
				HumanMessage: pkg_errors.ListProjectAPIKeyInvalidProject.ErrorMessage,
			},
		}, nil
	}
	if irRequest.GetProjectId() != tenant.ProjectID {
		return nil, status.Error(codes.PermissionDenied, "project context does not match request")
	}
	page := uint32(1)
	var paginate *protos.Paginate
	if irRequest.GetPaginate() != nil {
		page = irRequest.GetPaginate().GetPage()
		pageSize := irRequest.GetPaginate().GetPageSize()
		if page == 0 && pageSize == 0 {
			page, pageSize = 1, 25
		} else if page == 0 || !validator.Between(pageSize, uint32(1), uint32(100)) {
			return &protos.GetAllProjectCredentialResponse{
				Code: pkg_errors.ListProjectAPIKeyInvalidPagination.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode: uint64(pkg_errors.ListProjectAPIKeyInvalidPagination.Code), ErrorMessage: pkg_errors.ListProjectAPIKeyInvalidPagination.Error,
					HumanMessage: pkg_errors.ListProjectAPIKeyInvalidPagination.ErrorMessage,
				},
			}, nil
		}
		paginate = &protos.Paginate{Page: page, PageSize: pageSize}
	}
	if len(irRequest.GetCriterias()) > 2 {
		return &protos.GetAllProjectCredentialResponse{
			Code: pkg_errors.ListProjectAPIKeyInvalidCriteria.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode: uint64(pkg_errors.ListProjectAPIKeyInvalidCriteria.Code), ErrorMessage: pkg_errors.ListProjectAPIKeyInvalidCriteria.Error,
				HumanMessage: pkg_errors.ListProjectAPIKeyInvalidCriteria.ErrorMessage,
			},
		}, nil
	}
	criterias := make([]*protos.Criteria, 0, len(irRequest.GetCriterias()))
	seen := make(map[string]struct{}, len(irRequest.GetCriterias()))
	for _, criteria := range irRequest.GetCriterias() {
		if criteria == nil {
			return &protos.GetAllProjectCredentialResponse{
				Code: pkg_errors.ListProjectAPIKeyInvalidCriteria.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode: uint64(pkg_errors.ListProjectAPIKeyInvalidCriteria.Code), ErrorMessage: pkg_errors.ListProjectAPIKeyInvalidCriteria.Error,
					HumanMessage: pkg_errors.ListProjectAPIKeyInvalidCriteria.ErrorMessage,
				},
			}, nil
		}
		key, value := criteria.GetKey(), strings.TrimSpace(criteria.GetValue())
		_, duplicate := seen[key]
		if duplicate || !validator.OneOf(key, "name", "role") || criteria.GetLogic() != "contains" || strings.ContainsRune(value, '\x00') || !validator.Between(len([]rune(value)), 1, 200) {
			return &protos.GetAllProjectCredentialResponse{
				Code: pkg_errors.ListProjectAPIKeyInvalidCriteria.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode: uint64(pkg_errors.ListProjectAPIKeyInvalidCriteria.Code), ErrorMessage: pkg_errors.ListProjectAPIKeyInvalidCriteria.Error,
					HumanMessage: pkg_errors.ListProjectAPIKeyInvalidCriteria.ErrorMessage,
				},
			}, nil
		}
		seen[key] = struct{}{}
		criterias = append(criterias, &protos.Criteria{Key: key, Logic: criteria.GetLogic(), Value: value})
	}
	count, credentials, err := wProjectApi.projectService.GetAllCredential(c, principal, irRequest.GetProjectId(), tenant.OrganizationID, criterias, paginate)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		default:
			return nil, status.Error(codes.Internal, "project API key listing failed")
		}
	}
	if count < 0 || count > math.MaxUint32 {
		return nil, status.Error(codes.Internal, "project API key listing failed")
	}
	out := make([]*protos.ProjectCredential, 0, len(credentials))
	for _, credential := range credentials {
		if credential == nil {
			continue
		}
		item := &protos.ProjectCredential{
			Id: credential.Id, ProjectId: credential.ProjectId, OrganizationId: credential.OrganizationId,
			Name: credential.Name, Key: credential.Key, Status: credential.Status.String(),
			RoleDefinitionId: credential.RoleDefinitionID,
			CreatedDate:      timestamppb.New(time.Time(credential.CreatedDate)),
		}
		if credential.RoleDefinition != nil {
			item.RoleName = credential.RoleDefinition.Name
		}
		if updatedDate := time.Time(credential.UpdatedDate); !updatedDate.IsZero() {
			item.UpdatedDate = timestamppb.New(updatedDate)
		}
		if credential.CreatedActorType != "" && credential.CreatedActorID != 0 {
			item.CreatedActor = &protos.AuditActor{Type: credential.CreatedActorType, Id: credential.CreatedActorID}
		}
		if credential.UpdatedActorType != "" && credential.UpdatedActorID != 0 {
			item.UpdatedActor = &protos.AuditActor{Type: credential.UpdatedActorType, Id: credential.UpdatedActorID}
		}
		out = append(out, item)
	}
	totalItem, err := utils.Int64ToUint32(count)
	if err != nil {
		return nil, err
	}
	return &protos.GetAllProjectCredentialResponse{
		Code: 200, Success: true, Data: out,
		Paginated: &protos.Paginated{CurrentPage: page, TotalItem: totalItem},
	}, nil
}

func (wProjectApi *webProjectGRPCApi) RevokeProjectCredential(c context.Context, irRequest *protos.RevokeProjectCredentialRequest) (*protos.RevokeProjectCredentialResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if !validator.AllNonZero(tenant.OrganizationID, tenant.ProjectID) {
		return nil, status.Error(codes.PermissionDenied, "project API key revocation permission denied")
	}
	if !validator.Between(irRequest.GetProjectId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.RevokeProjectCredentialResponse{
			Code: pkg_errors.RevokeProjectAPIKeyInvalidProject.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode: uint64(pkg_errors.RevokeProjectAPIKeyInvalidProject.Code), ErrorMessage: pkg_errors.RevokeProjectAPIKeyInvalidProject.Error,
				HumanMessage: pkg_errors.RevokeProjectAPIKeyInvalidProject.ErrorMessage,
			},
		}, nil
	}
	if irRequest.GetProjectId() != tenant.ProjectID {
		return nil, status.Error(codes.PermissionDenied, "project context does not match request")
	}
	if !validator.Between(irRequest.GetCredentialId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.RevokeProjectCredentialResponse{
			Code: pkg_errors.RevokeProjectAPIKeyInvalidCredential.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode: uint64(pkg_errors.RevokeProjectAPIKeyInvalidCredential.Code), ErrorMessage: pkg_errors.RevokeProjectAPIKeyInvalidCredential.Error,
				HumanMessage: pkg_errors.RevokeProjectAPIKeyInvalidCredential.ErrorMessage,
			},
		}, nil
	}
	revokedID, err := wProjectApi.projectService.RevokeCredential(c, principal, irRequest.GetCredentialId(), irRequest.GetProjectId(), tenant.OrganizationID)
	if err != nil {
		var notFound pkg_errors.ProjectAPIKeyNotFoundError
		switch {
		case errors.As(err, &notFound):
			return &protos.RevokeProjectCredentialResponse{
				Code: pkg_errors.RevokeProjectAPIKeyNotFound.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode: uint64(pkg_errors.RevokeProjectAPIKeyNotFound.Code), ErrorMessage: pkg_errors.RevokeProjectAPIKeyNotFound.Error,
					HumanMessage: pkg_errors.RevokeProjectAPIKeyNotFound.ErrorMessage,
				},
			}, nil
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		default:
			return nil, status.Error(codes.Internal, "project API key revocation failed")
		}
	}
	return &protos.RevokeProjectCredentialResponse{Code: 200, Success: true, Id: revokedID}, nil
}
