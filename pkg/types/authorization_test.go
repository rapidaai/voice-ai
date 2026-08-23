package types

import (
	"context"
	"errors"
	"testing"
)

func TestAuthorize(t *testing.T) {
	actor := ActorIdentity{Type: ActorTypeUser, ID: 1}
	authentication := &Authentication{
		AuthType:          AuthTypeUser,
		ActorValue:        &actor,
		UserValue:         &UserContext{UserID: 1},
		OrganizationValue: &OrganizationContext{OrganizationID: 2},
	}
	tests := []struct {
		name string
		ctx  context.Context
		err  error
	}{
		{name: "missing", ctx: context.Background(), err: ErrUnauthenticated},
		{name: "wrong context value", ctx: context.WithValue(context.Background(), CTX_, "invalid"), err: ErrUnauthenticated},
		{name: "unauthenticated", ctx: context.WithValue(context.Background(), CTX_, &Authentication{}), err: ErrUnauthenticated},
		{name: "authenticated", ctx: context.WithValue(context.Background(), CTX_, authentication)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, err := Authorize(test.ctx)
			if !errors.Is(err, test.err) {
				t.Fatalf("Authorize() error = %v, want %v", err, test.err)
			}
			if test.err == nil && auth != authentication {
				t.Fatal("Authorize() did not return request authentication")
			}
		})
	}
}

func TestAuthenticationScopeAndContexts(t *testing.T) {
	actor := ActorIdentity{Type: ActorTypeUser, ID: 1}
	auth := &Authentication{
		AuthType:          AuthTypeUser,
		ActorValue:        &actor,
		UserValue:         &UserContext{UserID: 1},
		OrganizationValue: &OrganizationContext{OrganizationID: 2},
		ProjectValue:      &ProjectContext{OrganizationID: 2, ProjectID: 3},
	}
	if scoped, err := auth.Scope(AuthTypeProject, AuthTypeUser); err != nil || scoped != auth {
		t.Fatalf("Scope() = %v, %v", scoped, err)
	}
	if _, err := auth.Scope(); !errors.Is(err, ErrAuthenticationScopeNotAllowed) {
		t.Fatalf("Scope() error = %v", err)
	}
	if value := auth.Actor(); value != actor {
		t.Fatalf("Actor() = %+v", value)
	}
	if value, err := auth.UserContext(); err != nil || value.UserID != 1 {
		t.Fatalf("UserContext() = %+v, %v", value, err)
	}
	if value, err := auth.OrganizationContext(); err != nil || value.OrganizationID != 2 {
		t.Fatalf("OrganizationContext() = %+v, %v", value, err)
	}
	if value, err := auth.ProjectContext(); err != nil || value.ProjectID != 3 {
		t.Fatalf("ProjectContext() = %+v, %v", value, err)
	}
}

func TestAuthenticationUnavailableValues(t *testing.T) {
	auth := &Authentication{AuthType: AuthTypeOrg}
	if actor := auth.Actor(); actor != (ActorIdentity{}) {
		t.Fatalf("Actor() = %+v, want zero value", actor)
	}
	for name, run := range map[string]func() error{
		"user":         func() error { _, err := auth.UserContext(); return err },
		"organization": func() error { _, err := auth.OrganizationContext(); return err },
		"project":      func() error { _, err := auth.ProjectContext(); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(); err == nil {
				t.Fatal("error = nil")
			}
		})
	}
}
