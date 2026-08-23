// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// GetDuration converts provider callback duration values into seconds-based durations.
func GetDuration(value interface{}) *time.Duration {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		seconds, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil
		}
		duration := time.Duration(seconds * float64(time.Second))
		return &duration
	case float64:
		duration := time.Duration(v * float64(time.Second))
		return &duration
	case float32:
		duration := time.Duration(float64(v) * float64(time.Second))
		return &duration
	case int:
		duration := time.Duration(v) * time.Second
		return &duration
	case int64:
		duration := time.Duration(v) * time.Second
		return &duration
	case uint64:
		seconds, err := Uint64ToInt64(v)
		if err != nil || seconds > int64(math.MaxInt64/time.Second) {
			return nil
		}
		duration := time.Duration(seconds) * time.Second
		return &duration
	default:
		return nil
	}
}
