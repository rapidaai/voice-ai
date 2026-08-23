// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// MinUint64 returns the minimum of two uint64 numbers
func MinUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Int64ToUint32 converts an int64 to uint32 when the value is in range.
func Int64ToUint32(value int64) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("int64 value %d exceeds uint32 range", value)
	}
	return uint32(value), nil
}

// Uint64ToInt64 converts a uint64 to int64 when the value is in range.
func Uint64ToInt64(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("uint64 value %d exceeds int64 range", value)
	}
	return int64(value), nil
}

// UintToUint32 converts a uint to uint32 when the value is in range.
func UintToUint32(value uint) (uint32, error) {
	return StringToUint32(strconv.FormatUint(uint64(value), 10))
}

// IntToInt32 converts an int to int32 when the value is in range.
func IntToInt32(value int) (int32, error) {
	parsed, err := strconv.ParseInt(strconv.Itoa(value), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("int value %d exceeds int32 range: %w", value, err)
	}
	return int32(parsed), nil
}

// IntToUint64 converts a non-negative int to uint64.
func IntToUint64(value int) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("value %d cannot be converted to uint64", value)
	}
	return uint64(value), nil
}

// Int64ToString converts an int64 to its decimal representation.
func Int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func StringToUint32(value string) (uint32, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as uint32: %w", value, err)
	}
	return uint32(parsed), nil
}

func StringToUint64(value string) (uint64, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as uint64: %w", value, err)
	}
	return parsed, nil
}
