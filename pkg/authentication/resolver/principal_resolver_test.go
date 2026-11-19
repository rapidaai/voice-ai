// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resolver_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/resolver"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

type credentialAuthenticationService struct {
	authentication authentication.Authentication
	err            error
	token          string
	userID         uint64
	organizationID uint64
	projectID      uint64
	calls          int
	onResolve      func()
}

func (service *credentialAuthenticationService) ResolveUserCredential(_ context.Context, token string, userID uint64, organizationID uint64, projectID uint64) (authentication.Authentication, error) {
	service.calls++
	service.token, service.userID, service.organizationID, service.projectID = token, userID, organizationID, projectID
	if service.onResolve != nil {
		service.onResolve()
	}
	return service.authentication, service.err
}

func (service *credentialAuthenticationService) ResolveProjectCredential(_ context.Context, token string) (authentication.Authentication, error) {
	service.calls++
	service.token = token
	if service.onResolve != nil {
		service.onResolve()
	}
	return service.authentication, service.err
}

func (service *credentialAuthenticationService) ResolveOrganizationCredential(_ context.Context, token string) (authentication.Authentication, error) {
	service.calls++
	service.token = token
	if service.onResolve != nil {
		service.onResolve()
	}
	return service.authentication, service.err
}

func resolvedUserAuthentication(t *testing.T, tenant authentication.Tenant) authentication.Authentication {
	t.Helper()
	organizations := []authentication.Organization(nil)
	if tenant.OrganizationID != 0 {
		organization := authentication.Organization{ID: tenant.OrganizationID, Name: "Organization", Roles: []authentication.Role{{
			ID: 11, Key: "organization_reader", Name: "Organization Reader", Scope: authentication.RoleScopeOrganization,
			Subject: authentication.RoleSubjectUser, DefinitionType: authentication.RoleDefinitionBuiltIn,
			Permissions: []authentication.Permission{{ID: 1, Key: "organization.projects.read", Scope: authentication.RoleScopeOrganization, Name: "Read projects"}},
		}}}
		if tenant.ProjectID != 0 {
			organization.Projects = []authentication.Project{{ID: tenant.ProjectID, Name: "Project", Roles: []authentication.Role{{
				ID: 12, Key: "project_reader", Name: "Project Reader", Scope: authentication.RoleScopeProject,
				Subject: authentication.RoleSubjectUser, DefinitionType: authentication.RoleDefinitionBuiltIn,
				Permissions: []authentication.Permission{{ID: 2, Key: "project.resources.read", Scope: authentication.RoleScopeProject, Name: "Read resources"}},
			}}}}
		}
		organizations = []authentication.Organization{organization}
	}
	value, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7}, tenant,
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "8"}},
		&authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"}, organizations,
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func resolvedProjectAuthentication(t *testing.T) authentication.Authentication {
	t.Helper()
	value, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 7},
		authentication.Tenant{OrganizationID: 8, ProjectID: 9},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "7"}},
		nil, []authentication.Organization{{ID: 8, Name: "Organization", Projects: []authentication.Project{{
			ID: 9, Name: "Project", Roles: []authentication.Role{{
				ID: 10, Key: "token_reader", Name: "Token Reader", Scope: authentication.RoleScopeProject,
				Subject: authentication.RoleSubjectProjectToken, DefinitionType: authentication.RoleDefinitionBuiltIn,
				Permissions: []authentication.Permission{{ID: 11, Key: "project.resources.read", Scope: authentication.RoleScopeProject, Name: "Read resources"}},
			}},
		}}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func resolvedOrganizationAuthentication(t *testing.T) authentication.Authentication {
	t.Helper()
	value, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 7},
		authentication.Tenant{OrganizationID: 8},
		authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "7"}},
		nil, []authentication.Organization{{ID: 8, Name: "Organization"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestUserPrincipalResolverReturnsValidatedAggregatePrincipal(t *testing.T) {
	tests := []struct {
		name                   string
		tenant                 authentication.Tenant
		selectedOrganizationID uint64
		authorities            []authentication.PermissionKey
	}{
		{name: "onboarding user"},
		{name: "organization tenant", tenant: authentication.Tenant{OrganizationID: 9}, selectedOrganizationID: 9, authorities: []authentication.PermissionKey{"organization.projects.read"}},
		{name: "project tenant", tenant: authentication.Tenant{OrganizationID: 9, ProjectID: 10}, selectedOrganizationID: 9, authorities: []authentication.PermissionKey{"organization.projects.read", "project.resources.read"}},
		{name: "project infers organization", tenant: authentication.Tenant{OrganizationID: 9, ProjectID: 10}, authorities: []authentication.PermissionKey{"organization.projects.read", "project.resources.read"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &credentialAuthenticationService{authentication: resolvedUserAuthentication(t, test.tenant)}
			principal, err := resolver.NewUserPrincipalResolver(service).Resolve(context.Background(), resolver.UserCredential{Token: "fixture", UserID: 7, OrganizationID: test.selectedOrganizationID, ProjectID: test.tenant.ProjectID})
			if err != nil {
				t.Fatal(err)
			}
			if service.token != "fixture" || service.userID != 7 || service.organizationID != test.selectedOrganizationID || service.projectID != test.tenant.ProjectID || principal.Tenant() != test.tenant ||
				!reflect.DeepEqual(principal.Authorities(), test.authorities) {
				t.Fatalf("Principal() = %+v", principal)
			}
		})
	}
}

func TestProjectAndOrganizationResolversReturnValidatedAggregatePrincipal(t *testing.T) {
	projectService := &credentialAuthenticationService{authentication: resolvedProjectAuthentication(t)}
	project, err := resolver.NewProjectPrincipalResolver(projectService).Resolve(context.Background(), resolver.ProjectCredential{Token: "project-token"})
	if err != nil || projectService.token != "project-token" || project.Subject().Type != authentication.IdentityProjectCredential ||
		!reflect.DeepEqual(project.Authorities(), []authentication.PermissionKey{"project.resources.read"}) {
		t.Fatalf("project principal = %+v, error = %v", project, err)
	}
	organizationService := &credentialAuthenticationService{authentication: resolvedOrganizationAuthentication(t)}
	organization, err := resolver.NewOrganizationPrincipalResolver(organizationService).Resolve(context.Background(), resolver.OrganizationCredential{Token: "organization-token"})
	if err != nil || organizationService.token != "organization-token" || organization.Subject().Type != authentication.IdentityOrganizationCredential ||
		len(organization.Authorities()) != 0 {
		t.Fatalf("organization principal = %+v, error = %v", organization, err)
	}
}

func TestCredentialResolversRejectInvalidResponses(t *testing.T) {
	sourceFailure := errors.New("authentication unavailable")
	wrongSubject := resolvedOrganizationAuthentication(t)
	multiOrganization, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: 9},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "8"}},
		&authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"},
		[]authentication.Organization{{ID: 9, Name: "First"}, {ID: 10, Name: "Second"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		resolve func(*credentialAuthenticationService) error
	}{
		{name: "user blank token", resolve: func(service *credentialAuthenticationService) error {
			_, err := resolver.NewUserPrincipalResolver(service).Resolve(context.Background(), resolver.UserCredential{UserID: 7})
			return err
		}},
		{name: "user project mismatch", resolve: func(service *credentialAuthenticationService) error {
			service.authentication = resolvedUserAuthentication(t, authentication.Tenant{OrganizationID: 9, ProjectID: 10})
			_, err := resolver.NewUserPrincipalResolver(service).Resolve(context.Background(), resolver.UserCredential{Token: "token", UserID: 7, ProjectID: 11})
			return err
		}},
		{name: "user organization mismatch", resolve: func(service *credentialAuthenticationService) error {
			service.authentication = resolvedUserAuthentication(t, authentication.Tenant{OrganizationID: 9})
			_, err := resolver.NewUserPrincipalResolver(service).Resolve(context.Background(), resolver.UserCredential{Token: "token", UserID: 7, OrganizationID: 10})
			return err
		}},
		{name: "user arbitrary default organization", resolve: func(service *credentialAuthenticationService) error {
			service.authentication = multiOrganization
			_, err := resolver.NewUserPrincipalResolver(service).Resolve(context.Background(), resolver.UserCredential{Token: "token", UserID: 7})
			return err
		}},
		{name: "user wrong subject", resolve: func(service *credentialAuthenticationService) error {
			service.authentication = wrongSubject
			_, err := resolver.NewUserPrincipalResolver(service).Resolve(context.Background(), resolver.UserCredential{Token: "token", UserID: 7})
			return err
		}},
		{name: "project wrong subject", resolve: func(service *credentialAuthenticationService) error {
			service.authentication = resolvedUserAuthentication(t, authentication.Tenant{})
			_, err := resolver.NewProjectPrincipalResolver(service).Resolve(context.Background(), resolver.ProjectCredential{Token: "token"})
			return err
		}},
		{name: "organization zero aggregate", resolve: func(service *credentialAuthenticationService) error {
			_, err := resolver.NewOrganizationPrincipalResolver(service).Resolve(context.Background(), resolver.OrganizationCredential{Token: "token"})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.resolve(&credentialAuthenticationService{}); err == nil {
				t.Fatal("expected resolution failure")
			}
		})
	}
	service := &credentialAuthenticationService{err: sourceFailure}
	if _, err := resolver.NewProjectPrincipalResolver(service).Resolve(context.Background(), resolver.ProjectCredential{Token: "token"}); !errors.Is(err, sourceFailure) {
		t.Fatalf("error = %v, want source failure", err)
	}
}

