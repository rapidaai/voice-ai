// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication_test

import (
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

func userProjectPrincipal(t *testing.T, permissions ...authentication.PermissionKey) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: 20, ProjectID: 30},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-8"}},
		permissions,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func userRole(id uint64, scope string, permission authentication.Permission) authentication.Role {
	return authentication.Role{
		ID: id, Key: "role", Name: "Role", Scope: scope,
		Subject: authentication.RoleSubjectUser, DefinitionType: authentication.RoleDefinitionBuiltIn,
		Permissions: []authentication.Permission{permission},
	}
}

func newAuthentication(principal authentication.Principal, user *authentication.User, organizations []authentication.Organization) (authentication.Authentication, error) {
	actor, delegated := principal.Actor()
	if delegated {
		return authentication.NewDelegatedAuthentication(principal.Subject(), actor, principal.Tenant(), principal.Proof(), user, organizations)
	}
	return authentication.NewDirectAuthentication(principal.Subject(), principal.Tenant(), principal.Proof(), user, organizations)
}

func TestNewAuthenticationBuildsImmutableOrderedAggregate(t *testing.T) {
	organizations := []authentication.Organization{
		{
			ID: 20, Name: "Organization",
			Roles: []authentication.Role{
				userRole(12, authentication.RoleScopeOrganization, authentication.Permission{ID: 2, Key: "organization.users.read", Scope: authentication.RoleScopeOrganization, Name: "Read users"}),
				userRole(11, authentication.RoleScopeOrganization, authentication.Permission{ID: 1, Key: "organization.projects.read", Scope: authentication.RoleScopeOrganization, Name: "Read projects"}),
			},
			BillingPlan: &authentication.BillingPlan{
				ID: 40, Name: "Growth", Slug: "growth", IsActive: true, Currency: "USD",
				Quotas: []authentication.BillingPlanQuota{{ID: 2, ResourceType: "minutes", QuotaLimit: 1000, Unit: "minute"}, {ID: 1, ResourceType: "projects", QuotaLimit: 20, Unit: "project"}},
			},
			Projects: []authentication.Project{
				{ID: 31, Name: "Later", Roles: []authentication.Role{userRole(14, authentication.RoleScopeProject, authentication.Permission{ID: 4, Key: "project.assistants.update", Scope: authentication.RoleScopeProject, Name: "Update assistants"})}},
				{ID: 30, Name: "Selected", Roles: []authentication.Role{userRole(13, authentication.RoleScopeProject, authentication.Permission{ID: 3, Key: "project.assistants.read", Scope: authentication.RoleScopeProject, Name: "Read assistants"})}},
			},
		},
	}
	principal := userProjectPrincipal(t, "project.assistants.read", "organization.users.read", "organization.projects.read")
	value, err := newAuthentication(principal, &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}, organizations)
	if err != nil {
		t.Fatal(err)
	}

	organizations[0].Name = "changed input"
	organizations[0].Roles[0].Permissions[0].Key = "changed.input"
	organizations[0].BillingPlan.Quotas[0].ResourceType = "changed-input"

	if value.Principal().Subject().ID != 7 {
		t.Fatalf("Principal() = %+v", value.Principal())
	}
	user, ok := value.User()
	if !ok || user.Name != "Ada" {
		t.Fatalf("User() = %+v, %t", user, ok)
	}
	if got := value.Permissions(); !reflect.DeepEqual(got, []authentication.PermissionKey{"organization.projects.read", "organization.users.read", "project.assistants.read"}) {
		t.Fatalf("Permissions() = %v", got)
	}

	returned := value.Organizations()
	if len(returned) != 1 || returned[0].Roles[0].ID != 11 || returned[0].Projects[0].ID != 30 || returned[0].BillingPlan.Quotas[0].ID != 1 {
		t.Fatalf("Organizations() = %+v", returned)
	}
	returned[0].Name = "changed output"
	returned[0].Roles[0].Permissions[0].Key = "changed.output"
	returned[0].BillingPlan.Quotas[0].ResourceType = "changed-output"
	selectedOrganization, ok := value.SelectedOrganization()
	if !ok || selectedOrganization.Name != "Organization" || selectedOrganization.Roles[0].Permissions[0].Key != "organization.projects.read" || selectedOrganization.BillingPlan.Quotas[0].ResourceType != "projects" {
		t.Fatalf("SelectedOrganization() = %+v, %t", selectedOrganization, ok)
	}
	selectedProject, ok := value.SelectedProject()
	if !ok || selectedProject.ID != 30 {
		t.Fatalf("SelectedProject() = %+v, %t", selectedProject, ok)
	}
}

