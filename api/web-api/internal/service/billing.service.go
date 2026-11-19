// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
)

// BillingService owns billing-plan persistence reads for Organization authentication projections.
type BillingService interface {
	GetSubscriptions(context.Context, []uint64) (map[uint64]internal_entity.BillingSubscription, error)
}
