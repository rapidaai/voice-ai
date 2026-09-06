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
	// #nosec G115, value is checked above.
	return uint32(value), nil
}

// Int64ToUint64 converts a non-negative int64 to uint64.
func Int64ToUint64(value int64) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("int64 value %d cannot be converted to uint64", value)
	}
	// #nosec G115, value is checked above.
	return uint64(value), nil
}

// Int64ToInt converts an int64 to int when the value is in range.
func Int64ToInt(value int64) (int, error) {
	if strconv.IntSize == 32 && (value < math.MinInt32 || value > math.MaxInt32) {
		return 0, fmt.Errorf("int64 value %d exceeds int range", value)
	}
	// #nosec G115, value is checked above.
	return int(value), nil
}

// Uint64ToInt64 converts a uint64 to int64 when the value is in range.
func Uint64ToInt64(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("uint64 value %d exceeds int64 range", value)
	}
	// #nosec G115, value is checked above.
	return int64(value), nil
}

// Uint64ToUint32 converts a uint64 to uint32 when the value is in range.
func Uint64ToUint32(value uint64) (uint32, error) {
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("uint64 value %d exceeds uint32 range", value)
	}
	// #nosec G115, value is checked above.
	return uint32(value), nil
}

// Uint32ToInt converts a uint32 to int when the value is in range.
func Uint32ToInt(value uint32) (int, error) {
	if strconv.IntSize == 32 && value > math.MaxInt32 {
		return 0, fmt.Errorf("uint32 value %d exceeds int range", value)
	}
	// #nosec G115, value is checked above.
	return int(value), nil
}

// Uint32ToInt64 converts a uint32 to int64.
func Uint32ToInt64(value uint32) int64 {
	// #nosec G115, every uint32 value fits in int64.
	return int64(value)
}

// Uint32ToUint8 converts a uint32 to uint8 when the value is in range.
func Uint32ToUint8(value uint32) (uint8, error) {
	if value > math.MaxUint8 {
		return 0, fmt.Errorf("uint32 value %d exceeds uint8 range", value)
	}
	// #nosec G115, value is checked above.
	return uint8(value), nil
}

// UintToUint32 converts a uint to uint32 when the value is in range.
func UintToUint32(value uint) (uint32, error) {
	// #nosec G115, Uint64ToUint32 validates the target uint32 range.
	return StringToUint32(strconv.FormatUint(uint64(value), 10))
}

// IntToUint32 converts a non-negative int to uint32 when the value is in range.
func IntToUint32(value int) (uint32, error) {
	if value < 0 {
		return 0, fmt.Errorf("int value %d cannot be converted to uint32", value)
	}
	// #nosec G115, value is non-negative and checked by Uint64ToUint32.
	return Uint64ToUint32(uint64(value))
}

// IntToUint8 converts a non-negative int to uint8 when the value is in range.
func IntToUint8(value int) (uint8, error) {
	if value < 0 {
		return 0, fmt.Errorf("int value %d cannot be converted to uint8", value)
	}
	parsed, err := strconv.ParseUint(strconv.Itoa(value), 10, 8)
	if err != nil {
		return 0, fmt.Errorf("int value %d exceeds uint8 range: %w", value, err)
	}
	// #nosec G115, parsed is produced by ParseUint with bitSize 8.
	return uint8(parsed), nil
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
	// #nosec G115, value is checked above.
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
	// #nosec G115, parsed is produced by ParseUint with bitSize 32.
	return uint32(parsed), nil
}

func StringToUint64(value string) (uint64, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as uint64: %w", value, err)
	}
	return parsed, nil
}
