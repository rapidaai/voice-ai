// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"testing"

	"github.com/rapidaai/pkg/authentication"
)

func TestSessionAuthPreservesAuthenticationPrincipal(t *testing.T) {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: 11, ProjectID: 13},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceUserSession,
				Value: "session-1",
			},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("NewDirectPrincipal() error = %v", err)
	}

	var session Session
	WithSessionAuth(principal)(&session)
	if session.GetAuth().Subject().ID != 7 {
		t.Fatalf("session subject ID = %d, want 7", session.GetAuth().Subject().ID)
	}
}