func TestCredentialResolversRejectMissingSourceAndCancellation(t *testing.T) {
	assertInvalid := func(err error) {
		t.Helper()
		var invalid pkg_errors.AuthenticationResolverInvalidError
		if !errors.As(err, &invalid) {
			t.Fatalf("error = %v, want AuthenticationResolverInvalidError", err)
		}
	}
	_, err := resolver.NewUserPrincipalResolver(nil).Resolve(context.Background(), resolver.UserCredential{Token: "token", UserID: 7})
	assertInvalid(err)
	_, err = resolver.NewProjectPrincipalResolver(nil).Resolve(context.Background(), resolver.ProjectCredential{Token: "token"})
	assertInvalid(err)
	_, err = resolver.NewOrganizationPrincipalResolver(nil).Resolve(context.Background(), resolver.OrganizationCredential{Token: "token"})
	assertInvalid(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &credentialAuthenticationService{authentication: resolvedProjectAuthentication(t)}
	if _, err = resolver.NewProjectPrincipalResolver(service).Resolve(ctx, resolver.ProjectCredential{Token: "token"}); !errors.Is(err, context.Canceled) || service.calls != 0 {
		t.Fatalf("Resolve() error = %v, calls = %d", err, service.calls)
	}
	ctx, cancel = context.WithCancel(context.Background())
	service = &credentialAuthenticationService{authentication: resolvedProjectAuthentication(t), onResolve: cancel}
	if _, err = resolver.NewProjectPrincipalResolver(service).Resolve(ctx, resolver.ProjectCredential{Token: "token"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Resolve() error = %v", err)
	}
}
