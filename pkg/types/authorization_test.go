package types

import (
	"context"
	"errors"
	"testing"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

func TestAuthorize(t *testing.T) {
	user := &PlainAuthPrinciple{
		User:             UserInfo{Id: 1},
		OrganizationRole: &OrganizaitonRole{OrganizationId: 2},
	}

	tests := []struct {
		name string
		ctx  context.Context
		err  error
	}{
		{name: "missing", ctx: context.Background(), err: ErrUnauthenticated},
		{name: "wrong context value", ctx: context.WithValue(context.Background(), CTX_, "invalid"), err: ErrUnauthenticated},
		{name: "unauthenticated", ctx: context.WithValue(context.Background(), CTX_, &OrganizationScope{}), err: ErrUnauthenticated},
		{name: "authenticated", ctx: context.WithValue(context.Background(), CTX_, user)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, err := Authorize(test.ctx)
			if !errors.Is(err, test.err) {
				t.Fatalf("Authorize() error = %v, want %v", err, test.err)
			}
			if test.err == nil && auth != user {
				t.Fatalf("Authorize() = %T, want original user principle", auth)
			}
		})
	}
}

func TestAuthenticationScope(t *testing.T) {
	projectID := uint64(10)
	organizationID := uint64(20)
	tests := []struct {
		name    string
		auth    Authentication
		allowed []AuthType
		err     error
	}{
		{
			name: "user allowed",
			auth: &PlainAuthPrinciple{
				User:             UserInfo{Id: 1},
				OrganizationRole: &OrganizaitonRole{OrganizationId: organizationID},
			},
			allowed: []AuthType{AuthTypeProject, AuthTypeUser},
		},
		{
			name: "project allowed",
			auth: &ProjectScope{
				ProjectId:      &projectID,
				OrganizationId: &organizationID,
				Status:         type_enums.RECORD_ACTIVE.String(),
			},
			allowed: []AuthType{AuthTypeProject},
		},
		{
			name: "organization allowed",
			auth: &OrganizationScope{
				OrganizationId: &organizationID,
				Status:         type_enums.RECORD_ACTIVE.String(),
			},
			allowed: []AuthType{AuthTypeOrg},
		},
		{
			name:    "service allowed",
			auth:    &ServiceScope{OrganizationId: &organizationID},
			allowed: []AuthType{AuthTypeService, AuthTypeService},
		},
		{
			name: "empty allowlist",
			auth: &ServiceScope{OrganizationId: &organizationID},
			err:  ErrAuthenticationScopeNotAllowed,
		},
		{
			name:    "scope rejected",
			auth:    &ServiceScope{OrganizationId: &organizationID},
			allowed: []AuthType{AuthTypeUser, AuthTypeProject},
			err:     ErrAuthenticationScopeNotAllowed,
		},
		{
			name:    "unknown scope rejected",
			auth:    &ServiceScope{OrganizationId: &organizationID},
			allowed: []AuthType{AuthType("unknown")},
			err:     ErrAuthenticationScopeNotAllowed,
		},
		{
			name:    "unauthenticated rejected",
			auth:    &OrganizationScope{},
			allowed: []AuthType{AuthTypeOrg},
			err:     ErrUnauthenticated,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scoped, err := test.auth.Scope(test.allowed...)
			if !errors.Is(err, test.err) {
				t.Fatalf("Scope() error = %v, want %v", err, test.err)
			}
			if test.err == nil && scoped != test.auth {
				t.Fatalf("Scope() did not return original authentication")
			}
		})
	}
}
