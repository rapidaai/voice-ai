// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_metadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	principal_authentication "github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/authorizer"
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

var authenticationMiddlewareTestLogger = func() commons.Logger {
	logger, err := commons.NewApplicationLogger(
		commons.Name("authentication-middleware-test"),
		commons.EnableConsole(true),
		commons.EnableFile(false),
	)
	if err != nil {
		panic(err)
	}
	return logger
}()

type grpcChainServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (stream *grpcChainServerStream) Context() context.Context { return stream.ctx }

type grpcProjectSource struct {
	calls int
	err   error
	token string
}

func (source *grpcProjectSource) ResolveProjectCredential(_ context.Context, token string) (principal_authentication.Authentication, error) {
	source.calls++
	source.token = token
	if source.err != nil {
		return principal_authentication.Authentication{}, source.err
	}
	return principal_authentication.NewDirectAuthentication(
		principal_authentication.Identity{Type: principal_authentication.IdentityProjectCredential, ID: 31},
		principal_authentication.Tenant{OrganizationID: 7, ProjectID: 9},
		principal_authentication.Proof{Method: principal_authentication.AuthenticationProjectCredential, Audience: principal_authentication.AudienceManagement, Reference: principal_authentication.ProofReference{Type: principal_authentication.ProofReferenceCredential, Value: "31"}},
		nil,
		[]principal_authentication.Organization{{ID: 7, Name: "Organization", Projects: []principal_authentication.Project{{ID: 9, Name: "Project", Roles: []principal_authentication.Role{{ID: 21, Key: "project_reader", Name: "Project Reader", Scope: principal_authentication.RoleScopeProject, Subject: principal_authentication.RoleSubjectProjectToken, DefinitionType: principal_authentication.RoleDefinitionBuiltIn, Permissions: []principal_authentication.Permission{{ID: 22, Key: "project.assistant.read", Scope: principal_authentication.RoleScopeProject, Name: "Read assistants"}}}}}}}},
	)
}

type grpcOrganizationSource struct {
	calls int
	err   error
}

func (source *grpcOrganizationSource) ResolveOrganizationCredential(context.Context, string) (principal_authentication.Authentication, error) {
	source.calls++
	if source.err != nil {
		return principal_authentication.Authentication{}, source.err
	}
	return principal_authentication.NewDirectAuthentication(
		principal_authentication.Identity{Type: principal_authentication.IdentityOrganizationCredential, ID: 41},
		principal_authentication.Tenant{OrganizationID: 7},
		principal_authentication.Proof{Method: principal_authentication.AuthenticationOrganizationCredential, Reference: principal_authentication.ProofReference{Type: principal_authentication.ProofReferenceCredential, Value: "41"}},
		nil, []principal_authentication.Organization{{ID: 7, Name: "Organization"}},
	)
}

type grpcUserSource struct {
	calls          int
	organizationID uint64
	err            error
}

func (source *grpcUserSource) ResolveUserCredential(_ context.Context, _ string, userID uint64, organizationID uint64, projectID uint64) (principal_authentication.Authentication, error) {
	source.calls++
	source.organizationID = organizationID
	if source.err != nil {
		return principal_authentication.Authentication{}, source.err
	}
	if organizationID == 0 {
		organizationID = 7
	}
	organization := principal_authentication.Organization{ID: organizationID, Name: "Organization", Roles: []principal_authentication.Role{{ID: 31, Key: "organization_reader", Name: "Organization Reader", Scope: principal_authentication.RoleScopeOrganization, Subject: principal_authentication.RoleSubjectUser, DefinitionType: principal_authentication.RoleDefinitionBuiltIn, Permissions: []principal_authentication.Permission{{ID: 32, Key: "organization.user.read", Scope: principal_authentication.RoleScopeOrganization, Name: "Read users"}}}}}
	if projectID != 0 {
		organization.Projects = []principal_authentication.Project{{ID: projectID, Name: "Project", Roles: []principal_authentication.Role{{ID: 33, Key: "project_reader", Name: "Project Reader", Scope: principal_authentication.RoleScopeProject, Subject: principal_authentication.RoleSubjectUser, DefinitionType: principal_authentication.RoleDefinitionBuiltIn, Permissions: []principal_authentication.Permission{{ID: 34, Key: "project.assistant.read", Scope: principal_authentication.RoleScopeProject, Name: "Read assistants"}}}}}}
	}
	return principal_authentication.NewDirectAuthentication(
		principal_authentication.Identity{Type: principal_authentication.IdentityUser, ID: userID},
		principal_authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		principal_authentication.Proof{Method: principal_authentication.AuthenticationUserSession, Reference: principal_authentication.ProofReference{Type: principal_authentication.ProofReferenceUserSession, Value: "12"}},
		&principal_authentication.User{ID: userID, Name: "User", Email: "user@example.com", Status: "ACTIVE"},
		[]principal_authentication.Organization{organization},
	)
}

type grpcServicePolicy struct {
	policies map[string]authorizer.RoutePolicy
}

func (policy grpcServicePolicy) GRPCPolicy(route string) (authorizer.RoutePolicy, bool) {
	value, exists := policy.policies[route]
	return value, exists
}

func (grpcServicePolicy) HTTPPolicy(string) (authorizer.RoutePolicy, bool) {
	return authorizer.RoutePolicy{}, false
}

