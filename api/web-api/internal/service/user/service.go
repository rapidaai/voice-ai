// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_user_service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	internal_accessmanagement "github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	internal_billing_service "github.com/rapidaai/api/web-api/internal/service/billing"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/ciphers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	web_api "github.com/rapidaai/protos"
)

type userService struct {
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	billingService internal_services.BillingService
}

func NewUserService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.UserService {
	return &userService{
		logger:         logger,
		postgres:       postgres,
		billingService: internal_billing_service.NewBillingService(postgres),
	}
}

// AuthenticateUser verifies primary credentials and returns one complete session-backed Authentication.
func (aS *userService) AuthenticateUser(ctx context.Context, email string, password string) (*internal_services.UserAuthentication, error) {
	snapshot, err := aS.authenticate(ctx, email, password)
	if err != nil {
		return nil, err
	}
	return aS.createAuthentication(ctx, snapshot)
}

// VerifyPassword verifies primary credentials without issuing a session.
func (aS *userService) VerifyPassword(ctx context.Context, email string, password string) error {
	var user internal_entity.UserAuth
	err := aS.postgres.DB(ctx).
		Where("email = ? AND password = ? AND status = ?", strings.ToLower(email), ciphers.Hash(password), type_enums.RECORD_ACTIVE).
		Take(&user).Error
	if err != nil {
		aS.logger.Errorf("password verification failed: %v", err)
	}
	return err
}

func (aS *userService) authenticate(ctx context.Context, email string, password string) (*authenticationSnapshot, error) {
	db := aS.postgres.DB(ctx)
	var aUser internal_entity.UserAuth
	tx := db.First(&aUser, "email = ? AND password = ? AND status = ?", strings.ToLower(email), ciphers.Hash(password), type_enums.RECORD_ACTIVE)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var organizationRoles []*internal_entity.UserOrganizationRole
	var prjs []*internal_entity.UserProjectRole
	g, authenticationContext := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		organizationRoles, err = aS.loadCanonicalUserOrganizationRoles(authenticationContext, aUser.Id)
		return err
	})

	g.Go(func() error {
		var err error
		prjs, err = aS.loadCanonicalUserProjectRoles(authenticationContext, aUser.Id)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &authenticationSnapshot{user: &aUser, userOrgRoles: organizationRoles, userProjectRoles: prjs}, nil
}

func (aS *userService) Create(ctx context.Context, auth authentication.Principal, name string, email string, password string, status type_enums.RecordState, source *string) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_entity.UserAuth{
		Name:     name,
		Email:    strings.ToLower(email),
		Password: ciphers.Hash(password),
		Mutable: gorm_models.Mutable{
			Status: status,
		},
	}
	if source != nil {
		user.Source = *source
	}
	if auth.Caller().ID != 0 {
		user.CreatedActorType = string(auth.Caller().Type)
		user.CreatedActorID = auth.Caller().ID
	}
	tx := db.Save(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, err
	}
	return user, nil
}

func (aS *userService) Get(ctx context.Context, email string) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var aUser internal_entity.UserAuth
	tx := db.First(&aUser, "email = ?", strings.ToLower(email))
	if tx.Error != nil {
		aS.logger.Errorf("unable to find the user %s", email)
		return nil, tx.Error
	}
	return &aUser, nil
}

func (aS *userService) Activate(ctx context.Context, userId uint64, name string, source *string) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	ct := internal_entity.UserAuth{
		Name: name,
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
	}
	if source != nil {
		ct.Source = *source
	}

	tx := db.Where("id=?", userId).Clauses(clause.Returning{}).Updates(&ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) CreateNewAuthToken(ctx context.Context, auth authentication.Principal, userId uint64) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_entity.UserAuthToken{
		UserAuthId: userId,
		TokenType:  "auth-token", Token: ciphers.Token("at"),
		ExpireAt: time.Now().Add(200 * time.Hour),
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
	}
	if auth.Caller().ID != 0 {
		ct.CreatedActorType = string(auth.Caller().Type)
		ct.CreatedActorID = auth.Caller().ID
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) CreatePasswordToken(ctx context.Context, userId uint64) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_entity.UserAuthToken{
		UserAuthId: userId,
		TokenType:  "password-token", Token: ciphers.Token("pt"),
		ExpireAt: time.Now().Add(200 * time.Hour),
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) GetAuthToken(ctx context.Context, userId uint64) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserAuthToken
	tx := db.Where("user_auth_id = ? AND token_type = ? AND status = ? AND expire_at > ?", userId, "auth-token", type_enums.RECORD_ACTIVE, time.Now()).
		Order("id DESC").Take(&ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) GetToken(ctx context.Context, tokenType string, token string) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserAuthToken
	tx := db.First(&ct, "token_type = ? AND token = ?", tokenType, token)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) CreateOrganizationRole(ctx context.Context, auth authentication.Principal, roleDefinitionID uint64, userId uint64, organizationId uint64, status type_enums.RecordState) (*internal_entity.UserOrganizationRole, error) {
	assignment := &internal_entity.UserOrganizationRole{
		UserAuthId:       userId,
		OrganizationId:   organizationId,
		RoleDefinitionID: roleDefinitionID,
		RoleScopeType:    internal_accessmanagement.ScopeOrganization,
		RoleSubjectType:  internal_accessmanagement.SubjectUser,
		Mutable: gorm_models.Mutable{
			Status:           status,
			CreatedActorType: string(auth.Caller().Type),
			CreatedActorID:   auth.Caller().ID,
		},
	}
	if err := aS.postgres.DB(ctx).Create(assignment).Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return assignment, nil
}

