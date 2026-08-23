// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"encoding/json"
	"testing"
)

func TestOption_GetUint64(t *testing.T) {
	m := Option{
		"uint64":  uint64(42),
		"uint32":  uint32(43),
		"int64":   int64(44),
		"float64": 45.0,
		"string":  "46",
		"jsonNum": json.Number("47"),
		"bytes":   []byte("48"),
		"nil":     nil,
	}

	tests := []struct {
		name     string
		key      string
		expected uint64
		hasError bool
	}{
		{"uint64", "uint64", 42, false},
		{"uint32", "uint32", 43, false},
		{"int64", "int64", 44, false},
		{"float64", "float64", 45, false},
		{"string", "string", 46, false},
		{"jsonNum", "jsonNum", 47, false},
		{"bytes", "bytes", 48, false},
		{"not found", "missing", 0, true},
		{"nil", "nil", 0, true},
		{"negative int", "neg", 0, true}, // add to map
	}

	// Add negative for error case
	m["neg"] = int64(-1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.GetUint64(tt.key)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestOption_GetString(t *testing.T) {
	m := Option{
		"string": "hello",
		"bytes":  []byte("world"),
		"int":    42,
		"float":  3.14,
	}

	tests := []struct {
		name     string
		key      string
		expected string
		hasError bool
	}{
		{"string", "string", "hello", false},
		{"bytes", "bytes", "world", false},
		{"int", "int", "42", false},
		{"float", "float", "3.14", false},
		{"not found", "missing", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.GetString(tt.key)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestOption_GetUint32(t *testing.T) {
	m := Option{
		"uint32":  uint32(42),
		"uint64":  uint64(43),
		"uint":    uint(46),
		"float64": 44.0,
		"string":  "45",
	}

	tests := []struct {
		name     string
		key      string
		expected uint32
		hasError bool
	}{
		{"uint32", "uint32", 42, false},
		{"uint64", "uint64", 43, false},
		{"uint", "uint", 46, false},
		{"float64", "float64", 44, false},
		{"string", "string", 45, false},
		{"not found", "missing", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.GetUint32(tt.key)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestOption_GetFloat64(t *testing.T) {
	m := Option{
		"float64": 3.14,
		"float32": float32(2.0),
		"int":     42,
		"string":  "1.23",
	}

	tests := []struct {
		name     string
		key      string
		expected float64
		hasError bool
	}{
		{"float64", "float64", 3.14, false},
		{"float32", "float32", 2.0, false},
		{"int", "int", 42.0, false},
		{"string", "string", 1.23, false},
		{"not found", "missing", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.GetFloat64(tt.key)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %f, got %f", tt.expected, result)
				}
			}
		})
	}
}

func TestOption_GetStringMap(t *testing.T) {
	m := Option{
		"jsonStr": `{"key1":"value1","key2":"value2"}`,
		"map":     map[string]interface{}{"key3": "value3"},
	}

	tests := []struct {
		name     string
		key      string
		expected map[string]string
		hasError bool
	}{
		{"json string", "jsonStr", map[string]string{"key1": "value1", "key2": "value2"}, false},
		{"map", "map", map[string]string{"key3": "value3"}, false},
		{"not found", "missing", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.GetStringMap(tt.key)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				for k, v := range tt.expected {
					if result[k] != v {
						t.Errorf("key %s: expected %s, got %s", k, v, result[k])
					}
				}
			}
		})
	}
}

func TestOption_GetBool(t *testing.T) {
	m := Option{
		"bool":   true,
		"string": "true",
		"int1":   1,
		"int0":   0,
		"float1": 1.0,
		"float0": 0.0,
	}

	tests := []struct {
		name     string
		key      string
		expected bool
		hasError bool
	}{
		{"bool true", "bool", true, false},
		{"string true", "string", true, false},
		{"int 1", "int1", true, false},
		{"int 0", "int0", false, false},
		{"float 1", "float1", true, false},
		{"float 0", "float0", false, false},
		{"not found", "missing", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.GetBool(tt.key)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %t, got %t", tt.expected, result)
				}
			}
		})
	}
}

func TestNormalizeInterface(t *testing.T) {
	input := map[string]interface{}{
		"string":  "hello",
		"jsonStr": `{"nested":"value"}`,
		"number":  42.0,
		"bool":    true,
		"nil":     nil,
		"array":   []interface{}{"item1", `{"key":"val"}`},
	}

	result := NormalizeInterface(input)

	if result["string"] != "hello" {
		t.Errorf("expected 'hello', got %v", result["string"])
	}

	if nested, ok := result["jsonStr"].(map[string]interface{}); !ok || nested["nested"] != "value" {
		t.Errorf("expected nested map, got %v", result["jsonStr"])
	}

	if result["number"] != 42.0 {
		t.Errorf("expected 42.0, got %v", result["number"])
	}

	if result["bool"] != true {
		t.Errorf("expected true, got %v", result["bool"])
	}

	if _, exists := result["nil"]; exists {
		t.Error("nil key should be removed")
	}

	if arr, ok := result["array"].([]interface{}); ok {
		if arr[0] != "item1" {
			t.Errorf("array[0] expected 'item1', got %v", arr[0])
		}
		if nestedArr, ok := arr[1].(map[string]interface{}); !ok || nestedArr["key"] != "val" {
			t.Errorf("array[1] expected nested map, got %v", arr[1])
		}
	} else {
		t.Errorf("expected array, got %v", result["array"])
	}
}
