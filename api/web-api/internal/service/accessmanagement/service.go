// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package accessmanagement

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ScopeOrganization = "ORGANIZATION"
	ScopeProject      = "PROJECT"

	SubjectUser         = "USER"
	SubjectProjectToken = "PROJECT_TOKEN"

	DefinitionBuiltIn = "BUILT_IN"
	DefinitionCustom  = "CUSTOM"
)

type accessManagementService struct {
	postgres connectors.PostgresConnector
}

func NewAccessManagementService(postgres connectors.PostgresConnector) internal_services.AccessManagementService {
	return &accessManagementService{postgres: postgres}
}

func (s *accessManagementService) ListPermissions(ctx context.Context, principal authentication.Principal, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.PermissionDefinition, error) {
	tenant := principal.Tenant()
	var count int64
	var permissions []*internal_entity.PermissionDefinition
	err := s.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&internal_entity.PermissionDefinition{}).
			Where("status = ?", type_enums.RECORD_ACTIVE)
		if tenant.ProjectID != 0 {
			query = query.Where("scope_type = ?", ScopeProject)
		}
		for _, criteria := range criterias {
			switch criteria.GetKey() {
			case "scope":
				if tenant.ProjectID != 0 && criteria.GetValue() != ScopeProject {
					return pkg_errors.AccessManagementPermissionDeniedError{}
				}
				query = query.Where("scope_type = ?", criteria.GetValue())
			}
		}
		if countErr := query.Count(&count).Error; countErr != nil {
			return countErr
		}
		offset, conversionErr := utils.Uint64ToInt(uint64(paginate.GetPage()-1) * uint64(paginate.GetPageSize()))
		if conversionErr != nil {
			return conversionErr
		}
		return query.Order("key ASC").Order("id ASC").
			Offset(offset).Limit(int(paginate.GetPageSize())).Find(&permissions).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return 0, nil, err
	}
	return count, permissions, nil
}

func (s *accessManagementService) ListRoles(ctx context.Context, principal authentication.Principal, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.RoleDefinition, error) {
	tenant := principal.Tenant()
	var count int64
	var roles []*internal_entity.RoleDefinition
	err := s.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&internal_entity.RoleDefinition{}).
			Where("status = ? AND (organization_id IS NULL OR organization_id = ?)", type_enums.RECORD_ACTIVE, tenant.OrganizationID)
		if tenant.ProjectID != 0 {
			query = query.Where("scope_type = ?", ScopeProject)
		}
		for _, criteria := range criterias {
			switch criteria.GetKey() {
			case "name":
				pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(criteria.GetValue()) + "%"
				query = query.Where(`LOWER(role_definitions.name) LIKE LOWER(?) ESCAPE '\'`, pattern)
			case "key":
				pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(criteria.GetValue()) + "%"
				query = query.Where(`LOWER(role_definitions.key) LIKE LOWER(?) ESCAPE '\'`, pattern)
			case "scope":
				if tenant.ProjectID != 0 && criteria.GetValue() != ScopeProject {
					return pkg_errors.AccessManagementPermissionDeniedError{}
				}
				query = query.Where("scope_type = ?", criteria.GetValue())
			case "subject":
				query = query.Where("subject_type = ?", criteria.GetValue())
			case "type":
				query = query.Where("definition_type = ?", criteria.GetValue())
			}
		}
		if countErr := query.Count(&count).Error; countErr != nil {
			return countErr
		}
		offset, conversionErr := utils.Uint64ToInt(uint64(paginate.GetPage()-1) * uint64(paginate.GetPageSize()))
		if conversionErr != nil {
			return conversionErr
		}
		if queryErr := query.Order("LOWER(name) ASC").Order("key ASC").Order("id ASC").
			Offset(offset).Limit(int(paginate.GetPageSize())).Find(&roles).Error; queryErr != nil {
			return queryErr
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return 0, nil, err
	}
	return count, roles, nil
}

