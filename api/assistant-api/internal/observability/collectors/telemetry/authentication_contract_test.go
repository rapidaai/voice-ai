// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package telemetry

import (
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
)

func TestConfigCarriesAuthenticationPrincipal(t *testing.T) {
	field, ok := reflect.TypeOf(Config{}).FieldByName("Auth")
	if !ok {
		t.Fatal("Config.Auth is missing")
	}
	if field.Type != reflect.TypeOf(authentication.Principal{}) {
		t.Fatalf("Config.Auth type = %v, want authentication.Principal", field.Type)
	}
}
