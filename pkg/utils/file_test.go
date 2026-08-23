// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOrganizationKnowledgeCollection(t *testing.T) {
	result := OrganizationKnowledgeCollection(1, 2, 3)
	expected := "1__2__3"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestOrganizationObjectPrefix(t *testing.T) {
	result := OrganizationObjectPrefix(1, 2, "test")
	expected := "1/2/test"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestPtr(t *testing.T) {
	v := 42
	result := Ptr(v)
	if *result != v {
		t.Errorf("expected %d, got %d", v, *result)
	}
}

func TestUnPtr(t *testing.T) {
	v := 42
	p := &v
	result := UnPtr(p)
	if result != v {
		t.Errorf("expected %d, got %d", v, result)
	}

	var nilPtr *int
	result2 := UnPtr(nilPtr)
	if result2 != 0 {
		t.Errorf("expected 0 for nil, got %d", result2)
	}
}

func TestIntToString(t *testing.T) {
	result := IntToString(123)
	expected := "123"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestDurationToString(t *testing.T) {
	d := time.Second * 5
	result := DurationToString(d)
	expected := "5000000000"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestDurationToStringNegative(t *testing.T) {
	if result := DurationToString(-time.Nanosecond); result != "-1" {
		t.Fatalf("expected -1, got %s", result)
	}
}

func TestToJson(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	s := TestStruct{Name: "John", Age: 30}
	result := ToJson(s)
	expected := map[string]interface{}{
		"name": "John",
		"age":  float64(30), // json unmarshals to float64
	}
	if len(result) != len(expected) {
		t.Errorf("expected %d keys, got %d", len(expected), len(result))
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("key %s: expected %v, got %v", k, v, result[k])
		}
	}
}

func TestSerialize(t *testing.T) {
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	result, err := Serialize(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var deserialized map[string]interface{}
	err = json.Unmarshal(result, &deserialized)
	if err != nil {
		t.Fatalf("error unmarshaling: %v", err)
	}

	if deserialized["key1"] != "value1" || deserialized["key2"] != float64(42) {
		t.Errorf("deserialized data does not match")
	}
}