func grpcUserPrincipal(t *testing.T) principal_authentication.Principal {
	t.Helper()
	value, err := principal_authentication.NewDirectPrincipal(
		principal_authentication.Identity{Type: principal_authentication.IdentityUser, ID: 11},
		principal_authentication.Tenant{OrganizationID: 7},
		principal_authentication.Proof{
			Method: principal_authentication.AuthenticationUserSession,
			Reference: principal_authentication.ProofReference{
				Type: principal_authentication.ProofReferenceUserSession, Value: "12",
			},
		},
		[]principal_authentication.PermissionKey{"organization.user.read"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

type grpcAuthenticationFixture struct {
	name      string
	unary     grpc.UnaryServerInterceptor
	stream    grpc.StreamServerInterceptor
	valid     grpc_metadata.MD
	malformed grpc_metadata.MD
	duplicate grpc_metadata.MD
	rejected  grpc_metadata.MD
	calls     func() int
	reject    func(bool)
}

func grpcAuthenticationFixtures(t *testing.T) []grpcAuthenticationFixture {
	t.Helper()

	projectSource := &grpcProjectSource{}
	projectResolver := resolver.NewProjectPrincipalResolver(projectSource)
	organizationSource := &grpcOrganizationSource{}
	organizationResolver := resolver.NewOrganizationPrincipalResolver(organizationSource)
	userSource := &grpcUserSource{}
	userResolver := resolver.NewUserPrincipalResolver(userSource)
	key := bytes.Repeat([]byte{1}, sha256.Size)
	issuer, err := authentication_jwt.NewIssuer("web-api", key)
	if err != nil {
		t.Fatal(err)
	}
	assertion, err := issuer.Issue(
		principal_authentication.Identity{Type: principal_authentication.IdentityService, ID: 71, Issuer: "workflow-api"},
		principal_authentication.Tenant{OrganizationID: 7}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	serviceResolver := resolver.NewServicePrincipalResolver("web-api", key)

	return []grpcAuthenticationFixture{
		{
			name: "project", unary: NewProjectUnaryMiddleware(projectResolver, authenticationMiddlewareTestLogger), stream: NewProjectStreamMiddleware(projectResolver, authenticationMiddlewareTestLogger),
			valid:     grpc_metadata.Pairs(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX+"secret"),
			malformed: grpc_metadata.Pairs(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX),
			duplicate: grpc_metadata.MD{types.PROJECT_SCOPE_KEY: {"one", "two"}},
			rejected:  grpc_metadata.Pairs(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX+"rejected"),
			calls:     func() int { return projectSource.calls },
			reject: func(enabled bool) {
				projectSource.err = nil
				if enabled {
					projectSource.err = errors.New("project credential rejected")
				}
			},
		},
		{
			name: "organization", unary: NewOrganizationUnaryMiddleware(organizationResolver, authenticationMiddlewareTestLogger), stream: NewOrganizationStreamMiddleware(organizationResolver, authenticationMiddlewareTestLogger),
			valid:     grpc_metadata.Pairs(types.ORG_SCOPE_KEY, "organization-secret"),
			malformed: grpc_metadata.Pairs(types.ORG_SCOPE_KEY, " "),
			duplicate: grpc_metadata.MD{types.ORG_SCOPE_KEY: {"one", "two"}},
			rejected:  grpc_metadata.Pairs(types.ORG_SCOPE_KEY, "rejected"),
			calls:     func() int { return organizationSource.calls },
			reject: func(enabled bool) {
				organizationSource.err = nil
				if enabled {
					organizationSource.err = errors.New("organization credential rejected")
				}
			},
		},
		{
			name: "service", unary: NewServiceUnaryMiddleware(serviceResolver, authenticationMiddlewareTestLogger), stream: NewServiceStreamMiddleware(serviceResolver, authenticationMiddlewareTestLogger),
			valid:     grpc_metadata.Pairs(types.SERVICE_SCOPE_KEY, assertion),
			malformed: grpc_metadata.Pairs(types.SERVICE_SCOPE_KEY, " "),
			duplicate: grpc_metadata.MD{types.SERVICE_SCOPE_KEY: {assertion, assertion}},
			rejected:  grpc_metadata.Pairs(types.SERVICE_SCOPE_KEY, "legacy-token"),
		},
		{
			name: "user", unary: NewUserUnaryMiddleware(userResolver, authenticationMiddlewareTestLogger), stream: NewUserStreamMiddleware(userResolver, authenticationMiddlewareTestLogger),
			valid: grpc_metadata.Pairs(
				types.AUTHORIZATION_KEY, "session", types.AUTH_KEY, "11", types.ORGANIZATION_KEY, "7",
			),
			malformed: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, "session"),
			duplicate: grpc_metadata.MD{
				types.AUTHORIZATION_KEY: {"session", "duplicate"}, types.AUTH_KEY: {"11"},
			},
			rejected: grpc_metadata.Pairs(
				types.AUTHORIZATION_KEY, "rejected", types.AUTH_KEY, "11",
			),
			calls: func() int { return userSource.calls },
			reject: func(enabled bool) {
				userSource.err = nil
				if enabled {
					userSource.err = errors.New("user credential rejected")
				}
			},
		},
	}
}

func grpcUnaryChain(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		chained := handler
		for index := len(interceptors) - 1; index >= 0; index-- {
			interceptor := interceptors[index]
			next := chained
			chained = func(ctx context.Context, req any) (any, error) {
				return interceptor(ctx, req, info, next)
			}
		}
		return chained(ctx, req)
	}
}

func grpcStreamChain(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		chained := handler
		for index := len(interceptors) - 1; index >= 0; index-- {
			interceptor := interceptors[index]
			next := chained
			chained = func(srv any, stream grpc.ServerStream) error {
				return interceptor(srv, stream, info, next)
			}
		}
		return chained(srv, stream)
	}
}

func TestProjectGRPCAuthentication(t *testing.T) {
	source := &grpcProjectSource{}
	principalResolver := resolver.NewProjectPrincipalResolver(source)
	ctx := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.Pairs(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX+"secret"))
	called := false
	_, err := NewProjectUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger)(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Project/Unary"}, func(ctx context.Context, _ any) (any, error) {
		called = true
		value, exists := Principal(ctx)
		if !exists || value.Subject() != (principal_authentication.Identity{Type: principal_authentication.IdentityProjectCredential, ID: 31}) {
			t.Fatalf("principal = %+v, %v", value.Subject(), exists)
		}
		return nil, nil
	})
	if err != nil || !called || source.calls != 1 || source.token != "secret" {
		t.Fatalf("unary error = %v, called = %v, calls = %d, token = %q", err, called, source.calls, source.token)
	}

	called = false
	err = NewProjectStreamMiddleware(principalResolver, authenticationMiddlewareTestLogger)(nil, &grpcChainServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/test.Project/Stream"}, func(_ any, stream grpc.ServerStream) error {
		called = true
		if _, exists := Principal(stream.Context()); !exists {
			t.Fatal("stream principal missing")
		}
		return nil
	})
	if err != nil || !called || source.calls != 2 || source.token != "secret" {
		t.Fatalf("stream error = %v, called = %v, calls = %d, token = %q", err, called, source.calls, source.token)
	}
}

