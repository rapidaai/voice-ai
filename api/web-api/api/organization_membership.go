package web_api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/ciphers"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func (orgG *webOrganizationGRPCApi) InviteUserToOrganization(ctx context.Context, irRequest *protos.InviteUserToOrganizationRequest) (*protos.InviteUserToOrganizationResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationUnauthenticated.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationUnauthenticated.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.InviteUserToOrganizationUnauthenticated.ErrorMessage)
	}
	currentOrgRole := auth.GetOrganizationRole()
	if currentOrgRole == nil {
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationMissingOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationMissingOrganization.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationMissingOrganization.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationMissingOrganization.ErrorMessage,
			},
		}, nil
	}
	if !validator.OneOf(currentOrgRole.Role, type_enums.ORGANIZATION_ROLE_OWNER.String(), type_enums.ORGANIZATION_ROLE_ADMIN.String()) {
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationUnauthorized.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationUnauthorized.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationUnauthorized.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationUnauthorized.ErrorMessage,
			},
		}, nil
	}

	if !validator.Email(irRequest.GetEmail()) {
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationInvalidEmail.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidEmail.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidEmail.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationInvalidEmail.ErrorMessage,
			},
		}, nil
	}
	if !validator.OneOf(irRequest.GetOrganizationRole(), type_enums.ORGANIZATION_ROLE_ADMIN.String(), type_enums.ORGANIZATION_ROLE_MEMBER.String()) {
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.ErrorMessage,
			},
		}, nil
	}

	projectIds := make([]uint64, 0, len(irRequest.GetProjectRoles()))
	projectRoles := map[uint64]string{}
	for _, projectRole := range irRequest.GetProjectRoles() {
		if !validator.NonZero(projectRole.GetProjectId()) {
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationInvalidProjects.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjects.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.ErrorMessage,
				},
			}, nil
		}
		if _, ok := projectRoles[projectRole.GetProjectId()]; ok {
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationDuplicateProject.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationDuplicateProject.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationDuplicateProject.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationDuplicateProject.ErrorMessage,
				},
			}, nil
		}
		if !validator.OneOf(projectRole.GetProjectRole(), type_enums.PROJECT_ROLE_SUPER_ADMIN.String(), type_enums.PROJECT_ROLE_ADMIN.String(), type_enums.PROJECT_ROLE_WRITER.String(), type_enums.PROJECT_ROLE_READER.String()) {
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationInvalidProjectRole.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjectRole.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.ErrorMessage,
				},
			}, nil
		}
		projectIds = append(projectIds, projectRole.GetProjectId())
		projectRoles[projectRole.GetProjectId()] = projectRole.GetProjectRole()
	}

	var projects []*internal_entity.Project
	if validator.NotEmpty(projectIds) {
		var err error
		projects, err = orgG.projectService.GetAllByOrganization(ctx, auth, currentOrgRole.OrganizationId, projectIds)
		if err != nil {
			orgG.logger.Errorf("projectService.GetAllByOrganization from grpc with err %v", err)
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationInvalidProjects.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjects.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.ErrorMessage,
				},
			}, nil
		}
	}

	projectNames := make([]string, 0, len(projects))
	for _, project := range projects {
		projectNames = append(projectNames, project.Name)
	}

	eUser, err := orgG.userService.Get(ctx, irRequest.GetEmail())
	if err != nil {
		source := "invited-by-other"
		parts := strings.Split(irRequest.GetEmail(), "@")
		ePrinciple, err := orgG.userService.Create(ctx, parts[0], irRequest.GetEmail(), ciphers.RandomHash("rpd_"), type_enums.RECORD_INVITED, &source)
		if err != nil {
			orgG.logger.Errorf("unable to create user for invite err %v", err)
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationCreateUser.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateUser.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateUser.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateUser.ErrorMessage,
				},
			}, nil
		}
		invitedUserID, err := types.RequireUser(ePrinciple)
		if err != nil {
			orgG.logger.Errorf("unable to resolve invited user identity err %v", err)
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationCreateUser.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateUser.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateUser.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateUser.ErrorMessage,
				},
			}, nil
		}

		_, err = orgG.userService.CreateOrganizationRole(ctx, auth, irRequest.GetOrganizationRole(), invitedUserID, currentOrgRole.OrganizationId, type_enums.RECORD_INVITED)
		if err != nil {
			orgG.logger.Errorf("unable to create organization role err %v", err)
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationCreateOrganizationRole.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.ErrorMessage,
				},
			}, nil
		}
		for _, projectRole := range irRequest.GetProjectRoles() {
			_, err = orgG.userService.CreateProjectRole(ctx, auth, invitedUserID, projectRole.GetProjectRole(), projectRole.GetProjectId(), type_enums.RECORD_INVITED)
			if err != nil {
				orgG.logger.Errorf("unable to create project role for invite err %v", err)
				return &protos.InviteUserToOrganizationResponse{
					Code:    pkg_errors.InviteUserToOrganizationCreateProjectRoles.HTTPStatusCodeInt32(),
					Success: false,
					Error: &protos.Error{
						ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateProjectRoles.Code),
						ErrorMessage: pkg_errors.InviteUserToOrganizationCreateProjectRoles.Error,
						HumanMessage: pkg_errors.InviteUserToOrganizationCreateProjectRoles.ErrorMessage,
					},
				}, nil
			}
		}
		if err = orgG.emailerClient.EmailRichText(
			ctx,
			external_clients.Contact{Email: irRequest.GetEmail()},
			fmt.Sprintf("[RapidaAI] %s has invited you to join the %s organization", auth.GetUserInfo().Name, currentOrgRole.OrganizationName),
			external_emailer_template.INVITE_MEMBER_TEMPLATE,
			map[string]string{
				"inviter_name": auth.GetUserInfo().Name,
				"project_name": strings.Join(projectNames, ","),
				"invite_url":   fmt.Sprintf("%s/auth/signup?utm_source=invite&utm_param=%d", orgG.cfg.BaseUrl(), currentOrgRole.OrganizationId),
			},
		); err != nil {
			orgG.logger.Errorf("error while sending invite email %v", err)
		}
		return &protos.InviteUserToOrganizationResponse{
			Code:    200,
			Success: true,
			Data: &protos.User{
				Id:     invitedUserID,
				Name:   ePrinciple.GetUserInfo().Name,
				Email:  irRequest.GetEmail(),
				Role:   irRequest.GetOrganizationRole(),
				Status: type_enums.RECORD_INVITED.String(),
			},
		}, nil
	}

	org, err := orgG.userService.GetActiveOrInvitedOrganizationRole(ctx, eUser.GetId())
	if err == nil && org.GetOrganizationId() != currentOrgRole.OrganizationId {
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.ErrorMessage,
			},
		}, nil
	} else if err == nil {
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.ErrorMessage,
			},
		}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		orgG.logger.Errorf("unable to get organization role for invite err %v", err)
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationCreateOrganizationRole.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.ErrorMessage,
			},
		}, nil
	}

	roleStatus := eUser.Status
	if eUser.Status == type_enums.RECORD_ARCHIEVE {
		roleStatus = type_enums.RECORD_ACTIVE
		if err = orgG.userService.UpdateUserStatus(ctx, auth, eUser.GetId(), roleStatus); err != nil {
			orgG.logger.Errorf("unable to restore archived user for invite err %v", err)
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationCreateUser.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateUser.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateUser.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateUser.ErrorMessage,
				},
			}, nil
		}
	}

	_, err = orgG.userService.CreateOrganizationRole(ctx, auth, irRequest.GetOrganizationRole(), eUser.GetId(), currentOrgRole.OrganizationId, roleStatus)
	if err != nil {
		orgG.logger.Errorf("unable to create organization role err %v", err)
		return &protos.InviteUserToOrganizationResponse{
			Code:    pkg_errors.InviteUserToOrganizationCreateOrganizationRole.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.ErrorMessage,
			},
		}, nil
	}

	for _, projectRole := range irRequest.GetProjectRoles() {
		_, err = orgG.userService.CreateProjectRole(ctx, auth, eUser.Id, projectRole.GetProjectRole(), projectRole.GetProjectId(), roleStatus)
		if err != nil {
			orgG.logger.Errorf("unable to create project role for invite err %v", err)
			return &protos.InviteUserToOrganizationResponse{
				Code:    pkg_errors.InviteUserToOrganizationCreateProjectRoles.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateProjectRoles.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateProjectRoles.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateProjectRoles.ErrorMessage,
				},
			}, nil
		}
	}
	inviteURL := fmt.Sprintf("%s/auth/signup?utm_source=invite&utm_param=%d", orgG.cfg.BaseUrl(), currentOrgRole.OrganizationId)
	if roleStatus == type_enums.RECORD_ACTIVE {
		inviteURL = fmt.Sprintf("%s/auth/signin?utm_source=invite&utm_param=%d", orgG.cfg.BaseUrl(), currentOrgRole.OrganizationId)
	}
	if err = orgG.emailerClient.EmailRichText(
		ctx,
		external_clients.Contact{Email: irRequest.GetEmail()},
		fmt.Sprintf("[RapidaAI] %s has invited you to join the %s organization", auth.GetUserInfo().Name, currentOrgRole.OrganizationName),
		external_emailer_template.INVITE_MEMBER_TEMPLATE,
		map[string]string{
			"inviter_name": auth.GetUserInfo().Name,
			"project_name": strings.Join(projectNames, ","),
			"invite_url":   inviteURL,
		},
	); err != nil {
		orgG.logger.Errorf("error while sending invite email %v", err)
	}
	return &protos.InviteUserToOrganizationResponse{
		Code:    200,
		Success: true,
		Data: &protos.User{
			Id:          eUser.Id,
			Name:        eUser.Name,
			Email:       eUser.Email,
			Role:        irRequest.GetOrganizationRole(),
			Status:      roleStatus.String(),
			CreatedDate: timestamppb.New(time.Time(eUser.CreatedDate)),
		},
	}, nil
}

