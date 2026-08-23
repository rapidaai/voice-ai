package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type organizationAuthenticatorStub struct {
	token        string
	err          error
	credentialID uint64
}

func (authenticator *organizationAuthenticatorStub) Claim(_ context.Context, token string) (*types.PlainClaimPrinciple[*types.OrganizationScope], error) {
	authenticator.token = token
	if authenticator.err != nil {
		return nil, authenticator.err
	}
	credentialID := authenticator.credentialID
	if credentialID == 0 {
		credentialID = 1
	}
	organizationID := uint64(2)
	return &types.PlainClaimPrinciple[*types.OrganizationScope]{Info: &types.OrganizationScope{
		CredentialId:   &credentialID,
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}}, nil
}

type recordingUserAuthenticator struct {
	token      string
	userID     uint64
	projectIDs []uint64
}

func (authenticator *recordingUserAuthenticator) Authorize(_ context.Context, token string, userID uint64) (types.Principle, error) {
	authenticator.token = token
	authenticator.userID = userID
	projectIDs := authenticator.projectIDs
	if len(projectIDs) == 0 {
		projectIDs = []uint64{10, 20, 30}
	}
	projectRoles := make([]*types.ProjectRole, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		projectRoles = append(projectRoles, &types.ProjectRole{ProjectId: projectID})
	}
	return &types.PlainAuthPrinciple{
		User:             types.UserInfo{Id: userID},
		OrganizationRole: &types.OrganizaitonRole{OrganizationId: 2},
		ProjectRoles:     projectRoles,
	}, nil
}

func (*recordingUserAuthenticator) AuthPrinciple(context.Context, uint64) (types.Principle, error) {
	return nil, errors.New("Not used")
}

type recordingServiceAuthenticator struct{ token string }

func (authenticator *recordingServiceAuthenticator) Claim(_ context.Context, token string) (*types.PlainClaimPrinciple[*types.ServiceScope], error) {
	authenticator.token = token
	organizationID := uint64(2)
	return &types.PlainClaimPrinciple[*types.ServiceScope]{Info: &types.ServiceScope{
		ActorId:        3,
		Issuer:         "assistant-api",
		Audience:       types.ServiceAssertionAudience,
		OrganizationId: &organizationID,
	}}, nil
}

type delegatedServiceAuthenticatorStub struct {
	authType types.AuthType
	actorID  uint64
}

func (stub delegatedServiceAuthenticatorStub) Claim(context.Context, string) (*types.PlainClaimPrinciple[*types.ServiceScope], error) {
	organizationID := uint64(2)
	projectID := uint64(3)
	scope := &types.ServiceScope{
		ActorId:           4,
		Issuer:            "web-api",
		Audience:          types.ServiceAssertionAudience,
		DelegatedAuthType: stub.authType,
		DelegatedActorId:  &stub.actorID,
		OrganizationId:    &organizationID,
	}
	if stub.authType == types.AuthTypeUser || stub.authType == types.AuthTypeProject {
		scope.ProjectId = &projectID
	}
	return &types.PlainClaimPrinciple[*types.ServiceScope]{Info: scope}, nil
}

type invalidActorUserPrinciple struct {
	types.Principle
}

func (invalidActorUserPrinciple) AuditActor() (types.ActorIdentity, bool) {
	return types.ActorIdentity{}, false
}

type invalidActorUserAuthenticator struct{}

func (invalidActorUserAuthenticator) Authorize(context.Context, string, uint64) (types.Principle, error) {
	return invalidActorUserPrinciple{Principle: testUserPrinciple()}, nil
}

func (invalidActorUserAuthenticator) AuthPrinciple(context.Context, uint64) (types.Principle, error) {
	return invalidActorUserPrinciple{Principle: testUserPrinciple()}, nil
}

type typedNilUserAuthenticator struct {
	types.Authenticator
}

type nilPrincipleUserAuthenticator struct{}

func (nilPrincipleUserAuthenticator) Authorize(context.Context, string, uint64) (types.Principle, error) {
	var principle *types.PlainAuthPrinciple
	return principle, nil
}

func (nilPrincipleUserAuthenticator) AuthPrinciple(context.Context, uint64) (types.Principle, error) {
	var principle *types.PlainAuthPrinciple
	return principle, nil
}

