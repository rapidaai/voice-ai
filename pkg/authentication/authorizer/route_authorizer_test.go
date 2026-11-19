// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authorizer_test

import (
	"errors"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/authorizer"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

func TestRouteAuthorizerHandlesRouteStates(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	if err := routeAuthorizer.Authorize(nil, authorizer.RoutePolicy{Authorization: authorizer.AuthorizationPublic}); err != nil {
		t.Fatalf("public route denied: %v", err)
	}

	err := routeAuthorizer.Authorize(nil, authorizer.RoutePolicy{Authorization: authorizer.AuthorizationDisabled})
	var disabled pkg_errors.RouteAuthorizationDisabledError
	if !errors.As(err, &disabled) {
		t.Fatalf("disabled route error = %v", err)
	}

	err = routeAuthorizer.Authorize(nil, authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAuthenticated,
		Actors:        []authentication.IdentityType{authentication.IdentityUser},
		Context:       authorizer.ContextUser,
	})
	var denied pkg_errors.RouteAuthorizationDeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("protected route error = %v", err)
	}
}

func TestRouteAuthorizerEvaluatesActorsContextsAndPermissions(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	user := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9},
		"organization.projects.read", "project.resources.read", "project.resources.update")

	tests := []struct {
		name   string
		policy authorizer.RoutePolicy
		allow  bool
	}{
		{
			name: "authenticated user",
			policy: authorizer.RoutePolicy{
				Authorization: authorizer.AuthorizationAuthenticated,
				Actors:        []authentication.IdentityType{authentication.IdentityUser},
				Context:       authorizer.ContextUser,
			},
			allow: true,
		},
		{
			name: "organization context from project tenant",
			policy: authorizer.RoutePolicy{
				Authorization: authorizer.AuthorizationPermission,
				Actors:        []authentication.IdentityType{authentication.IdentityUser},
				Context:       authorizer.ContextOrganization,
				Permissions:   []authentication.PermissionKey{"organization.projects.read"},
			},
			allow: true,
		},
		{
			name: "project permissions are all of",
			policy: authorizer.RoutePolicy{
				Authorization: authorizer.AuthorizationPermission,
				Actors:        []authentication.IdentityType{authentication.IdentityUser},
				Context:       authorizer.ContextProject,
				Permissions: []authentication.PermissionKey{
					"project.resources.read",
					"project.resources.update",
				},
			},
			allow: true,
		},
		{
			name: "missing permission",
			policy: authorizer.RoutePolicy{
				Authorization: authorizer.AuthorizationPermission,
				Actors:        []authentication.IdentityType{authentication.IdentityUser},
				Context:       authorizer.ContextProject,
				Permissions:   []authentication.PermissionKey{"project.resources.delete"},
			},
		},
		{
			name: "wrong actor",
			policy: authorizer.RoutePolicy{
				Authorization: authorizer.AuthorizationAuthenticated,
				Actors:        []authentication.IdentityType{authentication.IdentityProjectCredential},
				Context:       authorizer.ContextProject,
				ProjectToken: &authorizer.ProjectTokenPolicy{
					Audience: authorizer.ProjectTokenAudienceManagement,
					Binding:  authorizer.ProjectTokenBindingProject,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := routeAuthorizer.Authorize(&user, test.policy)
			if test.allow && err != nil {
				t.Fatalf("Authorize() error = %v", err)
			}
			if !test.allow && err == nil {
				t.Fatal("Authorize() allowed denied route")
			}
		})
	}
	partialUser := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9}, "project.resources.read")
	if err := routeAuthorizer.Authorize(&partialUser, authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationPermission,
		Actors:        []authentication.IdentityType{authentication.IdentityUser},
		Context:       authorizer.ContextProject,
		Permissions: []authentication.PermissionKey{
			"project.resources.read",
			"project.resources.update",
		},
	}); err == nil {
		t.Fatal("one of two required permissions satisfied an all-of policy")
	}
}

