// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

func TestNewDirectPrincipal(t *testing.T) {
	tests := []struct {
		name        string
		subject     authentication.Identity
		tenant      authentication.Tenant
		proof       authentication.Proof
		authorities []authentication.PermissionKey
	}{
		{
			name:    "user without tenant",
			subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11},
			proof:   authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-11"}},
		},
		{
			name:        "user with organization tenant",
			subject:     authentication.Identity{Type: authentication.IdentityUser, ID: 16},
			tenant:      authentication.Tenant{OrganizationID: 25},
			proof:       authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-16"}},
			authorities: []authentication.PermissionKey{"organization.projects.read"},
		},
		{
			name:        "user with project tenant",
			subject:     authentication.Identity{Type: authentication.IdentityUser, ID: 12},
			tenant:      authentication.Tenant{OrganizationID: 21, ProjectID: 31},
			proof:       authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-12"}},
			authorities: []authentication.PermissionKey{"organization.projects.read", "project.resources.read"},
		},
		{
			name:        "project credential",
			subject:     authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 13},
			tenant:      authentication.Tenant{OrganizationID: 22, ProjectID: 32},
			proof:       authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "13"}},
			authorities: []authentication.PermissionKey{"project.resources.read"},
		},
		{
			name:    "organization credential",
			subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 14},
			tenant:  authentication.Tenant{OrganizationID: 23},
			proof:   authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "14"}},
		},
		{
			name:    "service without tenant",
			subject: authentication.Identity{Type: authentication.IdentityService, ID: 17, Issuer: "workflow-api"},
			proof:   authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-key-1"}},
		},
		{
			name:    "service with organization tenant",
			subject: authentication.Identity{Type: authentication.IdentityService, ID: 18, Issuer: "workflow-api"},
			tenant:  authentication.Tenant{OrganizationID: 24},
			proof:   authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-key-1"}},
		},
		{
			name:    "service",
			subject: authentication.Identity{Type: authentication.IdentityService, ID: 15, Issuer: "workflow-api"},
			tenant:  authentication.Tenant{OrganizationID: 24, ProjectID: 34},
			proof:   authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-key-1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principal, err := authentication.NewDirectPrincipal(test.subject, test.tenant, test.proof, test.authorities)
			if err != nil {
				t.Fatal(err)
			}
			if principal.Subject() != test.subject || principal.Caller() != test.subject || principal.Tenant() != test.tenant || principal.Proof() != test.proof {
				t.Fatalf("principal does not preserve its verified values")
			}
			if _, ok := principal.Actor(); ok {
				t.Fatal("direct principal has an actor")
			}
		})
	}
}

func TestNewDirectPrincipalRejectsInvalidCombinations(t *testing.T) {
	validUser := authentication.Identity{Type: authentication.IdentityUser, ID: 11}
	validUserProof := authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-11"}}
	validProject := authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 13}
	validProjectTenant := authentication.Tenant{OrganizationID: 21, ProjectID: 31}
	validProjectProof := authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "13"}}

	tests := []struct {
		name        string
		subject     authentication.Identity
		tenant      authentication.Tenant
		proof       authentication.Proof
		authorities []authentication.PermissionKey
	}{
		{name: "unknown subject", subject: authentication.Identity{Type: authentication.IdentityType("system"), ID: 1}, proof: validUserProof},
		{name: "zero subject id", subject: authentication.Identity{Type: authentication.IdentityUser}, proof: validUserProof},
		{name: "large subject id", subject: authentication.Identity{Type: authentication.IdentityUser, ID: math.MaxInt64 + 1}, proof: validUserProof},
		{name: "issuer on user", subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11, Issuer: "issuer"}, proof: validUserProof},
		{name: "service without issuer", subject: authentication.Identity{Type: authentication.IdentityService, ID: 11}, proof: authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key"}}},
		{name: "project without organization", subject: validUser, tenant: authentication.Tenant{ProjectID: 31}, proof: validUserProof},
		{name: "large organization id", subject: validUser, tenant: authentication.Tenant{OrganizationID: math.MaxInt64 + 1}, proof: validUserProof},
		{name: "project credential without project tenant", subject: validProject, tenant: authentication.Tenant{OrganizationID: 21}, proof: validProjectProof},
		{name: "organization credential without organization tenant", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 14}, proof: authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "14"}}},
		{name: "organization credential with project tenant", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 14}, tenant: validProjectTenant, proof: authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "14"}}},
		{name: "project credential runtime proof", subject: validProject, tenant: validProjectTenant, proof: authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.Audience("runtime"), Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "13"}}},
		{name: "project credential mismatched reference", subject: validProject, tenant: validProjectTenant, proof: authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "013"}}},
		{name: "user with credential proof", subject: validUser, proof: validProjectProof},
		{name: "organization credential mismatched reference", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 14}, tenant: authentication.Tenant{OrganizationID: 21}, proof: authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "15"}}},
		{name: "service with user proof", subject: authentication.Identity{Type: authentication.IdentityService, ID: 15, Issuer: "workflow-api"}, proof: validUserProof},
		{name: "service issuer with surrounding space", subject: authentication.Identity{Type: authentication.IdentityService, ID: 15, Issuer: " workflow-api"}, proof: authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key"}}},
		{name: "blank proof reference", subject: validUser, proof: authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: " "}}},
		{name: "proof reference with nul", subject: validUser, proof: authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session\x00id"}}},
		{name: "long proof reference", subject: validUser, proof: authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: strings.Repeat("a", 201)}}},
		{name: "blank authority", subject: validUser, proof: validUserProof, authorities: []authentication.PermissionKey{"project.resources.read", " "}},
		{name: "authority with surrounding space", subject: validUser, proof: validUserProof, authorities: []authentication.PermissionKey{" project.resources.read"}},
		{name: "tenantless user with organization authority", subject: validUser, proof: validUserProof, authorities: []authentication.PermissionKey{"organization.projects.read"}},
		{name: "organization user with project authority", subject: validUser, tenant: authentication.Tenant{OrganizationID: 21}, proof: validUserProof, authorities: []authentication.PermissionKey{"project.resources.read"}},
		{name: "project credential with organization authority", subject: validProject, tenant: validProjectTenant, proof: validProjectProof, authorities: []authentication.PermissionKey{"organization.projects.read"}},
		{name: "organization credential with authority", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 14}, tenant: authentication.Tenant{OrganizationID: 21}, proof: authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "14"}}, authorities: []authentication.PermissionKey{"organization.projects.read"}},
		{name: "service with authority", subject: authentication.Identity{Type: authentication.IdentityService, ID: 15, Issuer: "workflow-api"}, proof: authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key"}}, authorities: []authentication.PermissionKey{"project.resources.read"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := authentication.NewDirectPrincipal(test.subject, test.tenant, test.proof, test.authorities)
			if err == nil {
				t.Fatal("expected invalid principal")
			}
			if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
				t.Fatalf("error = %T, want AuthenticationPrincipalInvalidError", err)
			}
		})
	}
}

