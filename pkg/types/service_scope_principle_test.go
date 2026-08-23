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
	organizationID := uint64(2)
	projectID := uint64(3)
	scope := &ServiceScope{
		ActorId:        4,
		Issuer:         "assistant-api",
		Audience:       ServiceAssertionAudience,
		OrganizationId: &organizationID,
		ProjectId:      &projectID,
	}

	delegatedContext, ok := scope.DelegatedContext()
	if !ok {
		t.Fatal("DelegatedContext() ok = false")
	}
	if delegatedContext.OrganizationID != organizationID || delegatedContext.UserID != nil || delegatedContext.ProjectID == nil || *delegatedContext.ProjectID != projectID {
		t.Fatalf("DelegatedContext() = %+v", delegatedContext)
	}
	if !scope.IsAuthenticated() {
		t.Fatal("IsAuthenticated() = false, want true")
	}

	if actor, ok := scope.AuditActor(); !ok || actor != (ActorIdentity{Type: ActorTypeService, ID: 4}) {
		t.Fatalf("AuditActor() = %+v, %v", actor, ok)
	}
}

func TestServiceScopeAuthenticationUsesDelegatedActor(t *testing.T) {
	organizationID := uint64(2)
	projectID := uint64(3)
	tests := []struct {
		name      string
		authType  AuthType
		actorType ActorType
		projectID *uint64
	}{
		{name: "user", authType: AuthTypeUser, actorType: ActorTypeUser, projectID: &projectID},
		{name: "project", authType: AuthTypeProject, actorType: ActorTypeProject, projectID: &projectID},
		{name: "organization", authType: AuthTypeOrg, actorType: ActorTypeOrganization},
		{name: "service", authType: AuthTypeService, actorType: ActorTypeService},
		{name: "system", authType: AuthTypeSystem, actorType: ActorTypeSystem},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actorID := uint64(5)
			scope := &ServiceScope{
				ActorId:           4,
				Issuer:            "assistant-api",
				Audience:          ServiceAssertionAudience,
				DelegatedAuthType: test.authType,
				DelegatedActorId:  &actorID,
				OrganizationId:    &organizationID,
				ProjectId:         test.projectID,
			}
			auth, err := scope.Authentication()
			if err != nil {
				t.Fatalf("Authentication() error = %v", err)
			}
			actor, err := auth.Actor()
			if err != nil || actor != (ActorIdentity{Type: test.actorType, ID: actorID}) {
				t.Fatalf("Actor() = %+v, %v", actor, err)
			}
			caller, err := auth.Caller()
			if err != nil || caller != (ActorIdentity{Type: ActorTypeService, ID: 4}) {
				t.Fatalf("Caller() = %+v, %v", caller, err)
			}
			if auth.Type() != test.authType {
				t.Fatalf("Type() = %q, want %q", auth.Type(), test.authType)
			}
		})
	}
}

func TestServiceScopeAuthenticationUsesImmediateServiceActor(t *testing.T) {
	organizationID := uint64(2)
	scope := &ServiceScope{
		ActorId:        4,
		Issuer:         "assistant-api",
		Audience:       ServiceAssertionAudience,
		OrganizationId: &organizationID,
	}
	auth, err := scope.Authentication()
	if err != nil {
		t.Fatalf("Authentication() error = %v", err)
	}
	actor, err := auth.Actor()
	if err != nil || actor != (ActorIdentity{Type: ActorTypeService, ID: 4}) {
		t.Fatalf("Actor() = %+v, %v", actor, err)
	}
	caller, err := auth.Caller()
	if err != nil || caller != actor {
		t.Fatalf("Caller() = %+v, %v; want %+v", caller, err, actor)
	}
}

func TestServiceScopeRejectsMalformedDelegatedContext(t *testing.T) {
	zero := uint64(0)
	organizationID := uint64(2)
	for _, scope := range []*ServiceScope{
		{},
		{OrganizationId: &zero},
		{OrganizationId: &organizationID, ProjectId: &zero},
		{ActorId: 1, OrganizationId: &organizationID},
		{ActorId: 1, Issuer: "assistant-api", Audience: ServiceAssertionAudience, OrganizationId: &organizationID, DelegatedAuthType: AuthTypeUser},
		{ActorId: 1, Issuer: "assistant-api", Audience: ServiceAssertionAudience, OrganizationId: &organizationID, DelegatedActorId: &organizationID},
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
