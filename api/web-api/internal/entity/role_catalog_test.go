package internal_entity

import (
	"reflect"
	"strings"
	"testing"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

func TestRoleCatalogTableNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "permission definition", got: (PermissionDefinition{}).TableName(), want: "permission_definitions"},
		{name: "role definition", got: (RoleDefinition{}).TableName(), want: "role_definitions"},
		{name: "role permission", got: (RolePermission{}).TableName(), want: "role_permissions"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("TableName() = %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestRoleCatalogUsesRepositoryAuditEntities(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(PermissionDefinition{}),
		reflect.TypeOf(RoleDefinition{}),
		reflect.TypeOf(RolePermission{}),
	}

	for _, entityType := range types {
		audited, ok := entityType.FieldByName("Audited")
		if !ok || !audited.Anonymous || audited.Type != reflect.TypeOf(gorm_model.Audited{}) {
			t.Fatalf("%s must embed gorm_model.Audited", entityType.Name())
		}
		mutable, ok := entityType.FieldByName("Mutable")
		if !ok || !mutable.Anonymous || mutable.Type != reflect.TypeOf(gorm_model.Mutable{}) {
			t.Fatalf("%s must embed gorm_model.Mutable", entityType.Name())
		}
	}
}

func TestRoleCatalogPersistedColumns(t *testing.T) {
	tests := []struct {
		entity any
		fields map[string]string
	}{
		{
			entity: PermissionDefinition{},
			fields: map[string]string{
				"Key": "key", "ScopeType": "scope_type", "Name": "name",
				"Description": "description",
			},
		},
		{
			entity: RoleDefinition{},
			fields: map[string]string{
				"OrganizationID": "organization_id", "Key": "key", "Name": "name",
				"Description": "description", "ScopeType": "scope_type",
				"SubjectType": "subject_type", "DefinitionType": "definition_type",
			},
		},
		{
			entity: RolePermission{},
			fields: map[string]string{
				"RoleDefinitionID":       "role_definition_id",
				"PermissionDefinitionID": "permission_definition_id",
				"ScopeType":              "scope_type",
			},
		},
	}
	for _, test := range tests {
		entityType := reflect.TypeOf(test.entity)
		for fieldName, columnName := range test.fields {
			field, ok := entityType.FieldByName(fieldName)
			if !ok {
				t.Fatalf("%s.%s is missing", entityType.Name(), fieldName)
			}
			if !strings.Contains(field.Tag.Get("gorm"), "column:"+columnName) {
				t.Fatalf("%s.%s gorm tag = %q, want column:%s", entityType.Name(), fieldName, field.Tag.Get("gorm"), columnName)
			}
		}
	}

	organizationID, ok := reflect.TypeOf(RoleDefinition{}).FieldByName("OrganizationID")
	if !ok || organizationID.Type != reflect.TypeOf((*uint64)(nil)) {
		t.Fatal("RoleDefinition.OrganizationID must be nullable *uint64")
	}
	for _, entity := range []any{RoleDefinition{}, RolePermission{}} {
		entityType := reflect.TypeOf(entity)
		for _, fieldName := range []string{
			"ReductionReason", "ReductionCorrelationID", "ReductionAssuranceCondition",
			"CreatedActorDisplayName", "UpdatedActorDisplayName", "TokenEligible",
		} {
			if _, ok := entityType.FieldByName(fieldName); ok {
				t.Fatalf("%s.%s must not exist", entityType.Name(), fieldName)
			}
		}
	}
	if _, ok := reflect.TypeOf(RolePermission{}).FieldByName("SubjectType"); ok {
		t.Fatal("RolePermission.SubjectType must not exist")
	}
}

func TestRoleCatalogAssociationsAreOptionalPointers(t *testing.T) {
	rolePermissions, ok := reflect.TypeOf(RoleDefinition{}).FieldByName("RolePermissions")
	if !ok || rolePermissions.Type != reflect.TypeOf([]*RolePermission(nil)) {
		t.Fatalf("RoleDefinition.RolePermissions type = %v, want []*RolePermission", rolePermissions.Type)
	}
	if !strings.Contains(rolePermissions.Tag.Get("gorm"), "foreignKey:RoleDefinitionID") {
		t.Fatalf("RoleDefinition.RolePermissions gorm tag = %q", rolePermissions.Tag.Get("gorm"))
	}

	permissionDefinition, ok := reflect.TypeOf(RolePermission{}).FieldByName("PermissionDefinition")
	if !ok || permissionDefinition.Type != reflect.TypeOf((*PermissionDefinition)(nil)) {
		t.Fatalf("RolePermission.PermissionDefinition type = %v, want *PermissionDefinition", permissionDefinition.Type)
	}
	if !strings.Contains(permissionDefinition.Tag.Get("gorm"), "foreignKey:PermissionDefinitionID") {
		t.Fatalf("RolePermission.PermissionDefinition gorm tag = %q", permissionDefinition.Tag.Get("gorm"))
	}
}
