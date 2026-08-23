package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
)

type testServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (stream *testServerStream) Context() context.Context {
	return stream.ctx
}

func incomingContext(values ...string) context.Context {
	return grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(values...))
}

type userAuthenticatorStub struct {
	err error
}

func (stub userAuthenticatorStub) Authorize(context.Context, string, uint64) (types.Principle, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return testUserPrinciple(), nil
}

func (stub userAuthenticatorStub) AuthPrinciple(context.Context, uint64) (types.Principle, error) {
	return testUserPrinciple(), stub.err
}

func testUserPrinciple() *types.PlainAuthPrinciple {
	return &types.PlainAuthPrinciple{
		User:             types.UserInfo{Id: 1},
		OrganizationRole: &types.OrganizaitonRole{OrganizationId: 2},
	}
}

func TestUserUnaryRejectsPartialCredential(t *testing.T) {
	handlerCalled := false
	_, err := NewAuthenticationUnaryServerMiddleware(userAuthenticatorStub{}, nil)(
		incomingContext(types.AUTHORIZATION_KEY, "token"),
		nil,
		nil,
		func(context.Context, any) (any, error) {
			handlerCalled = true
			return nil, nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
	if handlerCalled {
		t.Fatal("handler called for partial user credential")
	}
}

func TestUserUnaryAcceptsCredentialWithoutProject(t *testing.T) {
	handlerCalled := false
	_, err := NewAuthenticationUnaryServerMiddleware(userAuthenticatorStub{}, nil)(
		incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"),
		nil,
		nil,
		func(ctx context.Context, _ any) (any, error) {
			handlerCalled = true
			if _, err := types.Authorize(ctx); err != nil {
				t.Fatalf("authenticated request missing from unary context: %v", err)
			}
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("middleware error = %v", err)
	}
	if !handlerCalled {
		t.Fatal("handler not called")
	}
}

func TestUserStreamAcceptsCredentialWithoutProject(t *testing.T) {
	handlerCalled := false
	err := NewAuthenticationStreamServerMiddleware(userAuthenticatorStub{}, nil)(
		nil,
		&testServerStream{ctx: incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1")},
		nil,
		func(_ any, stream grpc.ServerStream) error {
			handlerCalled = true
			if _, err := types.Authorize(stream.Context()); err != nil {
				t.Fatalf("authenticated request missing from stream context: %v", err)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("middleware error = %v", err)
	}
	if !handlerCalled {
		t.Fatal("handler not called")
	}
}

func TestUserStreamRejectsMalformedAndRejectedCredentials(t *testing.T) {
	tests := []struct {
		name     string
		resolver userAuthenticatorStub
		values   []string
	}{
		{
			name:   "partial credential",
			values: []string{types.AUTHORIZATION_KEY, "token"},
		},
		{
			name:     "resolver rejection",
			resolver: userAuthenticatorStub{err: errors.New("rejected")},
			values:   []string{types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerCalled := false
			err := NewAuthenticationStreamServerMiddleware(test.resolver, nil)(
				nil,
				&testServerStream{ctx: incomingContext(test.values...)},
				nil,
				func(any, grpc.ServerStream) error {
					handlerCalled = true
					return nil
				},
			)
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
			}
			if handlerCalled {
				t.Fatal("handler called for invalid user credential")
			}
		})
	}
}

func TestProjectUnaryRejectsResolverFailure(t *testing.T) {
	resolver := &projectScopeClaimAuthenticatorStub{err: errors.New("rejected")}
	handlerCalled := false
	_, err := NewProjectAuthenticatorUnaryServerMiddleware(resolver, nil)(
		incomingContext(types.PROJECT_SCOPE_KEY, "key"),
		nil,
		nil,
		func(context.Context, any) (any, error) {
			handlerCalled = true
			return nil, nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
	if handlerCalled {
		t.Fatal("handler called after project credential rejection")
	}
}

func TestProjectUnaryRemovesOnlyLeadingPrefix(t *testing.T) {
	resolver := &projectScopeClaimAuthenticatorStub{}
	_, err := NewProjectAuthenticatorUnaryServerMiddleware(resolver, nil)(
		incomingContext(types.PROJECT_SCOPE_KEY, "prefix-"+types.PROJECT_KEY_PREFIX+"inside"),
		nil,
		nil,
		func(context.Context, any) (any, error) { return nil, nil },
	)
	if err != nil {
		t.Fatalf("middleware error = %v", err)
	}
	if resolver.lastToken != "prefix-"+types.PROJECT_KEY_PREFIX+"inside" {
		t.Fatalf("resolver token = %q", resolver.lastToken)
	}
}

func TestProjectUnaryRemovesLeadingPrefix(t *testing.T) {
	resolver := &projectScopeClaimAuthenticatorStub{}
	_, err := NewProjectAuthenticatorUnaryServerMiddleware(resolver, nil)(
		incomingContext(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX+"secret"),
		nil,
		nil,
		func(context.Context, any) (any, error) { return nil, nil },
	)
	if err != nil {
		t.Fatalf("middleware error = %v", err)
	}
	if resolver.lastToken != "secret" {
		t.Fatalf("resolver token = %q, want %q", resolver.lastToken, "secret")
	}
}

func TestProjectStreamRejectsResolverFailure(t *testing.T) {
	handlerCalled := false
	err := NewProjectAuthenticatorStreamServerMiddleware(
		&projectScopeClaimAuthenticatorStub{err: errors.New("rejected")},
		nil,
	)(
		nil,
		&testServerStream{ctx: incomingContext(types.PROJECT_SCOPE_KEY, "key")},
		nil,
		func(any, grpc.ServerStream) error {
			handlerCalled = true
			return nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
	if handlerCalled {
		t.Fatal("handler called after project credential rejection")
	}
}

type serviceScopeClaimAuthenticatorStub struct {
	err     error
	actorID uint64
}

func (stub serviceScopeClaimAuthenticatorStub) Claim(
	context.Context,
	string,
) (*types.PlainClaimPrinciple[*types.ServiceScope], error) {
	if stub.err != nil {
		return nil, stub.err
	}
	actorID := stub.actorID
	if actorID == 0 {
		actorID = 4
	}
	organizationID := uint64(2)
	return &types.PlainClaimPrinciple[*types.ServiceScope]{
		Info: &types.ServiceScope{
			ActorId:        actorID,
			Issuer:         "assistant-api",
			Audience:       types.ServiceAssertionAudience,
			OrganizationId: &organizationID,
		},
	}, nil
}

func TestServiceUnaryRejectsResolverFailure(t *testing.T) {
	handlerCalled := false
	_, err := NewServiceAuthenticatorUnaryServerMiddleware(
		serviceScopeClaimAuthenticatorStub{err: errors.New("rejected")},
		nil,
	)(
		incomingContext(types.SERVICE_SCOPE_KEY, "service"),
		nil,
		nil,
		func(context.Context, any) (any, error) {
			handlerCalled = true
			return nil, nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
	if handlerCalled {
		t.Fatal("handler called after service credential rejection")
	}
}

func TestServiceStreamRejectsResolverFailure(t *testing.T) {
	handlerCalled := false
	err := NewServiceAuthenticatorStreamServerMiddleware(
		serviceScopeClaimAuthenticatorStub{err: errors.New("rejected")},
		nil,
	)(
		nil,
		&testServerStream{ctx: incomingContext(types.SERVICE_SCOPE_KEY, "service")},
		nil,
		func(any, grpc.ServerStream) error {
			handlerCalled = true
			return nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
	if handlerCalled {
		t.Fatal("handler called after service credential rejection")
	}
}

func TestUserGinRejectsPartialCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(NewAuthenticationMiddleware(userAuthenticatorStub{}, nil))
	handlerCalled := false
	engine.GET("/test", func(ctx *gin.Context) {
		handlerCalled = true
		ctx.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(types.AUTHORIZATION_KEY, "token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Fatal("handler called for partial user credential")
	}
}

func TestProjectGinRejectsResolverFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(NewProjectAuthenticatorMiddleware(
		&projectScopeClaimAuthenticatorStub{err: errors.New("rejected")},
		nil,
	))
	handlerCalled := false
	engine.GET("/test", func(ctx *gin.Context) {
		handlerCalled = true
		ctx.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(types.PROJECT_SCOPE_KEY, "key")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Fatal("handler called after project credential rejection")
	}
}