func (aS *userService) CreateProjectRole(ctx context.Context, auth authentication.Principal, userId uint64, roleDefinitionID uint64, projectId uint64, status type_enums.RecordState) (*internal_entity.UserProjectRole, error) {
	assignment := &internal_entity.UserProjectRole{
		UserAuthId:       userId,
		ProjectId:        projectId,
		RoleDefinitionID: roleDefinitionID,
		RoleScopeType:    internal_accessmanagement.ScopeProject,
		RoleSubjectType:  internal_accessmanagement.SubjectUser,
		Mutable: gorm_models.Mutable{
			Status:           status,
			CreatedActorType: string(auth.Caller().Type),
			CreatedActorID:   auth.Caller().ID,
		},
	}
	if err := aS.postgres.DB(ctx).Create(assignment).Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return assignment, nil
}

func (aS *userService) InviteUserToOrganization(ctx context.Context, auth authentication.Principal, email string, organizationID uint64, organizationRoleDefinitionIDs []uint64, projectRoles []internal_services.ProjectRoleSelection) (*internal_entity.UserAuth, []*internal_entity.Project, error) {
	if tenant := auth.Tenant(); tenant.OrganizationID == 0 || tenant.OrganizationID != organizationID {
		return nil, nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	organizationRoleDefinitionIDs, ok := uniqueRoleDefinitionIDs(organizationRoleDefinitionIDs)
	if !ok || len(organizationRoleDefinitionIDs) == 0 {
		return nil, nil, pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentOrganizationRole}
	}
	projectRoles, ok = orderedProjectRoleSelections(projectRoles)
	if !ok {
		return nil, nil, pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentProjectRole}
	}

	var invitedUser internal_entity.UserAuth
	var projects []*internal_entity.Project
	err := aS.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockActiveOrganization(tx, organizationID); err != nil {
			return err
		}
		identityErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", strings.ToLower(email)).Take(&invitedUser).Error
		switch {
		case errors.Is(identityErr, gorm.ErrRecordNotFound):
			parts := strings.Split(email, "@")
			invitedUser = internal_entity.UserAuth{
				Name:     parts[0],
				Email:    strings.ToLower(email),
				Password: ciphers.Hash(ciphers.RandomHash("rpd_")),
				Source:   "invited-by-other",
				Mutable: gorm_models.Mutable{
					Status:           type_enums.RECORD_INVITED,
					CreatedActorType: string(auth.Caller().Type),
					CreatedActorID:   auth.Caller().ID,
				},
			}
			if err := tx.Create(&invitedUser).Error; err != nil {
				return err
			}
			token := &internal_entity.UserAuthToken{
				UserAuthId: invitedUser.Id,
				TokenType:  "auth-token",
				Token:      ciphers.Token("at"),
				ExpireAt:   time.Now().Add(200 * time.Hour),
				Mutable: gorm_models.Mutable{
					Status:           type_enums.RECORD_ACTIVE,
					CreatedActorType: string(auth.Caller().Type),
					CreatedActorID:   auth.Caller().ID,
				},
			}
			if err := tx.Create(token).Error; err != nil {
				return err
			}
		case identityErr != nil:
			return identityErr
		default:
			var currentAssignments []internal_entity.UserOrganizationRole
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_auth_id = ? AND status IN ?", invitedUser.Id, currentAssignmentStates()).
				Order("organization_id ASC, role_definition_id ASC").Find(&currentAssignments).Error; err != nil {
				return err
			}
			if len(currentAssignments) > 0 {
				return pkg_errors.OrganizationMembershipConflictError{SameOrganization: currentAssignments[0].OrganizationId == organizationID}
			}
			if invitedUser.Status == type_enums.RECORD_ARCHIEVE {
				if err := tx.Model(&invitedUser).Updates(map[string]interface{}{
					"status":             type_enums.RECORD_ACTIVE,
					"updated_actor_type": string(auth.Caller().Type),
					"updated_actor_id":   auth.Caller().ID,
					"updated_date":       time.Now(),
				}).Error; err != nil {
					return err
				}
				invitedUser.Status = type_enums.RECORD_ACTIVE
			}
		}

		organizationRoles, err := lockRoleDefinitions(tx, organizationRoleDefinitionIDs, organizationID, internal_accessmanagement.ScopeOrganization)
		if err != nil {
			return err
		}
		for _, role := range organizationRoles {
			if role.Id == internal_services.OrganizationOwnerRoleDefinitionID {
				return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentOrganizationRole}
			}
		}
		projects, err = lockProjects(tx, organizationID, projectRoles)
		if err != nil {
			return err
		}
		if _, err = lockProjectRoleDefinitions(tx, organizationID, projectRoles); err != nil {
			return err
		}

		assignmentStatus := assignmentStatusForIdentity(invitedUser.Status)
		organizationAssignments := make([]internal_entity.UserOrganizationRole, 0, len(organizationRoleDefinitionIDs))
		for _, roleDefinitionID := range organizationRoleDefinitionIDs {
			organizationAssignments = append(organizationAssignments, internal_entity.UserOrganizationRole{
				UserAuthId: invitedUser.Id, OrganizationId: organizationID, RoleDefinitionID: roleDefinitionID,
				RoleScopeType: internal_accessmanagement.ScopeOrganization, RoleSubjectType: internal_accessmanagement.SubjectUser,
				Mutable: gorm_models.Mutable{Status: assignmentStatus, CreatedActorType: string(auth.Caller().Type), CreatedActorID: auth.Caller().ID},
			})
		}
		if err := tx.Create(&organizationAssignments).Error; err != nil {
			return pkg_errors.OrganizationRolePersistenceError{Cause: err}
		}
		projectAssignments := projectAssignmentsForSelections(auth, invitedUser.Id, assignmentStatus, projectRoles)
		if len(projectAssignments) > 0 {
			if err := tx.Create(&projectAssignments).Error; err != nil {
				return pkg_errors.ProjectRolePersistenceError{Cause: err}
			}
		}
		return nil
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, nil, ctxErr
		}
		return nil, nil, err
	}
	return &invitedUser, projects, nil
}

