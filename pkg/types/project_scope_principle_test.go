// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales/rapida.ai for commercial usage.
package types

import (
	"reflect"
	"testing"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

func TestProjectScopeCapabilities(t *testing.T) {
	credentialID := uint64(42)
	projectID := uint64(2)
	organizationID := uint64(1)
	scope := &ProjectScope{
		CredentialId:   &credentialID,
		ProjectId:      &projectID,
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}

	projectContext, ok := scope.ProjectContext()
	if !ok || projectContext != (ProjectContext{OrganizationID: organizationID, ProjectID: projectID}) {
		t.Fatalf("ProjectContext() = %+v, %v", projectContext, ok)
	}
	if got, ok := scope.OrganizationContext(); !ok || got != organizationID {
		t.Fatalf("OrganizationContext() = %d, %v", got, ok)
	}
	if !scope.IsAuthenticated() {
		t.Fatal("IsAuthenticated() = false, want true")
	}
	actor, ok := scope.AuditActor()
	if !ok || actor != (ActorIdentity{Type: ActorTypeProject, ID: "42"}) {
		t.Fatalf("AuditActor() = %+v, %v", actor, ok)
	}
}

func TestProjectScopeRejectsMissingOrZeroContext(t *testing.T) {
	active := type_enums.RECORD_ACTIVE.String()
	zero := uint64(0)
	organizationID := uint64(1)
	projectID := uint64(2)
	tests := []struct {
		name  string
		scope *ProjectScope
	}{
		{name: "missing organization", scope: &ProjectScope{ProjectId: &projectID, Status: active}},
		{name: "zero organization", scope: &ProjectScope{ProjectId: &projectID, OrganizationId: &zero, Status: active}},
		{name: "missing project", scope: &ProjectScope{OrganizationId: &organizationID, Status: active}},
		{name: "zero project", scope: &ProjectScope{ProjectId: &zero, OrganizationId: &organizationID, Status: active}},
		{name: "inactive", scope: &ProjectScope{ProjectId: &projectID, OrganizationId: &organizationID}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.scope.IsAuthenticated() {
				t.Fatal("IsAuthenticated() = true, want false")
			}
			if _, err := RequireProject(test.scope); err == nil {
				t.Fatal("RequireProject() error = nil")
			}
		})
	}
}

func TestProjectScopeDoesNotExposeUserIdentity(t *testing.T) {
	typeOfScope := reflect.TypeOf(&ProjectScope{})
	for _, method := range []string{"GetUserId", "HasUser", "UserIdentity"} {
		if _, ok := typeOfScope.MethodByName(method); ok {
			t.Fatalf("ProjectScope unexpectedly exposes %s", method)
		}
	}
}

var (
	_ AuthenticationPrinciple     = (*ProjectScope)(nil)
	_ OrganizationContextProvider = (*ProjectScope)(nil)
	_ ProjectContextProvider      = (*ProjectScope)(nil)
	_ ActorIdentityProvider       = (*ProjectScope)(nil)
)