func TestRouteAuthorizerEvaluatesContextMatrix(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	userEmpty := directUserPrincipal(t, authentication.Tenant{})
	userOrganization := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8})
	userProject := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9})
	project := directProjectPrincipal(t)
	organization := directOrganizationPrincipal(t)
	serviceOrganization := directServicePrincipal(t, authentication.Tenant{OrganizationID: 8})
	serviceProject := directServicePrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9})

	tests := []struct {
		name      string
		principal authentication.Principal
		actor     authentication.IdentityType
		context   authorizer.Context
		allow     bool
	}{
		{name: "user context empty tenant", principal: userEmpty, actor: authentication.IdentityUser, context: authorizer.ContextUser, allow: true},
		{name: "user context organization tenant", principal: userOrganization, actor: authentication.IdentityUser, context: authorizer.ContextUser, allow: true},
		{name: "user context project tenant", principal: userProject, actor: authentication.IdentityUser, context: authorizer.ContextUser, allow: true},
		{name: "organization context user", principal: userOrganization, actor: authentication.IdentityUser, context: authorizer.ContextOrganization, allow: true},
		{name: "organization context project tenant user", principal: userProject, actor: authentication.IdentityUser, context: authorizer.ContextOrganization, allow: true},
		{name: "organization context credential", principal: organization, actor: authentication.IdentityOrganizationCredential, context: authorizer.ContextOrganization, allow: true},
		{name: "organization context service", principal: serviceOrganization, actor: authentication.IdentityService, context: authorizer.ContextOrganization, allow: true},
		{name: "organization context rejects project credential", principal: project, actor: authentication.IdentityProjectCredential, context: authorizer.ContextOrganization},
		{name: "project context user", principal: userProject, actor: authentication.IdentityUser, context: authorizer.ContextProject, allow: true},
		{name: "project context credential", principal: project, actor: authentication.IdentityProjectCredential, context: authorizer.ContextProject, allow: true},
		{name: "project context service", principal: serviceProject, actor: authentication.IdentityService, context: authorizer.ContextProject, allow: true},
		{name: "project context rejects organization credential", principal: organization, actor: authentication.IdentityOrganizationCredential, context: authorizer.ContextProject},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := authorizer.RoutePolicy{
				Authorization: authorizer.AuthorizationAuthenticated,
				Actors:        []authentication.IdentityType{test.actor},
				Context:       test.context,
			}
			if test.actor == authentication.IdentityProjectCredential {
				policy.ProjectToken = &authorizer.ProjectTokenPolicy{Audience: authorizer.ProjectTokenAudienceManagement, Binding: authorizer.ProjectTokenBindingProject}
			}
			err := routeAuthorizer.Authorize(&test.principal, policy)
			if test.allow && err != nil {
				t.Fatalf("Authorize() error = %v", err)
			}
			if !test.allow && err == nil {
				t.Fatal("Authorize() allowed incompatible context")
			}
		})
	}
}

func TestRouteAuthorizerEvaluatesImmediateServiceCaller(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	delegatedUser := delegatedUserPrincipal(t, authentication.Tenant{OrganizationID: 8})
	directService := directServicePrincipal(t, authentication.Tenant{OrganizationID: 8})
	directUser := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8})

	policy := authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAuthenticated,
		Actors:        []authentication.IdentityType{authentication.IdentityUser, authentication.IdentityService},
		Context:       authorizer.ContextOrganization,
		CallerServices: []authorizer.CallerServicePolicy{
			{Issuer: "workflow-api", ActorID: 9007},
		},
	}

	if err := routeAuthorizer.Authorize(&delegatedUser, policy); err != nil {
		t.Fatalf("delegated service caller denied: %v", err)
	}
	if err := routeAuthorizer.Authorize(&directService, policy); err != nil {
		t.Fatalf("direct service caller denied: %v", err)
	}
	if err := routeAuthorizer.Authorize(&directUser, policy); err == nil {
		t.Fatal("direct user satisfied caller_services")
	}

	policy.CallerServices[0].ActorID = 9008
	if err := routeAuthorizer.Authorize(&delegatedUser, policy); err == nil {
		t.Fatal("wrong immediate service caller allowed")
	}
}

