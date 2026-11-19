// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_user_service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	internal_accessmanagement "github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	internal_billing_service "github.com/rapidaai/api/web-api/internal/service/billing"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/ciphers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

// PrincipalAuthenticationService owns canonical User credential and authority reads.
type PrincipalAuthenticationService interface {
	resolver.UserAuthenticationService
}

// NewPrincipalAuthenticationService constructs the canonical User principal source.
func NewPrincipalAuthenticationService(logger commons.Logger, postgres connectors.PostgresConnector) PrincipalAuthenticationService {
	return &userService{logger: logger, postgres: postgres, billingService: internal_billing_service.NewBillingService(postgres)}
}

// ResolveUserCredential binds the exact presented session to one canonical Authentication.
func (service *userService) ResolveUserCredential(ctx context.Context, token string, userID uint64, selectedOrganizationID uint64, selectedProjectID uint64) (authentication.Authentication, error) {
	snapshot, err := service.canonicalUserSnapshot(ctx, &token, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	if err != nil {
		return authentication.Authentication{}, err
	}
	if snapshot.userAuthToken.Status != type_enums.RECORD_ACTIVE || !snapshot.userAuthToken.ExpireAt.After(time.Now()) {
		return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}

	organizationIDs := make(map[uint64]struct{})
	for _, assignment := range snapshot.userOrgRoles {
		if assignment == nil {
			return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
		organizationIDs[assignment.OrganizationId] = struct{}{}
	}
	organizationID := selectedOrganizationID
	if organizationID != 0 {
		if _, exists := organizationIDs[organizationID]; !exists {
			return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	if selectedProjectID != 0 {
		projectOrganizationID := uint64(0)
		for _, assignment := range snapshot.userProjectRoles {
			if assignment != nil && assignment.ProjectId == selectedProjectID {
				projectOrganizationID = assignment.Project.OrganizationId
				break
			}
		}
		if projectOrganizationID == 0 {
			return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
		if _, exists := organizationIDs[projectOrganizationID]; !exists {
			return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
		if organizationID == 0 {
			organizationID = projectOrganizationID
		} else if organizationID != projectOrganizationID {
			return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	if organizationID == 0 && len(organizationIDs) == 1 {
		for candidate := range organizationIDs {
			organizationID = candidate
		}
	}
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: snapshot.user.Id},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: selectedProjectID},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceUserSession,
				Value: strconv.FormatUint(snapshot.userAuthToken.Id, 10),
			},
		},
		nil,
	)
	if err != nil {
		return authentication.Authentication{}, err
	}
	result, err := service.authentication(ctx, snapshot, principal)
	if err != nil {
		return authentication.Authentication{}, err
	}
	return result.Authentication, nil
}

func (service *userService) canonicalUserSnapshot(ctx context.Context, token *string, userID uint64) (*authenticationSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	database := service.postgres.DB(ctx)
	user := internal_entity.UserAuth{}
	if err := database.Where("id = ? AND status = ?", userID, type_enums.RECORD_ACTIVE).Take(&user).Error; err != nil {
		return nil, err
	}
	session := internal_entity.UserAuthToken{}
	if token != nil {
		if err := database.Where("user_auth_id = ? AND token = ? AND token_type = ?", userID, *token, "auth-token").Take(&session).Error; err != nil {
			return nil, err
		}
	}
	organizationRoles, err := service.loadCanonicalUserOrganizationRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	projectRoles, err := service.loadCanonicalUserProjectRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &authenticationSnapshot{
		user: &user, userAuthToken: &session, userOrgRoles: organizationRoles,
		userProjectRoles: projectRoles,
	}, nil
}

func (service *userService) loadCanonicalUserOrganizationRoles(ctx context.Context, userID uint64) ([]*internal_entity.UserOrganizationRole, error) {
	assignments := make([]*internal_entity.UserOrganizationRole, 0)
	err := service.postgres.DB(ctx).Table("user_organization_roles").
		Preload("Organization").
		Preload("RoleDefinition", "status = ?", type_enums.RECORD_ACTIVE).
		Preload("RoleDefinition.RolePermissions", func(query *gorm.DB) *gorm.DB {
			return query.Where("role_permissions.status = ? AND EXISTS (SELECT 1 FROM permission_definitions WHERE permission_definitions.id = role_permissions.permission_definition_id AND permission_definitions.status = ?)", type_enums.RECORD_ACTIVE, type_enums.RECORD_ACTIVE)
		}).
		Preload("RoleDefinition.RolePermissions.PermissionDefinition", "status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN organizations ON organizations.id = user_organization_roles.organization_id AND organizations.status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN role_definitions ON role_definitions.id = user_organization_roles.role_definition_id AND role_definitions.status = ? AND role_definitions.scope_type = ? AND role_definitions.subject_type = ?", type_enums.RECORD_ACTIVE, internal_accessmanagement.ScopeOrganization, internal_accessmanagement.SubjectUser).
		Where("user_organization_roles.user_auth_id = ? AND user_organization_roles.status = ? AND user_organization_roles.role_scope_type = ? AND user_organization_roles.role_subject_type = ?", userID, type_enums.RECORD_ACTIVE, internal_accessmanagement.ScopeOrganization, internal_accessmanagement.SubjectUser).
		Where("(role_definitions.definition_type = ? AND role_definitions.organization_id IS NULL) OR (role_definitions.definition_type = ? AND role_definitions.organization_id = user_organization_roles.organization_id)", internal_accessmanagement.DefinitionBuiltIn, internal_accessmanagement.DefinitionCustom).
		Order("user_organization_roles.organization_id ASC, user_organization_roles.role_definition_id ASC").Find(&assignments).Error
	return assignments, err
}

func (service *userService) loadCanonicalUserProjectRoles(ctx context.Context, userID uint64) ([]*internal_entity.UserProjectRole, error) {
	assignments := make([]*internal_entity.UserProjectRole, 0)
	err := service.postgres.DB(ctx).Table("user_project_roles").
		Preload("Project").
		Preload("RoleDefinition", "status = ?", type_enums.RECORD_ACTIVE).
		Preload("RoleDefinition.RolePermissions", func(query *gorm.DB) *gorm.DB {
			return query.Where("role_permissions.status = ? AND EXISTS (SELECT 1 FROM permission_definitions WHERE permission_definitions.id = role_permissions.permission_definition_id AND permission_definitions.status = ?)", type_enums.RECORD_ACTIVE, type_enums.RECORD_ACTIVE)
		}).
		Preload("RoleDefinition.RolePermissions.PermissionDefinition", "status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN projects ON projects.id = user_project_roles.project_id AND projects.status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN role_definitions ON role_definitions.id = user_project_roles.role_definition_id AND role_definitions.status = ? AND role_definitions.scope_type = ? AND role_definitions.subject_type = ?", type_enums.RECORD_ACTIVE, internal_accessmanagement.ScopeProject, internal_accessmanagement.SubjectUser).
		Where("user_project_roles.user_auth_id = ? AND user_project_roles.status = ? AND user_project_roles.role_scope_type = ? AND user_project_roles.role_subject_type = ?", userID, type_enums.RECORD_ACTIVE, internal_accessmanagement.ScopeProject, internal_accessmanagement.SubjectUser).
		Where("(role_definitions.definition_type = ? AND role_definitions.organization_id IS NULL) OR (role_definitions.definition_type = ? AND role_definitions.organization_id = projects.organization_id)", internal_accessmanagement.DefinitionBuiltIn, internal_accessmanagement.DefinitionCustom).
		Order("user_project_roles.project_id ASC, user_project_roles.role_definition_id ASC").Find(&assignments).Error
	return assignments, err
}

// authenticationFromSnapshot constructs the unified aggregate from one verified User snapshot.
func (service *userService) authenticationFromSnapshot(ctx context.Context, snapshot *authenticationSnapshot) (*internal_services.UserAuthentication, error) {
	if snapshot == nil || snapshot.user == nil || snapshot.userAuthToken == nil ||
		snapshot.userAuthToken.Status != type_enums.RECORD_ACTIVE || snapshot.userAuthToken.TokenType != "auth-token" ||
		!snapshot.userAuthToken.ExpireAt.After(time.Now()) {
		return nil, pkg_errors.AuthenticationInvalidError{}
	}

	organizationIDs := make(map[uint64]struct{})
	for _, assignment := range snapshot.userOrgRoles {
		if assignment == nil {
			return nil, pkg_errors.AuthenticationInvalidError{}
		}
		organizationIDs[assignment.OrganizationId] = struct{}{}
	}
	organizationID := uint64(0)
	if len(organizationIDs) == 1 {
		for candidate := range organizationIDs {
			organizationID = candidate
		}
	}

	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: snapshot.user.Id},
		authentication.Tenant{OrganizationID: organizationID},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceUserSession,
				Value: strconv.FormatUint(snapshot.userAuthToken.Id, 10),
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	return service.authentication(ctx, snapshot, principal)
}

// CreateAuthentication issues one fresh User session and builds Authentication from that exact session.
func (service *userService) CreateAuthentication(ctx context.Context, userID uint64) (*internal_services.UserAuthentication, error) {
	snapshot, err := service.canonicalUserSnapshot(ctx, nil, userID)
	if err != nil {
		return nil, err
	}
	return service.createAuthentication(ctx, snapshot)
}

func (service *userService) createAuthentication(ctx context.Context, snapshot *authenticationSnapshot) (*internal_services.UserAuthentication, error) {
	if snapshot == nil || snapshot.user == nil {
		return nil, pkg_errors.AuthenticationInvalidError{}
	}
	session := &internal_entity.UserAuthToken{
		Audited:    gorm_models.Audited{Id: gorm_generator.ID()},
		UserAuthId: snapshot.user.Id,
		TokenType:  "auth-token",
		Token:      ciphers.Token("at"),
		ExpireAt:   time.Now().Add(200 * time.Hour),
		Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
	}
	snapshot.userAuthToken = session
	result, err := service.authenticationFromSnapshot(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	if err := service.postgres.DB(ctx).Create(session).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// AuthenticationForPrincipal preserves an authenticated request Principal and reloads its exact User session.
func (service *userService) AuthenticationForPrincipal(ctx context.Context, principal authentication.Principal) (*internal_services.UserAuthentication, error) {
	subject := principal.Subject()
	proof := principal.Proof()
	if subject.Type != authentication.IdentityUser || proof.Method != authentication.AuthenticationUserSession ||
		proof.Reference.Type != authentication.ProofReferenceUserSession {
		return nil, pkg_errors.AuthenticationInvalidError{}
	}
	sessionID, err := strconv.ParseUint(proof.Reference.Value, 10, 64)
	if err != nil || sessionID == 0 {
		return nil, pkg_errors.AuthenticationInvalidError{}
	}

	snapshot, err := service.canonicalUserSnapshot(ctx, nil, subject.ID)
	if err != nil {
		return nil, err
	}
	session := internal_entity.UserAuthToken{}
	if err := service.postgres.DB(ctx).
		Where("id = ? AND user_auth_id = ? AND token_type = ? AND status = ?", sessionID, subject.ID, "auth-token", type_enums.RECORD_ACTIVE).
		Take(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkg_errors.AuthenticationCredentialInvalidError{}
		}
		return nil, err
	}
	if !session.ExpireAt.After(time.Now()) {
		return nil, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	snapshot.userAuthToken = &session
	return service.authentication(ctx, snapshot, principal)
}

func (service *userService) authentication(ctx context.Context, snapshot *authenticationSnapshot, principal authentication.Principal) (*internal_services.UserAuthentication, error) {
	if snapshot == nil || snapshot.user == nil || snapshot.userAuthToken == nil || snapshot.user.Id != principal.Subject().ID {
		return nil, pkg_errors.AuthenticationInvalidError{}
	}
	snapshot.user.Name = strings.TrimSpace(snapshot.user.Name)
	snapshot.user.Email = strings.TrimSpace(snapshot.user.Email)

	organizationsByID := make(map[uint64]*authentication.Organization)
	organizationIDs := make([]uint64, 0)
	for _, assignment := range snapshot.userOrgRoles {
		if assignment == nil || assignment.Organization.Id != assignment.OrganizationId {
			return nil, pkg_errors.AuthenticationInvalidError{}
		}
		assignment.Organization.Name = strings.TrimSpace(assignment.Organization.Name)
		assignment.RoleDefinition.Name = strings.TrimSpace(assignment.RoleDefinition.Name)
		assignment.RoleDefinition.Description = strings.TrimSpace(assignment.RoleDefinition.Description)
		organization := organizationsByID[assignment.OrganizationId]
		if organization == nil {
			organization = &authentication.Organization{ID: assignment.OrganizationId, Name: assignment.Organization.Name}
			organizationsByID[assignment.OrganizationId] = organization
			organizationIDs = append(organizationIDs, assignment.OrganizationId)
		}
		role := authentication.Role{
			ID: assignment.RoleDefinition.Id, Key: assignment.RoleDefinition.Key,
			Name: assignment.RoleDefinition.Name, Description: assignment.RoleDefinition.Description,
			Scope: assignment.RoleDefinition.ScopeType, Subject: assignment.RoleDefinition.SubjectType,
			DefinitionType: assignment.RoleDefinition.DefinitionType,
		}
		for _, rolePermission := range assignment.RoleDefinition.RolePermissions {
			if rolePermission == nil || rolePermission.PermissionDefinition == nil {
				return nil, pkg_errors.AuthenticationInvalidError{}
			}
			rolePermission.PermissionDefinition.Name = strings.TrimSpace(rolePermission.PermissionDefinition.Name)
			rolePermission.PermissionDefinition.Description = strings.TrimSpace(rolePermission.PermissionDefinition.Description)
			permission := rolePermission.PermissionDefinition
			role.Permissions = append(role.Permissions, authentication.Permission{
				ID: permission.Id, Key: permission.Key, Scope: permission.ScopeType,
				Name: permission.Name, Description: permission.Description,
			})
		}
		organization.Roles = append(organization.Roles, role)
	}

	projectsByID := make(map[uint64]*authentication.Project)
	for _, assignment := range snapshot.userProjectRoles {
		if assignment == nil || assignment.Project.Id != assignment.ProjectId {
			return nil, pkg_errors.AuthenticationInvalidError{}
		}
		assignment.Project.Name = strings.TrimSpace(assignment.Project.Name)
		assignment.RoleDefinition.Name = strings.TrimSpace(assignment.RoleDefinition.Name)
		assignment.RoleDefinition.Description = strings.TrimSpace(assignment.RoleDefinition.Description)
		organization := organizationsByID[assignment.Project.OrganizationId]
		if organization == nil {
			return nil, pkg_errors.AuthenticationInvalidError{}
		}
		project := projectsByID[assignment.ProjectId]
		if project == nil {
			project = &authentication.Project{ID: assignment.ProjectId, Name: assignment.Project.Name}
			projectsByID[assignment.ProjectId] = project
			organization.Projects = append(organization.Projects, *project)
		}
		role := authentication.Role{
			ID: assignment.RoleDefinition.Id, Key: assignment.RoleDefinition.Key,
			Name: assignment.RoleDefinition.Name, Description: assignment.RoleDefinition.Description,
			Scope: assignment.RoleDefinition.ScopeType, Subject: assignment.RoleDefinition.SubjectType,
			DefinitionType: assignment.RoleDefinition.DefinitionType,
		}
		for _, rolePermission := range assignment.RoleDefinition.RolePermissions {
			if rolePermission == nil || rolePermission.PermissionDefinition == nil {
				return nil, pkg_errors.AuthenticationInvalidError{}
			}
			rolePermission.PermissionDefinition.Name = strings.TrimSpace(rolePermission.PermissionDefinition.Name)
			rolePermission.PermissionDefinition.Description = strings.TrimSpace(rolePermission.PermissionDefinition.Description)
			permission := rolePermission.PermissionDefinition
			role.Permissions = append(role.Permissions, authentication.Permission{
				ID: permission.Id, Key: permission.Key, Scope: permission.ScopeType,
				Name: permission.Name, Description: permission.Description,
			})
		}
		project.Roles = append(project.Roles, role)
		for index := range organization.Projects {
			if organization.Projects[index].ID == project.ID {
				organization.Projects[index] = *project
				break
			}
		}
	}

	subscriptions, err := service.billingService.GetSubscriptions(ctx, organizationIDs)
	if err != nil {
		return nil, err
	}
	organizations := make([]authentication.Organization, 0, len(organizationsByID))
	for _, organizationID := range organizationIDs {
		organization := organizationsByID[organizationID]
		if subscription, exists := subscriptions[organizationID]; exists {
			if subscription.Plan == nil {
				return nil, pkg_errors.BillingSubscriptionInvalidError{}
			}
			plan := subscription.Plan
			organization.BillingPlan = &authentication.BillingPlan{
				ID: plan.Id, Name: plan.Name, Slug: plan.Slug, Description: plan.Description,
				IsDefault: plan.IsDefault, IsActive: plan.IsActive, SortOrder: int32(plan.SortOrder),
				PriceMonthly: plan.PriceMonthly, PriceYearly: plan.PriceYearly,
				Currency: plan.Currency, StripeURL: plan.StripeURL,
			}
			for _, quota := range plan.Quotas {
				if quota == nil {
					return nil, pkg_errors.BillingSubscriptionInvalidError{}
				}
				organization.BillingPlan.Quotas = append(organization.BillingPlan.Quotas, authentication.BillingPlanQuota{
					ID: quota.Id, ResourceType: quota.ResourceType, QuotaLimit: quota.QuotaLimit, Unit: quota.Unit,
				})
			}
		}
		organizations = append(organizations, *organization)
	}

	value, err := authentication.NewDirectAuthentication(principal.Subject(), principal.Tenant(), principal.Proof(), &authentication.User{
		ID: snapshot.user.Id, Name: snapshot.user.Name, Email: snapshot.user.Email, Status: snapshot.user.Status.String(),
	}, organizations)
	if err != nil {
		return nil, err
	}
	return &internal_services.UserAuthentication{
		Authentication: value,
		Token:          *snapshot.userAuthToken,
	}, nil
}
