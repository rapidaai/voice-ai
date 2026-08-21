package middlewares

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type organizationScopeClaimAuthenticatorStub struct {
	err error
}

type projectSelectingUserAuthenticatorStub struct {
	auth *types.PlainAuthPrinciple
}

func (stub projectSelectingUserAuthenticatorStub) Authorize(context.Context, string, uint64) (types.Principle, error) {
	return stub.auth, nil
}

func (stub projectSelectingUserAuthenticatorStub) AuthPrinciple(context.Context, uint64) (types.Principle, error) {
	return stub.auth, nil
}

func (stub organizationScopeClaimAuthenticatorStub) Claim(
	context.Context,
	string,
) (*types.PlainClaimPrinciple[*types.OrganizationScope], error) {
	if stub.err != nil {
		return nil, stub.err
	}
	organizationID := uint64(2)
	return &types.PlainClaimPrinciple[*types.OrganizationScope]{
		Info: &types.OrganizationScope{
			OrganizationId: &organizationID,
			Status:         type_enums.RECORD_ACTIVE.String(),
		},
	}, nil
}

func TestAuthenticationBoundaryUnaryAcceptsEveryCredentialClass(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected types.AuthType
	}{
		{name: "user", values: []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"}, expected: types.AuthTypeUser},
		{name: "project", values: []string{types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX + "project"}, expected: types.AuthTypeProject},
		{name: "organization", values: []string{types.ORG_SCOPE_KEY, "organization"}, expected: types.AuthTypeOrg},
		{name: "service", values: []string{types.SERVICE_SCOPE_KEY, "service"}, expected: types.AuthTypeService},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
				userAuthenticatorStub{},
				&projectScopeClaimAuthenticatorStub{},
				organizationScopeClaimAuthenticatorStub{},
				serviceScopeClaimAuthenticatorStub{},
				nil,
			)(incomingContext(test.values...), nil, nil, func(ctx context.Context, _ any) (any, error) {
				auth, err := types.Authorize(ctx)
				if err != nil {
					t.Fatalf("Authorize() error = %v", err)
				}
				if auth.Type() != test.expected {
					t.Fatalf("authentication type = %v, want %v", auth.Type(), test.expected)
				}
				return nil, nil
			})
			if err != nil {
				t.Fatalf("middleware error = %v", err)
			}
		})
	}
}

func TestAuthenticationBoundaryStreamAcceptsEveryCredentialClass(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected types.AuthType
	}{
		{name: "user", values: []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"}, expected: types.AuthTypeUser},
		{name: "project", values: []string{types.PROJECT_SCOPE_KEY, "project"}, expected: types.AuthTypeProject},
		{name: "organization", values: []string{types.ORG_SCOPE_KEY, "organization"}, expected: types.AuthTypeOrg},
		{name: "service", values: []string{types.SERVICE_SCOPE_KEY, "service"}, expected: types.AuthTypeService},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewAuthenticationBoundaryStreamServerMiddleware(
				userAuthenticatorStub{},
				&projectScopeClaimAuthenticatorStub{},
				organizationScopeClaimAuthenticatorStub{},
				serviceScopeClaimAuthenticatorStub{},
				nil,
			)(nil, &testServerStream{ctx: incomingContext(test.values...)}, nil, func(_ any, stream grpc.ServerStream) error {
				auth, err := types.Authorize(stream.Context())
				if err != nil {
					t.Fatalf("Authorize() error = %v", err)
				}
				if auth.Type() != test.expected {
					t.Fatalf("authentication type = %v, want %v", auth.Type(), test.expected)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("middleware error = %v", err)
			}
		})
	}
}