func TestRouteAuthorizerRequiresDirectTenantlessServiceContext(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	policy := authorizer.RoutePolicy{
		Authorization:  authorizer.AuthorizationAuthenticated,
		Actors:         []authentication.IdentityType{authentication.IdentityService},
		Context:        authorizer.ContextService,
		CallerServices: []authorizer.CallerServicePolicy{{Issuer: "workflow-api", ActorID: 9007}},
	}
	direct := directServicePrincipal(t, authentication.Tenant{})
	if err := routeAuthorizer.Authorize(&direct, policy); err != nil {
		t.Fatalf("direct tenantless Service denied: %v", err)
	}

	tenantBound := directServicePrincipal(t, authentication.Tenant{OrganizationID: 8})
	if err := routeAuthorizer.Authorize(&tenantBound, policy); err == nil {
		t.Fatal("tenant-bound Service satisfied Service context")
	}
	delegated := delegatedUserPrincipal(t, authentication.Tenant{})
	if err := routeAuthorizer.Authorize(&delegated, policy); err == nil {
		t.Fatal("delegated Service caller satisfied direct Service context")
	}
}

func TestRouteAuthorizerEvaluatesProjectTokenAudience(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	directProject := directProjectPrincipal(t)
	delegatedProject := delegatedProjectPrincipal(t)
	user := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9})

	policy := authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAuthenticated,
		Actors:        []authentication.IdentityType{authentication.IdentityProjectCredential},
		Context:       authorizer.ContextProject,
		ProjectToken: &authorizer.ProjectTokenPolicy{
			Audience: authorizer.ProjectTokenAudienceManagement,
			Binding:  authorizer.ProjectTokenBindingProject,
		},
	}
	if err := routeAuthorizer.Authorize(&directProject, policy); err != nil {
		t.Fatalf("management project credential denied: %v", err)
	}
	if err := routeAuthorizer.Authorize(&delegatedProject, policy); err == nil {
		t.Fatal("delegated runtime project satisfied management policy")
	}

	policy.ProjectToken.Audience = authorizer.ProjectTokenAudienceRuntime
	if err := routeAuthorizer.Authorize(&directProject, policy); err != nil {
		t.Fatalf("management project credential denied on runtime route: %v", err)
	}
	if err := routeAuthorizer.Authorize(&delegatedProject, policy); err != nil {
		t.Fatalf("delegated runtime project denied: %v", err)
	}

	policy.Actors = []authentication.IdentityType{authentication.IdentityUser, authentication.IdentityProjectCredential}
	if err := routeAuthorizer.Authorize(&user, policy); err != nil {
		t.Fatalf("conditional project-token policy applied to user: %v", err)
	}
}

func TestRouteAuthorizerEvaluatesRouteAlternatives(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	policy := authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAnyOf,
		AnyOf: []authorizer.AlternativePolicy{
			{
				Actors:      []authentication.IdentityType{authentication.IdentityUser},
				Context:     authorizer.ContextProject,
				Permissions: []authentication.PermissionKey{"organization.provider_credentials.read_metadata"},
			},
			{
				Actors:  []authentication.IdentityType{authentication.IdentityUser, authentication.IdentityProjectCredential},
				Context: authorizer.ContextProject,
				ProjectToken: &authorizer.ProjectTokenPolicy{
					Audience: authorizer.ProjectTokenAudienceManagement,
					Binding:  authorizer.ProjectTokenBindingProject,
				},
				Permissions: []authentication.PermissionKey{"project.provider_credentials.read_metadata"},
			},
		},
	}
	organizationUser := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9}, "organization.provider_credentials.read_metadata")
	projectUser := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9}, "project.provider_credentials.read_metadata")
	bothUser := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9},
		"organization.provider_credentials.read_metadata", "project.provider_credentials.read_metadata")
	project := directProjectPrincipalWithAuthorities(t, "project.provider_credentials.read_metadata")
	deniedUser := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8, ProjectID: 9})

	for name, principal := range map[string]authentication.Principal{
		"organization alternative": organizationUser,
		"project alternative":      projectUser,
		"both alternatives":        bothUser,
		"project actor":            project,
	} {
		t.Run(name, func(t *testing.T) {
			if err := routeAuthorizer.Authorize(&principal, policy); err != nil {
				t.Fatalf("Authorize() error = %v", err)
			}
		})
	}
	if err := routeAuthorizer.Authorize(&deniedUser, policy); err == nil {
		t.Fatal("route allowed without a complete alternative")
	}
}

