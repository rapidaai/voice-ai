// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/protos"
)

type AccessManagementService interface {
	ListPermissions(context.Context, authentication.Principal, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.PermissionDefinition, error)
	ListRoles(context.Context, authentication.Principal, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.RoleDefinition, error)
	GetRole(context.Context, authentication.Principal, uint64) (*internal_entity.RoleDefinition, error)
	CreateCustomRole(ctx context.Context, principal authentication.Principal, key, name, description, scopeType, subjectType string, permissionDefinitionIDs []uint64) (*internal_entity.RoleDefinition, error)
	UpdateCustomRole(ctx context.Context, principal authentication.Principal, roleDefinitionID uint64, name, description string) (*internal_entity.RoleDefinition, error)
	ReduceCustomRolePermissions(ctx context.Context, principal authentication.Principal, roleDefinitionID uint64, permissionDefinitionIDs []uint64, reason, correlationID, confirmationDigest string, preview bool) (*internal_entity.RoleDefinition, *RoleReductionPreview, error)
	ArchiveCustomRole(ctx context.Context, principal authentication.Principal, roleDefinitionID uint64, reason, correlationID string) error
}

type RoleReductionImpact struct {
	OrganizationID   uint64
	ProjectID        uint64
	SubjectType      string
	SubjectID        uint64
	AssignmentStatus string
}

type RoleReductionPreview struct {
	ResultingRole      *internal_entity.RoleDefinition
	Impacts            []RoleReductionImpact
	ConfirmationDigest string
}
