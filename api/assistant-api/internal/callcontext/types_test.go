// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_callcontext

import (
	"math"
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
)

func TestCallContextAuthenticationSnapshotRoundTrip(t *testing.T) {
	proof := authentication.Proof{
		Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal,
		Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "assistant-test-1"},
	}
	tests := []struct {
		name      string
		principal authentication.Principal
	}{
		{
			name: "direct service",
			principal: mustCallContextPrincipal(authentication.NewDirectPrincipal(
				authentication.Identity{Type: authentication.IdentityService, ID: 66, Issuer: "assistant-api"},
				authentication.Tenant{OrganizationID: 33, ProjectID: 22}, proof, nil,
			)),
		},
		{
			name: "delegated user with authorities",
			principal: mustCallContextPrincipal(authentication.NewDelegatedPrincipal(
				authentication.Identity{Type: authentication.IdentityUser, ID: 11},
				authentication.ServiceActor{ID: 66, Issuer: "assistant-api"},
				authentication.Tenant{OrganizationID: 33, ProjectID: 22}, proof,
				[]authentication.PermissionKey{"project.read", "organization.read"},
			)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callContext := &CallContext{}
			if err := callContext.SetAuthentication(test.principal); err != nil {
				t.Fatalf("SetAuthentication() error = %v", err)
			}
			reconstructed, err := callContext.ToAuthentication()
			if err != nil {
				t.Fatalf("ToAuthentication() error = %v", err)
			}
			if !reflect.DeepEqual(reconstructed, test.principal) {
				t.Fatalf("ToAuthentication() = %+v, want %+v", reconstructed, test.principal)
			}
		})
	}
}

func TestCallContextAuthenticationSnapshotFailsClosed(t *testing.T) {
	actorID := uint64(66)
	tests := []struct {
		name        string
		callContext *CallContext
	}{
		{name: "nil", callContext: nil},
		{name: "legacy snapshot", callContext: &CallContext{AuthType: "user", OrganizationID: 33}},
		{name: "unsupported version", callContext: &CallContext{AuthPrincipalVersion: 2}},
		{
			name: "incomplete delegated actor",
			callContext: &CallContext{
				AuthPrincipalVersion: 1, AuthSubjectType: string(authentication.IdentityUser), AuthSubjectID: 11,
				OrganizationID: 33, AuthServiceActorID: &actorID,
				AuthProofMethod: string(authentication.AuthenticationServiceAssertion), AuthProofAudience: string(authentication.AudienceRapidaInternal),
				AuthProofReferenceType: string(authentication.ProofReferenceSigningKey), AuthProofReferenceValue: "assistant-test-1",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.callContext.ToAuthentication(); err == nil {
				t.Fatal("ToAuthentication() error = nil")
			}
		})
	}
}

func TestCallContextRejectsZeroPrincipal(t *testing.T) {
	if err := (&CallContext{}).SetAuthentication(authentication.Principal{}); err == nil {
		t.Fatal("SetAuthentication() error = nil")
	}
}

func TestCallContextSupportsMaximumDatabaseIdentity(t *testing.T) {
	principal := mustCallContextPrincipal(authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: math.MaxInt64},
		authentication.Tenant{OrganizationID: 33},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-max"}}, nil,
	))
	callContext := &CallContext{}
	if err := callContext.SetAuthentication(principal); err != nil {
		t.Fatalf("SetAuthentication() error = %v", err)
	}
	reconstructed, err := callContext.ToAuthentication()
	if err != nil {
		t.Fatalf("ToAuthentication() error = %v", err)
	}
	if reconstructed.Subject().ID != math.MaxInt64 {
		t.Fatalf("Subject().ID = %d", reconstructed.Subject().ID)
	}
}

func mustCallContextPrincipal(principal authentication.Principal, err error) authentication.Principal {
	if err != nil {
		panic(err)
	}
	return principal
}