func TestNewDelegatedPrincipal(t *testing.T) {
	tests := []struct {
		name        string
		subject     authentication.Identity
		tenant      authentication.Tenant
		authorities []authentication.PermissionKey
	}{
		{name: "user without tenant", subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11}},
		{name: "user with organization tenant", subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11}, tenant: authentication.Tenant{OrganizationID: 21}, authorities: []authentication.PermissionKey{"organization.projects.read"}},
		{name: "user with project tenant", subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11}, tenant: authentication.Tenant{OrganizationID: 21, ProjectID: 31}, authorities: []authentication.PermissionKey{"organization.projects.read", "project.resources.read"}},
		{name: "project credential", subject: authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12}, tenant: authentication.Tenant{OrganizationID: 21, ProjectID: 31}, authorities: []authentication.PermissionKey{"project.resources.read"}},
		{name: "organization credential", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13}, tenant: authentication.Tenant{OrganizationID: 21}},
	}
	actor := authentication.ServiceActor{ID: 9007, Issuer: "workflow-api"}
	proof := authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-key-1"}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principal, err := authentication.NewDelegatedPrincipal(test.subject, actor, test.tenant, proof, test.authorities)
			if err != nil {
				t.Fatal(err)
			}
			actualActor, ok := principal.Actor()
			if !ok || actualActor != actor {
				t.Fatalf("actor = %+v, %t", actualActor, ok)
			}
			wantCaller := authentication.Identity{Type: authentication.IdentityService, ID: actor.ID, Issuer: actor.Issuer}
			if principal.Caller() != wantCaller || principal.Subject() != test.subject {
				t.Fatal("delegated caller or subject is incorrect")
			}
		})
	}
}

func TestNewDelegatedPrincipalRejectsInvalidCombinations(t *testing.T) {
	validSubject := authentication.Identity{Type: authentication.IdentityUser, ID: 11}
	validActor := authentication.ServiceActor{ID: 9007, Issuer: "workflow-api"}
	validProof := authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-key-1"}}

	tests := []struct {
		name        string
		subject     authentication.Identity
		actor       authentication.ServiceActor
		tenant      authentication.Tenant
		proof       authentication.Proof
		authorities []authentication.PermissionKey
	}{
		{name: "service subject", subject: authentication.Identity{Type: authentication.IdentityService, ID: 7, Issuer: "billing-api"}, actor: validActor, proof: validProof},
		{name: "actor without id", subject: validSubject, actor: authentication.ServiceActor{Issuer: "workflow-api"}, proof: validProof},
		{name: "actor without issuer", subject: validSubject, actor: authentication.ServiceActor{ID: 9007}, proof: validProof},
		{name: "user session proof", subject: validSubject, actor: validActor, proof: authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session"}}},
		{name: "project subject without project tenant", subject: authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12}, actor: validActor, tenant: authentication.Tenant{OrganizationID: 21}, proof: validProof},
		{name: "organization subject with project tenant", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13}, actor: validActor, tenant: authentication.Tenant{OrganizationID: 21, ProjectID: 31}, proof: validProof},
		{name: "tenantless user with organization authority", subject: validSubject, actor: validActor, proof: validProof, authorities: []authentication.PermissionKey{"organization.projects.read"}},
		{name: "organization user with project authority", subject: validSubject, actor: validActor, tenant: authentication.Tenant{OrganizationID: 21}, proof: validProof, authorities: []authentication.PermissionKey{"project.resources.read"}},
		{name: "project subject with organization authority", subject: authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12}, actor: validActor, tenant: authentication.Tenant{OrganizationID: 21, ProjectID: 31}, proof: validProof, authorities: []authentication.PermissionKey{"organization.projects.read"}},
		{name: "organization subject with authority", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13}, actor: validActor, tenant: authentication.Tenant{OrganizationID: 21}, proof: validProof, authorities: []authentication.PermissionKey{"organization.projects.read"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := authentication.NewDelegatedPrincipal(test.subject, test.actor, test.tenant, test.proof, test.authorities)
			if err == nil {
				t.Fatal("expected invalid delegated principal")
			}
		})
	}
}