func TestNewAuthenticationSupportsOnboardingAndDirectService(t *testing.T) {
	onboarding, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7}, authentication.Tenant{},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-8"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = newAuthentication(onboarding, &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}, nil); err != nil {
		t.Fatal(err)
	}

	service, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: 9, Issuer: "workflow-api"}, authentication.Tenant{},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = newAuthentication(service, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestNewAuthenticationSupportsCredentialAndDelegatedSubjects(t *testing.T) {
	projectPrincipal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 9},
		authentication.Tenant{OrganizationID: 20, ProjectID: 30},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "9"}},
		[]authentication.PermissionKey{"project.assistants.read"},
	)
	if err != nil {
		t.Fatal(err)
	}
	projectGraph := []authentication.Organization{{
		ID: 20, Name: "Organization",
		Projects: []authentication.Project{{ID: 30, Name: "Project", Roles: []authentication.Role{{
			ID: 12, Key: "token_reader", Name: "Token Reader", Scope: authentication.RoleScopeProject,
			Subject: authentication.RoleSubjectProjectToken, DefinitionType: authentication.RoleDefinitionBuiltIn,
			Permissions: []authentication.Permission{{ID: 2, Key: "project.assistants.read", Scope: authentication.RoleScopeProject, Name: "Read assistants"}},
		}}}},
	}}
	if _, err = newAuthentication(projectPrincipal, nil, projectGraph); err != nil {
		t.Fatal(err)
	}

	organizationPrincipal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 10},
		authentication.Tenant{OrganizationID: 20},
		authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "10"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = newAuthentication(organizationPrincipal, nil, []authentication.Organization{{ID: 20, Name: "Organization"}}); err != nil {
		t.Fatal(err)
	}

	delegatedPrincipal, err := authentication.NewDelegatedPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.ServiceActor{ID: 11, Issuer: "workflow-api"},
		authentication.Tenant{OrganizationID: 20, ProjectID: 30},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}},
		[]authentication.PermissionKey{"organization.projects.read", "project.assistants.read"},
	)
	if err != nil {
		t.Fatal(err)
	}
	delegatedGraph := []authentication.Organization{{
		ID: 20, Name: "Organization",
		Roles:    []authentication.Role{userRole(11, authentication.RoleScopeOrganization, authentication.Permission{ID: 1, Key: "organization.projects.read", Scope: authentication.RoleScopeOrganization, Name: "Read projects"})},
		Projects: []authentication.Project{{ID: 30, Name: "Project", Roles: []authentication.Role{userRole(12, authentication.RoleScopeProject, authentication.Permission{ID: 2, Key: "project.assistants.read", Scope: authentication.RoleScopeProject, Name: "Read assistants"})}}},
	}}
	if _, err = newAuthentication(delegatedPrincipal, &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}, delegatedGraph); err != nil {
		t.Fatal(err)
	}
}