func TestRouteAuthorizerEvaluatesCompleteAlternativeRestrictions(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	callerPolicy := authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAnyOf,
		AnyOf: []authorizer.AlternativePolicy{
			{
				Actors:         []authentication.IdentityType{authentication.IdentityUser},
				CallerServices: []authorizer.CallerServicePolicy{{Issuer: "workflow-api", ActorID: 9007}},
				Context:        authorizer.ContextOrganization,
				Authenticated:  true,
			},
		},
	}
	delegatedUser := delegatedUserPrincipal(t, authentication.Tenant{OrganizationID: 8})
	directUser := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8})
	if err := routeAuthorizer.Authorize(&delegatedUser, callerPolicy); err != nil {
		t.Fatalf("caller restriction denied matching caller: %v", err)
	}
	if err := routeAuthorizer.Authorize(&directUser, callerPolicy); err == nil {
		t.Fatal("route bypassed caller restriction")
	}

	projectPolicy := authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAnyOf,
		AnyOf: []authorizer.AlternativePolicy{
			{
				Actors:  []authentication.IdentityType{authentication.IdentityProjectCredential},
				Context: authorizer.ContextProject,
				ProjectToken: &authorizer.ProjectTokenPolicy{
					Audience: authorizer.ProjectTokenAudienceManagement,
					Binding:  authorizer.ProjectTokenBindingProject,
				},
				Authenticated: true,
			},
		},
	}
	directProject := directProjectPrincipal(t)
	delegatedProject := delegatedProjectPrincipal(t)
	if err := routeAuthorizer.Authorize(&directProject, projectPolicy); err != nil {
		t.Fatalf("management project denied: %v", err)
	}
	if err := routeAuthorizer.Authorize(&delegatedProject, projectPolicy); err == nil {
		t.Fatal("route bypassed management audience")
	}
}

func TestRouteAuthorizerEvaluatesAuthenticatedAlternatives(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	policy := authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAnyOf,
		AnyOf: []authorizer.AlternativePolicy{
			{
				Actors:        []authentication.IdentityType{authentication.IdentityOrganizationCredential},
				Context:       authorizer.ContextOrganization,
				Authenticated: true,
			},
			{
				Actors:  []authentication.IdentityType{authentication.IdentityProjectCredential},
				Context: authorizer.ContextProject,
				ProjectToken: &authorizer.ProjectTokenPolicy{
					Audience: authorizer.ProjectTokenAudienceRuntime,
					Binding:  authorizer.ProjectTokenBindingProject,
				},
				Authenticated: true,
			},
		},
	}
	organization := directOrganizationPrincipal(t)
	project := directProjectPrincipal(t)
	if err := routeAuthorizer.Authorize(&organization, policy); err != nil {
		t.Fatalf("organization alternative denied: %v", err)
	}
	if err := routeAuthorizer.Authorize(&project, policy); err != nil {
		t.Fatalf("project alternative denied: %v", err)
	}
}

