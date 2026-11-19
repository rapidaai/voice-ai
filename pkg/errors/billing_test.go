// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "testing"

func TestBillingErrorsExposeSafeMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "organization selection", err: BillingOrganizationSelectionInvalidError{}, want: "billing organization selection is invalid"},
		{name: "subscription", err: BillingSubscriptionInvalidError{}, want: "billing subscription is invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
		})
	}
}
