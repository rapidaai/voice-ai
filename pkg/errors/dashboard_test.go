// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import (
	"errors"
	"testing"
)

func TestDashboardErrorsPreserveSafeMessagesAndCauses(t *testing.T) {
	cause := errors.New("downstream failure")

	invalid := DashboardAnalyticsRequestInvalidError{Message: "invalid analytics range", Cause: cause}
	if invalid.Error() != "invalid analytics range" || !errors.Is(invalid, cause) {
		t.Fatalf("invalid error = %q, cause preserved = %t", invalid.Error(), errors.Is(invalid, cause))
	}
	if message := (DashboardAnalyticsRequestInvalidError{}).Error(); message != "" {
		t.Fatalf("empty invalid message = %q", message)
	}

	unavailable := DashboardUnavailableError{Cause: cause}
	if unavailable.Error() != "dashboard is unavailable" || !errors.Is(unavailable, cause) {
		t.Fatalf("unavailable error = %q, cause preserved = %t", unavailable.Error(), errors.Is(unavailable, cause))
	}
}
