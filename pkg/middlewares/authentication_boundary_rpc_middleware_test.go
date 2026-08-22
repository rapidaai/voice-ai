package middlewares

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/types"
)

func TestAuthenticationBoundaryGinAcceptsEveryCredentialClass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		headers  map[string]string
		expected types.AuthType
	}{
		{name: "user", headers: map[string]string{types.AUTHORIZATION_KEY: "token", types.AUTH_KEY: "1"}, expected: types.AuthTypeUser},
		{name: "project", headers: map[string]string{types.PROJECT_SCOPE_KEY: types.PROJECT_KEY_PREFIX + "project"}, expected: types.AuthTypeProject},
		{name: "organization", headers: map[string]string{types.ORG_SCOPE_KEY: "organization"}, expected: types.AuthTypeOrg},
		{name: "service", headers: map[string]string{types.SERVICE_SCOPE_KEY: "service"}, expected: types.AuthTypeService},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(NewAuthenticationBoundaryMiddleware(
				userAuthenticatorStub{},
				boundaryProjectScopeClaimAuthenticatorStub{},
				organizationScopeClaimAuthenticatorStub{},
				serviceScopeClaimAuthenticatorStub{},
				nil,
			))
			engine.GET("/test", func(ctx *gin.Context) {
				auth, err := types.Authorize(ctx.Request.Context())
				if err != nil {
					t.Errorf("Authorize() error = %v", err)
					ctx.Status(http.StatusInternalServerError)
					return
				}
				if auth.Type() != test.expected {
					t.Errorf("authentication type = %v, want %v", auth.Type(), test.expected)
					ctx.Status(http.StatusInternalServerError)
					return
				}
				if test.expected == types.AuthTypeOrg {
					actor, err := auth.Actor()
					if err != nil || actor != (types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 3}) {
						t.Errorf("organization actor = %+v, %v", actor, err)
						ctx.Status(http.StatusInternalServerError)
						return
					}
				}
				ctx.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			for key, value := range test.headers {
				request.Header.Set(key, value)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
			}
		})
	}
}

func TestAuthenticationBoundaryGinAllowsPublicRequestWithoutCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(NewAuthenticationBoundaryMiddleware(nil, nil, nil, nil, nil))
	engine.GET("/health", func(ctx *gin.Context) {
		if _, err := types.Authorize(ctx.Request.Context()); err == nil {
			t.Error("Authorize() succeeded without credentials")
		}
		ctx.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestAuthenticationBoundaryGinRejectsOutOfRangeUserActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	organizationID := uint64(2)
	user := &types.PlainAuthPrinciple{
		User:             types.UserInfo{Id: uint64(math.MaxInt64) + 1},
		OrganizationRole: &types.OrganizaitonRole{OrganizationId: organizationID},
	}
	engine := gin.New()
	engine.Use(NewAuthenticationBoundaryMiddleware(
		projectSelectingUserAuthenticatorStub{auth: user},
		boundaryProjectScopeClaimAuthenticatorStub{},
		organizationScopeClaimAuthenticatorStub{},
		serviceScopeClaimAuthenticatorStub{},
		nil,
	))
	handlerCalled := false
	engine.GET("/test", func(ctx *gin.Context) {
		handlerCalled = true
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(types.AUTHORIZATION_KEY, "token")
	request.Header.Set(types.AUTH_KEY, "1")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Fatal("handler called for out-of-range user actor")
	}
}

func TestAuthenticationBoundaryGinRejectsInvalidProjectActor(t *testing.T) {
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
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			engine.Use(NewAuthenticationBoundaryMiddleware(
				userAuthenticatorStub{},
				test.authenticator,
				organizationScopeClaimAuthenticatorStub{},
				serviceScopeClaimAuthenticatorStub{},
				nil,
			))
			handlerCalled := false
			engine.GET("/test", func(ctx *gin.Context) {
				handlerCalled = true
				ctx.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set(types.PROJECT_SCOPE_KEY, "project")
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if handlerCalled {
				t.Fatal("handler called for invalid project actor")
			}
		})
	}
}

func TestAuthenticationBoundaryGinRejectsInvalidOrganizationActor(t *testing.T) {
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
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			engine.Use(NewAuthenticationBoundaryMiddleware(
				userAuthenticatorStub{},
				boundaryProjectScopeClaimAuthenticatorStub{},
				test.authenticator,
				serviceScopeClaimAuthenticatorStub{},
				nil,
			))
			handlerCalled := false
			engine.GET("/test", func(ctx *gin.Context) {
				handlerCalled = true
				ctx.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set(types.ORG_SCOPE_KEY, "organization")
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if handlerCalled {
				t.Fatal("handler called for invalid organization actor")
			}
		})
	}
}

func TestAuthenticationBoundaryGinRejectsConflictingCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(NewAuthenticationBoundaryMiddleware(
		userAuthenticatorStub{},
		boundaryProjectScopeClaimAuthenticatorStub{},
		organizationScopeClaimAuthenticatorStub{},
		serviceScopeClaimAuthenticatorStub{},
		nil,
	))
	handlerCalled := false
	engine.GET("/test", func(ctx *gin.Context) {
		handlerCalled = true
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(types.ORG_SCOPE_KEY, "organization")
	request.Header.Set(types.SERVICE_SCOPE_KEY, "service")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Fatal("handler called for conflicting credentials")
	}
}

func TestAuthenticationBoundaryGinRejectsUnsupportedCredentialClass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(NewAuthenticationBoundaryMiddleware(userAuthenticatorStub{}, nil, nil, nil, nil))
	handlerCalled := false
	engine.GET("/test", func(ctx *gin.Context) {
		handlerCalled = true
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set(types.ORG_SCOPE_KEY, "organization")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Fatal("handler called for unsupported credential class")
	}
}
