package middlewares

import (
	"context"
	"errors"
	"math"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type organizationScopeClaimAuthenticatorStub struct {
	err          error
	credentialID *uint64
	omitActor    bool
}

type projectSelectingUserAuthenticatorStub struct {
	auth *types.PlainAuthPrinciple
}

type boundaryProjectScopeClaimAuthenticatorStub struct {
	credentialID *uint64
	omitActor    bool
}

func (stub boundaryProjectScopeClaimAuthenticatorStub) Claim(
	context.Context,
	string,
) (*types.PlainClaimPrinciple[*types.ProjectScope], error) {
	credentialID := uint64(1)
	if stub.credentialID != nil {
		credentialID = *stub.credentialID
	}
	projectID := uint64(1)
	organizationID := uint64(1)
	projectScope := &types.ProjectScope{
		CredentialId:   &credentialID,
		ProjectId:      &projectID,
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}
	if stub.omitActor {
		projectScope.CredentialId = nil
	}
	return &types.PlainClaimPrinciple[*types.ProjectScope]{Info: projectScope}, nil
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
	credentialID := uint64(3)
	if stub.credentialID != nil {
		credentialID = *stub.credentialID
	}
	organizationScope := &types.OrganizationScope{
		CredentialId:   &credentialID,
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}
	if stub.omitActor {
		organizationScope.CredentialId = nil
	}
	return &types.PlainClaimPrinciple[*types.OrganizationScope]{
		Info: organizationScope,
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
				boundaryProjectScopeClaimAuthenticatorStub{},
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
				if test.expected == types.AuthTypeOrg {
					actor, err := auth.Actor()
					if err != nil || actor != (types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 3}) {
						t.Fatalf("organization actor = %+v, %v", actor, err)
					}
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
				boundaryProjectScopeClaimAuthenticatorStub{},
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
		boundaryProjectScopeClaimAuthenticatorStub{},
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

func TestAuthenticationBoundaryUnaryRejectsOutOfRangeUserActor(t *testing.T) {
	organizationID := uint64(2)
	user := &types.PlainAuthPrinciple{
		User:             types.UserInfo{Id: uint64(math.MaxInt64) + 1},
		OrganizationRole: &types.OrganizaitonRole{OrganizationId: organizationID},
	}
	handlerCalled := false

	_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
		projectSelectingUserAuthenticatorStub{auth: user},
		boundaryProjectScopeClaimAuthenticatorStub{},
		organizationScopeClaimAuthenticatorStub{},
		serviceScopeClaimAuthenticatorStub{},
		nil,
	)(incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"), nil, nil, func(context.Context, any) (any, error) {
		handlerCalled = true
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("middleware error = %v, want unauthenticated", err)
	}
	if handlerCalled {
		t.Fatal("handler called for out-of-range user actor")
	}
}

func TestAuthenticationBoundaryUnaryRejectsInvalidProjectActor(t *testing.T) {
	zeroActorID := uint64(0)
	aboveMaxActorID := uint64(math.MaxInt64) + 1
	tests := []struct {
		name          string
		authenticator boundaryProjectScopeClaimAuthenticatorStub
	}{
		{name: "missing", authenticator: boundaryProjectScopeClaimAuthenticatorStub{omitActor: true}},
		{name: "zero", authenticator: boundaryProjectScopeClaimAuthenticatorStub{credentialID: &zeroActorID}},
		{name: "above max bigint", authenticator: boundaryProjectScopeClaimAuthenticatorStub{credentialID: &aboveMaxActorID}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerCalled := false
			_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
				userAuthenticatorStub{},
				test.authenticator,
				organizationScopeClaimAuthenticatorStub{},
				serviceScopeClaimAuthenticatorStub{},
				nil,
			)(incomingContext(types.PROJECT_SCOPE_KEY, "project"), nil, nil, func(context.Context, any) (any, error) {
				handlerCalled = true
				return nil, nil
			})

			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("middleware error = %v, want unauthenticated", err)
			}
			if handlerCalled {
				t.Fatal("handler called for invalid project actor")
			}
		})
	}
}

func TestAuthenticationBoundaryUnaryRejectsInvalidOrganizationActor(t *testing.T) {
	zeroActorID := uint64(0)
	aboveMaxActorID := uint64(math.MaxInt64) + 1
	tests := []struct {
		name          string
		authenticator organizationScopeClaimAuthenticatorStub
	}{
		{name: "missing", authenticator: organizationScopeClaimAuthenticatorStub{omitActor: true}},
		{name: "zero", authenticator: organizationScopeClaimAuthenticatorStub{credentialID: &zeroActorID}},
		{name: "above max bigint", authenticator: organizationScopeClaimAuthenticatorStub{credentialID: &aboveMaxActorID}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerCalled := false
			_, err := NewAuthenticationBoundaryUnaryServerMiddleware(
				userAuthenticatorStub{},
				boundaryProjectScopeClaimAuthenticatorStub{},
				test.authenticator,
				serviceScopeClaimAuthenticatorStub{},
				nil,
			)(incomingContext(types.ORG_SCOPE_KEY, "organization"), nil, nil, func(context.Context, any) (any, error) {
				handlerCalled = true
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
			}
			if handlerCalled {
				t.Fatal("handler called for invalid organization actor")
			}
		})
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
		boundaryProjectScopeClaimAuthenticatorStub{},
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
		boundaryProjectScopeClaimAuthenticatorStub{},
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
				boundaryProjectScopeClaimAuthenticatorStub{},
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
			user: userAuthenticatorStub{err: errors.New("rejected")}, project: boundaryProjectScopeClaimAuthenticatorStub{},
			organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "project", values: []string{types.PROJECT_SCOPE_KEY, "project"},
			user: userAuthenticatorStub{}, project: &projectScopeClaimAuthenticatorStub{err: errors.New("rejected")},
			organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "organization", values: []string{types.ORG_SCOPE_KEY, "organization"},
			user: userAuthenticatorStub{}, project: boundaryProjectScopeClaimAuthenticatorStub{},
			organization: organizationScopeClaimAuthenticatorStub{err: errors.New("rejected")}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "service", values: []string{types.SERVICE_SCOPE_KEY, "service"},
			user: userAuthenticatorStub{}, project: boundaryProjectScopeClaimAuthenticatorStub{},
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
			project: boundaryProjectScopeClaimAuthenticatorStub{}, organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "project", values: []string{types.PROJECT_SCOPE_KEY, "project"},
			user: userAuthenticatorStub{}, organization: organizationScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "organization", values: []string{types.ORG_SCOPE_KEY, "organization"},
			user: userAuthenticatorStub{}, project: boundaryProjectScopeClaimAuthenticatorStub{}, service: serviceScopeClaimAuthenticatorStub{},
		},
		{
			name: "service", values: []string{types.SERVICE_SCOPE_KEY, "service"},
			user: userAuthenticatorStub{}, project: boundaryProjectScopeClaimAuthenticatorStub{}, organization: organizationScopeClaimAuthenticatorStub{},
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
