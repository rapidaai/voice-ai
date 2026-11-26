// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "testing"

func TestAuthenticationTransportErrorsDoNotExposeDetails(t *testing.T) {
	if got := (AuthenticationFailureError{}).Error(); got != "Invalid authentication credentials" {
		t.Fatalf("AuthenticationFailureError.Error() = %q", got)
	}
	if got := (AuthorizationFailureError{}).Error(); got != "Access denied" {
		t.Fatalf("AuthorizationFailureError.Error() = %q", got)
	}
}
