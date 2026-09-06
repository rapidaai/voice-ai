// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"math"
	"testing"
)

func TestMaxUint64(t *testing.T) {
	tests := []struct {
		name     string
		a, b     uint64
		expected uint64
	}{
		{"a > b", 10, 5, 10},
		{"a < b", 5, 10, 10},
		{"equal", 5, 5, 5},
		{"zero", 0, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaxUint64(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestMinUint64(t *testing.T) {
	tests := []struct {
		name     string
		a, b     uint64
		expected uint64
	}{
		{"a > b", 10, 5, 5},
		{"a < b", 5, 10, 5},
		{"equal", 5, 5, 5},
		{"zero", 0, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MinUint64(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestInt64ToUint32(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		expected  uint32
		expectErr bool
	}{
		{name: "zero", value: 0, expected: 0},
		{name: "positive", value: 42, expected: 42},
		{name: "maximum", value: math.MaxUint32, expected: math.MaxUint32},
		{name: "negative", value: -1, expectErr: true},
		{name: "overflow", value: int64(math.MaxUint32) + 1, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Int64ToUint32(tt.value)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCheckedIntegerConversions(t *testing.T) {
	if value, err := Uint64ToInt64(math.MaxInt64); err != nil || value != math.MaxInt64 {
		t.Fatalf("Uint64ToInt64() = %d, %v", value, err)
	}
	if _, err := Uint64ToInt64(uint64(math.MaxInt64) + 1); err == nil {
		t.Fatal("expected uint64 to int64 overflow error")
	}
	if value, err := Uint64ToUint32(math.MaxUint32); err != nil || value != math.MaxUint32 {
		t.Fatalf("Uint64ToUint32() = %d, %v", value, err)
	}
	if _, err := Uint64ToUint32(uint64(math.MaxUint32) + 1); err == nil {
		t.Fatal("expected uint64 to uint32 overflow error")
	}
	if value, err := Int64ToUint64(math.MaxInt64); err != nil || value != math.MaxInt64 {
		t.Fatalf("Int64ToUint64() = %d, %v", value, err)
	}
	if _, err := Int64ToUint64(-1); err == nil {
		t.Fatal("expected int64 to uint64 signedness error")
	}
	if value, err := Int64ToInt(42); err != nil || value != 42 {
		t.Fatalf("Int64ToInt() = %d, %v", value, err)
	}
	if value, err := Uint32ToInt(42); err != nil || value != 42 {
		t.Fatalf("Uint32ToInt() = %d, %v", value, err)
	}
	if value := Uint32ToInt64(math.MaxUint32); value != math.MaxUint32 {
		t.Fatalf("Uint32ToInt64() = %d", value)
	}
	if value, err := Uint32ToUint8(math.MaxUint8); err != nil || value != math.MaxUint8 {
		t.Fatalf("Uint32ToUint8() = %d, %v", value, err)
	}
	if _, err := Uint32ToUint8(uint32(math.MaxUint8) + 1); err == nil {
		t.Fatal("expected uint32 to uint8 overflow error")
	}
	if value, err := UintToUint32(uint(math.MaxUint32)); err != nil || value != math.MaxUint32 {
		t.Fatalf("UintToUint32() = %d, %v", value, err)
	}
	if value, err := IntToUint32(math.MaxInt32); err != nil || value != math.MaxInt32 {
		t.Fatalf("IntToUint32() = %d, %v", value, err)
	}
	if _, err := IntToUint32(-1); err == nil {
		t.Fatal("expected int to uint32 signedness error")
	}
	if value, err := IntToUint8(math.MaxUint8); err != nil || value != math.MaxUint8 {
		t.Fatalf("IntToUint8() = %d, %v", value, err)
	}
	if _, err := IntToUint8(math.MaxUint8 + 1); err == nil {
		t.Fatal("expected int to uint8 overflow error")
	}
	if value, err := IntToInt32(math.MaxInt32); err != nil || value != math.MaxInt32 {
		t.Fatalf("IntToInt32() = %d, %v", value, err)
	}
	if _, err := IntToInt32(int(math.MaxInt32) + 1); err == nil {
		t.Fatal("expected int to int32 overflow error")
	}
	if _, err := IntToUint64(-1); err == nil {
		t.Fatal("expected int to uint64 signedness error")
	}
	if value := Int64ToString(-42); value != "-42" {
		t.Fatalf("Int64ToString() = %q", value)
	}
}

func TestStringToUint32(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  uint32
		expectErr bool
	}{
		{name: "valid", value: "42", expected: 42},
		{name: "trimmed", value: " 120 ", expected: 120},
		{name: "max", value: "4294967295", expected: 4294967295},
		{name: "overflow", value: "4294967296", expectErr: true},
		{name: "negative", value: "-1", expectErr: true},
		{name: "invalid", value: "abc", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToUint32(tt.value)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestStringToUint64(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  uint64
		expectErr bool
	}{
		{name: "valid", value: "42", expected: 42},
		{name: "trimmed", value: " 120 ", expected: 120},
		{name: "max", value: "18446744073709551615", expected: 18446744073709551615},
		{name: "overflow", value: "18446744073709551616", expectErr: true},
		{name: "negative", value: "-1", expectErr: true},
		{name: "invalid", value: "abc", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToUint64(tt.value)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
