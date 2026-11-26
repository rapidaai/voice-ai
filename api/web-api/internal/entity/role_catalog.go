package internal_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

// PermissionDefinition is a platform-owned authorization capability.
type PermissionDefinition struct {
	gorm_model.Audited
	gorm_model.Mutable

	// Key is the stable machine-readable capability, such as project.read.
	Key string `json:"key" gorm:"column:key;type:varchar(200);not null"`
	// ScopeType identifies the resource boundary: ORGANIZATION or PROJECT.
	ScopeType   string `json:"scopeType" gorm:"column:scope_type;type:varchar(32);not null"`
	Name        string `json:"name" gorm:"column:name;type:varchar(200);not null"`
	Description string `json:"description" gorm:"column:description;type:varchar(400);not null"`
}

func (PermissionDefinition) TableName() string {
	return "permission_definitions"
}

// RoleDefinition is a named permission set for one scope and subject type.
type RoleDefinition struct {
	gorm_model.Audited
	gorm_model.Mutable

	// OrganizationID is nil for platform roles and set for organization-owned custom roles.
	OrganizationID *uint64 `json:"organizationId" gorm:"column:organization_id;type:bigint"`
	// Key is the immutable machine-readable role identifier within its owner boundary.
	Key         string `json:"key" gorm:"column:key;type:varchar(200);not null"`
	Name        string `json:"name" gorm:"column:name;type:varchar(200);not null"`
	Description string `json:"description" gorm:"column:description;type:varchar(400);not null"`
	// ScopeType defines where the role applies: ORGANIZATION or PROJECT.
	ScopeType string `json:"scopeType" gorm:"column:scope_type;type:varchar(32);not null"`
	// SubjectType defines who can receive the role: USER or PROJECT_TOKEN.
	SubjectType string `json:"subjectType" gorm:"column:subject_type;type:varchar(32);not null"`
	// DefinitionType distinguishes platform BUILT_IN roles from organization CUSTOM roles.
	DefinitionType  string            `json:"definitionType" gorm:"column:definition_type;type:varchar(32);not null"`
	RolePermissions []*RolePermission `json:"rolePermissions,omitempty" gorm:"foreignKey:RoleDefinitionID;references:Id"`
}

func (RoleDefinition) TableName() string {
	return "role_definitions"
}

// RolePermission grants one catalog permission to one role definition.
type RolePermission struct {
	gorm_model.Audited
	gorm_model.Mutable

	// RoleDefinitionID identifies the role receiving the permission.
	RoleDefinitionID uint64 `json:"roleDefinitionId" gorm:"column:role_definition_id;type:bigint;not null"`
	// PermissionDefinitionID identifies the granted catalog capability.
	PermissionDefinitionID uint64                `json:"permissionDefinitionId" gorm:"column:permission_definition_id;type:bigint;not null"`
	PermissionDefinition   *PermissionDefinition `json:"permissionDefinition,omitempty" gorm:"foreignKey:PermissionDefinitionID;references:Id"`
	// ScopeType mirrors both references so the database can reject incompatible grants.
	ScopeType string `json:"scopeType" gorm:"column:scope_type;type:varchar(32);not null"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
