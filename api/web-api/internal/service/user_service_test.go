// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service

import (
	"reflect"
	"testing"
)

func TestUserRoleAssignmentContractUsesRoleDefinitionIDs(t *testing.T) {
	if OrganizationOwnerRoleDefinitionID != 101 {
		t.Fatalf("OrganizationOwnerRoleDefinitionID = %d, want 101", OrganizationOwnerRoleDefinitionID)
	}
	if ProjectAdminRoleDefinitionID != 109 {
		t.Fatalf("ProjectAdminRoleDefinitionID = %d, want 109", ProjectAdminRoleDefinitionID)
	}

	selection := ProjectRoleSelection{
		ProjectID:         7,
		RoleDefinitionIDs: []uint64{109, 201},
	}
	if selection.ProjectID != 7 {
		t.Fatalf("ProjectID = %d, want 7", selection.ProjectID)
	}
	if !reflect.DeepEqual(selection.RoleDefinitionIDs, []uint64{109, 201}) {
		t.Fatalf("RoleDefinitionIDs = %v, want [109 201]", selection.RoleDefinitionIDs)
	}
}
