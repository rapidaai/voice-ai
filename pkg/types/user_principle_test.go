// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"testing"
)

func TestUserInfo_GetId(t *testing.T) {
	u := &UserInfo{Id: 123}
	if u.GetId() != 123 {
		t.Errorf("GetId() = %v, want %v", u.GetId(), 123)
	}
}

func TestUserInfo_GetName(t *testing.T) {
	u := &UserInfo{Name: "test"}
	if u.GetName() != "test" {
		t.Errorf("GetName() = %v, want %v", u.GetName(), "test")
	}
}

func TestUserInfo_GetEmail(t *testing.T) {
	u := &UserInfo{Email: "test@example.com"}
	if u.GetEmail() != "test@example.com" {
		t.Errorf("GetEmail() = %v, want %v", u.GetEmail(), "test@example.com")
	}
}

func TestProjectRole_GetRole(t *testing.T) {
	p := &ProjectRole{Role: "admin"}
	if p.GetRole() != "admin" {
		t.Errorf("GetRole() = %v, want %v", p.GetRole(), "admin")
	}
}

func TestProjectRole_GetProjectId(t *testing.T) {
	p := &ProjectRole{ProjectId: 456}
	if p.GetProjectId() != 456 {
		t.Errorf("GetProjectId() = %v, want %v", p.GetProjectId(), 456)
	}
}

func TestPlainAuthPrincipleCapabilities(t *testing.T) {
	principle := &PlainAuthPrinciple{
		User:               UserInfo{Id: 7},
		OrganizationRole:   &OrganizaitonRole{OrganizationId: 8},
		CurrentProjectRole: &ProjectRole{ProjectId: 9},
	}

	if userID, ok := principle.UserIdentity(); !ok || userID != 7 {
		t.Fatalf("UserIdentity() = %d, %v", userID, ok)
	}
	if organizationID, ok := principle.OrganizationContext(); !ok || organizationID != 8 {
		t.Fatalf("OrganizationContext() = %d, %v", organizationID, ok)
	}
	projectContext, ok := principle.ProjectContext()
	if !ok || projectContext != (ProjectContext{OrganizationID: 8, ProjectID: 9}) {
		t.Fatalf("ProjectContext() = %+v, %v", projectContext, ok)
	}
}

func TestPlainAuthPrincipleAuthenticatesWithoutProject(t *testing.T) {
	principle := &PlainAuthPrinciple{
		User:             UserInfo{Id: 7},
		OrganizationRole: &OrganizaitonRole{OrganizationId: 8},
	}

	if !principle.IsAuthenticated() {
		t.Fatal("IsAuthenticated() = false, want true")
	}
	if _, ok := principle.ProjectContext(); ok {
		t.Fatal("ProjectContext() ok = true without selected project")
	}
}

var (
	_ AuthenticationPrinciple     = (*PlainAuthPrinciple)(nil)
	_ UserIdentityProvider        = (*PlainAuthPrinciple)(nil)
	_ OrganizationContextProvider = (*PlainAuthPrinciple)(nil)
	_ ProjectContextProvider      = (*PlainAuthPrinciple)(nil)
	_ ActorIdentityProvider       = (*PlainAuthPrinciple)(nil)
	_ Principle                   = (*PlainAuthPrinciple)(nil)
)
