// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"context"
	"strings"
	"testing"
)

func TestClientInfo_ToJson(t *testing.T) {
	ci := &ClientInfo{
		UserAgent: "test",
		Language:  "en",
	}
	jsonStr, err := ci.ToJson()
	if err != nil {
		t.Errorf("ToJson() error = %v", err)
	}
	if jsonStr == "" {
		t.Errorf("ToJson() returned empty string")
	}
	// Check if contains the fields
	if !strings.Contains(jsonStr, "test") {
		t.Errorf("ToJson() = %v, should contain 'test'", jsonStr)
	}
}

func TestNewClientInfoFromContext(t *testing.T) {
	ctx := context.Background()
	// With nil logger, but since it's not used in the function, ok
	ci := NewClientInfoFromContext(ctx, nil)
	if ci == nil {
		t.Errorf("NewClientInfoFromContext() returned nil")
	}
}

func TestGetClientInfoFromGrpcContext(t *testing.T) {
	ctx := context.Background()
	ci := GetClientInfoFromGrpcContext(ctx)
	if ci != nil {
		t.Errorf("GetClientInfoFromGrpcContext() = %v, want nil", ci)
	}
	// With value
	ctx = context.WithValue(ctx, CLIENT_CTX_KEY, &ClientInfo{UserAgent: "test"})
	ci = GetClientInfoFromGrpcContext(ctx)
	if ci == nil || ci.UserAgent != "test" {
		t.Errorf("GetClientInfoFromGrpcContext() = %v, want UserAgent 'test'", ci)
	}
}

func TestAuthenticationSelectionHeadersAreDistinct(t *testing.T) {
	if ORGANIZATION_KEY != "x-organization-id" || ORGANIZATION_KEY == ORG_SCOPE_KEY || ORGANIZATION_KEY == PROJECT_KEY {
		t.Fatalf("authentication selection headers = organization %q, credential %q, project %q", ORGANIZATION_KEY, ORG_SCOPE_KEY, PROJECT_KEY)
	}
}