func (aS *userService) AddUserToProjects(ctx context.Context, auth authentication.Principal, userID uint64, organizationID uint64, projectRoles []internal_services.ProjectRoleSelection) ([]*internal_entity.Project, error) {
	if tenant := auth.Tenant(); tenant.OrganizationID == 0 || tenant.OrganizationID != organizationID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	projectRoles, ok := orderedProjectRoleSelections(projectRoles)
	if !ok || len(projectRoles) == 0 {
		return nil, pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentProjectRole}
	}
	var projects []*internal_entity.Project
	err := aS.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockActiveOrganization(tx, organizationID); err != nil {
			return err
		}
		var err error
		projects, err = lockProjects(tx, organizationID, projectRoles)
		if err != nil {
			return err
		}
		var user internal_entity.UserAuth
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status IN ?", userID, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INVITED}).Take(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentMembership}
			}
			return err
		}
		var memberships []internal_entity.UserOrganizationRole
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_auth_id = ? AND organization_id = ? AND status IN ?", userID, organizationID, currentAssignmentStates()).
			Order("role_definition_id ASC").Find(&memberships).Error; err != nil {
			return err
		}
		if len(memberships) == 0 {
			return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentMembership}
		}
		if _, err := lockProjectRoleDefinitions(tx, organizationID, projectRoles); err != nil {
			return err
		}
		projectIDs := make([]uint64, 0, len(projectRoles))
		for _, selection := range projectRoles {
			projectIDs = append(projectIDs, selection.ProjectID)
		}
		var existing []internal_entity.UserProjectRole
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_auth_id = ? AND project_id IN ? AND status IN ?", userID, projectIDs, currentAssignmentStates()).
			Order("project_id ASC, role_definition_id ASC").Find(&existing).Error; err != nil {
			return err
		}
		if len(existing) > 0 {
			return pkg_errors.ProjectMembershipConflictError{}
		}
		assignments := projectAssignmentsForSelections(auth, userID, assignmentStatusForIdentity(user.Status), projectRoles)
		return tx.Create(&assignments).Error
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	return projects, nil
}

