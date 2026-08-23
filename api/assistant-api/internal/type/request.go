// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"context"

	"github.com/rapidaai/pkg/types"
)

type Talking interface {
	Communication
	Talk(ctx context.Context, auth *types.Authentication) error
}
