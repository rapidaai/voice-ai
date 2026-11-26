// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_entity

import (
	"reflect"
	"strings"
	"testing"
)

func TestUserRoleAssignmentsUseRoleDefinitionReferences(t *testing.T) {
	for _, assignment := range []any{UserOrganizationRole{}, UserProjectRole{}} {
		assignmentType := reflect.TypeOf(assignment)
		if _, ok := assignmentType.FieldByName("Role"); ok {
			t.Fatalf("%s still exposes a legacy Role field", assignmentType.Name())
		}
		roleID, ok := assignmentType.FieldByName("RoleDefinitionID")
		if !ok || !strings.Contains(roleID.Tag.Get("gorm"), "column:role_definition_id") {
			t.Fatalf("%s RoleDefinitionID tag = %q", assignmentType.Name(), roleID.Tag.Get("gorm"))
		}
		roleDefinition, ok := assignmentType.FieldByName("RoleDefinition")
		if !ok || !strings.Contains(roleDefinition.Tag.Get("gorm"), "foreignKey:RoleDefinitionID") {
			t.Fatalf("%s RoleDefinition relation tag = %q", assignmentType.Name(), roleDefinition.Tag.Get("gorm"))
		}
	}
}
