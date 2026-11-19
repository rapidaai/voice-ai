// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_project_service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"

	internal_accessmanagement "github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

// PrincipalAuthenticationService owns canonical Project credential and authority reads.
type PrincipalAuthenticationService interface {
	resolver.ProjectAuthenticationService
}

// NewPrincipalAuthenticationService constructs the canonical Project principal source.
func NewPrincipalAuthenticationService(logger commons.Logger, postgres connectors.PostgresConnector) PrincipalAuthenticationService {
	return &projectService{logger: logger, postgres: postgres}
}

// ResolveProjectCredential resolves one exact active Project credential and its assigned Role graph.
func (service *projectService) ResolveProjectCredential(ctx context.Context, token string) (authentication.Authentication, error) {
	var credential struct {
		CredentialID       uint64
		OrganizationID     uint64
		OrganizationName   string
		ProjectID          uint64
		ProjectName        string
		RoleID             uint64
		RoleKey            string
		RoleName           string
		RoleDescription    string
		RoleScope          string
		RoleSubject        string
		RoleDefinitionType string
	}
	query := service.postgres.DB(ctx).Table("project_credentials AS credential").
		Select("credential.id AS credential_id, credential.organization_id, organization.name AS organization_name, credential.project_id, project.name AS project_name, role.id AS role_id, role.key AS role_key, role.name AS role_name, role.description AS role_description, role.scope_type AS role_scope, role.subject_type AS role_subject, role.definition_type AS role_definition_type").
		Joins("JOIN projects AS project ON project.id = credential.project_id AND project.organization_id = credential.organization_id AND project.status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN organizations AS organization ON organization.id = credential.organization_id AND organization.status = ?", type_enums.RECORD_ACTIVE).
		Joins("JOIN role_definitions AS role ON role.id = credential.role_definition_id AND role.status = ? AND role.scope_type = ? AND role.subject_type = ?", type_enums.RECORD_ACTIVE, internal_accessmanagement.ScopeProject, internal_accessmanagement.SubjectProjectToken).
		Where("credential.status = ? AND credential.role_subject_type = ?", type_enums.RECORD_ACTIVE, internal_accessmanagement.SubjectProjectToken).
		Where("(role.definition_type = ? AND role.organization_id IS NULL) OR (role.definition_type = ? AND role.organization_id = credential.organization_id)", internal_accessmanagement.DefinitionBuiltIn, internal_accessmanagement.DefinitionCustom)
	query = query.Where("credential.key = ?", token)
	err := query.Take(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	if err != nil {
		return authentication.Authentication{}, err
	}

	type permissionRow struct {
		ID          uint64
		Key         string
		Scope       string
		Name        string
		Description string
	}
	rows := make([]permissionRow, 0)
	err = service.postgres.DB(ctx).Table("role_permissions AS grant_row").
		Select("permission.id, permission.key, permission.scope_type AS scope, permission.name, permission.description").
		Joins("JOIN permission_definitions AS permission ON permission.id = grant_row.permission_definition_id AND permission.status = ? AND permission.scope_type = ?", type_enums.RECORD_ACTIVE, internal_accessmanagement.ScopeProject).
		Where("grant_row.role_definition_id = ? AND grant_row.status = ?", credential.RoleID, type_enums.RECORD_ACTIVE).
		Order("permission.key ASC, permission.id ASC").Scan(&rows).Error
	if err != nil {
		return authentication.Authentication{}, err
	}
	role := authentication.Role{
		ID: credential.RoleID, Key: credential.RoleKey, Name: strings.TrimSpace(credential.RoleName),
		Description: strings.TrimSpace(credential.RoleDescription), Scope: credential.RoleScope,
		Subject: credential.RoleSubject, DefinitionType: credential.RoleDefinitionType,
	}
	for _, row := range rows {
		role.Permissions = append(role.Permissions, authentication.Permission{
			ID: row.ID, Key: row.Key, Scope: row.Scope, Name: strings.TrimSpace(row.Name),
			Description: strings.TrimSpace(row.Description),
		})
	}
	return authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: credential.CredentialID},
		authentication.Tenant{OrganizationID: credential.OrganizationID, ProjectID: credential.ProjectID},
		authentication.Proof{
			Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: strconv.FormatUint(credential.CredentialID, 10)},
		},
		nil,
		[]authentication.Organization{{
			ID: credential.OrganizationID, Name: strings.TrimSpace(credential.OrganizationName),
			Projects: []authentication.Project{{ID: credential.ProjectID, Name: strings.TrimSpace(credential.ProjectName), Roles: []authentication.Role{role}}},
		}},
	)
}
