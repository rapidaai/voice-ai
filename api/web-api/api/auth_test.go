// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"math"
	"testing"

	"github.com/rapidaai/pkg/utils"
)

func TestPaginationCountConversion(t *testing.T) {
	if _, err := utils.Int64ToUint32(int64(math.MaxUint32) + 1); err == nil {
		t.Fatal("expected pagination count overflow error")
	}
}