func TestNewAuthenticationDerivesUniquePermissionKeysAcrossRoles(t *testing.T) {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7}, authentication.Tenant{OrganizationID: 20},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-8"}},
		[]authentication.PermissionKey{"organization.projects.read"},
	)
	if err != nil {
		t.Fatal(err)
	}
	permission := authentication.Permission{ID: 1, Key: "organization.projects.read", Scope: authentication.RoleScopeOrganization, Name: "Read projects"}
	value, err := newAuthentication(principal, &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}, []authentication.Organization{{
		ID: 20, Name: "Organization", Roles: []authentication.Role{
			userRole(11, authentication.RoleScopeOrganization, permission),
			userRole(12, authentication.RoleScopeOrganization, permission),
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := value.Permissions(); !reflect.DeepEqual(got, []authentication.PermissionKey{"organization.projects.read"}) {
		t.Fatalf("Permissions() = %v", got)
	}
}

func TestNewAuthenticationRejectsInvalidGraphs(t *testing.T) {
	validOrganizationPermission := authentication.Permission{ID: 1, Key: "organization.projects.read", Scope: authentication.RoleScopeOrganization, Name: "Read projects"}
	validProjectPermission := authentication.Permission{ID: 2, Key: "project.assistants.read", Scope: authentication.RoleScopeProject, Name: "Read assistants"}
	valid := authentication.Organization{
		ID: 20, Name: "Organization",
		Roles:    []authentication.Role{userRole(11, authentication.RoleScopeOrganization, validOrganizationPermission)},
		Projects: []authentication.Project{{ID: 30, Name: "Project", Roles: []authentication.Role{userRole(12, authentication.RoleScopeProject, validProjectPermission)}}},
	}
	validPrincipal := userProjectPrincipal(t, "organization.projects.read", "project.assistants.read")

	tests := []struct {
		name          string
		principal     authentication.Principal
		user          *authentication.User
		organizations []authentication.Organization
	}{
		{name: "missing user", principal: validPrincipal, organizations: []authentication.Organization{valid}},
		{name: "mismatched user", principal: validPrincipal, user: &authentication.User{ID: 8, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}, organizations: []authentication.Organization{valid}},
		{name: "duplicate organization", principal: validPrincipal, user: &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}, organizations: []authentication.Organization{valid, valid}},
		{
			name:          "organization role with project key and matching cache",
			principal:     userProjectPrincipal(t, "project.assistants.read", "project.resources.write"),
			user:          &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"},
			organizations: []authentication.Organization{{ID: 20, Name: "Organization", Roles: []authentication.Role{userRole(11, authentication.RoleScopeOrganization, authentication.Permission{ID: 1, Key: "project.resources.write", Scope: authentication.RoleScopeOrganization, Name: "Wrong"})}, Projects: valid.Projects}},
		},
		{
			name:          "project role with organization key and matching cache",
			principal:     userProjectPrincipal(t, "organization.projects.read", "organization.users.read"),
			user:          &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"},
			organizations: []authentication.Organization{{ID: 20, Name: "Organization", Roles: valid.Roles, Projects: []authentication.Project{{ID: 30, Name: "Project", Roles: []authentication.Role{userRole(12, authentication.RoleScopeProject, authentication.Permission{ID: 2, Key: "organization.users.read", Scope: authentication.RoleScopeProject, Name: "Wrong"})}}}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newAuthentication(test.principal, test.user, test.organizations)
			if _, ok := err.(pkg_errors.AuthenticationInvalidError); !ok {
				t.Fatalf("error = %T, want AuthenticationInvalidError", err)
			}
		})
	}
}

func TestNewAuthenticationAcceptsOnlyDocumentedUnlimitedQuotaSentinel(t *testing.T) {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: 20},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-7"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	user := &authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}
	organization := authentication.Organization{ID: 20, Name: "Organization", BillingPlan: &authentication.BillingPlan{
		ID: 1, Name: "Enterprise", Slug: "enterprise", IsActive: true, Currency: "usd",
		Quotas: []authentication.BillingPlanQuota{{ID: 2, ResourceType: "projects", QuotaLimit: -1, Unit: "count"}},
	}}

	if _, err := newAuthentication(principal, user, []authentication.Organization{organization}); err != nil {
		t.Fatalf("unlimited quota error = %v", err)
	}
	organization.BillingPlan.Quotas[0].QuotaLimit = -2
	if _, err := newAuthentication(principal, user, []authentication.Organization{organization}); err == nil {
		t.Fatal("quota below the unlimited sentinel was accepted")
	}
}