func (orgG *webOrganizationGRPCApi) UpdateUserOrganizationRole(ctx context.Context, irRequest *protos.UpdateUserOrganizationRoleRequest) (*protos.UpdateUserOrganizationRoleResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleUnauthenticated.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleUnauthenticated.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.UpdateUserOrganizationRoleUnauthenticated.ErrorMessage)
	}

	currentOrgRole := auth.GetOrganizationRole()
	if currentOrgRole == nil {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleMissingOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleMissingOrganization.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleMissingOrganization.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleMissingOrganization.ErrorMessage,
			},
		}, nil
	}
	if !validator.OneOf(currentOrgRole.Role, type_enums.ORGANIZATION_ROLE_OWNER.String(), type_enums.ORGANIZATION_ROLE_ADMIN.String()) {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleUnauthorized.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleUnauthorized.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleUnauthorized.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleUnauthorized.ErrorMessage,
			},
		}, nil
	}

	if !validator.NonZero(irRequest.GetUserId()) {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleInvalidUser.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleInvalidUser.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleInvalidUser.ErrorMessage,
			},
		}, nil
	}
	if !validator.OneOf(irRequest.GetOrganizationRole(), type_enums.ORGANIZATION_ROLE_ADMIN.String(), type_enums.ORGANIZATION_ROLE_MEMBER.String()) {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleInvalidRole.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleInvalidRole.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.ErrorMessage,
			},
		}, nil
	}

	eUser, err := orgG.userService.GetUser(ctx, irRequest.GetUserId())
	if err != nil {
		orgG.logger.Errorf("unable to get user for organization role update err %v", err)
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleInvalidUser.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleInvalidUser.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleInvalidUser.ErrorMessage,
			},
		}, nil
	}

	org, err := orgG.userService.GetActiveOrInvitedOrganizationRoleForOrganization(ctx, eUser.GetId(), currentOrgRole.OrganizationId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.ErrorMessage,
			},
		}, nil
	} else if err != nil {
		orgG.logger.Errorf("unable to get organization role for organization role update err %v", err)
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleUpdateRole.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleUpdateRole.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleUpdateRole.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleUpdateRole.ErrorMessage,
			},
		}, nil
	}
	if strings.EqualFold(org.Role, type_enums.ORGANIZATION_ROLE_OWNER.String()) {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleOwner.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleOwner.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleOwner.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleOwner.ErrorMessage,
			},
		}, nil
	}

	if err = orgG.userService.UpdateOrganizationRole(ctx, auth, eUser.GetId(), currentOrgRole.OrganizationId, irRequest.GetOrganizationRole()); err != nil {
		orgG.logger.Errorf("unable to update organization role err %v", err)
		return &protos.UpdateUserOrganizationRoleResponse{
			Code:    pkg_errors.UpdateUserOrganizationRoleUpdateRole.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleUpdateRole.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleUpdateRole.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleUpdateRole.ErrorMessage,
			},
		}, nil
	}

	return &protos.UpdateUserOrganizationRoleResponse{
		Code:    200,
		Success: true,
	}, nil
}