type nilProjectInfoAuthenticator struct{}

func (nilProjectInfoAuthenticator) Claim(context.Context, string) (*types.PlainClaimPrinciple[*types.ProjectScope], error) {
	return &types.PlainClaimPrinciple[*types.ProjectScope]{}, nil
}

type typedNilLogger struct {
	commons.Logger
}

func TestOrganizationAuthenticationSupportsEveryTransport(t *testing.T) {
	t.Run("gin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		resolver := &organizationAuthenticatorStub{}
		engine := gin.New()
		engine.Use(NewOrganizationAuthenticatorMiddleware(resolver, nil))
		engine.GET("/test", func(ctx *gin.Context) {
			auth, err := types.Authorize(ctx.Request.Context())
			if err != nil || auth.Type() != types.AuthTypeOrg {
				t.Fatalf("authentication = %v, %v", auth, err)
			}
			ctx.Status(http.StatusNoContent)
		})
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set(types.ORG_SCOPE_KEY, "organization")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
	})

	t.Run("unary", func(t *testing.T) {
		_, err := NewOrganizationAuthenticatorUnaryServerMiddleware(&organizationAuthenticatorStub{}, nil)(
			incomingContext(types.ORG_SCOPE_KEY, "organization"), nil, nil,
			func(ctx context.Context, _ any) (any, error) {
				auth, err := types.Authorize(ctx)
				if err != nil || auth.Type() != types.AuthTypeOrg {
					t.Fatalf("authentication = %v, %v", auth, err)
				}
				return nil, nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("stream", func(t *testing.T) {
		err := NewOrganizationAuthenticatorStreamServerMiddleware(&organizationAuthenticatorStub{}, nil)(
			nil, &testServerStream{ctx: incomingContext(types.ORG_SCOPE_KEY, "organization")}, nil,
			func(_ any, stream grpc.ServerStream) error {
				auth, err := types.Authorize(stream.Context())
				if err != nil || auth.Type() != types.AuthTypeOrg {
					t.Fatalf("authentication = %v, %v", auth, err)
				}
				return nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestServiceAuthenticationSupportsGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &recordingServiceAuthenticator{}
	engine := gin.New()
	engine.Use(NewServiceAuthenticatorMiddleware(resolver, nil))
	engine.GET("/test", func(ctx *gin.Context) {
		auth, err := types.Authorize(ctx.Request.Context())
		if err != nil || auth.Type() != types.AuthTypeService {
			t.Fatalf("authentication = %v, %v", auth, err)
		}
		ctx.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(types.SERVICE_SCOPE_KEY, "service")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestUserProjectAndServiceAuthenticationSuccessAcrossMissingTransports(t *testing.T) {
	t.Run("user gin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.Use(NewAuthenticationMiddleware(userAuthenticatorStub{}, nil))
		engine.GET("/test", func(ctx *gin.Context) {
			auth, err := types.Authorize(ctx.Request.Context())
			if err != nil || auth.Type() != types.AuthTypeUser {
				t.Fatalf("authentication = %v, %v", auth, err)
			}
			ctx.Status(http.StatusNoContent)
		})
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set(types.AUTHORIZATION_KEY, "user")
		request.Header.Set(types.AUTH_KEY, "1")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
	})

	t.Run("project stream", func(t *testing.T) {
		err := NewProjectAuthenticatorStreamServerMiddleware(&projectScopeClaimAuthenticatorStub{}, nil)(
			nil, &testServerStream{ctx: incomingContext(types.PROJECT_SCOPE_KEY, "project")}, nil,
			func(_ any, stream grpc.ServerStream) error {
				auth, err := types.Authorize(stream.Context())
				if err != nil || auth.Type() != types.AuthTypeProject {
					t.Fatalf("authentication = %v, %v", auth, err)
				}
				return nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	})

	for _, test := range []struct {
		name string
		run  func() error
	}{
		{name: "service unary", run: func() error {
			_, err := NewServiceAuthenticatorUnaryServerMiddleware(serviceScopeClaimAuthenticatorStub{}, nil)(
				incomingContext(types.SERVICE_SCOPE_KEY, "service"), nil, nil,
				func(ctx context.Context, _ any) (any, error) {
					auth, err := types.Authorize(ctx)
					if err != nil || auth.Type() != types.AuthTypeService {
						t.Fatalf("authentication = %v, %v", auth, err)
					}
					return nil, nil
				},
			)
			return err
		}},
		{name: "service stream", run: func() error {
			return NewServiceAuthenticatorStreamServerMiddleware(serviceScopeClaimAuthenticatorStub{}, nil)(
				nil, &testServerStream{ctx: incomingContext(types.SERVICE_SCOPE_KEY, "service")}, nil,
				func(_ any, stream grpc.ServerStream) error {
					auth, err := types.Authorize(stream.Context())
					if err != nil || auth.Type() != types.AuthTypeService {
						t.Fatalf("authentication = %v, %v", auth, err)
					}
					return nil
				},
			)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestServiceAuthenticationRestoresDelegatedActorAcrossGRPC(t *testing.T) {
	tests := []struct {
		name      string
		authType  types.AuthType
		actorType types.ActorType
	}{
		{name: "user", authType: types.AuthTypeUser, actorType: types.ActorTypeUser},
		{name: "project", authType: types.AuthTypeProject, actorType: types.ActorTypeProject},
		{name: "organization", authType: types.AuthTypeOrg, actorType: types.ActorTypeOrganization},
		{name: "service", authType: types.AuthTypeService, actorType: types.ActorTypeService},
		{name: "system", authType: types.AuthTypeSystem, actorType: types.ActorTypeSystem},
	}
	assertAuthentication := func(t *testing.T, ctx context.Context, test struct {
		name      string
		authType  types.AuthType
		actorType types.ActorType
	}) {
		t.Helper()
		auth, err := types.Authorize(ctx)
		if err != nil {
			t.Fatalf("Authorize() error = %v", err)
		}
		if auth.Type() != test.authType {
			t.Fatalf("Type() = %q, want %q", auth.Type(), test.authType)
		}
		actor, err := auth.Actor()
		if err != nil || actor != (types.ActorIdentity{Type: test.actorType, ID: 5}) {
			t.Fatalf("Actor() = %+v, %v", actor, err)
		}
		caller, err := auth.Caller()
		if err != nil || caller != (types.ActorIdentity{Type: types.ActorTypeService, ID: 4}) {
			t.Fatalf("Caller() = %+v, %v", caller, err)
		}
	}

	for _, test := range tests {
		t.Run(test.name+" unary", func(t *testing.T) {
			_, err := NewServiceAuthenticatorUnaryServerMiddleware(
				delegatedServiceAuthenticatorStub{authType: test.authType, actorID: 5}, nil,
			)(incomingContext(types.SERVICE_SCOPE_KEY, "service"), nil, nil, func(ctx context.Context, _ any) (any, error) {
				assertAuthentication(t, ctx, test)
				return nil, nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run(test.name+" stream", func(t *testing.T) {
			err := NewServiceAuthenticatorStreamServerMiddleware(
				delegatedServiceAuthenticatorStub{authType: test.authType, actorID: 5}, nil,
			)(nil, &testServerStream{ctx: incomingContext(types.SERVICE_SCOPE_KEY, "service")}, nil, func(_ any, stream grpc.ServerStream) error {
				assertAuthentication(t, stream.Context(), test)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNewMiddlewareVariantsRejectCredentials(t *testing.T) {
	t.Run("organization gin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.Use(NewOrganizationAuthenticatorMiddleware(&organizationAuthenticatorStub{err: errors.New("Rejected")}, nil))
		handlerCalled := false
		engine.GET("/test", func(ctx *gin.Context) { handlerCalled = true })
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set(types.ORG_SCOPE_KEY, "organization")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized || handlerCalled {
			t.Fatalf("status = %d, handlerCalled = %t", recorder.Code, handlerCalled)
		}
	})

	t.Run("organization unary", func(t *testing.T) {
		handlerCalled := false
		_, err := NewOrganizationAuthenticatorUnaryServerMiddleware(&organizationAuthenticatorStub{err: errors.New("Rejected")}, nil)(
			incomingContext(types.ORG_SCOPE_KEY, "organization"), nil, nil,
			func(context.Context, any) (any, error) { handlerCalled = true; return nil, nil },
		)
		if status.Code(err) != codes.Unauthenticated || handlerCalled {
			t.Fatalf("status = %v, handlerCalled = %t", status.Code(err), handlerCalled)
		}
	})

	t.Run("organization stream", func(t *testing.T) {
		handlerCalled := false
		err := NewOrganizationAuthenticatorStreamServerMiddleware(&organizationAuthenticatorStub{err: errors.New("Rejected")}, nil)(
			nil, &testServerStream{ctx: incomingContext(types.ORG_SCOPE_KEY, "organization")}, nil,
			func(any, grpc.ServerStream) error { handlerCalled = true; return nil },
		)
		if status.Code(err) != codes.Unauthenticated || handlerCalled {
			t.Fatalf("status = %v, handlerCalled = %t", status.Code(err), handlerCalled)
		}
	})

	t.Run("service gin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.Use(NewServiceAuthenticatorMiddleware(serviceScopeClaimAuthenticatorStub{err: errors.New("Rejected")}, nil))
		handlerCalled := false
		engine.GET("/test", func(ctx *gin.Context) { handlerCalled = true })
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set(types.SERVICE_SCOPE_KEY, "service")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized || handlerCalled {
			t.Fatalf("status = %d, handlerCalled = %t", recorder.Code, handlerCalled)
		}
	})
}

func TestEveryAuthenticationMiddlewareRejectsInvalidAuditActor(t *testing.T) {
	invalidCredentialID := uint64(math.MaxInt64) + 1

	tests := []struct {
		name   string
		key    string
		value  string
		gin    gin.HandlerFunc
		unary  grpc.UnaryServerInterceptor
		stream grpc.StreamServerInterceptor
	}{
		{
			name:   "user",
			key:    types.AUTHORIZATION_KEY,
			value:  "user",
			gin:    NewAuthenticationMiddleware(invalidActorUserAuthenticator{}, nil),
			unary:  NewAuthenticationUnaryServerMiddleware(invalidActorUserAuthenticator{}, nil),
			stream: NewAuthenticationStreamServerMiddleware(invalidActorUserAuthenticator{}, nil),
		},
		{
			name:   "project",
			key:    types.PROJECT_SCOPE_KEY,
			value:  "project",
			gin:    NewProjectAuthenticatorMiddleware(&projectScopeClaimAuthenticatorStub{credentialID: invalidCredentialID}, nil),
			unary:  NewProjectAuthenticatorUnaryServerMiddleware(&projectScopeClaimAuthenticatorStub{credentialID: invalidCredentialID}, nil),
			stream: NewProjectAuthenticatorStreamServerMiddleware(&projectScopeClaimAuthenticatorStub{credentialID: invalidCredentialID}, nil),
		},
		{
			name:   "organization",
			key:    types.ORG_SCOPE_KEY,
			value:  "organization",
			gin:    NewOrganizationAuthenticatorMiddleware(&organizationAuthenticatorStub{credentialID: invalidCredentialID}, nil),
			unary:  NewOrganizationAuthenticatorUnaryServerMiddleware(&organizationAuthenticatorStub{credentialID: invalidCredentialID}, nil),
			stream: NewOrganizationAuthenticatorStreamServerMiddleware(&organizationAuthenticatorStub{credentialID: invalidCredentialID}, nil),
		},
		{
			name:   "service",
			key:    types.SERVICE_SCOPE_KEY,
			value:  "service",
			gin:    NewServiceAuthenticatorMiddleware(serviceScopeClaimAuthenticatorStub{actorID: invalidCredentialID}, nil),
			unary:  NewServiceAuthenticatorUnaryServerMiddleware(serviceScopeClaimAuthenticatorStub{actorID: invalidCredentialID}, nil),
			stream: NewServiceAuthenticatorStreamServerMiddleware(serviceScopeClaimAuthenticatorStub{actorID: invalidCredentialID}, nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name+" gin", func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			engine.Use(test.gin)
			handlerCalled := false
			engine.GET("/test", func(*gin.Context) { handlerCalled = true })
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set(test.key, test.value)
			if test.name == "user" {
				request.Header.Set(types.AUTH_KEY, "1")
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized || handlerCalled {
				t.Fatalf("status = %d, handlerCalled = %t", recorder.Code, handlerCalled)
			}
		})

		t.Run(test.name+" unary", func(t *testing.T) {
			values := []string{test.key, test.value}
			if test.name == "user" {
				values = append(values, types.AUTH_KEY, "1")
			}
			handlerCalled := false
			_, err := test.unary(incomingContext(values...), nil, nil, func(context.Context, any) (any, error) {
				handlerCalled = true
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated || handlerCalled {
				t.Fatalf("status = %v, handlerCalled = %t", status.Code(err), handlerCalled)
			}
		})

		t.Run(test.name+" stream", func(t *testing.T) {
			values := []string{test.key, test.value}
			if test.name == "user" {
				values = append(values, types.AUTH_KEY, "1")
			}
			handlerCalled := false
			err := test.stream(nil, &testServerStream{ctx: incomingContext(values...)}, nil, func(any, grpc.ServerStream) error {
				handlerCalled = true
				return nil
			})
			if status.Code(err) != codes.Unauthenticated || handlerCalled {
				t.Fatalf("status = %v, handlerCalled = %t", status.Code(err), handlerCalled)
			}
		})
	}
}

func TestEveryAuthenticationMiddlewarePassesWithoutCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginMiddlewares := []gin.HandlerFunc{
		NewAuthenticationMiddleware(nil, nil),
		NewProjectAuthenticatorMiddleware(nil, nil),
		NewOrganizationAuthenticatorMiddleware(nil, nil),
		NewServiceAuthenticatorMiddleware(nil, nil),
	}
	for index, middleware := range ginMiddlewares {
		handlerCalled := false
		engine := gin.New()
		engine.Use(middleware)
		engine.GET("/test", func(ctx *gin.Context) { handlerCalled = true; ctx.Status(http.StatusNoContent) })
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
		if recorder.Code != http.StatusNoContent || !handlerCalled {
			t.Fatalf("gin middleware %d did not pass through", index)
		}
	}

	unaryMiddlewares := []grpc.UnaryServerInterceptor{
		NewAuthenticationUnaryServerMiddleware(nil, nil),
		NewProjectAuthenticatorUnaryServerMiddleware(nil, nil),
		NewOrganizationAuthenticatorUnaryServerMiddleware(nil, nil),
		NewServiceAuthenticatorUnaryServerMiddleware(nil, nil),
	}
	for index, middleware := range unaryMiddlewares {
		handlerCalled := false
		_, err := middleware(context.Background(), nil, nil, func(context.Context, any) (any, error) {
			handlerCalled = true
			return nil, nil
		})
		if err != nil || !handlerCalled {
			t.Fatalf("unary middleware %d error = %v, handlerCalled = %t", index, err, handlerCalled)
		}
	}

	streamMiddlewares := []grpc.StreamServerInterceptor{
		NewAuthenticationStreamServerMiddleware(nil, nil),
		NewProjectAuthenticatorStreamServerMiddleware(nil, nil),
		NewOrganizationAuthenticatorStreamServerMiddleware(nil, nil),
		NewServiceAuthenticatorStreamServerMiddleware(nil, nil),
	}
	for index, middleware := range streamMiddlewares {
		handlerCalled := false
		err := middleware(nil, &testServerStream{ctx: context.Background()}, nil, func(any, grpc.ServerStream) error {
			handlerCalled = true
			return nil
		})
		if err != nil || !handlerCalled {
			t.Fatalf("stream middleware %d error = %v, handlerCalled = %t", index, err, handlerCalled)
		}
	}
}

func TestAuthenticationChainsRejectConflictsInEitherOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "normal"
		if reverse {
			name = "reversed"
		}
		t.Run("gin "+name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			user := NewAuthenticationMiddleware(userAuthenticatorStub{}, nil)
			project := NewProjectAuthenticatorMiddleware(&projectScopeClaimAuthenticatorStub{}, nil)
			engine := gin.New()
			if reverse {
				engine.Use(project, user)
			} else {
				engine.Use(user, project)
			}
			handlerCalled := false
			engine.GET("/test", func(ctx *gin.Context) {
				handlerCalled = true
				ctx.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set(types.AUTHORIZATION_KEY, "user")
			request.Header.Set(types.AUTH_KEY, "1")
			request.Header.Set(types.PROJECT_SCOPE_KEY, "project")
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized || handlerCalled {
				t.Fatalf("status = %d, handlerCalled = %t", recorder.Code, handlerCalled)
			}
		})

		t.Run("unary "+name, func(t *testing.T) {
			user := NewAuthenticationUnaryServerMiddleware(userAuthenticatorStub{}, nil)
			project := NewProjectAuthenticatorUnaryServerMiddleware(&projectScopeClaimAuthenticatorStub{}, nil)
			first, second := user, project
			if reverse {
				first, second = project, user
			}
			handlerCalled := false
			_, err := first(incomingContext(types.AUTHORIZATION_KEY, "user", types.AUTH_KEY, "1", types.PROJECT_SCOPE_KEY, "project"), nil, nil,
				func(ctx context.Context, request any) (any, error) {
					return second(ctx, request, nil, func(context.Context, any) (any, error) {
						handlerCalled = true
						return nil, nil
					})
				})
			if status.Code(err) != codes.Unauthenticated || handlerCalled {
				t.Fatalf("status = %v, handlerCalled = %t", status.Code(err), handlerCalled)
			}
		})

		t.Run("stream "+name, func(t *testing.T) {
			user := NewAuthenticationStreamServerMiddleware(userAuthenticatorStub{}, nil)
			project := NewProjectAuthenticatorStreamServerMiddleware(&projectScopeClaimAuthenticatorStub{}, nil)
			first, second := user, project
			if reverse {
				first, second = project, user
			}
			handlerCalled := false
			stream := &testServerStream{ctx: incomingContext(types.AUTHORIZATION_KEY, "user", types.AUTH_KEY, "1", types.PROJECT_SCOPE_KEY, "project")}
			err := first(nil, stream, nil, func(server any, authenticated grpc.ServerStream) error {
				return second(server, authenticated, nil, func(any, grpc.ServerStream) error {
					handlerCalled = true
					return nil
				})
			})
			if status.Code(err) != codes.Unauthenticated || handlerCalled {
				t.Fatalf("status = %v, handlerCalled = %t", status.Code(err), handlerCalled)
			}
		})
	}
}

func TestGinAuthenticationPreservesSourcePrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &recordingUserAuthenticator{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test?"+types.AUTHORIZATION_KEY+"=query&"+types.AUTH_KEY+"=30&"+types.PROJECT_KEY+"=30", nil)
	ctx.Request.Header.Set(types.AUTHORIZATION_KEY, "header")
	ctx.Request.Header.Set(types.AUTH_KEY, "10")
	ctx.Request.Header.Set(types.PROJECT_KEY, "10")
	ctx.Params = gin.Params{{Key: types.AUTHORIZATION_KEY, Value: "path"}, {Key: types.AUTH_KEY, Value: "20"}, {Key: types.PROJECT_KEY, Value: "20"}}
	NewAuthenticationMiddleware(user, nil)(ctx)
	if user.token != "path" || user.userID != 10 {
		t.Fatalf("user credential = (%q, %d), want (%q, %d)", user.token, user.userID, "path", 10)
	}
	auth, err := types.Authorize(ctx.Request.Context())
	if err != nil {
		t.Fatal(err)
	}
	projectContext, err := auth.ProjectContext()
	if err != nil || projectContext.ProjectID != 10 {
		t.Fatalf("project context = %+v, %v", projectContext, err)
	}

	project := &projectScopeClaimAuthenticatorStub{}
	organization := &organizationAuthenticatorStub{}
	service := &recordingServiceAuthenticator{}
	tests := []struct {
		name       string
		key        string
		middleware gin.HandlerFunc
		token      func() string
	}{
		{name: "project", key: types.PROJECT_SCOPE_KEY, middleware: NewProjectAuthenticatorMiddleware(project, nil), token: func() string { return project.lastToken }},
		{name: "organization", key: types.ORG_SCOPE_KEY, middleware: NewOrganizationAuthenticatorMiddleware(organization, nil), token: func() string { return organization.token }},
		{name: "service", key: types.SERVICE_SCOPE_KEY, middleware: NewServiceAuthenticatorMiddleware(service, nil), token: func() string { return service.token }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/test?"+test.key+"=query", nil)
			ctx.Request.Header.Set(test.key, "header")
			ctx.Params = gin.Params{{Key: test.key, Value: "path"}}
			test.middleware(ctx)
			if test.token() != "header" {
				t.Fatalf("token = %q, want header", test.token())
			}
		})
	}
}

func TestAuthenticationLogsExcludeCredentialValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	malicious := "secret\nFORGED AUTH LOG ENTRY"
	logDirectory := t.TempDir()
	logger, err := commons.NewApplicationLogger(commons.Name("authentication-security"), commons.Path(logDirectory), commons.EnableConsole(false), commons.EnableFile(true))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewAuthenticationUnaryServerMiddleware(userAuthenticatorStub{err: errors.New("Rejected")}, logger)(incomingContext(types.AUTHORIZATION_KEY, malicious, types.AUTH_KEY, "1"), nil, nil, func(context.Context, any) (any, error) { return nil, nil }); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unary rejection status = %v", status.Code(err))
	}
	if err := NewAuthenticationStreamServerMiddleware(userAuthenticatorStub{err: errors.New("Rejected")}, logger)(nil, &testServerStream{ctx: incomingContext(types.AUTHORIZATION_KEY, malicious, types.AUTH_KEY, "1")}, nil, func(any, grpc.ServerStream) error { return nil }); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("stream rejection status = %v", status.Code(err))
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request.Header.Set(types.AUTHORIZATION_KEY, malicious)
	ctx.Request.Header.Set(types.AUTH_KEY, "1")
	NewAuthenticationMiddleware(userAuthenticatorStub{err: errors.New("Rejected")}, logger)(ctx)

	existingAuthentication := &types.Authentication{AuthType: types.AuthTypeUser}
	unaryContext := context.WithValue(incomingContext(types.PROJECT_SCOPE_KEY, malicious), types.CTX_, existingAuthentication)
	if _, err := NewProjectAuthenticatorUnaryServerMiddleware(&projectScopeClaimAuthenticatorStub{}, logger)(unaryContext, nil, nil, func(context.Context, any) (any, error) { return nil, nil }); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unary conflict status = %v", status.Code(err))
	}
	streamContext := context.WithValue(incomingContext(types.PROJECT_SCOPE_KEY, malicious), types.CTX_, existingAuthentication)
	if err := NewProjectAuthenticatorStreamServerMiddleware(&projectScopeClaimAuthenticatorStub{}, logger)(nil, &testServerStream{ctx: streamContext}, nil, func(any, grpc.ServerStream) error { return nil }); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("stream conflict status = %v", status.Code(err))
	}
	conflictRecorder := httptest.NewRecorder()
	conflictContext, _ := gin.CreateTestContext(conflictRecorder)
	conflictRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	conflictRequest.Header.Set(types.PROJECT_SCOPE_KEY, malicious)
	conflictContext.Request = conflictRequest.WithContext(context.WithValue(conflictRequest.Context(), types.CTX_, existingAuthentication))
	NewProjectAuthenticatorMiddleware(&projectScopeClaimAuthenticatorStub{}, logger)(conflictContext)

	if err := logger.Sync(); err != nil {
		t.Fatal(err)
	}
	logData, err := fs.ReadFile(os.DirFS(logDirectory), "authentication-security.log")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logData), malicious) || strings.Contains(string(logData), "FORGED") {
		t.Fatalf("credential value was written to logs: %s", logData)
	}
}

func TestAuthenticationFailureUsesExportedTypedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(NewAuthenticationMiddleware(nil, nil))
	engine.GET("/test", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(types.AUTHORIZATION_KEY, "token")
	request.Header.Set(types.AUTH_KEY, "1")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	var response AuthenticationError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != AuthenticationFailureMessage {
		t.Fatalf("error = %q, want %q", response.Error, AuthenticationFailureMessage)
	}

	_, err := NewAuthenticationUnaryServerMiddleware(nil, nil)(
		incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"), nil, nil,
		func(context.Context, any) (any, error) { return nil, nil },
	)
	if status.Code(err) != codes.Unauthenticated || status.Convert(err).Message() != AuthenticationFailureMessage {
		t.Fatalf("gRPC error = %v", err)
	}
}

func TestAuthenticationValidationHandlesTypedNilValues(t *testing.T) {
	t.Run("resolver", func(t *testing.T) {
		var resolver *typedNilUserAuthenticator
		_, err := NewAuthenticationUnaryServerMiddleware(resolver, nil)(
			incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"), nil, nil,
			func(context.Context, any) (any, error) { return nil, nil },
		)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("status = %v, want %v", status.Code(err), codes.Unauthenticated)
		}
	})

	t.Run("principle", func(t *testing.T) {
		_, err := NewAuthenticationUnaryServerMiddleware(nilPrincipleUserAuthenticator{}, nil)(
			incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"), nil, nil,
			func(context.Context, any) (any, error) { return nil, nil },
		)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("status = %v, want %v", status.Code(err), codes.Unauthenticated)
		}
	})

	t.Run("claim info", func(t *testing.T) {
		_, err := NewProjectAuthenticatorUnaryServerMiddleware(nilProjectInfoAuthenticator{}, nil)(
			incomingContext(types.PROJECT_SCOPE_KEY, "project"), nil, nil,
			func(context.Context, any) (any, error) { return nil, nil },
		)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("status = %v, want %v", status.Code(err), codes.Unauthenticated)
		}
	})

	t.Run("context", func(t *testing.T) {
		var existingAuthentication *types.Authentication
		ctx := context.WithValue(incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"), types.CTX_, existingAuthentication)
		handlerCalled := false
		_, err := NewAuthenticationUnaryServerMiddleware(userAuthenticatorStub{}, nil)(ctx, nil, nil, func(context.Context, any) (any, error) {
			handlerCalled = true
			return nil, nil
		})
		if err != nil || !handlerCalled {
			t.Fatalf("error = %v, handlerCalled = %t", err, handlerCalled)
		}
	})

	t.Run("logger", func(t *testing.T) {
		var logger *typedNilLogger
		_, err := NewAuthenticationUnaryServerMiddleware(nil, logger)(
			incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "1"), nil, nil,
			func(context.Context, any) (any, error) { return nil, nil },
		)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("status = %v, want %v", status.Code(err), codes.Unauthenticated)
		}
	})
}

func TestUserGinWhitespacePathRetainsPrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &recordingUserAuthenticator{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx.Request.Header.Set(types.AUTHORIZATION_KEY, "header-token")
	ctx.Request.Header.Set(types.AUTH_KEY, "1")
	ctx.Params = gin.Params{{Key: types.AUTHORIZATION_KEY, Value: " "}}

	NewAuthenticationMiddleware(resolver, nil)(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if resolver.token != "" {
		t.Fatalf("resolver token = %q, want no authentication attempt", resolver.token)
	}
}

func TestUserIdentifierBoundariesRemainUnchanged(t *testing.T) {
	maximumUserID := uint64(math.MaxInt64)
	resolver := &recordingUserAuthenticator{}
	_, err := NewAuthenticationUnaryServerMiddleware(resolver, nil)(
		incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "9223372036854775807"), nil, nil,
		func(context.Context, any) (any, error) { return nil, nil },
	)
	if err != nil || resolver.userID != maximumUserID {
		t.Fatalf("maximum user ID error = %v, userID = %d", err, resolver.userID)
	}

	_, err = NewAuthenticationUnaryServerMiddleware(resolver, nil)(
		incomingContext(types.AUTHORIZATION_KEY, "token", types.AUTH_KEY, "9223372036854775808"), nil, nil,
		func(context.Context, any) (any, error) { return nil, nil },
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("overflow user ID status = %v", status.Code(err))
	}

	largeProjectID := uint64(math.MaxInt64) + 1
	projectResolver := &recordingUserAuthenticator{projectIDs: []uint64{largeProjectID}}
	_, err = NewAuthenticationUnaryServerMiddleware(projectResolver, nil)(
		incomingContext(
			types.AUTHORIZATION_KEY, "token",
			types.AUTH_KEY, "1",
			types.PROJECT_KEY, strconv.FormatUint(largeProjectID, 10),
		), nil, nil,
		func(ctx context.Context, _ any) (any, error) {
			authentication, authErr := types.Authorize(ctx)
			if authErr != nil {
				return nil, authErr
			}
			projectContext, projectErr := authentication.ProjectContext()
			if projectErr != nil || projectContext.ProjectID != largeProjectID {
				t.Fatalf("project context = %+v, error = %v", projectContext, projectErr)
			}
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("large project ID error = %v", err)
	}
}