func TestPrincipalTenantMatrix(t *testing.T) {
	serviceProof := authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}}
	paths := []struct {
		name      string
		delegated bool
		subject   authentication.Identity
		proof     authentication.Proof
		allowed   [4]bool
	}{
		{name: "direct user", subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11}, proof: authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-11"}}, allowed: [4]bool{true, true, true, false}},
		{name: "direct project", subject: authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12}, proof: authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "12"}}, allowed: [4]bool{false, false, true, false}},
		{name: "direct organization", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13}, proof: authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "13"}}, allowed: [4]bool{false, true, false, false}},
		{name: "direct service", subject: authentication.Identity{Type: authentication.IdentityService, ID: 14, Issuer: "workflow-api"}, proof: serviceProof, allowed: [4]bool{true, true, true, false}},
		{name: "delegated user", delegated: true, subject: authentication.Identity{Type: authentication.IdentityUser, ID: 15}, proof: serviceProof, allowed: [4]bool{true, true, true, false}},
		{name: "delegated project", delegated: true, subject: authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 16}, proof: serviceProof, allowed: [4]bool{false, false, true, false}},
		{name: "delegated organization", delegated: true, subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 17}, proof: serviceProof, allowed: [4]bool{false, true, false, false}},
	}
	tenants := []struct {
		name   string
		tenant authentication.Tenant
	}{
		{name: "empty"},
		{name: "organization", tenant: authentication.Tenant{OrganizationID: 21}},
		{name: "project", tenant: authentication.Tenant{OrganizationID: 21, ProjectID: 31}},
		{name: "project without organization", tenant: authentication.Tenant{ProjectID: 31}},
	}
	actor := authentication.ServiceActor{ID: 9007, Issuer: "workflow-api"}

	for _, path := range paths {
		t.Run(path.name, func(t *testing.T) {
			for index, tenant := range tenants {
				t.Run(tenant.name, func(t *testing.T) {
					var err error
					if path.delegated {
						_, err = authentication.NewDelegatedPrincipal(path.subject, actor, tenant.tenant, path.proof, nil)
					} else {
						_, err = authentication.NewDirectPrincipal(path.subject, tenant.tenant, path.proof, nil)
					}
					if path.allowed[index] && err != nil {
						t.Fatalf("valid tenant rejected: %v", err)
					}
					if !path.allowed[index] && err == nil {
						t.Fatal("invalid tenant accepted")
					}
				})
			}
		})
	}
}

func TestDelegatedPrincipalActorIsImmutable(t *testing.T) {
	actor := authentication.ServiceActor{ID: 9007, Issuer: "workflow-api"}
	principal, err := authentication.NewDelegatedPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		actor,
		authentication.Tenant{},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor.Issuer = "changed-input"
	returned, ok := principal.Actor()
	if !ok {
		t.Fatal("delegated actor is unavailable")
	}
	returned.Issuer = "changed-output"
	actual, ok := principal.Actor()
	if !ok || actual.Issuer != "workflow-api" {
		t.Fatalf("Actor() = %+v, %t", actual, ok)
	}
}

func TestPrincipalAuthoritiesAreImmutable(t *testing.T) {
	authorities := []authentication.PermissionKey{"project.resources.update", "organization.projects.read", "project.resources.update"}
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		authentication.Tenant{OrganizationID: 21, ProjectID: 31},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-11"}},
		authorities,
	)
	if err != nil {
		t.Fatal(err)
	}

	authorities[0] = "changed.input"
	want := []authentication.PermissionKey{"organization.projects.read", "project.resources.update"}
	if got := principal.Authorities(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Authorities() = %v, want %v", got, want)
	}

	returned := principal.Authorities()
	returned[0] = "changed.output"
	if got := principal.Authorities(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Authorities() after output mutation = %v, want %v", got, want)
	}
}
