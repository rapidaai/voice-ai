// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

// BillingOrganizationSelectionInvalidError reports an invalid Organization billing selection.
type BillingOrganizationSelectionInvalidError struct{}

func (BillingOrganizationSelectionInvalidError) Error() string {
	return "billing organization selection is invalid"
}

// BillingSubscriptionInvalidError reports an inconsistent persisted subscription without exposing its data.
type BillingSubscriptionInvalidError struct{}

func (BillingSubscriptionInvalidError) Error() string {
	return "billing subscription is invalid"
}
