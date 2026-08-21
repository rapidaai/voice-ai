// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with RapidaAI
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestServiceScopeTokenRoundTrip(t *testing.T) {
	secretKey := "test-secret"
	userID := uint64(1)
	projectID := uint64(3)
	want := DelegatedContext{UserID: &userID, OrganizationID: 2, ProjectID: &projectID}

	token, err := CreateServiceScopeToken(want, secretKey)
	if err != nil {
		t.Fatalf("CreateServiceScopeToken() error = %v", err)
	}
	scope, err := ExtractServiceScope(token, secretKey)
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	got, err := ResolveDelegatedContext(scope)
	if err != nil {
		t.Fatalf("ResolveDelegatedContext() error = %v", err)
	}
	if got.OrganizationID != want.OrganizationID || got.UserID == nil || *got.UserID != userID || got.ProjectID == nil || *got.ProjectID != projectID {
		t.Fatalf("delegated context = %+v", got)
	}
	organizationID, err := RequireOrganization(scope)
	if err != nil || organizationID != want.OrganizationID {
		t.Fatalf("RequireOrganization() = %d, %v", organizationID, err)
	}
	projectContext, err := RequireProject(scope)
	if err != nil || projectContext != (ProjectContext{OrganizationID: want.OrganizationID, ProjectID: projectID}) {
		t.Fatalf("RequireProject() = %+v, %v", projectContext, err)
	}
	if _, err := RequireUser(scope); err == nil {
		t.Fatal("RequireUser() error = nil for delegated service scope")
	}
}

func TestCreateServiceScopeTokenAcceptsOrganizationOnly(t *testing.T) {
	token, err := CreateServiceScopeToken(DelegatedContext{OrganizationID: 2}, "test-secret")
	if err != nil || token == "" {
		t.Fatalf("CreateServiceScopeToken() = %q, %v", token, err)
	}
	scope, err := ExtractServiceScope(token, "test-secret")
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	if _, err := RequireProject(scope); err == nil {
		t.Fatal("RequireProject() error = nil without delegated project")
	}
}

func TestCreateServiceScopeTokenRejectsMalformedContext(t *testing.T) {
	zero := uint64(0)
	for _, delegatedContext := range []DelegatedContext{
		{},
		{OrganizationID: 2, UserID: &zero},
		{OrganizationID: 2, ProjectID: &zero},
	} {
		if _, err := CreateServiceScopeToken(delegatedContext, "test-secret"); err == nil {
			t.Fatalf("CreateServiceScopeToken(%+v) error = nil", delegatedContext)
		}
	}
}

func TestExtractServiceScopeRejectsMalformedRequiredOrganization(t *testing.T) {
	secretKey := "test-secret"
	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{name: "missing", claims: jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()}},
		{name: "zero", claims: jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix(), "organizationId": 0}},
		{name: "invalid", claims: jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix(), "organizationId": "invalid"}},
		{name: "fractional", claims: jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix(), "organizationId": 2.5}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, test.claims)
			tokenString, err := token.SignedString([]byte(secretKey))
			if err != nil {
				t.Fatalf("SignedString() error = %v", err)
			}
			if _, err := ExtractServiceScope(tokenString, secretKey); err == nil {
				t.Fatal("ExtractServiceScope() error = nil")
			}
		})
	}
}

func TestExtractServiceScopeRejectsInvalidTokens(t *testing.T) {
	validToken, err := CreateServiceScopeToken(DelegatedContext{OrganizationID: 2}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		token  string
		secret string
	}{
		{name: "invalid", token: "invalid.token.here", secret: "test-secret"},
		{name: "wrong secret", token: validToken, secret: "wrong-secret"},
		{name: "empty", token: "", secret: "test-secret"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ExtractServiceScope(test.token, test.secret); err == nil {
				t.Fatal("ExtractServiceScope() error = nil")
			}
		})
	}
}

func TestToUint64(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  uint64
		ok    bool
	}{
		{name: "float64", value: float64(123), want: 123, ok: true},
		{name: "int", value: 456, want: 456, ok: true},
		{name: "int64", value: int64(789), want: 789, ok: true},
		{name: "uint64", value: uint64(10), want: 10, ok: true},
		{name: "string valid", value: "101112", want: 101112, ok: true},
		{name: "zero", value: 0, ok: false},
		{name: "negative", value: -1, ok: false},
		{name: "fractional", value: 1.5, ok: false},
		{name: "string invalid", value: "not-a-number", ok: false},
		{name: "unsupported type", value: true, ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := toUint64(test.value)
			if ok != test.ok || ok && got != test.want {
				t.Fatalf("toUint64() = %d, %v; want %d, %v", got, ok, test.want, test.ok)
			}
		})
	}
}