func (orgG *webOrganizationGRPCApi) DeleteUserFromOrganization(ctx context.Context, irRequest *protos.DeleteUserFromOrganizationRequest) (*protos.DeleteUserFromOrganizationResponse, error) {
	auth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationUnauthenticated.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationUnauthenticated.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.DeleteUserFromOrganizationUnauthenticated.Error)
	}
	currentOrgRole := auth.GetOrganizationRole()
	if currentOrgRole == nil {
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationMissingOrganization.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationMissingOrganization.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationMissingOrganization.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationMissingOrganization.ErrorMessage,
			},
		}, nil
	}
	if !validator.OneOf(currentOrgRole.Role, type_enums.ORGANIZATION_ROLE_OWNER.String(), type_enums.ORGANIZATION_ROLE_ADMIN.String()) {
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationUnauthorized.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationUnauthorized.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationUnauthorized.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationUnauthorized.ErrorMessage,
			},
		}, nil
	}
	if !validator.NonZero(irRequest.GetUserId()) {
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationInvalidUser.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationInvalidUser.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationInvalidUser.ErrorMessage,
			},
		}, nil
	}

	eUser, err := orgG.userService.GetUser(ctx, irRequest.GetUserId())
	if err != nil {
		orgG.logger.Errorf("unable to get user for organization delete err %v", err)
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationInvalidUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationInvalidUser.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationInvalidUser.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationInvalidUser.ErrorMessage,
			},
		}, nil
	}
	org, err := orgG.userService.GetAnyOrganizationRole(ctx, eUser.GetId())
	if err != nil || org.GetOrganizationId() != currentOrgRole.OrganizationId || !validator.OneOf(org.Status.String(), type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_INVITED.String()) {
		if err != nil {
			orgG.logger.Errorf("unable to get organization role for organization delete err %v", err)
		}
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationUserNotInOrg.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationUserNotInOrg.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationUserNotInOrg.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationUserNotInOrg.ErrorMessage,
			},
		}, nil
	}
	if strings.EqualFold(org.Role, type_enums.ORGANIZATION_ROLE_OWNER.String()) {
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationOwner.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationOwner.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationOwner.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationOwner.ErrorMessage,
			},
		}, nil
	}

	if err = orgG.userService.ArchiveUserFromOrganization(ctx, auth, eUser.GetId(), currentOrgRole.OrganizationId); err != nil {
		orgG.logger.Errorf("unable to archive user from organization err %v", err)
		return &protos.DeleteUserFromOrganizationResponse{
			Code:    pkg_errors.DeleteUserFromOrganizationArchiveUser.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.DeleteUserFromOrganizationArchiveUser.Code),
				ErrorMessage: pkg_errors.DeleteUserFromOrganizationArchiveUser.Error,
				HumanMessage: pkg_errors.DeleteUserFromOrganizationArchiveUser.ErrorMessage,
			},
		}, nil
	}

	return &protos.DeleteUserFromOrganizationResponse{
		Code:    200,
		Success: true,
		Id:      eUser.GetId(),
	}, nil
}
