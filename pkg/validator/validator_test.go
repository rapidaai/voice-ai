// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package validator

import (
	"testing"

	"github.com/rapidaai/protos"
)

func TestOneOf(t *testing.T) {
	if !OneOf("admin", "owner", "admin") {
		t.Fatal("expected value to match one option")
	}
	if OneOf("member", "owner", "admin") {
		t.Fatal("expected value not to match any option")
	}
}

func TestNotEmpty(t *testing.T) {
	if !NotEmpty([]uint64{1}) {
		t.Fatal("expected non-empty slice to pass validation")
	}
	if NotEmpty([]uint64{}) {
		t.Fatal("expected empty slice to fail validation")
	}
}

func TestNonNil(t *testing.T) {
	value := "value"
	if !NonNil(&value) {
		t.Fatal("expected non-nil pointer to pass validation")
	}
	var valuePtr *string
	if NonNil(valuePtr) {
		t.Fatal("expected nil pointer to fail validation")
	}
	var iface interface{} = valuePtr
	if NonNil(iface) {
		t.Fatal("expected typed nil interface value to fail validation")
	}
	if !NonNil("value") {
		t.Fatal("expected non-pointer value to pass validation")
	}
}

func TestNotBlank(t *testing.T) {
	if !NotBlank("value") {
		t.Fatal("expected non-blank string to pass validation")
	}
	if NotBlank(" ") {
		t.Fatal("expected blank string to fail validation")
	}
}

func TestBetween(t *testing.T) {
	intTests := []struct {
		name  string
		value int
		min   int
		max   int
		want  bool
	}{
		{name: "inside", value: 5, min: 1, max: 10, want: true},
		{name: "lower bound", value: 1, min: 1, max: 10, want: true},
		{name: "upper bound", value: 10, min: 1, max: 10, want: true},
		{name: "below", value: 0, min: 1, max: 10, want: false},
		{name: "above", value: 11, min: 1, max: 10, want: false},
		{name: "invalid range", value: 5, min: 10, max: 1, want: false},
	}

	for _, tt := range intTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Between(tt.value, tt.min, tt.max); got != tt.want {
				t.Fatalf("Between(%d, %d, %d) = %v, want %v", tt.value, tt.min, tt.max, got, tt.want)
			}
		})
	}

	floatTests := []struct {
		name  string
		value float64
		min   float64
		max   float64
		want  bool
	}{
		{name: "float inside", value: 1.5, min: 0.5, max: 5, want: true},
		{name: "float lower bound", value: 0.5, min: 0.5, max: 5, want: true},
		{name: "float upper bound", value: 5, min: 0.5, max: 5, want: true},
		{name: "float below", value: 0.4, min: 0.5, max: 5, want: false},
		{name: "float above", value: 5.1, min: 0.5, max: 5, want: false},
	}

	for _, tt := range floatTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Between(tt.value, tt.min, tt.max); got != tt.want {
				t.Fatalf("Between(%f, %f, %f) = %v, want %v", tt.value, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	if !Email("user@example.com") {
		t.Fatal("expected valid email to pass validation")
	}
	if Email(" user@example.com") {
		t.Fatal("expected email with leading space to fail validation")
	}
	if Email("User <user@example.com>") {
		t.Fatal("expected named address to fail exact email validation")
	}
	if Email("not-email") {
		t.Fatal("expected invalid email to fail validation")
	}
}

func TestAllNonZero(t *testing.T) {
	if !AllNonZero(uint64(1), uint64(2)) {
		t.Fatal("expected all values to be non-zero")
	}
	if AllNonZero(uint64(1), uint64(0)) {
		t.Fatal("expected zero value to fail validation")
	}
}

func TestNonZero(t *testing.T) {
	if !NonZero(uint64(1)) {
		t.Fatal("expected non-zero uint64 to pass validation")
	}
	if NonZero(uint64(0)) {
		t.Fatal("expected zero uint64 to fail validation")
	}
}

func TestOfAssistantDefinition(t *testing.T) {
	tests := []struct {
		name      string
		assistant *protos.AssistantDefinition
		want      bool
	}{
		{name: "nil assistant", assistant: nil, want: false},
		{name: "zero assistant id", assistant: &protos.AssistantDefinition{}, want: false},
		{name: "empty version", assistant: &protos.AssistantDefinition{AssistantId: 1}, want: true},
		{name: "latest version", assistant: &protos.AssistantDefinition{AssistantId: 1, Version: "latest"}, want: true},
		{name: "explicit version", assistant: &protos.AssistantDefinition{AssistantId: 1, Version: "vrsn_123"}, want: true},
		{name: "numeric version without prefix", assistant: &protos.AssistantDefinition{AssistantId: 1, Version: "123"}, want: false},
		{name: "zero explicit version", assistant: &protos.AssistantDefinition{AssistantId: 1, Version: "vrsn_0"}, want: false},
		{name: "invalid explicit version", assistant: &protos.AssistantDefinition{AssistantId: 1, Version: "vrsn_abc"}, want: false},
		{name: "invalid version", assistant: &protos.AssistantDefinition{AssistantId: 1, Version: "abc"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OfAssistantDefinition(tt.assistant); got != tt.want {
				t.Fatalf("OfAssistantDefinition() = %v, want %v", got, tt.want)
			}
		})
	}
}
