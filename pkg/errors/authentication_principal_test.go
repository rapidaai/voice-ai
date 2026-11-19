// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "testing"

func TestAuthenticationPrincipalInvalidError(t *testing.T) {
	err := AuthenticationPrincipalInvalidError{}
	if got := err.Error(); got != "authentication principal is invalid" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestAuthenticationPrincipalConflictError(t *testing.T) {
	err := AuthenticationPrincipalConflictError{}
	if got := err.Error(); got != "authentication principal conflicts with request context" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestAuthenticationInvalidError(t *testing.T) {
	if got := (AuthenticationInvalidError{}).Error(); got != "authentication is invalid" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestAuthenticationResolverInvalidError(t *testing.T) {
	err := AuthenticationResolverInvalidError{}
	if got := err.Error(); got != "authentication resolver is invalid" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestAuthenticationSecretInvalidError(t *testing.T) {
	if got := (AuthenticationSecretInvalidError{}).Error(); got != "authentication secret is invalid" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestJWTErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "issuer configuration", err: JWTIssuerInvalidError{}, want: "authentication JWT issuer is invalid"},
		{name: "verifier configuration", err: JWTVerifierInvalidError{}, want: "authentication JWT verifier is invalid"},
		{name: "issuance", err: JWTIssueError{}, want: "authentication JWT could not be issued"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
		})
	}
}