func (aS *userService) ReplaceOrganizationRoles(ctx context.Context, auth authentication.Principal, userID uint64, organizationID uint64, roleDefinitionIDs []uint64) error {
	if tenant := auth.Tenant(); tenant.OrganizationID == 0 || tenant.OrganizationID != organizationID {
		return pkg_errors.RouteAuthorizationDeniedError{}
	}
	roleDefinitionIDs, ok := uniqueRoleDefinitionIDs(roleDefinitionIDs)
	if !ok || len(roleDefinitionIDs) == 0 {
		return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentOrganizationRole}
	}
	for _, roleDefinitionID := range roleDefinitionIDs {
		if roleDefinitionID == internal_services.OrganizationOwnerRoleDefinitionID {
			return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentOrganizationRole}
		}
	}
	return aS.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockActiveOrganization(tx, organizationID); err != nil {
			return err
		}
		var user internal_entity.UserAuth
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status IN ?", userID, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INVITED}).Take(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentMembership}
			}
			return err
		}
		var current []internal_entity.UserOrganizationRole
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_auth_id = ? AND organization_id = ? AND status IN ?", userID, organizationID, currentAssignmentStates()).
			Order("role_definition_id ASC").Find(&current).Error; err != nil {
			return err
		}
		if len(current) == 0 {
			return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentMembership}
		}
		for _, assignment := range current {
			if assignment.RoleDefinitionID == internal_services.OrganizationOwnerRoleDefinitionID {
				return pkg_errors.OrganizationOwnerMutationError{}
			}
		}
		if _, err := lockRoleDefinitions(tx, roleDefinitionIDs, organizationID, internal_accessmanagement.ScopeOrganization); err != nil {
			return err
		}
		wanted := make(map[uint64]struct{}, len(roleDefinitionIDs))
		for _, roleDefinitionID := range roleDefinitionIDs {
			wanted[roleDefinitionID] = struct{}{}
		}
		currentByRole := make(map[uint64]internal_entity.UserOrganizationRole, len(current))
		for _, assignment := range current {
			currentByRole[assignment.RoleDefinitionID] = assignment
		}
		status := assignmentStatusForIdentity(user.Status)
		now := time.Now()
		for _, assignment := range current {
			_, keep := wanted[assignment.RoleDefinitionID]
			updates := map[string]interface{}{"updated_actor_type": string(auth.Caller().Type), "updated_actor_id": auth.Caller().ID, "updated_date": now}
			if keep {
				updates["status"] = status
			} else {
				updates["status"] = type_enums.RECORD_ARCHIEVE
			}
			if err := tx.Model(&internal_entity.UserOrganizationRole{}).Where("id = ?", assignment.Id).Updates(updates).Error; err != nil {
				return err
			}
		}
		for _, roleDefinitionID := range roleDefinitionIDs {
			if _, exists := currentByRole[roleDefinitionID]; exists {
				continue
			}
			assignment := &internal_entity.UserOrganizationRole{
				UserAuthId: userID, OrganizationId: organizationID, RoleDefinitionID: roleDefinitionID,
				RoleScopeType: internal_accessmanagement.ScopeOrganization, RoleSubjectType: internal_accessmanagement.SubjectUser,
				Mutable: gorm_models.Mutable{Status: status, CreatedActorType: string(auth.Caller().Type), CreatedActorID: auth.Caller().ID},
			}
			if err := tx.Create(assignment).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (aS *userService) HasOrganizationPermissions(ctx context.Context, userID uint64, organizationID uint64, permissionKeys ...string) (bool, error) {
	if len(permissionKeys) == 0 {
		return true, nil
	}
	var keys []string
	err := aS.postgres.DB(ctx).Table("user_organization_roles").Distinct("permission_definitions.key").
		Joins("JOIN role_definitions ON role_definitions.id = user_organization_roles.role_definition_id AND role_definitions.status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN role_permissions ON role_permissions.role_definition_id = role_definitions.id AND role_permissions.status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN permission_definitions ON permission_definitions.id = role_permissions.permission_definition_id AND permission_definitions.status = ?", type_enums.RECORD_ACTIVE).
		Where("user_organization_roles.user_auth_id = ? AND user_organization_roles.organization_id = ? AND user_organization_roles.status = ?", userID, organizationID, type_enums.RECORD_ACTIVE).
		Where("permission_definitions.key IN ?", permissionKeys).Pluck("permission_definitions.key", &keys).Error
	if err != nil {
		return false, err
	}
	found := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		found[key] = struct{}{}
	}
	for _, key := range permissionKeys {
		if _, ok := found[key]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func currentAssignmentStates() []type_enums.RecordState {
	return []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INVITED}
}

func assignmentStatusForIdentity(status type_enums.RecordState) type_enums.RecordState {
	if status == type_enums.RECORD_INVITED {
		return type_enums.RECORD_INVITED
	}
	return type_enums.RECORD_ACTIVE
}

func uniqueRoleDefinitionIDs(ids []uint64) ([]uint64, bool) {
	seen := make(map[uint64]struct{}, len(ids))
	ordered := append([]uint64(nil), ids...)
	for _, id := range ordered {
		if id == 0 {
			return nil, false
		}
		if _, exists := seen[id]; exists {
			return nil, false
		}
		seen[id] = struct{}{}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered, true
}

func orderedProjectRoleSelections(selections []internal_services.ProjectRoleSelection) ([]internal_services.ProjectRoleSelection, bool) {
	projectIDs := make(map[uint64]struct{}, len(selections))
	ordered := make([]internal_services.ProjectRoleSelection, 0, len(selections))
	for _, selection := range selections {
		if selection.ProjectID == 0 || len(selection.RoleDefinitionIDs) == 0 {
			return nil, false
		}
		if _, exists := projectIDs[selection.ProjectID]; exists {
			return nil, false
		}
		projectIDs[selection.ProjectID] = struct{}{}
		roleDefinitionIDs, ok := uniqueRoleDefinitionIDs(selection.RoleDefinitionIDs)
		if !ok {
			return nil, false
		}
		ordered = append(ordered, internal_services.ProjectRoleSelection{ProjectID: selection.ProjectID, RoleDefinitionIDs: roleDefinitionIDs})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ProjectID < ordered[j].ProjectID })
	return ordered, true
}

func lockActiveOrganization(tx *gorm.DB, organizationID uint64) error {
	var organization internal_entity.Organization
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", organizationID, type_enums.RECORD_ACTIVE).Take(&organization).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentOrganizationSelection}
	}
	return err
}

