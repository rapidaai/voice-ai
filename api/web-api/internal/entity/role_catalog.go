package internal_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

type PermissionDefinition struct {
	gorm_model.Audited
	gorm_model.Mutable

	Key           string `json:"key" gorm:"column:key;type:varchar(200);not null"`
	ScopeType     string `json:"scopeType" gorm:"column:scope_type;type:varchar(32);not null"`
	Name          string `json:"name" gorm:"column:name;type:varchar(200);not null"`
	Description   string `json:"description" gorm:"column:description;type:varchar(400);not null"`
	TokenEligible bool   `json:"tokenEligible" gorm:"column:token_eligible;not null"`
}

func (PermissionDefinition) TableName() string {
	return "permission_definitions"
}

type RoleDefinition struct {
	gorm_model.Audited
	gorm_model.Mutable

	OrganizationID *uint64 `json:"organizationId" gorm:"column:organization_id;type:bigint"`
	Key            string  `json:"key" gorm:"column:key;type:varchar(200);not null"`
	Name           string  `json:"name" gorm:"column:name;type:varchar(200);not null"`
	Description    string  `json:"description" gorm:"column:description;type:varchar(400);not null"`
	ScopeType      string  `json:"scopeType" gorm:"column:scope_type;type:varchar(32);not null"`
	SubjectType    string  `json:"subjectType" gorm:"column:subject_type;type:varchar(32);not null"`
	DefinitionType string  `json:"definitionType" gorm:"column:definition_type;type:varchar(32);not null"`
}

func (RoleDefinition) TableName() string {
	return "role_definitions"
}

type RolePermission struct {
	gorm_model.Audited
	gorm_model.Mutable

	RoleDefinitionID       uint64 `json:"roleDefinitionId" gorm:"column:role_definition_id;type:bigint;not null"`
	PermissionDefinitionID uint64 `json:"permissionDefinitionId" gorm:"column:permission_definition_id;type:bigint;not null"`
	ScopeType              string `json:"scopeType" gorm:"column:scope_type;type:varchar(32);not null"`
	SubjectType            string `json:"subjectType" gorm:"column:subject_type;type:varchar(32);not null"`
	TokenEligible          bool   `json:"tokenEligible" gorm:"column:token_eligible;not null"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
