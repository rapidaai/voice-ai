// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "testing"

func TestRouteAuthorizationErrorsDoNotExposePolicyDetails(t *testing.T) {
	if got := (RouteAuthorizationDeniedError{}).Error(); got != "route authorization denied" {
		t.Fatalf("RouteAuthorizationDeniedError.Error() = %q", got)
	}
	if got := (RouteAuthorizationDisabledError{}).Error(); got != "route is disabled" {
		t.Fatalf("RouteAuthorizationDisabledError.Error() = %q", got)
	}
}