func TestGRPCAuthenticationAbsenceConflictDuplicatesAndCancellation(t *testing.T) {
	source := &grpcProjectSource{}
	principalResolver := resolver.NewProjectPrincipalResolver(source)
	called := false
	_, err := NewProjectUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger)(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Project/Absent"}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil || !called || source.calls != 0 {
		t.Fatalf("absent error = %v, called = %v, calls = %d", err, called, source.calls)
	}

	duplicate := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.MD{types.PROJECT_SCOPE_KEY: {"one", "two"}})
	if _, err := NewProjectUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger)(duplicate, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		t.Fatal("duplicate credential reached handler")
		return nil, nil
	}); status.Code(err) != codes.Unauthenticated || source.calls != 0 {
		t.Fatalf("duplicate code = %v, calls = %d", status.Code(err), source.calls)
	}

	conflict, err := ContextWithPrincipal(context.Background(), grpcUserPrincipal(t))
	if err != nil {
		t.Fatal(err)
	}
	conflict = grpc_metadata.NewIncomingContext(conflict, grpc_metadata.Pairs(types.PROJECT_SCOPE_KEY, "secret"))
	if _, err := NewProjectUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger)(conflict, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		t.Fatal("conflict reached handler")
		return nil, nil
	}); status.Code(err) != codes.Unauthenticated || source.calls != 0 {
		t.Fatalf("conflict code = %v, calls = %d", status.Code(err), source.calls)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	canceled = grpc_metadata.NewIncomingContext(canceled, grpc_metadata.Pairs(types.PROJECT_SCOPE_KEY, "secret"))
	if _, err := NewProjectUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger)(canceled, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		t.Fatal("canceled request reached handler")
		return nil, nil
	}); status.Code(err) != codes.Canceled || source.calls != 0 {
		t.Fatalf("canceled code = %v, calls = %d", status.Code(err), source.calls)
	}

	if _, err := NewProjectUnaryMiddleware(nil, authenticationMiddlewareTestLogger)(context.Background(), nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		t.Fatal("nil resolver reached handler")
		return nil, nil
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("nil resolver code = %v", status.Code(err))
	}
}

