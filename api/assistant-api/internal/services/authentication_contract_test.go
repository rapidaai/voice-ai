// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_services

import (
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
)

func TestAssistantServiceAcceptsAuthenticationPrincipal(t *testing.T) {
	method, ok := reflect.TypeOf((*AssistantService)(nil)).Elem().MethodByName("Get")
	if !ok {
		t.Fatal("AssistantService.Get is missing")
	}

	principalType := reflect.TypeOf(authentication.Principal{})
	for index := 0; index < method.Type.NumIn(); index++ {
		if method.Type.In(index) == principalType {
			return
		}
	}
	t.Fatal("AssistantService.Get must accept authentication.Principal")
}