func TestRouteAuthorizerRejectsMalformedPolicyAndPrincipal(t *testing.T) {
	routeAuthorizer := authorizer.RouteAuthorizer{}
	user := directUserPrincipal(t, authentication.Tenant{OrganizationID: 8}, "organization.projects.read")

	tests := []struct {
		name   string
		policy authorizer.RoutePolicy
	}{
		{name: "unknown authorization", policy: authorizer.RoutePolicy{Authorization: "unknown"}},
		{name: "public with actor", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationPublic, Actors: []authentication.IdentityType{authentication.IdentityUser}}},
		{name: "duplicate actor", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAuthenticated, Actors: []authentication.IdentityType{authentication.IdentityUser, authentication.IdentityUser}, Context: authorizer.ContextUser}},
		{name: "permission without permissions", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationPermission, Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextOrganization}},
		{name: "user context permission", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationPermission, Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextUser, Permissions: []authentication.PermissionKey{"organization.projects.read"}}},
		{name: "permission wrong context", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationPermission, Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextOrganization, Permissions: []authentication.PermissionKey{"project.resources.read"}}},
		{name: "duplicate permission", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationPermission, Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextOrganization, Permissions: []authentication.PermissionKey{"organization.projects.read", "organization.projects.read"}}},
		{name: "invalid caller", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAuthenticated, Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextUser, CallerServices: []authorizer.CallerServicePolicy{{Issuer: " workflow-api", ActorID: 9007}}}},
		{name: "service context without caller allowlist", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAuthenticated, Actors: []authentication.IdentityType{authentication.IdentityService}, Context: authorizer.ContextService}},
		{name: "service context wrong actor", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAuthenticated, Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextService, CallerServices: []authorizer.CallerServicePolicy{{Issuer: "workflow-api", ActorID: 9007}}}},
		{name: "project actor without token policy", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAuthenticated, Actors: []authentication.IdentityType{authentication.IdentityProjectCredential}, Context: authorizer.ContextProject}},
		{name: "any of without alternatives", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAnyOf}},
		{name: "any of with parent fields", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAnyOf, Actors: []authentication.IdentityType{authentication.IdentityUser}, AnyOf: []authorizer.AlternativePolicy{{Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextUser, Authenticated: true}}}},
		{name: "duplicate alternatives", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAnyOf, AnyOf: []authorizer.AlternativePolicy{{Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextUser, Authenticated: true}, {Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextUser, Authenticated: true}}}},
		{name: "alternative mixes predicates", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAnyOf, AnyOf: []authorizer.AlternativePolicy{{Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextOrganization, Authenticated: true, Permissions: []authentication.PermissionKey{"organization.projects.read"}}}}},
		{name: "alternative uses user permission", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAnyOf, AnyOf: []authorizer.AlternativePolicy{{Actors: []authentication.IdentityType{authentication.IdentityUser}, Context: authorizer.ContextUser, Permissions: []authentication.PermissionKey{"organization.projects.read"}}}}},
		{name: "alternative project actor without token", policy: authorizer.RoutePolicy{Authorization: authorizer.AuthorizationAnyOf, AnyOf: []authorizer.AlternativePolicy{{Actors: []authentication.IdentityType{authentication.IdentityProjectCredential}, Context: authorizer.ContextProject, Authenticated: true}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := routeAuthorizer.Authorize(&user, test.policy); err == nil {
				t.Fatal("malformed policy allowed")
			}
		})
	}

	zero := authentication.Principal{}
	if err := routeAuthorizer.Authorize(&zero, authorizer.RoutePolicy{
		Authorization: authorizer.AuthorizationAuthenticated,
		Actors:        []authentication.IdentityType{authentication.IdentityUser},
		Context:       authorizer.ContextUser,
	}); err == nil {
		t.Fatal("zero principal allowed")
	}
}

func directUserPrincipal(t *testing.T, tenant authentication.Tenant, authorities ...authentication.PermissionKey) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		tenant,
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "17"}},
		authorities,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func directProjectPrincipal(t *testing.T) authentication.Principal {
	t.Helper()
	return directProjectPrincipalWithAuthorities(t)
}

func directProjectPrincipalWithAuthorities(t *testing.T, authorities ...authentication.PermissionKey) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12},
		authentication.Tenant{OrganizationID: 8, ProjectID: 9},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "12"}},
		authorities,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func directOrganizationPrincipal(t *testing.T) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13},
		authentication.Tenant{OrganizationID: 8},
		authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "13"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func directServicePrincipal(t *testing.T, tenant authentication.Tenant) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: 9007, Issuer: "workflow-api"},
		tenant,
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func delegatedUserPrincipal(t *testing.T, tenant authentication.Tenant) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDelegatedPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		authentication.ServiceActor{ID: 9007, Issuer: "workflow-api"},
		tenant,
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func delegatedProjectPrincipal(t *testing.T) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDelegatedPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12},
		authentication.ServiceActor{ID: 9007, Issuer: "workflow-api"},
		authentication.Tenant{OrganizationID: 8, ProjectID: 9},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