func TestOrganizationAndUserGRPCAuthentication(t *testing.T) {
	organizationResolver := resolver.NewOrganizationPrincipalResolver(&grpcOrganizationSource{})
	organizationContext := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.Pairs(types.ORG_SCOPE_KEY, "organization-secret"))
	_, err := NewOrganizationUnaryMiddleware(organizationResolver, authenticationMiddlewareTestLogger)(organizationContext, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, _ any) (any, error) {
		value, exists := Principal(ctx)
		if !exists || value.Subject().Type != principal_authentication.IdentityOrganizationCredential {
			t.Fatalf("organization principal = %+v, %v", value.Subject(), exists)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewOrganizationStreamMiddleware(organizationResolver, authenticationMiddlewareTestLogger)(nil, &grpcChainServerStream{ctx: organizationContext}, &grpc.StreamServerInfo{}, func(_ any, stream grpc.ServerStream) error {
		value, exists := Principal(stream.Context())
		if !exists || value.Subject().Type != principal_authentication.IdentityOrganizationCredential {
			t.Fatalf("organization stream principal = %+v, %v", value.Subject(), exists)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	userSource := &grpcUserSource{}
	userResolver := resolver.NewUserPrincipalResolver(userSource)
	userContext := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.Pairs(
		types.AUTHORIZATION_KEY, "session", types.AUTH_KEY, "11", types.ORGANIZATION_KEY, "7",
	))
	_, err = NewUserUnaryMiddleware(userResolver, authenticationMiddlewareTestLogger)(userContext, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, _ any) (any, error) {
		value, exists := Principal(ctx)
		if !exists || value.Subject().Type != principal_authentication.IdentityUser {
			t.Fatalf("user principal = %+v, %v", value.Subject(), exists)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewUserStreamMiddleware(userResolver, authenticationMiddlewareTestLogger)(nil, &grpcChainServerStream{ctx: userContext}, &grpc.StreamServerInfo{}, func(_ any, stream grpc.ServerStream) error {
		value, exists := Principal(stream.Context())
		if !exists || value.Subject().Type != principal_authentication.IdentityUser {
			t.Fatalf("user stream principal = %+v, %v", value.Subject(), exists)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if userSource.organizationID != 7 {
		t.Fatalf("selected organization = %d", userSource.organizationID)
	}
}

func TestUserGRPCRejectsAmbiguousOrganizationSelection(t *testing.T) {
	userSource := &grpcUserSource{}
	userResolver := resolver.NewUserPrincipalResolver(userSource)
	for name, values := range map[string][]string{
		"blank":        {" "},
		"malformed":    {"not-an-id"},
		"out of range": {"9223372036854775808"},
		"duplicate":    {"7", "8"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.MD{
				types.AUTHORIZATION_KEY: {"session"},
				types.AUTH_KEY:          {"11"},
				types.ORGANIZATION_KEY:  values,
			})
			if _, err := NewUserUnaryMiddleware(userResolver, authenticationMiddlewareTestLogger)(ctx, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
				t.Fatal("invalid organization selection reached handler")
				return nil, nil
			}); status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status = %v", status.Code(err))
			}
			if err := NewUserStreamMiddleware(userResolver, authenticationMiddlewareTestLogger)(nil, &grpcChainServerStream{ctx: ctx}, &grpc.StreamServerInfo{}, func(any, grpc.ServerStream) error {
				t.Fatal("invalid organization selection reached stream handler")
				return nil
			}); status.Code(err) != codes.Unauthenticated {
				t.Fatalf("stream status = %v", status.Code(err))
			}
		})
	}
	if userSource.calls != 0 {
		t.Fatalf("resolver calls = %d", userSource.calls)
	}
}

func TestUserGRPCRejectsOrganizationSelectionWithoutUserCredential(t *testing.T) {
	fixtures := grpcAuthenticationFixtures(t)
	user := fixtures[len(fixtures)-1]
	selection := grpc_metadata.Pairs(types.ORGANIZATION_KEY, "7")
	type credentialCase struct {
		name       string
		metadata   grpc_metadata.MD
		middleware grpc.UnaryServerInterceptor
	}
	cases := []credentialCase{
		{name: "selection only", metadata: selection, middleware: user.unary},
	}
	for _, family := range fixtures[:len(fixtures)-1] {
		cases = append(cases,
			credentialCase{name: family.name + " before user", metadata: grpc_metadata.Join(selection, family.valid), middleware: grpcUnaryChain(family.unary, user.unary)},
			credentialCase{name: "user before " + family.name, metadata: grpc_metadata.Join(selection, family.valid), middleware: grpcUnaryChain(user.unary, family.unary)},
		)
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ctx := grpc_metadata.NewIncomingContext(context.Background(), test.metadata)
			if _, err := test.middleware(ctx, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
				t.Fatal("organization selection without user credential reached handler")
				return nil, nil
			}); status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status = %v", status.Code(err))
			}
		})
	}
}

func TestUserGRPCAuthenticationLogsEveryNonContextDenial(t *testing.T) {
	principalResolver := resolver.NewUserPrincipalResolver(&grpcUserSource{})
	logDirectory := t.TempDir()
	logger, err := commons.NewApplicationLogger(
		commons.Name("user-grpc-denials"),
		commons.Path(logDirectory),
		commons.EnableConsole(false),
		commons.EnableFile(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Sync()

	validToken := "user-secret-value"
	cases := []struct {
		name     string
		stream   bool
		shadowed bool
		metadata grpc_metadata.MD
	}{
		{name: "unary blank project", metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, validToken, types.AUTH_KEY, "11", types.PROJECT_KEY, " ")},
		{name: "unary invalid project", metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, validToken, types.AUTH_KEY, "11", types.PROJECT_KEY, "invalid")},
		{name: "unary attachment conflict", shadowed: true, metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, validToken, types.AUTH_KEY, "11")},
		{name: "stream blank token", stream: true, metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, " ", types.AUTH_KEY, "11")},
		{name: "stream invalid user", stream: true, metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, validToken, types.AUTH_KEY, "invalid")},
		{name: "stream blank project", stream: true, metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, validToken, types.AUTH_KEY, "11", types.PROJECT_KEY, " ")},
		{name: "stream invalid project", stream: true, metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, validToken, types.AUTH_KEY, "11", types.PROJECT_KEY, "invalid")},
		{name: "stream attachment conflict", stream: true, shadowed: true, metadata: grpc_metadata.Pairs(types.AUTHORIZATION_KEY, validToken, types.AUTH_KEY, "11")},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.shadowed {
				ctx = contextWithoutPrincipal(ctx)
			}
			ctx = grpc_metadata.NewIncomingContext(ctx, test.metadata)
			if test.stream {
				err := NewUserStreamMiddleware(principalResolver, logger)(nil, &grpcChainServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/test.User/Denied"}, func(any, grpc.ServerStream) error {
					t.Fatal("denied stream reached handler")
					return nil
				})
				if status.Code(err) != codes.Unauthenticated {
					t.Fatalf("code = %v", status.Code(err))
				}
				return
			}
			_, err := NewUserUnaryMiddleware(principalResolver, logger)(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.User/Denied"}, func(context.Context, any) (any, error) {
				t.Fatal("denied unary request reached handler")
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("code = %v", status.Code(err))
			}
		})
	}

	if err := logger.Sync(); err != nil {
		t.Fatal(err)
	}
	logOutput, err := os.ReadFile(filepath.Join(logDirectory, "user-grpc-denials.log"))
	if err != nil {
		t.Fatal(err)
	}
	const denial = "user authentication rejected for route /test.User/Denied"
	if count := strings.Count(string(logOutput), denial); count != len(cases) {
		t.Fatalf("denial log count = %d, want %d", count, len(cases))
	}
	if strings.Contains(string(logOutput), validToken) {
		t.Fatal("denial log contains credential value")
	}
}

