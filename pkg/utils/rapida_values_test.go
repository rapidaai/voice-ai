// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"math"
	"strconv"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestAnyMapToInterfaceMap(t *testing.T) {
	// Create some Any values
	boolAny, _ := anypb.New(wrapperspb.Bool(true))
	stringAny, _ := anypb.New(wrapperspb.String("hello"))

	anyMap := map[string]*anypb.Any{
		"bool":   boolAny,
		"string": stringAny,
	}

	result, err := AnyMapToInterfaceMap(anyMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["bool"] != true {
		t.Errorf("expected true, got %v", result["bool"])
	}
	if result["string"] != "hello" {
		t.Errorf("expected 'hello', got %v", result["string"])
	}
}

func TestAnyToBool(t *testing.T) {
	boolAny, _ := anypb.New(wrapperspb.Bool(true))
	result, err := AnyToBool(boolAny)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestAnyToString(t *testing.T) {
	stringAny, _ := anypb.New(wrapperspb.String("test"))
	result, err := AnyToString(stringAny)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}

func TestBoolToAny(t *testing.T) {
	anyValue, err := BoolToAny(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := AnyToBool(anyValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestStringToAny(t *testing.T) {
	anyValue, err := StringToAny("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := AnyToString(anyValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}

func TestToIntAny(t *testing.T) {
	anyValue := ToIntAny(42)
	result, err := AnyToInt(anyValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestIntAnyRejectsInt32Overflow(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("int32 overflow value requires a 64-bit int")
	}
	value := int64(math.MaxInt32) + 1
	if anyValue := ToIntAny(int(value)); anyValue != nil {
		t.Fatal("expected nil for int32 overflow")
	}
	if _, err := InterfaceToAnyValue(int(value)); err == nil {
		t.Fatal("expected error for int32 overflow")
	}
}

func TestToStringAny(t *testing.T) {
	anyValue := ToStringAny("test")
	result, err := AnyToString(anyValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}

func TestToUInt64Any(t *testing.T) {
	anyValue := ToUInt64Any(42)
	result, err := AnyToUInt64(anyValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

// TestInterfaceMapRoundTrip reproduces the exact path that tool call
// arguments take: map[string]interface{} → InterfaceMapToAnyMap → wire →
// AnyMapToInterfaceMap → type-assert string. This is the transfer_call flow.
func TestInterfaceMapRoundTrip(t *testing.T) {
	original := map[string]interface{}{
		"to":     "+14155551234",
		"reason": "customer requested agent",
	}

	anyMap, err := InterfaceMapToAnyMap(original)
	if err != nil {
		t.Fatalf("InterfaceMapToAnyMap error: %v", err)
	}

	recovered, err := AnyMapToInterfaceMap(anyMap)
	if err != nil {
		t.Fatalf("AnyMapToInterfaceMap error: %v", err)
	}

	to, ok := recovered["to"].(string)
	if !ok {
		t.Fatalf("expected recovered[\"to\"] to be string, got %T: %v", recovered["to"], recovered["to"])
	}
	if to != "+14155551234" {
		t.Errorf("expected '+14155551234', got %q", to)
	}

	reason, ok := recovered["reason"].(string)
	if !ok {
		t.Fatalf("expected recovered[\"reason\"] to be string, got %T: %v", recovered["reason"], recovered["reason"])
	}
	if reason != "customer requested agent" {
		t.Errorf("expected 'customer requested agent', got %q", reason)
	}
}
