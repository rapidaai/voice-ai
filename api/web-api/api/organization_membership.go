// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	internal_service "github.com/rapidaai/api/web-api/internal/service"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (orgG *webOrganizationGRPCApi) InviteUserToOrganization(ctx context.Context, irRequest *protos.InviteUserToOrganizationRequest) (*protos.InviteUserToOrganizationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if !validator.Email(irRequest.GetEmail()) {
		return &protos.InviteUserToOrganizationResponse{
			Code: pkg_errors.InviteUserToOrganizationInvalidEmail.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidEmail.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidEmail.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationInvalidEmail.ErrorMessage,
			},
		}, nil
	}
	organizationRoleDefinitionIDs := append([]uint64(nil), irRequest.GetOrganizationRoleDefinitionIds()...)
	if !validator.NotEmpty(organizationRoleDefinitionIDs) {
		return &protos.InviteUserToOrganizationResponse{
			Code: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.ErrorMessage,
			},
		}, nil
	}
	organizationRoleDefinitionIDSet := make(map[uint64]struct{}, len(organizationRoleDefinitionIDs))
	for _, roleDefinitionID := range organizationRoleDefinitionIDs {
		_, duplicate := organizationRoleDefinitionIDSet[roleDefinitionID]
		if !validator.NonZero(roleDefinitionID) || duplicate || roleDefinitionID == internal_service.OrganizationOwnerRoleDefinitionID {
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.ErrorMessage,
				},
			}, nil
		}
		organizationRoleDefinitionIDSet[roleDefinitionID] = struct{}{}
	}
	projectRoleAssignments := irRequest.GetProjectRoles()
	projectSelections := make([]internal_service.ProjectRoleSelection, 0, len(projectRoleAssignments))
	projectIDs := make(map[uint64]struct{}, len(projectRoleAssignments))
	for _, assignment := range projectRoleAssignments {
		if assignment == nil || !validator.NonZero(assignment.GetProjectId()) {
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationInvalidProjects.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjects.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.ErrorMessage,
				},
			}, nil
		}
		if _, duplicate := projectIDs[assignment.GetProjectId()]; duplicate {
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationDuplicateProject.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationDuplicateProject.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationDuplicateProject.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationDuplicateProject.ErrorMessage,
				},
			}, nil
		}
		projectIDs[assignment.GetProjectId()] = struct{}{}

		roleDefinitionIDs := append([]uint64(nil), assignment.GetRoleDefinitionIds()...)
		if !validator.NotEmpty(roleDefinitionIDs) {
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationInvalidProjectRole.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjectRole.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.ErrorMessage,
				},
			}, nil
		}
		roleDefinitionIDSet := make(map[uint64]struct{}, len(roleDefinitionIDs))
		for _, roleDefinitionID := range roleDefinitionIDs {
			_, duplicate := roleDefinitionIDSet[roleDefinitionID]
			if !validator.NonZero(roleDefinitionID) || duplicate {
				return &protos.InviteUserToOrganizationResponse{
					Code: pkg_errors.InviteUserToOrganizationInvalidProjectRole.HTTPStatusCodeInt32(),
					Error: &protos.Error{
						ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjectRole.Code),
						ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.Error,
						HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.ErrorMessage,
					},
				}, nil
			}
			roleDefinitionIDSet[roleDefinitionID] = struct{}{}
		}
		projectSelections = append(projectSelections, internal_service.ProjectRoleSelection{
			ProjectID:         assignment.GetProjectId(),
			RoleDefinitionIDs: roleDefinitionIDs,
		})
	}
	organization, err := orgG.organizationService.Get(ctx, tenant.OrganizationID)
	if err != nil {
		return &protos.InviteUserToOrganizationResponse{
			Code: pkg_errors.InviteUserToOrganizationMissingOrganization.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationMissingOrganization.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationMissingOrganization.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationMissingOrganization.ErrorMessage,
			},
		}, nil
	}
	inviter, err := orgG.userService.GetUser(ctx, principal.Subject().ID)
	if err != nil {
		return &protos.InviteUserToOrganizationResponse{
			Code: pkg_errors.InviteUserToOrganizationCreateUser.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateUser.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationCreateUser.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationCreateUser.ErrorMessage,
			},
		}, nil
	}
	invitedUser, projects, err := orgG.userService.InviteUserToOrganization(ctx, principal, irRequest.GetEmail(), tenant.OrganizationID, organizationRoleDefinitionIDs, projectSelections)
	if err != nil {
		var organizationConflict pkg_errors.OrganizationMembershipConflictError
		var selectionError pkg_errors.RoleAssignmentSelectionUnavailableError
		var organizationPersistenceError pkg_errors.OrganizationRolePersistenceError
		var projectPersistenceError pkg_errors.ProjectRolePersistenceError
		switch {
		case errors.As(err, &organizationConflict) && organizationConflict.SameOrganization:
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.ErrorMessage,
				},
			}, nil
		case errors.As(err, &organizationConflict):
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.ErrorMessage,
				},
			}, nil
		case errors.As(err, &selectionError):
			switch selectionError.Selection {
			case pkg_errors.RoleAssignmentProject:
				return &protos.InviteUserToOrganizationResponse{
					Code: pkg_errors.InviteUserToOrganizationInvalidProjects.HTTPStatusCodeInt32(),
					Error: &protos.Error{
						ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjects.Code),
						ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.Error,
						HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjects.ErrorMessage,
					},
				}, nil
			case pkg_errors.RoleAssignmentProjectRole:
				return &protos.InviteUserToOrganizationResponse{
					Code: pkg_errors.InviteUserToOrganizationInvalidProjectRole.HTTPStatusCodeInt32(),
					Error: &protos.Error{
						ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidProjectRole.Code),
						ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.Error,
						HumanMessage: pkg_errors.InviteUserToOrganizationInvalidProjectRole.ErrorMessage,
					},
				}, nil
			default:
				return &protos.InviteUserToOrganizationResponse{
					Code: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.HTTPStatusCodeInt32(),
					Error: &protos.Error{
						ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Code),
						ErrorMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.Error,
						HumanMessage: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole.ErrorMessage,
					},
				}, nil
			}
		case errors.As(err, &organizationPersistenceError):
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.ErrorMessage,
				},
			}, nil
		case errors.As(err, &projectPersistenceError):
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationCreateProjectRoles.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateProjectRoles.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateProjectRoles.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateProjectRoles.ErrorMessage,
				},
			}, nil
		default:
			orgG.logger.Errorf("unable to persist organization invitation: %v", err)
			return &protos.InviteUserToOrganizationResponse{
				Code: pkg_errors.InviteUserToOrganizationCreateUser.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateUser.Code),
					ErrorMessage: pkg_errors.InviteUserToOrganizationCreateUser.Error,
					HumanMessage: pkg_errors.InviteUserToOrganizationCreateUser.ErrorMessage,
				},
			}, nil
		}
	}
	assignments, err := orgG.userService.GetActiveOrInvitedOrganizationRolesForOrganization(ctx, invitedUser.Id, tenant.OrganizationID)
	if err != nil {
		return &protos.InviteUserToOrganizationResponse{
			Code: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Code),
				ErrorMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.Error,
				HumanMessage: pkg_errors.InviteUserToOrganizationCreateOrganizationRole.ErrorMessage,
			},
		}, nil
	}
	assignedRoles := make([]*protos.AssignedRole, 0, len(assignments))
	for _, assignment := range assignments {
		assignedRoles = append(assignedRoles, &protos.AssignedRole{RoleDefinitionId: assignment.RoleDefinitionID, Key: assignment.RoleDefinition.Key, Name: assignment.RoleDefinition.Name})
	}
	projectNames := make([]string, 0, len(projects))
	for _, project := range projects {
		projectNames = append(projectNames, project.Name)
	}
	inviteURL := fmt.Sprintf("%s/auth/signup?utm_source=invite&utm_param=%d", orgG.cfg.BaseUrl(), tenant.OrganizationID)
	if invitedUser.Status == type_enums.RECORD_ACTIVE {
		inviteURL = fmt.Sprintf("%s/auth/signin?utm_source=invite&utm_param=%d", orgG.cfg.BaseUrl(), tenant.OrganizationID)
	}
	if err = orgG.emailerClient.EmailRichText(
		ctx,
		external_clients.Contact{Email: irRequest.GetEmail()},
		fmt.Sprintf("[RapidaAI] %s has invited you to join the %s organization", inviter.Name, organization.Name),
		external_emailer_template.INVITE_MEMBER_TEMPLATE,
		map[string]string{
			"inviter_name": inviter.Name,
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
			Id:                invitedUser.Id,
			Name:              invitedUser.Name,
			Email:             invitedUser.Email,
			OrganizationRoles: assignedRoles,
			Status:            invitedUser.Status.String(),
			CreatedDate:       timestamppb.New(time.Time(invitedUser.CreatedDate)),
		},
	}, nil
}