func TestServiceGRPCAuthenticationDirectAndDelegated(t *testing.T) {
	key := bytes.Repeat([]byte{1}, sha256.Size)
	issuer, err := authentication_jwt.NewIssuer("web-api", key)
	if err != nil {
		t.Fatal(err)
	}
	principalResolver := resolver.NewServicePrincipalResolver("web-api", key)
	var directAssertion string
	for _, subject := range []principal_authentication.Identity{
		{Type: principal_authentication.IdentityService, ID: 71, Issuer: "workflow-api"},
		{Type: principal_authentication.IdentityUser, ID: 11},
	} {
		assertion, err := issuer.Issue(subject, principal_authentication.Tenant{OrganizationID: 7}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if subject.Type == principal_authentication.IdentityService {
			directAssertion = assertion
		}
		ctx := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.Pairs(types.SERVICE_SCOPE_KEY, assertion))
		_, err = NewServiceUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger)(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, _ any) (any, error) {
			value, exists := Principal(ctx)
			wantSubject := subject
			if subject.Type == principal_authentication.IdentityService {
				wantSubject = principal_authentication.Identity{Type: principal_authentication.IdentityService, ID: 1, Issuer: "internal"}
			}
			if !exists || value.Subject() != wantSubject || value.Proof().Reference.Value == "" {
				t.Fatalf("service principal = %+v, %v", value.Subject(), exists)
			}
			return nil, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	streamContext := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.Pairs(types.SERVICE_SCOPE_KEY, directAssertion))
	if err := NewServiceStreamMiddleware(principalResolver, authenticationMiddlewareTestLogger)(nil, &grpcChainServerStream{ctx: streamContext}, &grpc.StreamServerInfo{}, func(_ any, stream grpc.ServerStream) error {
		value, exists := Principal(stream.Context())
		if !exists || value.Subject().Type != principal_authentication.IdentityService {
			t.Fatalf("service stream principal = %+v, %v", value.Subject(), exists)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	legacy := grpc_metadata.NewIncomingContext(context.Background(), grpc_metadata.Pairs(types.SERVICE_SCOPE_KEY, "legacy-token"))
	if _, err := NewServiceUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger)(legacy, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		t.Fatal("legacy token reached handler")
		return nil, nil
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("legacy token code = %v", status.Code(err))
	}
}

func TestPolicyGRPCAuthentication(t *testing.T) {
	const (
		publicRoute       = "/test.Policy/Public"
		protectedRoute    = "/test.Policy/Protected"
		disabledRoute     = "/test.Policy/Disabled"
		unauthorizedRoute = "/test.Policy/Unauthorized"
	)
	policy := grpcServicePolicy{policies: map[string]authorizer.RoutePolicy{
		publicRoute: {Authorization: authorizer.AuthorizationPublic},
		protectedRoute: {
			Authorization: authorizer.AuthorizationPermission,
			Actors:        []principal_authentication.IdentityType{principal_authentication.IdentityUser},
			Context:       authorizer.ContextOrganization,
			Permissions:   []principal_authentication.PermissionKey{"organization.user.read"},
		},
		disabledRoute: {Authorization: authorizer.AuthorizationDisabled},
		unauthorizedRoute: {
			Authorization: authorizer.AuthorizationPermission,
			Actors:        []principal_authentication.IdentityType{principal_authentication.IdentityUser},
			Context:       authorizer.ContextOrganization,
			Permissions:   []principal_authentication.PermissionKey{"organization.billing.read"},
		},
	}}

	withPrincipal, err := ContextWithPrincipal(context.Background(), grpcUserPrincipal(t))
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewPolicyUnaryMiddleware(policy, authenticationMiddlewareTestLogger)(withPrincipal, nil, &grpc.UnaryServerInfo{FullMethod: publicRoute}, func(ctx context.Context, _ any) (any, error) {
		if _, exists := Principal(ctx); exists {
			t.Fatal("public principal was visible")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewPolicyUnaryMiddleware(policy, authenticationMiddlewareTestLogger)(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: protectedRoute}, func(context.Context, any) (any, error) {
		t.Fatal("missing principal reached handler")
		return nil, nil
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing principal code = %v", status.Code(err))
	}
	if _, err := NewPolicyUnaryMiddleware(policy, authenticationMiddlewareTestLogger)(withPrincipal, nil, &grpc.UnaryServerInfo{FullMethod: protectedRoute}, func(context.Context, any) (any, error) {
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPolicyUnaryMiddleware(policy, authenticationMiddlewareTestLogger)(withPrincipal, nil, &grpc.UnaryServerInfo{FullMethod: disabledRoute}, func(context.Context, any) (any, error) {
		t.Fatal("disabled route reached handler")
		return nil, nil
	}); status.Code(err) != codes.Unavailable {
		t.Fatalf("disabled code = %v", status.Code(err))
	}
	if _, err := NewPolicyUnaryMiddleware(policy, authenticationMiddlewareTestLogger)(withPrincipal, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Policy/Unknown"}, func(context.Context, any) (any, error) {
		t.Fatal("unknown route reached handler")
		return nil, nil
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("unknown code = %v", status.Code(err))
	}
	if _, err := NewPolicyUnaryMiddleware(nil, authenticationMiddlewareTestLogger)(withPrincipal, nil, &grpc.UnaryServerInfo{FullMethod: protectedRoute}, func(context.Context, any) (any, error) {
		t.Fatal("nil policy reached handler")
		return nil, nil
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("nil policy code = %v", status.Code(err))
	}

	stream := &grpcChainServerStream{ctx: withPrincipal}
	if err := NewPolicyStreamMiddleware(policy, authenticationMiddlewareTestLogger)(nil, stream, &grpc.StreamServerInfo{FullMethod: publicRoute}, func(_ any, stream grpc.ServerStream) error {
		if _, exists := Principal(stream.Context()); exists {
			t.Fatal("public stream principal was visible")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name       string
		policy     ServicePolicy
		ctx        context.Context
		route      string
		code       codes.Code
		handlerRan bool
	}{
		{name: "protected", policy: policy, ctx: withPrincipal, route: protectedRoute, code: codes.OK, handlerRan: true},
		{name: "missing principal", policy: policy, ctx: context.Background(), route: protectedRoute, code: codes.Unauthenticated},
		{name: "disabled", policy: policy, ctx: withPrincipal, route: disabledRoute, code: codes.Unavailable},
		{name: "unknown", policy: policy, ctx: withPrincipal, route: "/test.Policy/Unknown", code: codes.PermissionDenied},
		{name: "unauthorized", policy: policy, ctx: withPrincipal, route: unauthorizedRoute, code: codes.PermissionDenied},
		{name: "nil policy", ctx: withPrincipal, route: protectedRoute, code: codes.PermissionDenied},
	} {
		t.Run("stream "+test.name, func(t *testing.T) {
			handlerRan := false
			err := NewPolicyStreamMiddleware(test.policy, authenticationMiddlewareTestLogger)(nil, &grpcChainServerStream{ctx: test.ctx}, &grpc.StreamServerInfo{FullMethod: test.route}, func(any, grpc.ServerStream) error {
				handlerRan = true
				return nil
			})
			if status.Code(err) != test.code || handlerRan != test.handlerRan {
				t.Fatalf("code = %v, handler ran = %v", status.Code(err), handlerRan)
			}
		})
	}
}

func TestGRPCAuthenticationFamiliesRejectInvalidRequests(t *testing.T) {
	for _, fixture := range grpcAuthenticationFixtures(t) {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Run("unary", func(t *testing.T) {
				called := false
				before := 0
				if fixture.calls != nil {
					before = fixture.calls()
				}
				_, err := fixture.unary(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Authentication/Absent"}, func(context.Context, any) (any, error) {
					called = true
					return nil, nil
				})
				if err != nil || !called || fixture.calls != nil && fixture.calls() != before {
					t.Fatalf("absent error = %v, called = %v", err, called)
				}

				for _, request := range []struct {
					name     string
					metadata grpc_metadata.MD
					reject   bool
					code     codes.Code
					resolved bool
				}{
					{name: "malformed", metadata: fixture.malformed, code: codes.Unauthenticated},
					{name: "duplicate", metadata: fixture.duplicate, code: codes.Unauthenticated},
					{name: "rejected", metadata: fixture.rejected, reject: true, code: codes.Unauthenticated, resolved: fixture.calls != nil},
				} {
					t.Run(request.name, func(t *testing.T) {
						if fixture.reject != nil {
							fixture.reject(request.reject)
							defer fixture.reject(false)
						}
						before := 0
						if fixture.calls != nil {
							before = fixture.calls()
						}
						ctx := grpc_metadata.NewIncomingContext(context.Background(), request.metadata)
						_, err := fixture.unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Authentication/Rejected"}, func(context.Context, any) (any, error) {
							t.Fatal("invalid request reached unary handler")
							return nil, nil
						})
						if status.Code(err) != request.code {
							t.Fatalf("code = %v, want %v", status.Code(err), request.code)
						}
						if fixture.calls != nil {
							want := before
							if request.resolved {
								want++
							}
							if fixture.calls() != want {
								t.Fatalf("resolver calls = %d, want %d", fixture.calls(), want)
							}
						}
					})
				}

				for _, request := range []struct {
					name string
					ctx  func() (context.Context, context.CancelFunc)
					code codes.Code
				}{
					{
						name: "canceled", code: codes.Canceled,
						ctx: func() (context.Context, context.CancelFunc) {
							ctx, cancel := context.WithCancel(context.Background())
							cancel()
							return ctx, func() {}
						},
					},
					{
						name: "deadline", code: codes.DeadlineExceeded,
						ctx: func() (context.Context, context.CancelFunc) {
							return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
						},
					},
				} {
					t.Run(request.name, func(t *testing.T) {
						ctx, cancel := request.ctx()
						defer cancel()
						ctx = grpc_metadata.NewIncomingContext(ctx, fixture.valid)
						before := 0
						if fixture.calls != nil {
							before = fixture.calls()
						}
						_, err := fixture.unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Authentication/Canceled"}, func(context.Context, any) (any, error) {
							t.Fatal("canceled request reached unary handler")
							return nil, nil
						})
						if status.Code(err) != request.code {
							t.Fatalf("code = %v, want %v", status.Code(err), request.code)
						}
						if fixture.calls != nil && fixture.calls() != before {
							t.Fatalf("resolver calls = %d, want %d", fixture.calls(), before)
						}
					})
				}
			})

			t.Run("stream", func(t *testing.T) {
				called := false
				before := 0
				if fixture.calls != nil {
					before = fixture.calls()
				}
				err := fixture.stream(nil, &grpcChainServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/test.Authentication/Absent"}, func(any, grpc.ServerStream) error {
					called = true
					return nil
				})
				if err != nil || !called || fixture.calls != nil && fixture.calls() != before {
					t.Fatalf("absent error = %v, called = %v", err, called)
				}

				for _, request := range []struct {
					name     string
					metadata grpc_metadata.MD
					reject   bool
					resolved bool
				}{
					{name: "malformed", metadata: fixture.malformed},
					{name: "duplicate", metadata: fixture.duplicate},
					{name: "rejected", metadata: fixture.rejected, reject: true, resolved: fixture.calls != nil},
				} {
					t.Run(request.name, func(t *testing.T) {
						if fixture.reject != nil {
							fixture.reject(request.reject)
							defer fixture.reject(false)
						}
						before := 0
						if fixture.calls != nil {
							before = fixture.calls()
						}
						ctx := grpc_metadata.NewIncomingContext(context.Background(), request.metadata)
						err := fixture.stream(nil, &grpcChainServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/test.Authentication/Rejected"}, func(any, grpc.ServerStream) error {
							t.Fatal("invalid request reached stream handler")
							return nil
						})
						if status.Code(err) != codes.Unauthenticated {
							t.Fatalf("code = %v, want %v", status.Code(err), codes.Unauthenticated)
						}
						if fixture.calls != nil {
							want := before
							if request.resolved {
								want++
							}
							if fixture.calls() != want {
								t.Fatalf("resolver calls = %d, want %d", fixture.calls(), want)
							}
						}
					})
				}

				for _, request := range []struct {
					name string
					ctx  func() (context.Context, context.CancelFunc)
					code codes.Code
				}{
					{
						name: "canceled", code: codes.Canceled,
						ctx: func() (context.Context, context.CancelFunc) {
							ctx, cancel := context.WithCancel(context.Background())
							cancel()
							return ctx, func() {}
						},
					},
					{
						name: "deadline", code: codes.DeadlineExceeded,
						ctx: func() (context.Context, context.CancelFunc) {
							return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
						},
					},
				} {
					t.Run(request.name, func(t *testing.T) {
						ctx, cancel := request.ctx()
						defer cancel()
						ctx = grpc_metadata.NewIncomingContext(ctx, fixture.valid)
						before := 0
						if fixture.calls != nil {
							before = fixture.calls()
						}
						err := fixture.stream(nil, &grpcChainServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/test.Authentication/Canceled"}, func(any, grpc.ServerStream) error {
							t.Fatal("canceled request reached stream handler")
							return nil
						})
						if status.Code(err) != request.code {
							t.Fatalf("code = %v, want %v", status.Code(err), request.code)
						}
						if fixture.calls != nil && fixture.calls() != before {
							t.Fatalf("resolver calls = %d, want %d", fixture.calls(), before)
						}
					})
				}
			})
		})
	}
}