func lockRoleDefinitions(tx *gorm.DB, ids []uint64, organizationID uint64, scope string) ([]internal_entity.RoleDefinition, error) {
	var roles []internal_entity.RoleDefinition
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ? AND status = ? AND scope_type = ? AND subject_type = ?", ids, type_enums.RECORD_ACTIVE, scope, internal_accessmanagement.SubjectUser).
		Where("(definition_type = ? AND organization_id IS NULL) OR (definition_type = ? AND organization_id = ?)", internal_accessmanagement.DefinitionBuiltIn, internal_accessmanagement.DefinitionCustom, organizationID).
		Order("id ASC").Find(&roles).Error
	if err != nil {
		return nil, err
	}
	if len(roles) != len(ids) {
		selection := pkg_errors.RoleAssignmentOrganizationRole
		if scope == internal_accessmanagement.ScopeProject {
			selection = pkg_errors.RoleAssignmentProjectRole
		}
		return nil, pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: selection}
	}
	return roles, nil
}

func lockProjects(tx *gorm.DB, organizationID uint64, selections []internal_services.ProjectRoleSelection) ([]*internal_entity.Project, error) {
	ids := make([]uint64, 0, len(selections))
	for _, selection := range selections {
		ids = append(ids, selection.ProjectID)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var projects []*internal_entity.Project
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ? AND organization_id = ? AND status = ?", ids, organizationID, type_enums.RECORD_ACTIVE).Order("id ASC").Find(&projects).Error
	if err != nil {
		return nil, err
	}
	if len(projects) != len(ids) {
		return nil, pkg_errors.RoleAssignmentSelectionUnavailableError{Selection: pkg_errors.RoleAssignmentProject}
	}
	return projects, nil
}

func lockProjectRoleDefinitions(tx *gorm.DB, organizationID uint64, selections []internal_services.ProjectRoleSelection) ([]internal_entity.RoleDefinition, error) {
	roleIDs := make([]uint64, 0)
	seen := map[uint64]struct{}{}
	for _, selection := range selections {
		for _, roleID := range selection.RoleDefinitionIDs {
			if _, exists := seen[roleID]; exists {
				continue
			}
			seen[roleID] = struct{}{}
			roleIDs = append(roleIDs, roleID)
		}
	}
	sort.Slice(roleIDs, func(i, j int) bool { return roleIDs[i] < roleIDs[j] })
	if len(roleIDs) == 0 {
		return nil, nil
	}
	return lockRoleDefinitions(tx, roleIDs, organizationID, internal_accessmanagement.ScopeProject)
}

func projectAssignmentsForSelections(auth authentication.Principal, userID uint64, status type_enums.RecordState, selections []internal_services.ProjectRoleSelection) []internal_entity.UserProjectRole {
	count := 0
	for _, selection := range selections {
		count += len(selection.RoleDefinitionIDs)
	}
	assignments := make([]internal_entity.UserProjectRole, 0, count)
	for _, selection := range selections {
		for _, roleDefinitionID := range selection.RoleDefinitionIDs {
			assignments = append(assignments, internal_entity.UserProjectRole{
				UserAuthId: userID, ProjectId: selection.ProjectID, RoleDefinitionID: roleDefinitionID,
				RoleScopeType: internal_accessmanagement.ScopeProject, RoleSubjectType: internal_accessmanagement.SubjectUser,
				Mutable: gorm_models.Mutable{Status: status, CreatedActorType: string(auth.Caller().Type), CreatedActorID: auth.Caller().ID},
			})
		}
	}
	return assignments
}

func (aS *userService) ArchiveUserFromOrganization(ctx context.Context, auth authentication.Principal, userId uint64, organizationId uint64) error {
	if tenant := auth.Tenant(); tenant.OrganizationID == 0 || tenant.OrganizationID != organizationId {
		return pkg_errors.RouteAuthorizationDeniedError{}
	}

	db := aS.postgres.DB(ctx)
	tx := db.Begin()
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return tx.Error
	}

	if err := tx.Model(&internal_entity.UserAuth{}).
		Where("id = ?", userId).
		Updates(&internal_entity.UserAuth{
			Mutable: gorm_models.Mutable{
				Status:           type_enums.RECORD_ARCHIEVE,
				UpdatedActorType: string(auth.Caller().Type),
				UpdatedActorID:   auth.Caller().ID,
			},
		}).Error; err != nil {
		tx.Rollback()
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	}
	if err := tx.Model(&internal_entity.UserOrganizationRole{}).
		Where("user_auth_id = ? AND organization_id = ? AND status IN ?", userId, organizationId, []string{type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_INVITED.String()}).
		Updates(&internal_entity.UserOrganizationRole{
			Mutable: gorm_models.Mutable{
				Status:           type_enums.RECORD_ARCHIEVE,
				UpdatedActorType: string(auth.Caller().Type),
				UpdatedActorID:   auth.Caller().ID,
			},
		}).Error; err != nil {
		tx.Rollback()
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	}
	if err := tx.Model(&internal_entity.UserProjectRole{}).
		Where("user_auth_id = ? AND status IN ?", userId, []string{type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_INVITED.String()}).
		Updates(&internal_entity.UserProjectRole{
			Mutable: gorm_models.Mutable{
				Status:           type_enums.RECORD_ARCHIEVE,
				UpdatedActorType: string(auth.Caller().Type),
				UpdatedActorID:   auth.Caller().ID,
			},
		}).Error; err != nil {
		tx.Rollback()
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	}

	return tx.Commit().Error
}

