// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package resolver_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	"github.com/rapidaai/pkg/authentication/resolver"
)

func TestServicePrincipalResolverCreatesDirectAndDelegatedPrincipals(t *testing.T) {
	key := bytes.Repeat([]byte{1}, sha256.Size)
	issuer, err := authentication_jwt.NewIssuer("web-api", key)
	if err != nil {
		t.Fatal(err)
	}
	principalResolver := resolver.NewServicePrincipalResolver("web-api", key)
	tests := []struct {
		name            string
		subject         authentication.Identity
		tenant          authentication.Tenant
		authorities     []authentication.PermissionKey
		wantSubject     authentication.Identity
		wantAuthorities []authentication.PermissionKey
		wantActor       bool
	}{
		{
			name: "service", subject: authentication.Identity{Type: authentication.IdentityService, ID: 7, Issuer: "assistant-api"},
			wantSubject: authentication.Identity{Type: authentication.IdentityService, ID: 1, Issuer: "internal"},
		},
		{
			name: "user", subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11},
			wantSubject:     authentication.Identity{Type: authentication.IdentityUser, ID: 11},
			tenant:          authentication.Tenant{OrganizationID: 8, ProjectID: 9},
			authorities:     []authentication.PermissionKey{"project.assistant.read", "organization.user.read"},
			wantAuthorities: []authentication.PermissionKey{"organization.user.read", "project.assistant.read"}, wantActor: true,
		},
		{
			name: "project", subject: authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12},
			wantSubject:     authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12},
			tenant:          authentication.Tenant{OrganizationID: 8, ProjectID: 9},
			authorities:     []authentication.PermissionKey{"project.assistant.read"},
			wantAuthorities: []authentication.PermissionKey{"project.assistant.read"}, wantActor: true,
		},
		{
			name: "organization", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13},
			wantSubject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13},
			tenant:      authentication.Tenant{OrganizationID: 8}, wantActor: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertion, err := issuer.Issue(test.subject, test.tenant, test.authorities)
			if err != nil {
				t.Fatal(err)
			}
			principal, err := principalResolver.Resolve(context.Background(), resolver.ServiceAssertionCredential{Assertion: assertion})
			if err != nil {
				t.Fatal(err)
			}
			if principal.Subject() != test.wantSubject || principal.Tenant() != test.tenant {
				t.Fatalf("principal = subject %+v tenant %+v", principal.Subject(), principal.Tenant())
			}
			actor, hasActor := principal.Actor()
			if hasActor != test.wantActor || (hasActor && actor != (authentication.ServiceActor{ID: 1, Issuer: "internal"})) {
				t.Fatalf("Actor() = %+v, %v", actor, hasActor)
			}
			if !reflect.DeepEqual(principal.Authorities(), test.wantAuthorities) {
				t.Fatalf("Authorities() = %v", principal.Authorities())
			}
		})
	}
}

func TestServicePrincipalResolverRejectsInvalidInput(t *testing.T) {
	if _, err := (&resolver.ServicePrincipalResolver{}).Resolve(
		context.Background(), resolver.ServiceAssertionCredential{Assertion: "assertion"},
	); err == nil {
		t.Fatal("zero-value resolver accepted")
	}
	principalResolver := resolver.NewServicePrincipalResolver("web-api", bytes.Repeat([]byte{1}, sha256.Size))

	if _, err := principalResolver.Resolve(context.Background(), resolver.ServiceAssertionCredential{}); err == nil {
		t.Fatal("blank assertion accepted")
	}
	if _, err := principalResolver.Resolve(context.Background(), resolver.ServiceAssertionCredential{Assertion: "not-a-token"}); err == nil {
		t.Fatal("invalid assertion accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := principalResolver.Resolve(ctx, resolver.ServiceAssertionCredential{Assertion: "assertion"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Resolve() error = %v", err)
	}
}