func TestGRPCAuthenticationChainRejectsCredentialFamilyConflicts(t *testing.T) {
	fixtures := grpcAuthenticationFixtures(t)
	for firstIndex, first := range fixtures {
		for secondIndex, second := range fixtures {
			if firstIndex == secondIndex {
				continue
			}
			name := first.name + "_then_" + second.name
			metadata := grpc_metadata.Join(first.valid, second.valid)
			t.Run(name+"_unary", func(t *testing.T) {
				firstBefore, secondBefore := 0, 0
				if first.calls != nil {
					firstBefore = first.calls()
				}
				if second.calls != nil {
					secondBefore = second.calls()
				}
				called := false
				ctx := grpc_metadata.NewIncomingContext(context.Background(), metadata)
				_, err := grpcUnaryChain(first.unary, second.unary)(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Authentication/Conflict"}, func(context.Context, any) (any, error) {
					called = true
					return nil, nil
				})
				if status.Code(err) != codes.Unauthenticated || called {
					t.Fatalf("code = %v, called = %v", status.Code(err), called)
				}
				if first.calls != nil && first.calls() != firstBefore+1 {
					t.Fatalf("first resolver calls = %d, want %d", first.calls(), firstBefore+1)
				}
				if second.calls != nil && second.calls() != secondBefore {
					t.Fatalf("second resolver calls = %d, want %d", second.calls(), secondBefore)
				}
			})

			t.Run(name+"_stream", func(t *testing.T) {
				firstBefore, secondBefore := 0, 0
				if first.calls != nil {
					firstBefore = first.calls()
				}
				if second.calls != nil {
					secondBefore = second.calls()
				}
				called := false
				ctx := grpc_metadata.NewIncomingContext(context.Background(), metadata)
				err := grpcStreamChain(first.stream, second.stream)(nil, &grpcChainServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/test.Authentication/Conflict"}, func(any, grpc.ServerStream) error {
					called = true
					return nil
				})
				if status.Code(err) != codes.Unauthenticated || called {
					t.Fatalf("code = %v, called = %v", status.Code(err), called)
				}
				if first.calls != nil && first.calls() != firstBefore+1 {
					t.Fatalf("first resolver calls = %d, want %d", first.calls(), firstBefore+1)
				}
				if second.calls != nil && second.calls() != secondBefore {
					t.Fatalf("second resolver calls = %d, want %d", second.calls(), secondBefore)
				}
			})
		}
	}
}