func (aS *userService) ArchiveUserFromProject(ctx context.Context, auth authentication.Principal, userId uint64, projectId uint64) error {
	if tenant := auth.Tenant(); tenant.ProjectID == 0 || tenant.ProjectID != projectId {
		return pkg_errors.RouteAuthorizationDeniedError{}
	}

	db := aS.postgres.DB(ctx)
	tx := db.Model(&internal_entity.UserProjectRole{}).
		Where("user_auth_id = ? AND project_id = ? AND status IN ?", userId, projectId, []string{type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_INVITED.String()}).
		Updates(&internal_entity.UserProjectRole{
			Mutable: gorm_models.Mutable{
				Status:           type_enums.RECORD_ARCHIEVE,
				UpdatedActorType: string(auth.Caller().Type),
				UpdatedActorID:   auth.Caller().ID,
			},
		})
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("user project role not found")
	}
	return nil
}

func (aS *userService) GetActiveOrInvitedProjectRoles(ctx context.Context, userId uint64, projectId uint64) ([]*internal_entity.UserProjectRole, error) {
	var assignments []*internal_entity.UserProjectRole
	err := aS.postgres.DB(ctx).Preload("RoleDefinition").
		Where("user_auth_id = ? AND project_id = ? AND status IN ?", userId, projectId, currentAssignmentStates()).
		Order("role_definition_id ASC").Find(&assignments).Error
	return assignments, err
}

func (aS *userService) GetOrganizationRoles(ctx context.Context, userId uint64) ([]*internal_entity.UserOrganizationRole, error) {
	return aS.loadCanonicalUserOrganizationRoles(ctx, userId)
}

func (aS *userService) GetActiveOrInvitedOrganizationRoles(ctx context.Context, userId uint64) ([]*internal_entity.UserOrganizationRole, error) {
	var assignments []*internal_entity.UserOrganizationRole
	err := aS.postgres.DB(ctx).Preload("Organization").Preload("RoleDefinition").
		Where("user_auth_id = ? AND status IN ?", userId, currentAssignmentStates()).
		Order("organization_id ASC, role_definition_id ASC").Find(&assignments).Error
	return assignments, err
}

func (aS *userService) GetActiveOrInvitedOrganizationRolesForOrganization(ctx context.Context, userId uint64, organizationId uint64) ([]*internal_entity.UserOrganizationRole, error) {
	var assignments []*internal_entity.UserOrganizationRole
	err := aS.postgres.DB(ctx).Preload("Organization").Preload("RoleDefinition").
		Where("user_auth_id = ? AND organization_id = ? AND status IN ?", userId, organizationId, currentAssignmentStates()).
		Order("role_definition_id ASC").Find(&assignments).Error
	return assignments, err
}

