// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"reflect"
	"testing"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

func TestOrganizationScopeCapabilities(t *testing.T) {
	organizationID := uint64(1)
	scope := &OrganizationScope{
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}

	if got, ok := scope.OrganizationContext(); !ok || got != organizationID {
		t.Fatalf("OrganizationContext() = %d, %v", got, ok)
	}
	if !scope.IsAuthenticated() {
		t.Fatal("IsAuthenticated() = false, want true")
	}
	if _, ok := scope.AuditActor(); ok {
		t.Fatal("AuditActor() ok = true, want false until organization credentials have durable identity")
	}
}

func TestOrganizationScopeRejectsMissingOrZeroContext(t *testing.T) {
	zero := uint64(0)
	for _, scope := range []*OrganizationScope{
		{Status: type_enums.RECORD_ACTIVE.String()},
		{OrganizationId: &zero, Status: type_enums.RECORD_ACTIVE.String()},
	} {
		if scope.IsAuthenticated() {
			t.Fatal("IsAuthenticated() = true, want false")
		}
		if _, err := RequireOrganization(scope); err == nil {
			t.Fatal("RequireOrganization() error = nil")
		}
	}
}

func TestOrganizationScopeDoesNotExposeFakeCapabilities(t *testing.T) {
	typeOfScope := reflect.TypeOf(&OrganizationScope{})
	for _, method := range []string{
		"GetUserId", "HasUser", "UserIdentity",
		"GetCurrentProjectId", "HasProject", "ProjectContext",
	} {
		if _, ok := typeOfScope.MethodByName(method); ok {
			t.Fatalf("OrganizationScope unexpectedly exposes %s", method)
		}
	}
}

var (
	_ AuthenticationPrinciple     = (*OrganizationScope)(nil)
	_ OrganizationContextProvider = (*OrganizationScope)(nil)
	_ ActorIdentityProvider       = (*OrganizationScope)(nil)
)