func TestGRPCAuthenticationRunsBeforePolicy(t *testing.T) {
	const (
		publicRoute   = "/test.Precedence/Public"
		disabledRoute = "/test.Precedence/Disabled"
		unknownRoute  = "/test.Precedence/Unknown"
	)
	policy := grpcServicePolicy{policies: map[string]authorizer.RoutePolicy{
		publicRoute:   {Authorization: authorizer.AuthorizationPublic},
		disabledRoute: {Authorization: authorizer.AuthorizationDisabled},
	}}
	source := &grpcProjectSource{}
	principalResolver := resolver.NewProjectPrincipalResolver(source)
	malformed := grpc_metadata.Pairs(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX)
	for _, route := range []string{publicRoute, disabledRoute, unknownRoute} {
		t.Run(route+"_unary", func(t *testing.T) {
			called := false
			ctx := grpc_metadata.NewIncomingContext(context.Background(), malformed)
			_, err := grpcUnaryChain(NewProjectUnaryMiddleware(principalResolver, authenticationMiddlewareTestLogger), NewPolicyUnaryMiddleware(policy, authenticationMiddlewareTestLogger))(
				ctx, nil, &grpc.UnaryServerInfo{FullMethod: route}, func(context.Context, any) (any, error) {
					called = true
					return nil, nil
				},
			)
			if status.Code(err) != codes.Unauthenticated || called || source.calls != 0 {
				t.Fatalf("code = %v, called = %v, resolver calls = %d", status.Code(err), called, source.calls)
			}
		})
		t.Run(route+"_stream", func(t *testing.T) {
			called := false
			ctx := grpc_metadata.NewIncomingContext(context.Background(), malformed)
			err := grpcStreamChain(NewProjectStreamMiddleware(principalResolver, authenticationMiddlewareTestLogger), NewPolicyStreamMiddleware(policy, authenticationMiddlewareTestLogger))(
				nil, &grpcChainServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: route}, func(any, grpc.ServerStream) error {
					called = true
					return nil
				},
			)
			if status.Code(err) != codes.Unauthenticated || called || source.calls != 0 {
				t.Fatalf("code = %v, called = %v, resolver calls = %d", status.Code(err), called, source.calls)
			}
		})
	}
}