func (aS *userService) GetAnyOrganizationRoles(ctx context.Context, userId uint64) ([]*internal_entity.UserOrganizationRole, error) {
	var assignments []*internal_entity.UserOrganizationRole
	err := aS.postgres.DB(ctx).Preload("Organization").Preload("RoleDefinition").
		Where("user_auth_id = ?", userId).Order("organization_id ASC, role_definition_id ASC, id ASC").Find(&assignments).Error
	return assignments, err
}

func (aS *userService) GetUser(ctx context.Context, userId uint64) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserAuth
	tx := db.First(&ct, "id = ?", userId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) UpdateUser(ctx context.Context, auth authentication.Principal, userId uint64, name *string) (*internal_entity.UserAuth, error) {
	subject := auth.Subject()
	if subject.Type != authentication.IdentityUser || subject.ID != userId {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	db := aS.postgres.DB(ctx)
	user := &internal_entity.UserAuth{
		Mutable: gorm_models.Mutable{
			UpdatedActorType: string(auth.Caller().Type),
			UpdatedActorID:   auth.Caller().ID,
		},
	}
	if name != nil {
		user.Name = *name
	}
	tx := db.Where("id = ? and status = ?", userId, type_enums.RECORD_ACTIVE.String()).Updates(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	} else {
		return user, nil
	}
}

func (aS *userService) UpdateUserStatus(ctx context.Context, auth authentication.Principal, userId uint64, status type_enums.RecordState) error {
	subject := auth.Subject()
	if subject.Type != authentication.IdentityUser || subject.ID != userId {
		return pkg_errors.RouteAuthorizationDeniedError{}
	}
	db := aS.postgres.DB(ctx)
	tx := db.Model(&internal_entity.UserAuth{}).
		Where("id = ?", userId).
		Updates(&internal_entity.UserAuth{
			Mutable: gorm_models.Mutable{
				Status:           status,
				UpdatedActorType: string(auth.Caller().Type),
				UpdatedActorID:   auth.Caller().ID,
			},
		})
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return tx.Error
	}
	return nil
}

func (as *userService) CreateSocial(ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_entity.UserSocial, error) {
	db := as.postgres.DB(ctx)
	ct := &internal_entity.UserSocial{
		UserAuthId: userId,
		Social:     source,
		Identifier: id,
		Token:      token,
		Verified:   verified,
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		as.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) GetSocial(ctx context.Context, userId uint64) (*internal_entity.UserSocial, error) {
	db := aS.postgres.DB(ctx)
	var socialUser *internal_entity.UserSocial
	if err := db.Where("user_auth_id = ? and status = ? ", userId, type_enums.RECORD_ACTIVE.String()).Find(&socialUser).Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return socialUser, nil
}

func (aS *userService) GetUsers(ctx context.Context, uIds []uint64, limit uint32, offset uint32) ([]*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var u []*internal_entity.UserAuth
	if err := db.Limit(int(limit)).Offset(int(offset)).Where("id IN ?", uIds).Order("created_date ASC").Find(&u).Error; err != nil {
		return nil, err
	}

	return u, nil
}

func (aS *userService) UpdatePassword(ctx context.Context, auth authentication.Principal, userId uint64, password string) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_entity.UserAuth{
		Mutable:  gorm_models.Mutable{},
		Password: ciphers.Hash(password),
	}
	if auth.Caller().ID != 0 {
		user.UpdatedActorType = string(auth.Caller().Type)
		user.UpdatedActorID = auth.Caller().ID
	}
	tx := db.Where("id = ? ", userId).Updates(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return user, nil
}

func (aS *userService) ActivateAllProjectRoles(ctx context.Context, userId uint64) error {
	db := aS.postgres.DB(ctx)
	// Update with struct
	tx := db.Where("user_auth_id = ? AND status  = ?", userId, type_enums.RECORD_INVITED.String()).
		Updates(&internal_entity.UserProjectRole{
			Mutable: gorm_models.Mutable{
				Status: type_enums.RECORD_ACTIVE,
			},
		})
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	} else {
		return nil
	}
}

func (aS *userService) ActivateAllOrganizationRole(ctx context.Context, userId uint64) error {
	db := aS.postgres.DB(ctx)
	// Update with struct
	tx := db.Where("user_auth_id = ? AND status  = ?", userId, type_enums.RECORD_INVITED.String()).Updates(&internal_entity.UserOrganizationRole{
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
	})
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	}
	return nil
}

func (aS *userService) GetAllActiveProjectMember(ctx context.Context, projectId uint64) ([]*internal_entity.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var rt []*internal_entity.UserProjectRole
	tx := db.Preload("Member").Where("project_id = ? AND status = ?", projectId, type_enums.RECORD_ACTIVE.String()).Find(&rt)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return rt, err
	}
	return rt, nil
}

