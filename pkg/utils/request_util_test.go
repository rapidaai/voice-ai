// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestError(t *testing.T) {
	result, err := Error[map[string]interface{}](errors.New("test error"), "human message")
	if err == nil {
		t.Error("expected error")
	}
	if result == nil {
		t.Error("expected result")
	}
	// Check structure
	if code, ok := (*result)["Code"].(float64); !ok || code != 400 {
		t.Errorf("expected code 400, got %v", (*result)["Code"])
	}
	if success, ok := (*result)["Success"].(bool); !ok || success != false {
		t.Errorf("expected success false, got %v", (*result)["Success"])
	}
}

func TestErrorWithCode(t *testing.T) {
	result, err := ErrorWithCode[map[string]interface{}](500, errors.New("test"), "message")
	if err == nil {
		t.Error("expected error")
	}
	if code, ok := (*result)["Code"].(float64); !ok || code != 500 {
		t.Errorf("expected code 500, got %v", (*result)["Code"])
	}
}

func TestJustErrorHandlesNegativeCode(t *testing.T) {
	result := JustError(-1, errors.New("test"), "message")
	if result.GetErrorCode() != 0 {
		t.Fatalf("expected error code 0, got %d", result.GetErrorCode())
	}
}

func TestPaginatedSuccess(t *testing.T) {
	data := map[string]string{"key": "value"}
	result, err := PaginatedSuccess[map[string]interface{}, map[string]string](100, 1, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code, ok := (*result)["Code"].(float64); !ok || code != 200 {
		t.Errorf("expected code 200, got %v", (*result)["Code"])
	}
	if success, ok := (*result)["Success"].(bool); !ok || success != true {
		t.Errorf("expected success true, got %v", (*result)["Success"])
	}
}

func TestSuccess(t *testing.T) {
	data := "test data"
	result, err := Success[map[string]interface{}, string](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*result)["Data"] != "test data" {
		t.Errorf("expected 'test data', got %v", (*result)["Data"])
	}
}

func TestJustSuccess(t *testing.T) {
	result, err := JustSuccess()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success != true || result.Code != 200 {
		t.Errorf("expected success true and code 200, got %v", result)
	}
}

func TestCast(t *testing.T) {
	orig := map[string]interface{}{"key": "value"}
	var dst map[string]interface{}
	err := Cast(orig, &dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst["key"] != "value" {
		t.Errorf("expected 'value', got %v", dst["key"])
	}
}

func TestIndexFunc(t *testing.T) {
	s := []int{1, 2, 3, 4}
	index := IndexFunc(s, func(e int) bool { return e == 3 })
	if index != 2 {
		t.Errorf("expected 2, got %d", index)
	}
	index = IndexFunc(s, func(e int) bool { return e == 5 })
	if index != -1 {
		t.Errorf("expected -1, got %d", index)
	}
}

func TestToString(t *testing.T) {
	in := []string{"a", "b", "c"}
	result := ToString(in)
	var out []string
	err := json.NewDecoder(strings.NewReader(result)).Decode(&out)
	if err != nil {
		t.Fatalf("error decoding: %v", err)
	}
	if len(out) != 3 || out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Errorf("expected ['a','b','c'], got %v", out)
	}
}

func TestUint64SliceToString(t *testing.T) {
	in := []uint64{1, 2, 3}
	result := Uint64SliceToString(in)
	var out []uint64
	err := json.NewDecoder(strings.NewReader(result)).Decode(&out)
	if err != nil {
		t.Fatalf("error decoding: %v", err)
	}
	if len(out) != 3 || out[0] != 1 || out[1] != 2 || out[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", out)
	}
}

func TestMapToStruct(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	result := MapToStruct(m)
	if result == nil {
		t.Error("expected non-nil struct")
	}
	if result.Fields["key"].GetStringValue() != "value" {
		t.Errorf("expected 'value', got %v", result.Fields["key"])
	}
}

func TestProtoJson(t *testing.T) {
	s := &structpb.Struct{}
	result := ProtoJson(s)
	if result == "" {
		t.Error("expected non-empty json")
	}
	// Basic check that it's valid JSON
	var m map[string]interface{}
	err := json.Unmarshal([]byte(result), &m)
	if err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
}
