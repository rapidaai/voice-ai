// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package validator contains small reusable validation helpers.
package validator

import (
	"math"
	"net/mail"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/rapidaai/protos"
)

var phonePattern = regexp.MustCompile(`^\+?[0-9]+$`)

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// OneOf returns true when value matches one of the provided options.
func OneOf[T comparable](value T, options ...T) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

// NotEmpty returns true when the provided slice has at least one value.
func NotEmpty[T any](values []T) bool {
	return len(values) > 0
}

// NonNil returns true when value is not nil.
func NonNil(value interface{}) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return !v.IsNil()
	default:
		return true
	}
}

// NotBlank returns true when value has non-whitespace content.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// Numeric returns true when value can be parsed as a finite number.
func Numeric(value string) bool {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
}

// Between returns true when value is within the inclusive min/max range.
func Between[T numeric](value, min, max T) bool {
	return value >= min && value <= max
}

// Email returns true when value is a valid mailbox exactly as provided.
func Email(value string) bool {
	parsedEmail, err := mail.ParseAddress(value)
	return err == nil && parsedEmail.Address == value && parsedEmail.Name == ""
}

// Phone returns true when value contains only ASCII digits with an optional leading plus.
func Phone(value string) bool {
	return phonePattern.MatchString(value)
}

// AllNonZero returns true when every provided value is not its zero value.
func AllNonZero[T comparable](values ...T) bool {
	var zero T
	for _, value := range values {
		if value == zero {
			return false
		}
	}
	return true
}

// NonZero returns true when value is not its zero value.
func NonZero[T comparable](value T) bool {
	var zero T
	return value != zero
}

// OfAssistantDefinition returns true when an assistant definition has a valid
// assistant ID and either no version or a valid version.
func OfAssistantDefinition(assistant *protos.AssistantDefinition) bool {
	if assistant == nil || assistant.GetAssistantId() == 0 {
		return false
	}
	version := assistant.GetVersion()
	if version == "" || version == "latest" {
		return true
	}
	if !strings.HasPrefix(version, "vrsn_") {
		return false
	}
	versionID, err := strconv.ParseUint(strings.TrimPrefix(version, "vrsn_"), 10, 64)
	return err == nil && versionID > 0
}