func (orgG *webOrganizationGRPCApi) UpdateUserOrganizationRole(ctx context.Context, irRequest *protos.UpdateUserOrganizationRoleRequest) (*protos.UpdateUserOrganizationRoleResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	tenant := principal.Tenant()
	if !validator.NonZero(irRequest.GetUserId()) {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code: pkg_errors.UpdateUserOrganizationRoleInvalidUser.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleInvalidUser.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleInvalidUser.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleInvalidUser.ErrorMessage,
			},
		}, nil
	}
	roleDefinitionIDs := append([]uint64(nil), irRequest.GetRoleDefinitionIds()...)
	if !validator.NotEmpty(roleDefinitionIDs) {
		return &protos.UpdateUserOrganizationRoleResponse{
			Code: pkg_errors.UpdateUserOrganizationRoleInvalidRole.HTTPStatusCodeInt32(),
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleInvalidRole.Code),
				ErrorMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.Error,
				HumanMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.ErrorMessage,
			},
		}, nil
	}
	roleDefinitionIDSet := make(map[uint64]struct{}, len(roleDefinitionIDs))
	for _, roleDefinitionID := range roleDefinitionIDs {
		_, duplicate := roleDefinitionIDSet[roleDefinitionID]
		if !validator.NonZero(roleDefinitionID) || duplicate || roleDefinitionID == internal_service.OrganizationOwnerRoleDefinitionID {
			return &protos.UpdateUserOrganizationRoleResponse{
				Code: pkg_errors.UpdateUserOrganizationRoleInvalidRole.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleInvalidRole.Code),
					ErrorMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.Error,
					HumanMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.ErrorMessage,
				},
			}, nil
		}
		roleDefinitionIDSet[roleDefinitionID] = struct{}{}
	}
	if err := orgG.userService.ReplaceOrganizationRoles(ctx, principal, irRequest.GetUserId(), tenant.OrganizationID, roleDefinitionIDs); err != nil {
		orgG.logger.Errorf("unable to update organization role err %v", err)
		var ownerError pkg_errors.OrganizationOwnerMutationError
		var selectionError pkg_errors.RoleAssignmentSelectionUnavailableError
		switch {
		case errors.As(err, &ownerError):
			return &protos.UpdateUserOrganizationRoleResponse{
				Code: pkg_errors.UpdateUserOrganizationRoleOwner.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleOwner.Code),
					ErrorMessage: pkg_errors.UpdateUserOrganizationRoleOwner.Error,
					HumanMessage: pkg_errors.UpdateUserOrganizationRoleOwner.ErrorMessage,
				},
			}, nil
		case errors.As(err, &selectionError) && selectionError.Selection == pkg_errors.RoleAssignmentMembership:
			return &protos.UpdateUserOrganizationRoleResponse{
				Code: pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.Code),
					ErrorMessage: pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.Error,
					HumanMessage: pkg_errors.UpdateUserOrganizationRoleUserNotInOrg.ErrorMessage,
				},
			}, nil
		case errors.As(err, &selectionError):
			return &protos.UpdateUserOrganizationRoleResponse{
				Code: pkg_errors.UpdateUserOrganizationRoleInvalidRole.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleInvalidRole.Code),
					ErrorMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.Error,
					HumanMessage: pkg_errors.UpdateUserOrganizationRoleInvalidRole.ErrorMessage,
				},
			}, nil
		default:
			return &protos.UpdateUserOrganizationRoleResponse{
				Code: pkg_errors.UpdateUserOrganizationRoleUpdateRole.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.UpdateUserOrganizationRoleUpdateRole.Code),
					ErrorMessage: pkg_errors.UpdateUserOrganizationRoleUpdateRole.Error,
					HumanMessage: pkg_errors.UpdateUserOrganizationRoleUpdateRole.ErrorMessage,
				},
			}, nil
		}
	}

	return &protos.UpdateUserOrganizationRoleResponse{
		Code:    200,
		Success: true,
	}, nil
}

func (orgG *webOrganizationGRPCApi) DeleteUserFromOrganization(ctx context.Context, irRequest *protos.DeleteUserFromOrganizationRequest) (*protos.DeleteUserFromOrganizationResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
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
	organizationRoles, err := orgG.userService.GetActiveOrInvitedOrganizationRolesForOrganization(ctx, eUser.GetId(), tenant.OrganizationID)
	if err != nil || len(organizationRoles) == 0 {
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
	for _, organizationRole := range organizationRoles {
		if organizationRole.RoleDefinitionID != internal_service.OrganizationOwnerRoleDefinitionID {
			continue
		}
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

	if err = orgG.userService.ArchiveUserFromOrganization(ctx, principal, eUser.GetId(), tenant.OrganizationID); err != nil {
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
