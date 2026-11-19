// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	"context"
	"testing"

	principal_authentication "github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

func TestPrincipalContextIsWriteOnce(t *testing.T) {
	if value, exists := Principal(context.Background()); exists || value.Subject().ID != 0 {
		t.Fatalf("Principal() without attachment = %+v, %v", value.Subject(), exists)
	}

	first, err := principal_authentication.NewDirectPrincipal(
		principal_authentication.Identity{Type: principal_authentication.IdentityUser, ID: 7},
		principal_authentication.Tenant{},
		principal_authentication.Proof{
			Method: principal_authentication.AuthenticationUserSession,
			Reference: principal_authentication.ProofReference{
				Type:  principal_authentication.ProofReferenceUserSession,
				Value: "11",
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := principal_authentication.NewDirectPrincipal(
		principal_authentication.Identity{Type: principal_authentication.IdentityUser, ID: 8},
		principal_authentication.Tenant{},
		principal_authentication.Proof{
			Method: principal_authentication.AuthenticationUserSession,
			Reference: principal_authentication.ProofReference{
				Type:  principal_authentication.ProofReferenceUserSession,
				Value: "12",
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := ContextWithPrincipal(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ContextWithPrincipal(ctx, second); err == nil {
		t.Fatal("second principal attachment succeeded")
	} else if _, ok := err.(pkg_errors.AuthenticationPrincipalConflictError); !ok {
		t.Fatalf("duplicate attachment error = %T", err)
	}
	attached, exists := Principal(ctx)
	if !exists || attached.Subject() != first.Subject() {
		t.Fatalf("Principal() = %+v, %v", attached.Subject(), exists)
	}
	shadowed := contextWithoutPrincipal(ctx)
	if attached, exists = Principal(shadowed); exists || attached.Subject().ID != 0 {
		t.Fatalf("shadowed Principal() = %+v, %v", attached.Subject(), exists)
	}
	if _, err := ContextWithPrincipal(shadowed, second); err == nil {
		t.Fatal("principal attached after public-route shadowing")
	} else if _, ok := err.(pkg_errors.AuthenticationPrincipalConflictError); !ok {
		t.Fatalf("shadowed attachment error = %T", err)
	}
}

func TestContextWithPrincipalFailsClosed(t *testing.T) {
	ctx := context.Background()
	got, err := ContextWithPrincipal(ctx, principal_authentication.Principal{})
	if err == nil || got != ctx {
		t.Fatalf("invalid principal = context changed %v, error %v", got != ctx, err)
	}

	principal, err := principal_authentication.NewDirectPrincipal(
		principal_authentication.Identity{Type: principal_authentication.IdentityUser, ID: 7},
		principal_authentication.Tenant{},
		principal_authentication.Proof{
			Method: principal_authentication.AuthenticationUserSession,
			Reference: principal_authentication.ProofReference{
				Type:  principal_authentication.ProofReferenceUserSession,
				Value: "11",
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err = ContextWithPrincipal(ctx, principal)
	if err != nil {
		t.Fatal(err)
	}
	got, err = ContextWithPrincipal(ctx, principal)
	if err == nil || got != ctx {
		t.Fatalf("duplicate principal = context changed %v, error %v", got != ctx, err)
	}
	if _, ok := err.(pkg_errors.AuthenticationPrincipalConflictError); !ok {
		t.Fatalf("duplicate attachment error = %T", err)
	}
}