func (s *accessManagementService) GetRole(ctx context.Context, principal authentication.Principal, roleDefinitionID uint64) (*internal_entity.RoleDefinition, error) {
	tenant := principal.Tenant()
	db := s.postgres.DB(ctx)
	query := preloadOrderedPermissions(db).
		Where("id = ? AND status = ? AND (organization_id IS NULL OR organization_id = ?)", roleDefinitionID, type_enums.RECORD_ACTIVE, tenant.OrganizationID)
	if tenant.ProjectID != 0 {
		query = query.Where("scope_type = ?", ScopeProject)
	}
	var role internal_entity.RoleDefinition
	err := query.Take(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkg_errors.AccessManagementRoleNotFoundError{}
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *accessManagementService) CreateCustomRole(ctx context.Context, principal authentication.Principal, key, name, description, scopeType, subjectType string, permissionDefinitionIDs []uint64) (*internal_entity.RoleDefinition, error) {
	tenant := principal.Tenant()
	subject := principal.Subject()
	caller := principal.Caller()
	var created *internal_entity.RoleDefinition
	err := s.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedOrganization internal_entity.Organization
		organizationLockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", tenant.OrganizationID, type_enums.RECORD_ACTIVE).
			Take(&lockedOrganization).Error
		if errors.Is(organizationLockErr, gorm.ErrRecordNotFound) {
			return pkg_errors.AccessManagementPermissionDeniedError{}
		}
		if organizationLockErr != nil {
			return organizationLockErr
		}
		allowed, permissionErr := hasOrganizationPermission(tx, subject.ID, tenant.OrganizationID, "organization.roles.create")
		if permissionErr != nil {
			return permissionErr
		}
		if !allowed {
			return pkg_errors.AccessManagementPermissionDeniedError{}
		}
		var existingCount int64
		if queryErr := tx.Model(&internal_entity.RoleDefinition{}).
			Where("key = ? AND (definition_type = ? OR (definition_type = ? AND organization_id = ?))", key, DefinitionBuiltIn, DefinitionCustom, tenant.OrganizationID).
			Count(&existingCount).Error; queryErr != nil {
			return queryErr
		}
		if existingCount != 0 {
			return pkg_errors.AccessManagementRoleKeyAlreadyExistsError{}
		}

		permissions := make([]*internal_entity.PermissionDefinition, 0, len(permissionDefinitionIDs))
		if len(permissionDefinitionIDs) != 0 {
			if queryErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id IN ? AND status = ?", permissionDefinitionIDs, type_enums.RECORD_ACTIVE).
				Order("id ASC").Find(&permissions).Error; queryErr != nil {
				return queryErr
			}
			if len(permissions) != len(permissionDefinitionIDs) {
				return pkg_errors.AccessManagementRolePermissionSelectionInvalidError{}
			}
			for _, permission := range permissions {
				if permission.ScopeType != scopeType {
					return pkg_errors.AccessManagementRolePermissionSelectionInvalidError{}
				}
			}
		}
		role := &internal_entity.RoleDefinition{
			OrganizationID: &tenant.OrganizationID,
			Key:            key, Name: name, Description: description,
			ScopeType: scopeType, SubjectType: subjectType, DefinitionType: DefinitionCustom,
		}
		role.Status = type_enums.RECORD_ACTIVE
		role.CreatedActorType = string(caller.Type)
		role.CreatedActorID = caller.ID
		if createErr := tx.Omit("id").Create(role).Error; createErr != nil {
			var count int64
			lookupErr := tx.Model(&internal_entity.RoleDefinition{}).
				Where("organization_id = ? AND key = ?", tenant.OrganizationID, key).Count(&count).Error
			if lookupErr == nil && count != 0 {
				return pkg_errors.AccessManagementRoleKeyAlreadyExistsError{}
			}
			return createErr
		}

		rows := make([]internal_entity.RolePermission, 0, len(permissions))
		for _, permission := range permissions {
			row := internal_entity.RolePermission{
				RoleDefinitionID: role.Id, PermissionDefinitionID: permission.Id,
				ScopeType: role.ScopeType,
			}
			row.Status = type_enums.RECORD_ACTIVE
			row.CreatedActorType = string(caller.Type)
			row.CreatedActorID = caller.ID
			rows = append(rows, row)
		}
		if len(rows) != 0 {
			if createErr := tx.Omit("id").Create(&rows).Error; createErr != nil {
				return createErr
			}
		}
		role.RolePermissions = make([]*internal_entity.RolePermission, 0, len(rows))
		for index := range rows {
			rows[index].PermissionDefinition = permissions[index]
			role.RolePermissions = append(role.RolePermissions, &rows[index])
		}
		created = role
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *accessManagementService) UpdateCustomRole(ctx context.Context, principal authentication.Principal, roleDefinitionID uint64, name, description string) (*internal_entity.RoleDefinition, error) {
	tenant := principal.Tenant()
	subject := principal.Subject()
	caller := principal.Caller()
	var updated *internal_entity.RoleDefinition
	err := s.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedOrganization internal_entity.Organization
		organizationLockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", tenant.OrganizationID, type_enums.RECORD_ACTIVE).
			Take(&lockedOrganization).Error
		if errors.Is(organizationLockErr, gorm.ErrRecordNotFound) {
			return pkg_errors.AccessManagementPermissionDeniedError{}
		}
		if organizationLockErr != nil {
			return organizationLockErr
		}
		allowed, permissionErr := hasOrganizationPermission(tx, subject.ID, tenant.OrganizationID, "organization.roles.update")
		if permissionErr != nil {
			return permissionErr
		}
		if !allowed {
			return pkg_errors.AccessManagementPermissionDeniedError{}
		}
		var result *gorm.DB
		var role internal_entity.RoleDefinition
		roleErr := preloadOrderedPermissions(tx.Clauses(clause.Locking{Strength: "UPDATE"})).
			Where("id = ? AND status = ? AND (organization_id IS NULL OR organization_id = ?)", roleDefinitionID, type_enums.RECORD_ACTIVE, tenant.OrganizationID).
			Take(&role).Error
		if errors.Is(roleErr, gorm.ErrRecordNotFound) {
			return pkg_errors.AccessManagementRoleNotFoundError{}
		}
		if roleErr != nil {
			return roleErr
		}
		if role.DefinitionType != DefinitionCustom {
			return pkg_errors.AccessManagementRoleChangeNotAllowedError{}
		}
		updatedRole := &internal_entity.RoleDefinition{
			Name:        name,
			Description: description,
			Mutable: gorm_model.Mutable{
				UpdatedActorType: string(caller.Type),
				UpdatedActorID:   caller.ID,
			},
		}
		result = tx.Where("id = ?", role.Id).
			Select("name", "description", "updated_date", "updated_actor_type", "updated_actor_id").
			Clauses(clause.Returning{}).
			Updates(updatedRole)
		if result.Error != nil {
			return result.Error
		}
		role.RolePermissions = nil
		if loadErr := preloadOrderedPermissions(tx).Take(&role, role.Id).Error; loadErr != nil {
			return loadErr
		}
		updated = &role
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *accessManagementService) ReduceCustomRolePermissions(ctx context.Context, principal authentication.Principal, roleDefinitionID uint64, permissionDefinitionIDs []uint64, reason, correlationID, confirmationDigest string, preview bool) (*internal_entity.RoleDefinition, *internal_services.RoleReductionPreview, error) {
	_ = reason
	_ = correlationID
	caller := principal.Caller()
	requested := append([]uint64(nil), permissionDefinitionIDs...)
	sort.Slice(requested, func(i, j int) bool { return requested[i] < requested[j] })
	if preview {
		var snapshot *roleReductionSnapshot
		err := s.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
			var err error
			snapshot, err = readRoleReductionSnapshot(tx, principal, roleDefinitionID, requested)
			return err
		}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
		if err != nil {
			return nil, nil, err
		}
		return nil, snapshot.preview(), nil
	}

	var reduced *internal_entity.RoleDefinition
	err := s.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		candidate, err := readRoleReductionSnapshot(tx, principal, roleDefinitionID, requested)
		if err != nil {
			var roleNotFound pkg_errors.AccessManagementRoleNotFoundError
			var roleChangeNotAllowed pkg_errors.AccessManagementRoleChangeNotAllowedError
			if errors.As(err, &roleNotFound) || errors.As(err, &roleChangeNotAllowed) {
				return pkg_errors.AccessManagementReductionStaleError{}
			}
			return err
		}
		if err = lockRoleReductionGraph(tx, principal, candidate); err != nil {
			return err
		}
		locked, err := readRoleReductionSnapshot(tx, principal, roleDefinitionID, requested)
		if err != nil {
			var roleNotFound pkg_errors.AccessManagementRoleNotFoundError
			var roleChangeNotAllowed pkg_errors.AccessManagementRoleChangeNotAllowedError
			if errors.As(err, &roleNotFound) || errors.As(err, &roleChangeNotAllowed) {
				return pkg_errors.AccessManagementReductionStaleError{}
			}
			return err
		}
		if subtle.ConstantTimeCompare([]byte(locked.digest), []byte(confirmationDigest)) != 1 {
			return pkg_errors.AccessManagementReductionStaleError{}
		}
		rowIDs := append([]uint64(nil), locked.requestedPairIDs...)
		result := tx.Model(&internal_entity.RolePermission{}).Where("id IN ? AND status = ?", rowIDs, type_enums.RECORD_ACTIVE).Updates(&internal_entity.RolePermission{
			Mutable: gorm_model.Mutable{
				Status:           type_enums.RECORD_ARCHIEVE,
				UpdatedActorType: string(caller.Type),
				UpdatedActorID:   caller.ID,
			},
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(rowIDs)) {
			return pkg_errors.AccessManagementReductionStaleError{}
		}
		var role internal_entity.RoleDefinition
		if loadErr := preloadOrderedPermissions(tx).Where("id = ?", roleDefinitionID).Take(&role).Error; loadErr != nil {
			return loadErr
		}
		reduced = &role
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return reduced, nil, nil
}

type roleReductionImpact struct {
	AssignmentID     uint64
	OrganizationID   uint64
	ProjectID        uint64
	SubjectType      string
	SubjectID        uint64
	AssignmentStatus string
}

type roleReductionSnapshot struct {
	role             *internal_entity.RoleDefinition
	resultingRole    *internal_entity.RoleDefinition
	impacts          []roleReductionImpact
	requestedIDs     []uint64
	requestedPairIDs []uint64
	digest           string
}

func preloadOrderedPermissions(db *gorm.DB) *gorm.DB {
	return db.Preload("RolePermissions", func(preload *gorm.DB) *gorm.DB {
		return preload.Joins("JOIN permission_definitions ON permission_definitions.id = role_permissions.permission_definition_id").
			Where("role_permissions.status = ? AND permission_definitions.status = ?", type_enums.RECORD_ACTIVE, type_enums.RECORD_ACTIVE).
			Order("permission_definitions.key ASC").Order("permission_definitions.id ASC").Order("role_permissions.id ASC")
	}).Preload("RolePermissions.PermissionDefinition", "status = ?", type_enums.RECORD_ACTIVE)
}

func hasOrganizationPermission(db *gorm.DB, userID, organizationID uint64, permissionKey string) (bool, error) {
	var count int64
	err := db.Raw(`
SELECT COUNT(DISTINCT permission_definition.key)
FROM user_organization_roles assignment
JOIN organizations organization ON organization.id = assignment.organization_id
JOIN role_definitions role_definition ON role_definition.id = assignment.role_definition_id
JOIN role_permissions role_permission ON role_permission.role_definition_id = role_definition.id
JOIN permission_definitions permission_definition ON permission_definition.id = role_permission.permission_definition_id
WHERE assignment.user_auth_id = ? AND assignment.organization_id = ? AND assignment.status = 'ACTIVE'
  AND organization.status = 'ACTIVE'
  AND role_definition.status = 'ACTIVE' AND role_definition.scope_type = 'ORGANIZATION'
  AND role_definition.subject_type = 'USER'
  AND (role_definition.organization_id IS NULL OR role_definition.organization_id = ?)
  AND role_permission.status = 'ACTIVE' AND role_permission.scope_type = 'ORGANIZATION'
  AND permission_definition.status = 'ACTIVE' AND permission_definition.scope_type = 'ORGANIZATION'
  AND permission_definition.key = ?`, userID, organizationID, organizationID, permissionKey).Scan(&count).Error
	return count != 0, err
}

func readRoleReductionSnapshot(tx *gorm.DB, principal authentication.Principal, roleDefinitionID uint64, requested []uint64) (*roleReductionSnapshot, error) {
	tenant := principal.Tenant()
	subject := principal.Subject()
	if subject.Type != authentication.IdentityUser || tenant.OrganizationID == 0 {
		return nil, pkg_errors.AccessManagementPermissionDeniedError{}
	}
	allowed, permissionErr := hasOrganizationPermission(tx, subject.ID, tenant.OrganizationID, "organization.roles.update")
	if permissionErr != nil {
		return nil, permissionErr
	}
	if !allowed {
		return nil, pkg_errors.AccessManagementPermissionDeniedError{}
	}

	var role internal_entity.RoleDefinition
	roleErr := preloadOrderedPermissions(tx).
		Where("id = ? AND organization_id = ? AND status = ?", roleDefinitionID, tenant.OrganizationID, type_enums.RECORD_ACTIVE).
		Take(&role).Error
	if errors.Is(roleErr, gorm.ErrRecordNotFound) {
		return nil, pkg_errors.AccessManagementRoleNotFoundError{}
	}
	if roleErr != nil {
		return nil, roleErr
	}
	if role.DefinitionType != DefinitionCustom {
		return nil, pkg_errors.AccessManagementRoleChangeNotAllowedError{}
	}

	requestedSet := make(map[uint64]struct{}, len(requested))
	for _, id := range requested {
		requestedSet[id] = struct{}{}
	}
	requestedPairIDs := make([]uint64, 0, len(requested))
	remaining := make([]*internal_entity.RolePermission, 0, len(role.RolePermissions))
	for _, pair := range role.RolePermissions {
		if pair == nil || pair.PermissionDefinition == nil {
			continue
		}
		if _, exists := requestedSet[pair.PermissionDefinitionID]; exists {
			requestedPairIDs = append(requestedPairIDs, pair.Id)
			continue
		}
		remaining = append(remaining, pair)
	}
	if len(requestedPairIDs) != len(requested) {
		return nil, pkg_errors.AccessManagementRoleChangeNotAllowedError{}
	}
	sort.Slice(requestedPairIDs, func(i, j int) bool { return requestedPairIDs[i] < requestedPairIDs[j] })

	var impacts []roleReductionImpact
	impactErr := tx.Raw(`
SELECT assignment.id AS assignment_id, assignment.organization_id, 0 AS project_id,
       'USER' AS subject_type, assignment.user_auth_id AS subject_id, assignment.status AS assignment_status
FROM user_organization_roles assignment
WHERE assignment.role_definition_id = ? AND assignment.organization_id = ? AND assignment.status IN ('ACTIVE', 'INVITED')
UNION ALL
SELECT assignment.id AS assignment_id, project.organization_id, assignment.project_id,
       'USER' AS subject_type, assignment.user_auth_id AS subject_id, assignment.status AS assignment_status
FROM user_project_roles assignment
JOIN projects project ON project.id = assignment.project_id
WHERE assignment.role_definition_id = ? AND project.organization_id = ?
  AND assignment.status IN ('ACTIVE', 'INVITED')
UNION ALL
SELECT credential.id AS assignment_id, credential.organization_id, credential.project_id,
       'PROJECT_TOKEN' AS subject_type, credential.id AS subject_id, credential.status AS assignment_status
FROM project_credentials credential
JOIN projects project ON project.id = credential.project_id
WHERE credential.role_definition_id = ? AND credential.organization_id = ?
  AND credential.status = 'ACTIVE' AND project.organization_id = credential.organization_id`,
		role.Id, tenant.OrganizationID, role.Id, tenant.OrganizationID, role.Id, tenant.OrganizationID).
		Scan(&impacts).Error
	if impactErr != nil {
		return nil, impactErr
	}
	sort.Slice(impacts, func(i, j int) bool {
		left, right := impacts[i], impacts[j]
		if left.OrganizationID != right.OrganizationID {
			return left.OrganizationID < right.OrganizationID
		}
		if left.ProjectID != right.ProjectID {
			return left.ProjectID < right.ProjectID
		}
		if left.SubjectType != right.SubjectType {
			return left.SubjectType < right.SubjectType
		}
		if left.SubjectID != right.SubjectID {
			return left.SubjectID < right.SubjectID
		}
		if left.AssignmentStatus != right.AssignmentStatus {
			return left.AssignmentStatus < right.AssignmentStatus
		}
		return left.AssignmentID < right.AssignmentID
	})

	resulting := role
	resulting.RolePermissions = remaining
	snapshot := &roleReductionSnapshot{
		role: &role, resultingRole: &resulting, impacts: impacts,
		requestedIDs: append([]uint64(nil), requested...), requestedPairIDs: requestedPairIDs,
	}
	snapshot.digest = snapshotDigest(snapshot)
	return snapshot, nil
}

func (snapshot *roleReductionSnapshot) preview() *internal_services.RoleReductionPreview {
	impacts := make([]internal_services.RoleReductionImpact, 0, len(snapshot.impacts))
	for _, impact := range snapshot.impacts {
		impacts = append(impacts, internal_services.RoleReductionImpact{
			OrganizationID: impact.OrganizationID, ProjectID: impact.ProjectID,
			SubjectType: impact.SubjectType, SubjectID: impact.SubjectID, AssignmentStatus: impact.AssignmentStatus,
		})
	}
	return &internal_services.RoleReductionPreview{ResultingRole: snapshot.resultingRole, Impacts: impacts, ConfirmationDigest: snapshot.digest}
}

func snapshotDigest(snapshot *roleReductionSnapshot) string {
	var encoded bytes.Buffer
	writeDigestField(&encoded, "role-id", uint64Bytes(snapshot.role.Id))
	organizationID := uint64(0)
	if snapshot.role.OrganizationID != nil {
		organizationID = *snapshot.role.OrganizationID
	}
	writeDigestField(&encoded, "organization-id", uint64Bytes(organizationID))
	writeDigestField(&encoded, "scope", []byte(snapshot.role.ScopeType))
	writeDigestField(&encoded, "subject", []byte(snapshot.role.SubjectType))
	writeDigestField(&encoded, "status", []byte(snapshot.role.Status.String()))
	activePairIDs := make([]uint64, 0, len(snapshot.role.RolePermissions))
	for _, pair := range snapshot.role.RolePermissions {
		activePairIDs = append(activePairIDs, pair.Id)
	}
	sort.Slice(activePairIDs, func(i, j int) bool { return activePairIDs[i] < activePairIDs[j] })
	for _, id := range activePairIDs {
		writeDigestField(&encoded, "active-pair", uint64Bytes(id))
	}
	for _, impact := range snapshot.impacts {
		var fact bytes.Buffer
		fact.Write(uint64Bytes(impact.AssignmentID))
		fact.Write(uint64Bytes(impact.OrganizationID))
		fact.Write(uint64Bytes(impact.ProjectID))
		writeDigestField(&fact, "subject", []byte(impact.SubjectType))
		fact.Write(uint64Bytes(impact.SubjectID))
		writeDigestField(&fact, "assignment-status", []byte(impact.AssignmentStatus))
		writeDigestField(&encoded, "impact", fact.Bytes())
	}
	for _, id := range snapshot.requestedIDs {
		writeDigestField(&encoded, "requested-permission", uint64Bytes(id))
	}
	resultingIDs := make([]uint64, 0, len(snapshot.resultingRole.RolePermissions))
	for _, pair := range snapshot.resultingRole.RolePermissions {
		resultingIDs = append(resultingIDs, pair.PermissionDefinitionID)
	}
	sort.Slice(resultingIDs, func(i, j int) bool { return resultingIDs[i] < resultingIDs[j] })
	for _, id := range resultingIDs {
		writeDigestField(&encoded, "resulting-permission", uint64Bytes(id))
	}
	digest := sha256.Sum256(encoded.Bytes())
	return hex.EncodeToString(digest[:])
}

func writeDigestField(buffer *bytes.Buffer, name string, value []byte) {
	_ = binary.Write(buffer, binary.BigEndian, uint64(len(name)))
	buffer.WriteString(name)
	_ = binary.Write(buffer, binary.BigEndian, uint64(len(value)))
	buffer.Write(value)
}

func uint64Bytes(value uint64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, value)
	return encoded
}

