// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"reflect"
	"testing"
)

func TestServiceScopeDelegatedContext(t *testing.T) {
	userID := uint64(1)
	organizationID := uint64(2)
	projectID := uint64(3)
	scope := &ServiceScope{UserId: &userID, OrganizationId: &organizationID, ProjectId: &projectID}

	delegatedContext, ok := scope.DelegatedContext()
	if !ok {
		t.Fatal("DelegatedContext() ok = false")
	}
	if delegatedContext.OrganizationID != organizationID || delegatedContext.UserID == nil || *delegatedContext.UserID != userID || delegatedContext.ProjectID == nil || *delegatedContext.ProjectID != projectID {
		t.Fatalf("DelegatedContext() = %+v", delegatedContext)
	}
	if !scope.IsAuthenticated() {
		t.Fatal("IsAuthenticated() = false, want true")
	}

	*delegatedContext.UserID = 99
	if *scope.UserId != userID {
		t.Fatal("DelegatedContext() returned an aliased user ID")
	}
	if _, ok := scope.AuditActor(); ok {
		t.Fatal("AuditActor() ok = true, want false until service assertions have stable identity")
	}
}

func TestServiceScopeRejectsMalformedDelegatedContext(t *testing.T) {
	zero := uint64(0)
	organizationID := uint64(2)
	for _, scope := range []*ServiceScope{
		{},
		{OrganizationId: &zero},
		{OrganizationId: &organizationID, UserId: &zero},
		{OrganizationId: &organizationID, ProjectId: &zero},
	} {
		if scope.IsAuthenticated() {
			t.Fatal("IsAuthenticated() = true, want false")
		}
		if _, err := ResolveDelegatedContext(scope); err == nil {
			t.Fatal("ResolveDelegatedContext() error = nil")
		}
	}
}

func TestServiceScopeDoesNotExposeIdentityOrTenantProviders(t *testing.T) {
	typeOfScope := reflect.TypeOf(&ServiceScope{})
	for _, method := range []string{
		"GetUserId", "HasUser", "UserIdentity",
		"GetCurrentOrganizationId", "HasOrganization", "OrganizationContext",
		"GetCurrentProjectId", "HasProject", "ProjectContext",
	} {
		if _, ok := typeOfScope.MethodByName(method); ok {
			t.Fatalf("ServiceScope unexpectedly exposes %s", method)
		}
	}
}

var (
	_ AuthenticationPrinciple  = (*ServiceScope)(nil)
	_ DelegatedContextProvider = (*ServiceScope)(nil)
	_ ActorIdentityProvider    = (*ServiceScope)(nil)
)