func (aS *userService) GetAllOrganizationMember(ctx context.Context, organizationId uint64, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_entity.UserOrganizationRole, error) {
	db := aS.postgres.DB(ctx)
	var roles []internal_entity.UserOrganizationRole
	var count int64

	organizationMemberIDs := db.Model(&internal_entity.UserOrganizationRole{}).
		Select("DISTINCT user_auth_id").
		Where("organization_id = ? AND status IN ?", organizationId, currentAssignmentStates())
	usersQuery := db.Model(&internal_entity.UserAuth{}).
		Joins("JOIN (?) AS organization_members ON organization_members.user_auth_id = user_auths.id", organizationMemberIDs)
	for _, criterion := range criteria {
		if criterion == nil {
			return 0, &roles, errors.New("organization member criterion is required")
		}
		switch criterion.GetKey() {
		case "id":
			if criterion.GetLogic() != "" && criterion.GetLogic() != "=" {
				return 0, &roles, fmt.Errorf("unsupported logic %q for organization member criterion %q", criterion.GetLogic(), criterion.GetKey())
			}
			usersQuery = usersQuery.Where("user_auths.id = ?", criterion.GetValue())
		case "name", "email":
			column := "user_auths.name"
			if criterion.GetKey() == "email" {
				column = "user_auths.email"
			}
			switch criterion.GetLogic() {
			case "":
				usersQuery = usersQuery.Where(column+" = ?", criterion.GetValue())
			case "contains":
				pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(criterion.GetValue()) + "%"
				usersQuery = usersQuery.Where(`LOWER(`+column+`) LIKE LOWER(?) ESCAPE '\'`, pattern)
			default:
				return 0, &roles, fmt.Errorf("unsupported logic %q for organization member criterion %q", criterion.GetLogic(), criterion.GetKey())
			}
		case "status":
			if criterion.GetLogic() != "" && criterion.GetLogic() != "=" {
				return 0, &roles, fmt.Errorf("unsupported logic %q for organization member criterion %q", criterion.GetLogic(), criterion.GetKey())
			}
			usersQuery = usersQuery.Where("user_auths.status = ?", criterion.GetValue())
		case "source":
			if criterion.GetLogic() != "" && criterion.GetLogic() != "=" {
				return 0, &roles, fmt.Errorf("unsupported logic %q for organization member criterion %q", criterion.GetLogic(), criterion.GetKey())
			}
			usersQuery = usersQuery.Where("user_auths.source = ?", criterion.GetValue())
		default:
			return 0, &roles, fmt.Errorf("unsupported organization member criterion %q", criterion.GetKey())
		}
	}

	page, pageSize := uint32(1), uint32(25)
	if paginate != nil {
		page, pageSize = paginate.GetPage(), paginate.GetPageSize()
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 25
	}

	var users []internal_entity.UserAuth
	if err := usersQuery.Scopes(gorm_models.Paginate(gorm_models.NewPaginated(page, pageSize, &count, usersQuery))).
		Order("user_auths.created_date DESC, user_auths.id ASC").Find(&users).Error; err != nil {
		return count, &roles, err
	}
	userIDs := make([]uint64, 0, len(users))
	usersByID := make(map[uint64]internal_entity.UserAuth, len(users))
	userPosition := make(map[uint64]int, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.Id)
		usersByID[user.Id] = user
		userPosition[user.Id] = len(userIDs) - 1
	}
	if len(userIDs) == 0 {
		return count, &roles, nil
	}
	if err := db.Preload("RoleDefinition").
		Where("organization_id = ? AND user_auth_id IN ? AND status IN ?", organizationId, userIDs, currentAssignmentStates()).
		Find(&roles).Error; err != nil {
		return count, &roles, err
	}
	for index := range roles {
		roles[index].Member = usersByID[roles[index].UserAuthId]
	}
	sort.Slice(roles, func(i, j int) bool {
		leftPosition := userPosition[roles[i].UserAuthId]
		rightPosition := userPosition[roles[j].UserAuthId]
		if leftPosition != rightPosition {
			return leftPosition < rightPosition
		}
		return roles[i].RoleDefinitionID < roles[j].RoleDefinitionID
	})
	return count, &roles, nil
}

func (uS *userService) GetAllUserRolesForOrg(ctx context.Context, organizationId uint64) ([]*internal_entity.UserOrganizationRole, error) {
	db := uS.postgres.DB(ctx)
	var roles []*internal_entity.UserOrganizationRole
	if err := db.Where("organization_id = ? and status = ?", organizationId, type_enums.RECORD_ACTIVE.String()).Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

func (aS *userService) GetProjectRolesForUsers(ctx context.Context, pIds []uint64, uIds []uint64) ([]*internal_entity.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var pr []*internal_entity.UserProjectRole
	if err := db.Where("project_id IN ? and user_auth_id IN ? and status IN ?", pIds, uIds, []string{type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_INVITED.String()}).Find(&pr).Error; err != nil {
		return nil, err
	}
	return pr, nil
}
