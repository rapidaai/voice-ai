// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

import (
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
)

func TestAnalysisInputCarriesAuthenticationPrincipal(t *testing.T) {
	field, ok := reflect.TypeOf(AnalysisInput{}).FieldByName("Auth")
	if !ok {
		t.Fatal("AnalysisInput.Auth is missing")
	}
	if field.Type != reflect.TypeOf(authentication.Principal{}) {
		t.Fatalf("AnalysisInput.Auth type = %v, want authentication.Principal", field.Type)
	}
}