func TestGRPCTypedNilDependenciesFailClosed(t *testing.T) {
	var projectResolver *resolver.ProjectPrincipalResolver
	var organizationResolver *resolver.OrganizationPrincipalResolver
	var serviceResolver *resolver.ServicePrincipalResolver
	var userResolver *resolver.UserPrincipalResolver
	interceptors := []struct {
		name   string
		unary  grpc.UnaryServerInterceptor
		stream grpc.StreamServerInterceptor
	}{
		{name: "project", unary: NewProjectUnaryMiddleware(projectResolver, authenticationMiddlewareTestLogger), stream: NewProjectStreamMiddleware(projectResolver, authenticationMiddlewareTestLogger)},
		{name: "organization", unary: NewOrganizationUnaryMiddleware(organizationResolver, authenticationMiddlewareTestLogger), stream: NewOrganizationStreamMiddleware(organizationResolver, authenticationMiddlewareTestLogger)},
		{name: "service", unary: NewServiceUnaryMiddleware(serviceResolver, authenticationMiddlewareTestLogger), stream: NewServiceStreamMiddleware(serviceResolver, authenticationMiddlewareTestLogger)},
		{name: "user", unary: NewUserUnaryMiddleware(userResolver, authenticationMiddlewareTestLogger), stream: NewUserStreamMiddleware(userResolver, authenticationMiddlewareTestLogger)},
	}
	for _, interceptor := range interceptors {
		t.Run(interceptor.name+"_unary", func(t *testing.T) {
			_, err := interceptor.unary(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.TypedNil/Unary"}, func(context.Context, any) (any, error) {
				t.Fatal("typed-nil resolver reached unary handler")
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("code = %v", status.Code(err))
			}
		})
		t.Run(interceptor.name+"_stream", func(t *testing.T) {
			err := interceptor.stream(nil, &grpcChainServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/test.TypedNil/Stream"}, func(any, grpc.ServerStream) error {
				t.Fatal("typed-nil resolver reached stream handler")
				return nil
			})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("code = %v", status.Code(err))
			}
		})
	}

	var typedNilPolicy *grpcServicePolicy
	if _, err := NewPolicyUnaryMiddleware(typedNilPolicy, authenticationMiddlewareTestLogger)(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.TypedNil/Policy"}, func(context.Context, any) (any, error) {
		t.Fatal("typed-nil policy reached unary handler")
		return nil, nil
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("unary policy code = %v", status.Code(err))
	}
	if err := NewPolicyStreamMiddleware(typedNilPolicy, authenticationMiddlewareTestLogger)(nil, &grpcChainServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/test.TypedNil/Policy"}, func(any, grpc.ServerStream) error {
		t.Fatal("typed-nil policy reached stream handler")
		return nil
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("stream policy code = %v", status.Code(err))
	}
}
