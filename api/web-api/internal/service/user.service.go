// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	type_enums "github.com/rapidaai/pkg/types/enums"
	web_api "github.com/rapidaai/protos"
)

const (
	OrganizationOwnerRoleDefinitionID uint64 = 101
	ProjectAdminRoleDefinitionID      uint64 = 109
)

type ProjectRoleSelection struct {
	ProjectID         uint64
	RoleDefinitionIDs []uint64
}

// UserAuthentication is the canonical Web API authentication response source.
type UserAuthentication struct {
	Authentication authentication.Authentication
	Token          internal_entity.UserAuthToken
}

type UserService interface {
	AuthenticateUser(ctx context.Context, email string, password string) (*UserAuthentication, error)
	VerifyPassword(ctx context.Context, email string, password string) error
	CreateAuthentication(ctx context.Context, userId uint64) (*UserAuthentication, error)
	AuthenticationForPrincipal(ctx context.Context, principal authentication.Principal) (*UserAuthentication, error)

	Get(ctx context.Context, email string) (*internal_entity.UserAuth, error)
	GetUser(ctx context.Context, userId uint64) (*internal_entity.UserAuth, error)
	UpdateUser(ctx context.Context, auth authentication.Principal, userId uint64, name *string) (*internal_entity.UserAuth, error)
	UpdateUserStatus(ctx context.Context, auth authentication.Principal, userId uint64, status type_enums.RecordState) error
	UpdatePassword(ctx context.Context, auth authentication.Principal, userId uint64, password string) (*internal_entity.UserAuth, error)
	GetToken(ctx context.Context, tokenType string, token string) (*internal_entity.UserAuthToken, error)
	CreateNewAuthToken(ctx context.Context, auth authentication.Principal, userId uint64) (*internal_entity.UserAuthToken, error)
	Create(ctx context.Context, auth authentication.Principal, name string, email string, password string, status type_enums.RecordState, source *string) (*internal_entity.UserAuth, error)
	CreatePasswordToken(ctx context.Context, userId uint64) (*internal_entity.UserAuthToken, error)
	//
	CreateOrganizationRole(ctx context.Context, auth authentication.Principal, roleDefinitionID uint64, userId uint64, organizationId uint64, status type_enums.RecordState) (*internal_entity.UserOrganizationRole, error)
	CreateProjectRole(ctx context.Context, auth authentication.Principal, userId uint64, roleDefinitionID uint64, projectId uint64, status type_enums.RecordState) (*internal_entity.UserProjectRole, error)
	InviteUserToOrganization(ctx context.Context, auth authentication.Principal, email string, organizationID uint64, organizationRoleDefinitionIDs []uint64, projectRoles []ProjectRoleSelection) (*internal_entity.UserAuth, []*internal_entity.Project, error)
	AddUserToProjects(ctx context.Context, auth authentication.Principal, userID uint64, organizationID uint64, projectRoles []ProjectRoleSelection) ([]*internal_entity.Project, error)
	ReplaceOrganizationRoles(ctx context.Context, auth authentication.Principal, userID uint64, organizationID uint64, roleDefinitionIDs []uint64) error
	HasOrganizationPermissions(ctx context.Context, userID uint64, organizationID uint64, permissionKeys ...string) (bool, error)
	ArchiveUserFromOrganization(ctx context.Context, auth authentication.Principal, userId uint64, organizationId uint64) error
	ArchiveUserFromProject(ctx context.Context, auth authentication.Principal, userId uint64, projectId uint64) error

	//
	ActivateAllProjectRoles(ctx context.Context, userId uint64) error
	ActivateAllOrganizationRole(ctx context.Context, userId uint64) error
	GetAllOrganizationMember(ctx context.Context, organizationId uint64, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_entity.UserOrganizationRole, error)

	GetActiveOrInvitedProjectRoles(ctx context.Context, userId uint64, projectId uint64) ([]*internal_entity.UserProjectRole, error)
	GetOrganizationRoles(ctx context.Context, userId uint64) ([]*internal_entity.UserOrganizationRole, error)
	GetActiveOrInvitedOrganizationRoles(ctx context.Context, userId uint64) ([]*internal_entity.UserOrganizationRole, error)
	GetActiveOrInvitedOrganizationRolesForOrganization(ctx context.Context, userId uint64, organizationId uint64) ([]*internal_entity.UserOrganizationRole, error)
	GetAnyOrganizationRoles(ctx context.Context, userId uint64) ([]*internal_entity.UserOrganizationRole, error)
	Activate(ctx context.Context, Id uint64, name string, source *string) (*internal_entity.UserAuth, error)

	CreateSocial(ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_entity.UserSocial, error)
	GetSocial(ctx context.Context, userId uint64) (*internal_entity.UserSocial, error)

	GetAllActiveProjectMember(ctx context.Context, projectId uint64) ([]*internal_entity.UserProjectRole, error)
	GetAllUserRolesForOrg(ctx context.Context, organizationId uint64) ([]*internal_entity.UserOrganizationRole, error)
	GetProjectRolesForUsers(ctx context.Context, pIds []uint64, uIds []uint64) ([]*internal_entity.UserProjectRole, error)
}