func lockRoleReductionGraph(tx *gorm.DB, principal authentication.Principal, snapshot *roleReductionSnapshot) error {
	tenant := principal.Tenant()
	subject := principal.Subject()
	var organization internal_entity.Organization
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", tenant.OrganizationID, type_enums.RECORD_ACTIVE).Take(&organization).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkg_errors.AccessManagementPermissionDeniedError{}
		}
		return err
	}
	projectIDs := make([]uint64, 0)
	seenProjects := make(map[uint64]struct{})
	userIDs := []uint64{subject.ID}
	seenUsers := map[uint64]struct{}{subject.ID: {}}
	for _, impact := range snapshot.impacts {
		if impact.ProjectID != 0 {
			if _, exists := seenProjects[impact.ProjectID]; !exists {
				seenProjects[impact.ProjectID] = struct{}{}
				projectIDs = append(projectIDs, impact.ProjectID)
			}
		}
		if impact.SubjectType == SubjectUser {
			if _, exists := seenUsers[impact.SubjectID]; !exists {
				seenUsers[impact.SubjectID] = struct{}{}
				userIDs = append(userIDs, impact.SubjectID)
			}
		}
	}
	sort.Slice(projectIDs, func(i, j int) bool { return projectIDs[i] < projectIDs[j] })
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	if len(projectIDs) != 0 {
		var projects []internal_entity.Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ? AND organization_id = ?", projectIDs, tenant.OrganizationID).Order("id ASC").Find(&projects).Error; err != nil {
			return err
		}
		if len(projects) != len(projectIDs) {
			return pkg_errors.AccessManagementReductionStaleError{}
		}
	}
	var organizationAssignments []struct{ ID, RoleDefinitionID uint64 }
	if err := tx.Table("user_organization_roles").Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id, role_definition_id").Where("organization_id = ? AND user_auth_id IN ? AND status IN ?", tenant.OrganizationID, userIDs, []string{"ACTIVE", "INVITED"}).Order("id ASC").Scan(&organizationAssignments).Error; err != nil {
		return err
	}
	var projectAssignments []struct{ ID, RoleDefinitionID uint64 }
	if len(projectIDs) != 0 {
		if err := tx.Table("user_project_roles").Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, role_definition_id").Where("project_id IN ? AND user_auth_id IN ? AND status IN ?", projectIDs, userIDs, []string{"ACTIVE", "INVITED"}).Order("id ASC").Scan(&projectAssignments).Error; err != nil {
			return err
		}
	}
	roleIDs := []uint64{snapshot.role.Id}
	seenRoles := map[uint64]struct{}{snapshot.role.Id: {}}
	for _, assignment := range organizationAssignments {
		if _, exists := seenRoles[assignment.RoleDefinitionID]; !exists {
			seenRoles[assignment.RoleDefinitionID] = struct{}{}
			roleIDs = append(roleIDs, assignment.RoleDefinitionID)
		}
	}
	for _, assignment := range projectAssignments {
		if _, exists := seenRoles[assignment.RoleDefinitionID]; !exists {
			seenRoles[assignment.RoleDefinitionID] = struct{}{}
			roleIDs = append(roleIDs, assignment.RoleDefinitionID)
		}
	}
	sort.Slice(roleIDs, func(i, j int) bool { return roleIDs[i] < roleIDs[j] })
	var roles []internal_entity.RoleDefinition
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", roleIDs).Order("id ASC").Find(&roles).Error; err != nil {
		return err
	}
	var permissionPairs []internal_entity.RolePermission
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("role_definition_id IN ? AND status = ?", roleIDs, type_enums.RECORD_ACTIVE).Order("role_definition_id ASC").Order("id ASC").Find(&permissionPairs).Error; err != nil {
		return err
	}
	credentialIDs := make([]uint64, 0)
	for _, impact := range snapshot.impacts {
		if impact.SubjectType == SubjectProjectToken {
			credentialIDs = append(credentialIDs, impact.SubjectID)
		}
	}
	sort.Slice(credentialIDs, func(i, j int) bool { return credentialIDs[i] < credentialIDs[j] })
	if len(credentialIDs) != 0 {
		var credentials []internal_entity.ProjectCredential
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ? AND status = ?", credentialIDs, type_enums.RECORD_ACTIVE).Order("id ASC").Find(&credentials).Error; err != nil {
			return err
		}
		if len(credentials) != len(credentialIDs) {
			return pkg_errors.AccessManagementReductionStaleError{}
		}
	}
	return nil
}

