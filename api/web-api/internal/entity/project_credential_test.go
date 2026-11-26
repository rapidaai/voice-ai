// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_entity

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestProjectCredentialRoleFields(t *testing.T) {
	credential := ProjectCredential{
		RoleDefinitionID: 117,
		RoleSubjectType:  "PROJECT_TOKEN",
	}
	encoded, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal project credential: %v", err)
	}
	var fields map[string]any
	if err = json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal project credential: %v", err)
	}
	if fields["roleDefinitionId"] != float64(117) || fields["roleSubjectType"] != "PROJECT_TOKEN" {
		t.Fatalf("project credential role fields = %v", fields)
	}

	credentialType := reflect.TypeOf(ProjectCredential{})
	roleID, ok := credentialType.FieldByName("RoleDefinitionID")
	if !ok || !strings.Contains(roleID.Tag.Get("gorm"), "column:role_definition_id") {
		t.Fatalf("RoleDefinitionID gorm tag = %q", roleID.Tag.Get("gorm"))
	}
	roleSubject, ok := credentialType.FieldByName("RoleSubjectType")
	if !ok || !strings.Contains(roleSubject.Tag.Get("gorm"), "column:role_subject_type") {
		t.Fatalf("RoleSubjectType gorm tag = %q", roleSubject.Tag.Get("gorm"))
	}
	roleDefinition, ok := credentialType.FieldByName("RoleDefinition")
	if !ok || !strings.Contains(roleDefinition.Tag.Get("gorm"), "foreignKey:RoleDefinitionID") {
		t.Fatalf("RoleDefinition gorm tag = %q", roleDefinition.Tag.Get("gorm"))
	}
	if roleDefinition.Type != reflect.TypeOf((*RoleDefinition)(nil)) {
		t.Fatalf("RoleDefinition type = %v, want *RoleDefinition", roleDefinition.Type)
	}
}