func TestAuthenticationBoundaryUserProjectSelectionCompatibility(t *testing.T) {
	organizationID := uint64(2)
	selectedProjectID := uint64(20)
	user := &types.PlainAuthPrinciple{
		User:             types.UserInfo{Id: 1},
		OrganizationRole: &types.OrganizaitonRole{OrganizationId: organizationID},
		ProjectRoles: []*types.ProjectRole{
			{ProjectId: 10},
			{ProjectId: selectedProjectID},
		},
	}

	_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
		projectSelectingUserAuthenticatorStub{auth: user},
		&projectScopeClaimAuthenticatorStub{},
		organizationScopeClaimAuthenticatorStub{},
		serviceScopeClaimAuthenticatorStub{},
		nil,
	)(incomingContext(
		types.AUTHORIZATION_KEY, "token",
		types.AUTH_KEY, "1",
		types.PROJECT_KEY, "20",
	), nil, nil, func(ctx context.Context, _ any) (any, error) {
		auth, err := types.Authorize(ctx)
		if err != nil {
			t.Fatalf("Authorize() error = %v", err)
		}
		projectContext, err := auth.ProjectContext()
		if err != nil {
			t.Fatalf("ProjectContext() error = %v", err)
		}
		if projectContext.ProjectID != selectedProjectID {
			t.Fatalf("selected project = %d, want %d", projectContext.ProjectID, selectedProjectID)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("middleware error = %v", err)
	}
}

func TestAuthenticationBoundaryRejectsUnauthorizedUserProjectSelection(t *testing.T) {
	organizationID := uint64(2)
	user := &types.PlainAuthPrinciple{
		User:             types.UserInfo{Id: 1},
		OrganizationRole: &types.OrganizaitonRole{OrganizationId: organizationID},
		ProjectRoles:     []*types.ProjectRole{{ProjectId: 10}},
	}
	handlerCalled := false

	_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
		projectSelectingUserAuthenticatorStub{auth: user},
		&projectScopeClaimAuthenticatorStub{},
		organizationScopeClaimAuthenticatorStub{},
		serviceScopeClaimAuthenticatorStub{},
		nil,
	)(incomingContext(
		types.AUTHORIZATION_KEY, "token",
		types.AUTH_KEY, "1",
		types.PROJECT_KEY, "20",
	), nil, nil, func(context.Context, any) (any, error) {
		handlerCalled = true
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
	if handlerCalled {
		t.Fatal("handler called after unauthorized project selection")
	}
}

func TestAuthenticationBoundaryAllowsMissingCredentials(t *testing.T) {
	handlerCalled := false
	_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
		userAuthenticatorStub{},
		&projectScopeClaimAuthenticatorStub{},
		organizationScopeClaimAuthenticatorStub{},
		serviceScopeClaimAuthenticatorStub{},
		nil,
	)(context.Background(), nil, nil, func(ctx context.Context, _ any) (any, error) {
		handlerCalled = true
		if _, err := types.Authorize(ctx); err == nil {
			t.Fatal("Authorize() succeeded without credentials")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("middleware error = %v", err)
	}
	if !handlerCalled {
		t.Fatal("handler not called for credential-free request")
	}
}

func TestAuthenticationBoundaryRejectsConflictingCredentials(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{name: "user and project", values: []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1", types.PROJECT_SCOPE_KEY, "project"}},
		{name: "user and organization", values: []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1", types.ORG_SCOPE_KEY, "organization"}},
		{name: "user and service", values: []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1", types.SERVICE_SCOPE_KEY, "service"}},
		{name: "project and organization", values: []string{types.PROJECT_SCOPE_KEY, "project", types.ORG_SCOPE_KEY, "organization"}},
		{name: "project and service", values: []string{types.PROJECT_SCOPE_KEY, "project", types.SERVICE_SCOPE_KEY, "service"}},
		{name: "organization and service", values: []string{types.ORG_SCOPE_KEY, "organization", types.SERVICE_SCOPE_KEY, "service"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerCalled := false
			_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
				userAuthenticatorStub{},
				&projectScopeClaimAuthenticatorStub{},
				organizationScopeClaimAuthenticatorStub{},
				serviceScopeClaimAuthenticatorStub{},
				nil,
			)(incomingContext(test.values...), nil, nil, func(context.Context, any) (any, error) {
				handlerCalled = true
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
			}
			if handlerCalled {
				t.Fatal("handler called for invalid credential selection")
			}
		})
	}
}

func TestAuthenticationBoundaryRejectsResolverFailures(t *testing.T) {
	tests := []struct {
		name         string
		values       []string
		user         types.Authenticator
		project      types.ClaimAuthenticator[*types.ProjectScope]
		organization types.ClaimAuthenticator[*types.OrganizationScope]
		service      types.ClaimAuthenticator[*types.ServiceScope]
	}{
		{
			name: "user", values: []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"},
			user: userAuthenticatorStub{err: errors.New("rejected")}, project: &projectScopeClaimAuthenticatorStub{},
			organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "project", values: []string{types.PROJECT_SCOPE_KEY, "project"},
			user: userAuthenticatorStub{}, project: &projectScopeClaimAuthenticatorStub{err: errors.New("rejected")},
			organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "organization", values: []string{types.ORG_SCOPE_KEY, "organization"},
			user: userAuthenticatorStub{}, project: &projectScopeClaimAuthenticatorStub{},
			organization: organizationScopeClaimAuthenticatorStub{err: errors.New("rejected")}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "service", values: []string{types.SERVICE_SCOPE_KEY, "service"},
			user: userAuthenticatorStub{}, project: &projectScopeClaimAuthenticatorStub{},
			organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{err: errors.New("rejected")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerCalled := false
			_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
				test.user, test.project, test.organization, test.service, nil,
			)(incomingContext(test.values...), nil, nil, func(context.Context, any) (any, error) {
				handlerCalled = true
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
			}
			if handlerCalled {
				t.Fatal("handler called after resolver failure")
			}
		})
	}
}

func TestAuthenticationBoundaryRejectsUnsupportedCredentialClass(t *testing.T) {
	tests := []struct {
		name         string
		values       []string
		user         types.Authenticator
		project      types.ClaimAuthenticator[*types.ProjectScope]
		organization types.ClaimAuthenticator[*types.OrganizationScope]
		service      types.ClaimAuthenticator[*types.ServiceScope]
	}{
		{
			name: "user", values: []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"},
			project: &projectScopeClaimAuthenticatorStub{}, organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "project", values: []string{types.PROJECT_SCOPE_KEY, "project"},
			user: userAuthenticatorStub{}, organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "organization", values: []string{types.ORG_SCOPE_KEY, "organization"},
			user: userAuthenticatorStub{}, project: &projectScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "service", values: []string{types.SERVICE_SCOPE_KEY, "service"},
			user: userAuthenticatorStub{}, project: &projectScopeClaimAuthenticatorStub{}, organization: organizationScopeClaimAuthenticatorStub{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerCalled := false
			_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
				test.user, test.project, test.organization, test.service, nil,
			)(incomingContext(test.values...), nil, nil, func(context.Context, any) (any, error) {
				handlerCalled = true
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
			}
			if handlerCalled {
				t.Fatal("handler called for unsupported credential class")
			}
		})
	}
}