func (s *accessManagementService) ArchiveCustomRole(ctx context.Context, principal authentication.Principal, roleDefinitionID uint64, reason, correlationID string) error {
	tenant := principal.Tenant()
	subject := principal.Subject()
	caller := principal.Caller()
	err := s.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedOrganization internal_entity.Organization
		organizationLockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", tenant.OrganizationID, type_enums.RECORD_ACTIVE).
			Take(&lockedOrganization).Error
		if errors.Is(organizationLockErr, gorm.ErrRecordNotFound) {
			return pkg_errors.AccessManagementPermissionDeniedError{}
		}
		if organizationLockErr != nil {
			return organizationLockErr
		}
		allowed, permissionErr := hasOrganizationPermission(tx, subject.ID, tenant.OrganizationID, "organization.roles.archive")
		if permissionErr != nil {
			return permissionErr
		}
		if !allowed {
			return pkg_errors.AccessManagementPermissionDeniedError{}
		}
		var result *gorm.DB
		var role internal_entity.RoleDefinition
		roleErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ? AND (organization_id IS NULL OR organization_id = ?)", roleDefinitionID, type_enums.RECORD_ACTIVE, tenant.OrganizationID).
			Take(&role).Error
		if errors.Is(roleErr, gorm.ErrRecordNotFound) {
			return pkg_errors.AccessManagementRoleNotFoundError{}
		}
		if roleErr != nil {
			return roleErr
		}
		if role.DefinitionType != DefinitionCustom {
			return pkg_errors.AccessManagementRoleChangeNotAllowedError{}
		}
		var activeOrInvitedUserCount int64
		if countErr := tx.Raw(`
SELECT COUNT(*) FROM (
  SELECT assignment.id
  FROM user_organization_roles assignment
  WHERE assignment.role_definition_id = ? AND assignment.organization_id = ? AND assignment.status IN ('ACTIVE', 'INVITED')
  UNION ALL
  SELECT assignment.id
  FROM user_project_roles assignment
  JOIN projects project ON project.id = assignment.project_id
  WHERE assignment.role_definition_id = ? AND project.organization_id = ?
    AND project.status = 'ACTIVE' AND assignment.status IN ('ACTIVE', 'INVITED')
) assigned_users`, role.Id, tenant.OrganizationID, role.Id, tenant.OrganizationID).
			Scan(&activeOrInvitedUserCount).Error; countErr != nil {
			return countErr
		}
		if activeOrInvitedUserCount != 0 {
			return pkg_errors.AccessManagementRoleChangeNotAllowedError{}
		}
		var activeCredentialCount int64
		if countErr := tx.Table("project_credentials").
			Where("role_definition_id = ? AND status = ?", role.Id, type_enums.RECORD_ACTIVE).
			Count(&activeCredentialCount).Error; countErr != nil {
			return countErr
		}
		if activeCredentialCount != 0 {
			return pkg_errors.AccessManagementRoleChangeNotAllowedError{}
		}
		var rows []internal_entity.RolePermission
		if queryErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("role_definition_id = ? AND status = ?", role.Id, type_enums.RECORD_ACTIVE).
			Order("id ASC").Find(&rows).Error; queryErr != nil {
			return queryErr
		}
		if len(rows) != 0 {
			rowIDs := make([]uint64, 0, len(rows))
			for _, row := range rows {
				rowIDs = append(rowIDs, row.Id)
			}
			result = tx.Where("id IN ?", rowIDs).Updates(&internal_entity.RolePermission{
				Mutable: gorm_model.Mutable{
					Status:           type_enums.RECORD_ARCHIEVE,
					UpdatedActorType: string(caller.Type),
					UpdatedActorID:   caller.ID,
				},
			})
			if result.Error != nil {
				return result.Error
			}
		}
		return tx.Where("id = ?", role.Id).Updates(&internal_entity.RoleDefinition{
			Mutable: gorm_model.Mutable{
				Status:           type_enums.RECORD_ARCHIEVE,
				UpdatedActorType: string(caller.Type),
				UpdatedActorID:   caller.ID,
			},
		}).Error
	})
	if err != nil {
		return err
	}
	return nil
}
