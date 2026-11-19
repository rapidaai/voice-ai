// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import (
	"math"
	"strconv"
)

type ErrorCode uint64

type PlatformError struct {
	HTTPStatusCode int
	Code           ErrorCode
	Error          string
	ErrorMessage   string
}

func (platformError PlatformError) HTTPStatusCodeInt32() int32 {
	if platformError.HTTPStatusCode < math.MinInt32 || platformError.HTTPStatusCode > math.MaxInt32 {
		return 0
	}
	// #nosec G115, the status code range is checked above.
	return int32(platformError.HTTPStatusCode)
}

func (platformError PlatformError) CodeString() string {
	return strconv.FormatUint(uint64(platformError.Code), 10)
}
