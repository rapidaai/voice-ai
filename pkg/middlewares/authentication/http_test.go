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
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	principal "github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/authorizer"
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/types"
)

type httpTestProjectSource struct {
	calls int
	token string
	err   error
}

func (source *httpTestProjectSource) ResolveProjectCredential(_ context.Context, token string) (principal.Authentication, error) {
	source.calls++
	source.token = token
	if source.err != nil {
		return principal.Authentication{}, source.err
	}
	return principal.NewDirectAuthentication(
		principal.Identity{Type: principal.IdentityProjectCredential, ID: 31},
		principal.Tenant{OrganizationID: 8, ProjectID: 9},
		principal.Proof{Method: principal.AuthenticationProjectCredential, Audience: principal.AudienceManagement, Reference: principal.ProofReference{Type: principal.ProofReferenceCredential, Value: "31"}},
		nil,
		[]principal.Organization{{ID: 8, Name: "Organization", Projects: []principal.Project{{ID: 9, Name: "Project", Roles: []principal.Role{{ID: 21, Key: "project_reader", Name: "Project Reader", Scope: principal.RoleScopeProject, Subject: principal.RoleSubjectProjectToken, DefinitionType: principal.RoleDefinitionBuiltIn, Permissions: []principal.Permission{{ID: 22, Key: "project.assistant.read", Scope: principal.RoleScopeProject, Name: "Read assistants"}}}}}}}},
	)
}

type httpTestOrganizationSource struct {
	calls int
	token string
	err   error
}

func (source *httpTestOrganizationSource) ResolveOrganizationCredential(_ context.Context, token string) (principal.Authentication, error) {
	source.calls++
	source.token = token
	if source.err != nil {
		return principal.Authentication{}, source.err
	}
	return principal.NewDirectAuthentication(
		principal.Identity{Type: principal.IdentityOrganizationCredential, ID: 41},
		principal.Tenant{OrganizationID: 8},
		principal.Proof{Method: principal.AuthenticationOrganizationCredential, Reference: principal.ProofReference{Type: principal.ProofReferenceCredential, Value: "41"}},
		nil, []principal.Organization{{ID: 8, Name: "Organization"}},
	)
}

type httpTestUserSource struct {
	calls          int
	token          string
	userID         uint64
	organizationID uint64
	projectID      uint64
	err            error
}

func (source *httpTestUserSource) ResolveUserCredential(_ context.Context, token string, userID uint64, organizationID uint64, projectID uint64) (principal.Authentication, error) {
	source.calls++
	source.token = token
	source.userID = userID
	source.organizationID = organizationID
	source.projectID = projectID
	if source.err != nil {
		return principal.Authentication{}, source.err
	}
	if organizationID == 0 {
		organizationID = 8
	}
	organization := principal.Organization{ID: organizationID, Name: "Organization", Roles: []principal.Role{{ID: 31, Key: "organization_reader", Name: "Organization Reader", Scope: principal.RoleScopeOrganization, Subject: principal.RoleSubjectUser, DefinitionType: principal.RoleDefinitionBuiltIn, Permissions: []principal.Permission{{ID: 32, Key: "organization.user.read", Scope: principal.RoleScopeOrganization, Name: "Read users"}}}}}
	if projectID != 0 {
		organization.Projects = []principal.Project{{ID: projectID, Name: "Project", Roles: []principal.Role{{ID: 33, Key: "project_reader", Name: "Project Reader", Scope: principal.RoleScopeProject, Subject: principal.RoleSubjectUser, DefinitionType: principal.RoleDefinitionBuiltIn, Permissions: []principal.Permission{{ID: 34, Key: "project.assistant.read", Scope: principal.RoleScopeProject, Name: "Read assistants"}}}}}}
	}
	return principal.NewDirectAuthentication(
		principal.Identity{Type: principal.IdentityUser, ID: userID},
		principal.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		principal.Proof{Method: principal.AuthenticationUserSession, Reference: principal.ProofReference{Type: principal.ProofReferenceUserSession, Value: "51"}},
		&principal.User{ID: userID, Name: "User", Email: "user@example.com", Status: "ACTIVE"},
		[]principal.Organization{organization},
	)
}

type httpTestPolicy struct {
	http map[string]authorizer.RoutePolicy
}

func (policy httpTestPolicy) HTTPPolicy(route string) (authorizer.RoutePolicy, bool) {
	value, exists := policy.http[route]
	return value, exists
}

func (httpTestPolicy) GRPCPolicy(string) (authorizer.RoutePolicy, bool) {
	return authorizer.RoutePolicy{}, false
}

func httpTestPrincipal(t *testing.T, subject principal.Identity, tenant principal.Tenant, authorities []principal.PermissionKey) principal.Principal {
	t.Helper()
	proof := principal.Proof{Method: principal.AuthenticationUserSession, Reference: principal.ProofReference{Type: principal.ProofReferenceUserSession, Value: "51"}}
	if subject.Type == principal.IdentityProjectCredential {
		proof = principal.Proof{
			Method: principal.AuthenticationProjectCredential, Audience: principal.AudienceManagement,
			Reference: principal.ProofReference{Type: principal.ProofReferenceCredential, Value: "31"},
		}
	}
	value, err := principal.NewDirectPrincipal(subject, tenant, proof, authorities)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

type httpTestFamily struct {
	name      string
	handler   gin.HandlerFunc
	valid     func(*http.Request)
	malformed func(*http.Request)
	duplicate func(*http.Request)
	calls     func() int
}

func httpTestFamilies(t *testing.T, rejected bool) []httpTestFamily {
	t.Helper()
	rejection := error(nil)
	if rejected {
		rejection = errors.New("credential rejected")
	}

	projectSource := &httpTestProjectSource{err: rejection}
	projectResolver := resolver.NewProjectPrincipalResolver(projectSource)
	organizationSource := &httpTestOrganizationSource{err: rejection}
	organizationResolver := resolver.NewOrganizationPrincipalResolver(organizationSource)
	userSource := &httpTestUserSource{err: rejection}
	userResolver := resolver.NewUserPrincipalResolver(userSource)

	key := bytes.Repeat([]byte{1}, sha256.Size)
	issuer, err := authentication_jwt.NewIssuer("web-api", key)
	if err != nil {
		t.Fatal(err)
	}
	serviceResolver := resolver.NewServicePrincipalResolver("web-api", key)
	assertion, err := issuer.Issue(
		principal.Identity{Type: principal.IdentityService, ID: 7, Issuer: "workflow-api"}, principal.Tenant{}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if rejected {
		assertion = "legacy-token"
	}

	return []httpTestFamily{
		{
			name: "project", handler: NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger),
			valid:     func(request *http.Request) { request.Header.Set(types.PROJECT_SCOPE_KEY, "rpd-prj-project") },
			malformed: func(request *http.Request) { request.Header.Set(types.PROJECT_SCOPE_KEY, " ") },
			duplicate: func(request *http.Request) {
				request.Header[http.CanonicalHeaderKey(types.PROJECT_SCOPE_KEY)] = []string{"rpd-prj-one", "rpd-prj-two"}
			},
			calls: func() int { return projectSource.calls },
		},
		{
			name: "organization", handler: NewOrganizationHTTPMiddleware(organizationResolver, authenticationMiddlewareTestLogger),
			valid:     func(request *http.Request) { request.Header.Set(types.ORG_SCOPE_KEY, "organization-secret") },
			malformed: func(request *http.Request) { request.Header.Set(types.ORG_SCOPE_KEY, " ") },
			duplicate: func(request *http.Request) {
				request.Header[http.CanonicalHeaderKey(types.ORG_SCOPE_KEY)] = []string{"one", "two"}
			},
			calls: func() int { return organizationSource.calls },
		},
		{
			name: "service", handler: NewServiceHTTPMiddleware(serviceResolver, authenticationMiddlewareTestLogger),
			valid:     func(request *http.Request) { request.Header.Set(types.SERVICE_SCOPE_KEY, assertion) },
			malformed: func(request *http.Request) { request.Header.Set(types.SERVICE_SCOPE_KEY, " ") },
			duplicate: func(request *http.Request) {
				request.Header[http.CanonicalHeaderKey(types.SERVICE_SCOPE_KEY)] = []string{"one", "two"}
			},
			calls: func() int { return -1 },
		},
		{
			name: "user", handler: NewUserHTTPMiddleware(userResolver, authenticationMiddlewareTestLogger),
			valid: func(request *http.Request) {
				request.Header.Set(types.AUTHORIZATION_KEY, "session-token")
				request.Header.Set(types.AUTH_KEY, "11")
				request.Header.Set(types.ORGANIZATION_KEY, "8")
			},
			malformed: func(request *http.Request) {
				request.Header.Set(types.AUTHORIZATION_KEY, " ")
				request.Header.Set(types.AUTH_KEY, "11")
			},
			duplicate: func(request *http.Request) {
				request.Header[http.CanonicalHeaderKey(types.AUTHORIZATION_KEY)] = []string{"one", "two"}
				request.Header.Set(types.AUTH_KEY, "11")
			},
			calls: func() int { return userSource.calls },
		},
	}
}

func TestProjectHTTPPreservesSourcePrecedenceAndRejectsDuplicates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	source := &httpTestProjectSource{}
	projectResolver := resolver.NewProjectPrincipalResolver(source)
	called := 0
	engine := gin.New()
	engine.Use(NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger))
	engine.GET("/project", func(ctx *gin.Context) {
		called++
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/project?x-api-key=rpd-prj-query", nil)
	request.Header[http.CanonicalHeaderKey(types.PROJECT_SCOPE_KEY)] = []string{"rpd-prj-header"}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || source.token != "header" || source.calls != 1 || called != 1 {
		t.Fatalf("header precedence = status %d token %q calls %d handler %d", response.Code, source.token, source.calls, called)
	}

	request = httptest.NewRequest(http.MethodGet, "/project?x-api-key=rpd-prj-one&x-api-key=rpd-prj-two", nil)
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || source.calls != 1 || called != 1 {
		t.Fatalf("duplicate query = status %d calls %d handler %d", response.Code, source.calls, called)
	}

	request = httptest.NewRequest(http.MethodGet, "/project", nil)
	request.Header[http.CanonicalHeaderKey(types.PROJECT_SCOPE_KEY)] = []string{"rpd-prj-one", "rpd-prj-two"}
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || source.calls != 1 || called != 1 {
		t.Fatalf("duplicate header = status %d calls %d handler %d", response.Code, source.calls, called)
	}
}

func TestHTTPFamilyMiddlewareAuthenticatesOrganizationUserAndService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("organization", func(t *testing.T) {
		source := &httpTestOrganizationSource{}
		organizationResolver := resolver.NewOrganizationPrincipalResolver(source)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		ctx.Request.Header.Set(types.ORG_SCOPE_KEY, "organization-secret")
		NewOrganizationHTTPMiddleware(organizationResolver, authenticationMiddlewareTestLogger)(ctx)
		value, exists := Principal(ctx.Request.Context())
		if !exists || value.Subject() != (principal.Identity{Type: principal.IdentityOrganizationCredential, ID: 41}) ||
			source.token != "organization-secret" {
			t.Fatalf("organization principal = %+v, %v token %q", value.Subject(), exists, source.token)
		}
	})

	t.Run("user source order", func(t *testing.T) {
		userSource := &httpTestUserSource{}
		userResolver := resolver.NewUserPrincipalResolver(userSource)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodGet, "/?authorization=query-token&x-auth-id=99&x-project-id=99", nil)
		ctx.Request.Header.Set(types.AUTHORIZATION_KEY, "header-token")
		ctx.Request.Header.Set(types.AUTH_KEY, "11")
		ctx.Request.Header.Set(types.ORGANIZATION_KEY, "8")
		ctx.Request.Header.Set(types.PROJECT_KEY, "9")
		ctx.Params = gin.Params{
			{Key: types.AUTHORIZATION_KEY, Value: "path-token"},
			{Key: types.AUTH_KEY, Value: "12"},
			{Key: types.PROJECT_KEY, Value: "10"},
		}
		NewUserHTTPMiddleware(userResolver, authenticationMiddlewareTestLogger)(ctx)
		value, exists := Principal(ctx.Request.Context())
		if !exists || value.Subject() != (principal.Identity{Type: principal.IdentityUser, ID: 11}) ||
			userSource.token != "path-token" || userSource.userID != 11 || userSource.organizationID != 8 || userSource.projectID != 9 {
			t.Fatalf("user principal = %+v, %v token %q user %d tenant %+v", value.Subject(), exists, userSource.token, userSource.userID, value.Tenant())
		}
		if !reflect.DeepEqual(value.Authorities(), []principal.PermissionKey{"organization.user.read", "project.assistant.read"}) {
			t.Fatalf("Authorities() = %v", value.Authorities())
		}
	})

	t.Run("service", func(t *testing.T) {
		key := bytes.Repeat([]byte{1}, sha256.Size)
		issuer, err := authentication_jwt.NewIssuer("web-api", key)
		if err != nil {
			t.Fatal(err)
		}
		serviceResolver := resolver.NewServicePrincipalResolver("web-api", key)
		assertion, err := issuer.Issue(
			principal.Identity{Type: principal.IdentityUser, ID: 11},
			principal.Tenant{OrganizationID: 8, ProjectID: 9},
			[]principal.PermissionKey{"project.assistant.read"},
		)
		if err != nil {
			t.Fatal(err)
		}
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		ctx.Request.Header.Set(types.SERVICE_SCOPE_KEY, assertion)
		NewServiceHTTPMiddleware(serviceResolver, authenticationMiddlewareTestLogger)(ctx)
		value, exists := Principal(ctx.Request.Context())
		actor, hasActor := value.Actor()
		if !exists || value.Subject() != (principal.Identity{Type: principal.IdentityUser, ID: 11}) || !hasActor ||
			actor != (principal.ServiceActor{ID: 1, Issuer: "internal"}) || value.Proof().Reference.Value == "" {
			t.Fatalf("service principal = subject %+v actor %+v/%v proof %+v", value.Subject(), actor, hasActor, value.Proof())
		}

		invalidContext, _ := gin.CreateTestContext(httptest.NewRecorder())
		invalidContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		invalidContext.Request.Header.Set(types.SERVICE_SCOPE_KEY, "legacy-token")
		NewServiceHTTPMiddleware(serviceResolver, authenticationMiddlewareTestLogger)(invalidContext)
		if !invalidContext.IsAborted() {
			t.Fatal("legacy service token reached the handler")
		}
	})
}

func TestHTTPAuthenticationRejectsConflictCancellationAndNilResolver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	source := &httpTestProjectSource{}
	projectResolver := resolver.NewProjectPrincipalResolver(source)

	user := httpTestPrincipal(t, principal.Identity{Type: principal.IdentityUser, ID: 11}, principal.Tenant{OrganizationID: 8}, nil)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(types.PROJECT_SCOPE_KEY, "rpd-prj-project")
	requestContext, err := ContextWithPrincipal(request.Context(), user)
	if err != nil {
		t.Fatal(err)
	}
	request = request.WithContext(requestContext)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = request
	NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger)(ctx)
	if !ctx.IsAborted() || source.calls != 0 {
		t.Fatalf("conflict = aborted %v calls %d", ctx.IsAborted(), source.calls)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(canceled)
	request.Header.Set(types.PROJECT_SCOPE_KEY, "rpd-prj-project")
	response := httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(response)
	ctx.Request = request
	NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger)(ctx)
	if !ctx.IsAborted() || source.calls != 0 || response.Body.Len() != 0 {
		t.Fatalf("cancellation = aborted %v calls %d body %q", ctx.IsAborted(), source.calls, response.Body.String())
	}

	response = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	NewProjectHTTPMiddleware(nil, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusUnauthorized || !ctx.IsAborted() {
		t.Fatalf("nil resolver = status %d aborted %v", response.Code, ctx.IsAborted())
	}
}

func TestPolicyHTTPHandlesRouteStatesAndShadowsPublicPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	policy := httpTestPolicy{http: map[string]authorizer.RoutePolicy{
		"GET /public":    {Authorization: authorizer.AuthorizationPublic},
		"GET /disabled":  {Authorization: authorizer.AuthorizationDisabled},
		"GET /protected": {Authorization: authorizer.AuthorizationPermission, Actors: []principal.IdentityType{principal.IdentityUser}, Context: authorizer.ContextOrganization, Permissions: []principal.PermissionKey{"organization.user.read"}},
	}}
	user := httpTestPrincipal(
		t, principal.Identity{Type: principal.IdentityUser, ID: 11}, principal.Tenant{OrganizationID: 8},
		[]principal.PermissionKey{"organization.user.read"},
	)
	engine := gin.New()
	engine.Use(NewPolicyHTTPMiddleware(policy, authenticationMiddlewareTestLogger))
	publicHasPrincipal := true
	protectedHasPrincipal := false
	engine.GET("/public", func(ctx *gin.Context) {
		_, publicHasPrincipal = Principal(ctx.Request.Context())
		ctx.Status(http.StatusNoContent)
	})
	engine.GET("/protected", func(ctx *gin.Context) {
		_, protectedHasPrincipal = Principal(ctx.Request.Context())
		ctx.Status(http.StatusNoContent)
	})
	engine.GET("/disabled", func(ctx *gin.Context) { t.Fatal("disabled route reached controller") })
	engine.GET("/unknown", func(ctx *gin.Context) { t.Fatal("unknown route reached controller") })

	request := httptest.NewRequest(http.MethodGet, "/public", nil)
	requestContext, err := ContextWithPrincipal(request.Context(), user)
	if err != nil {
		t.Fatal(err)
	}
	request = request.WithContext(requestContext)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || publicHasPrincipal {
		t.Fatalf("public = status %d principal %v", response.Code, publicHasPrincipal)
	}

	request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	requestContext, err = ContextWithPrincipal(request.Context(), user)
	if err != nil {
		t.Fatal(err)
	}
	request = request.WithContext(requestContext)
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !protectedHasPrincipal {
		t.Fatalf("protected = status %d principal %v", response.Code, protectedHasPrincipal)
	}

	for path, want := range map[string]int{"/protected": http.StatusUnauthorized, "/disabled": http.StatusServiceUnavailable, "/unknown": http.StatusForbidden} {
		response = httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != want {
			t.Fatalf("%s status = %d, want %d", path, response.Code, want)
		}
	}

	response = httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	NewPolicyHTTPMiddleware(nil, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusForbidden || !ctx.IsAborted() {
		t.Fatalf("nil policy = status %d aborted %v", response.Code, ctx.IsAborted())
	}
}

func TestHTTPAuthenticationMapsResolverFailureToUnauthorized(t *testing.T) {
	source := &httpTestProjectSource{err: errors.New("backend unavailable")}
	projectResolver := resolver.NewProjectPrincipalResolver(source)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(types.PROJECT_SCOPE_KEY, "rpd-prj-project")
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = request
	NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusUnauthorized || !ctx.IsAborted() {
		t.Fatalf("status = %d aborted %v", response.Code, ctx.IsAborted())
	}
}

func TestHTTPFamilyFailureContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, family := range httpTestFamilies(t, false) {
		family := family
		t.Run(family.name, func(t *testing.T) {
			t.Run("absent", func(t *testing.T) {
				before := family.calls()
				controllerCalls := 0
				engine := gin.New()
				engine.Use(family.handler)
				engine.GET("/", func(ctx *gin.Context) {
					controllerCalls++
					ctx.Status(http.StatusNoContent)
				})
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
				if response.Code != http.StatusNoContent || controllerCalls != 1 || (before >= 0 && family.calls() != before) {
					t.Fatalf("absent = status %d controller calls %d resolver calls %d to %d", response.Code, controllerCalls, before, family.calls())
				}
			})

			for name, addCredential := range map[string]func(*http.Request){"malformed": family.malformed, "duplicate": family.duplicate} {
				name, addCredential := name, addCredential
				t.Run(name, func(t *testing.T) {
					before := family.calls()
					response := httptest.NewRecorder()
					ctx, _ := gin.CreateTestContext(response)
					ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
					addCredential(ctx.Request)
					family.handler(ctx)
					if response.Code != http.StatusUnauthorized || !ctx.IsAborted() || (before >= 0 && family.calls() != before) {
						t.Fatalf("%s = status %d aborted %v calls %d to %d", name, response.Code, ctx.IsAborted(), before, family.calls())
					}
				})
			}

			for name, requestContext := range map[string]context.Context{
				"canceled": func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				"deadline": func() context.Context {
					ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
					defer cancel()
					return ctx
				}(),
			} {
				name, requestContext := name, requestContext
				t.Run(name, func(t *testing.T) {
					before := family.calls()
					response := httptest.NewRecorder()
					ctx, _ := gin.CreateTestContext(response)
					ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestContext)
					family.handler(ctx)
					if !ctx.IsAborted() || response.Body.Len() != 0 || (before >= 0 && family.calls() != before) {
						t.Fatalf("%s = aborted %v body %q calls %d to %d", name, ctx.IsAborted(), response.Body.String(), before, family.calls())
					}
				})
			}
		})
	}

	for _, family := range httpTestFamilies(t, true) {
		family := family
		t.Run(family.name+" rejected", func(t *testing.T) {
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			family.valid(ctx.Request)
			family.handler(ctx)
			if response.Code != http.StatusUnauthorized || !ctx.IsAborted() {
				t.Fatalf("rejected = status %d aborted %v", response.Code, ctx.IsAborted())
			}
		})
	}
}

func TestHTTPCancellationPrecedesCredentialInspection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := httpTestPrincipal(t, principal.Identity{Type: principal.IdentityUser, ID: 11}, principal.Tenant{OrganizationID: 8}, nil)
	for _, contextName := range []string{"canceled", "deadline"} {
		contextName := contextName
		for _, shape := range []string{"valid", "malformed", "duplicate", "conflicting", "absent"} {
			shape := shape
			t.Run(contextName+" "+shape, func(t *testing.T) {
				source := &httpTestProjectSource{}
				projectResolver := resolver.NewProjectPrincipalResolver(source)
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				switch shape {
				case "valid":
					request.Header.Set(types.PROJECT_SCOPE_KEY, "rpd-prj-project")
				case "malformed":
					request.Header.Set(types.PROJECT_SCOPE_KEY, " ")
				case "duplicate":
					request.Header[http.CanonicalHeaderKey(types.PROJECT_SCOPE_KEY)] = []string{"one", "two"}
				case "conflicting":
					request.Header.Set(types.PROJECT_SCOPE_KEY, "rpd-prj-project")
					attachedContext, err := ContextWithPrincipal(request.Context(), user)
					if err != nil {
						t.Fatal(err)
					}
					request = request.WithContext(attachedContext)
				}
				if contextName == "canceled" {
					requestContext, cancel := context.WithCancel(request.Context())
					cancel()
					request = request.WithContext(requestContext)
				} else {
					requestContext, cancel := context.WithDeadline(request.Context(), time.Now().Add(-time.Second))
					defer cancel()
					request = request.WithContext(requestContext)
				}
				response := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(response)
				ctx.Request = request
				NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger)(ctx)
				if !ctx.IsAborted() || response.Body.Len() != 0 || source.calls != 0 {
					t.Fatalf("aborted %v body %q calls %d", ctx.IsAborted(), response.Body.String(), source.calls)
				}
			})
		}
	}
}

func TestUserHTTPRejectsBlankPresentedProjectSelectionAndAllowsFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSource := &httpTestUserSource{}
	userResolver := resolver.NewUserPrincipalResolver(userSource)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(types.AUTHORIZATION_KEY, "session-token")
	request.Header.Set(types.AUTH_KEY, "11")
	request.Header.Set(types.PROJECT_KEY, " ")
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = request
	NewUserHTTPMiddleware(userResolver, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusUnauthorized || !ctx.IsAborted() || userSource.calls != 0 {
		t.Fatalf("blank project = status %d aborted %v calls %d", response.Code, ctx.IsAborted(), userSource.calls)
	}

	request = httptest.NewRequest(http.MethodGet, "/?x-project-id=9", nil)
	request.Header.Set(types.AUTHORIZATION_KEY, "session-token")
	request.Header.Set(types.AUTH_KEY, "11")
	request.Header.Set(types.PROJECT_KEY, " ")
	response = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(response)
	ctx.Request = request
	NewUserHTTPMiddleware(userResolver, authenticationMiddlewareTestLogger)(ctx)
	value, exists := Principal(ctx.Request.Context())
	if response.Code != http.StatusOK || ctx.IsAborted() || !exists || value.Tenant().ProjectID != 9 || userSource.projectID != 9 {
		t.Fatalf("fallback = status %d aborted %v principal %v tenant %+v", response.Code, ctx.IsAborted(), exists, value.Tenant())
	}
}

func TestUserHTTPUsesOnlyTheOrganizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userSource := &httpTestUserSource{}
	userResolver := resolver.NewUserPrincipalResolver(userSource)

	request := httptest.NewRequest(http.MethodGet, "/?x-organization-id=99", nil)
	request.Header.Set(types.AUTHORIZATION_KEY, "session-token")
	request.Header.Set(types.AUTH_KEY, "11")
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = request
	ctx.Params = gin.Params{{Key: types.ORGANIZATION_KEY, Value: "77"}}
	NewUserHTTPMiddleware(userResolver, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusOK || ctx.IsAborted() || userSource.organizationID != 0 {
		t.Fatalf("query/path selection = status %d aborted %v organization %d", response.Code, ctx.IsAborted(), userSource.organizationID)
	}

	request = httptest.NewRequest(http.MethodGet, "/?x-organization-id=99", nil)
	request.Header.Set(types.AUTHORIZATION_KEY, "session-token")
	request.Header.Set(types.AUTH_KEY, "11")
	request.Header.Set(types.ORGANIZATION_KEY, "8")
	response = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(response)
	ctx.Request = request
	ctx.Params = gin.Params{{Key: types.ORGANIZATION_KEY, Value: "77"}}
	NewUserHTTPMiddleware(userResolver, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusOK || ctx.IsAborted() || userSource.organizationID != 8 {
		t.Fatalf("header selection = status %d aborted %v organization %d", response.Code, ctx.IsAborted(), userSource.organizationID)
	}

	for name, values := range map[string][]string{
		"blank":        {" "},
		"malformed":    {"not-an-id"},
		"out of range": {"9223372036854775808"},
		"duplicate":    {"8", "9"},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set(types.AUTHORIZATION_KEY, "session-token")
			request.Header.Set(types.AUTH_KEY, "11")
			request.Header[http.CanonicalHeaderKey(types.ORGANIZATION_KEY)] = values
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = request
			NewUserHTTPMiddleware(userResolver, authenticationMiddlewareTestLogger)(ctx)
			if response.Code != http.StatusUnauthorized || !ctx.IsAborted() {
				t.Fatalf("status = %d aborted %v", response.Code, ctx.IsAborted())
			}
		})
	}
	if userSource.calls != 2 {
		t.Fatalf("resolver calls = %d", userSource.calls)
	}
}

func TestUserHTTPRejectsOrganizationSelectionWithoutUserCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	families := httpTestFamilies(t, false)
	user := families[len(families)-1]
	type credentialCase struct {
		name     string
		handlers []gin.HandlerFunc
		prepare  func(*http.Request)
	}
	tests := []credentialCase{
		{name: "selection only", handlers: []gin.HandlerFunc{user.handler}, prepare: func(*http.Request) {}},
	}
	for _, family := range families[:len(families)-1] {
		family := family
		tests = append(tests,
			credentialCase{name: family.name + " before user", handlers: []gin.HandlerFunc{family.handler, user.handler}, prepare: family.valid},
			credentialCase{name: "user before " + family.name, handlers: []gin.HandlerFunc{user.handler, family.handler}, prepare: family.valid},
		)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controllerCalls := 0
			engine := gin.New()
			engine.Use(test.handlers...)
			engine.GET("/conflict", func(ctx *gin.Context) {
				controllerCalls++
				ctx.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/conflict", nil)
			request.Header.Set(types.ORGANIZATION_KEY, "8")
			test.prepare(request)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || controllerCalls != 0 {
				t.Fatalf("status = %d, controller calls = %d", response.Code, controllerCalls)
			}
		})
	}
}

func TestHTTPComposedFamilyMiddlewareRejectsEveryConflictOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const familyCount = 4
	for left := 0; left < familyCount; left++ {
		for right := left + 1; right < familyCount; right++ {
			for _, order := range [][2]int{{left, right}, {right, left}} {
				order := order
				families := httpTestFamilies(t, false)
				first, second := families[order[0]], families[order[1]]
				t.Run(first.name+" then "+second.name, func(t *testing.T) {
					controllerCalls := 0
					engine := gin.New()
					engine.Use(first.handler, second.handler)
					engine.GET("/conflict", func(ctx *gin.Context) {
						controllerCalls++
						ctx.Status(http.StatusNoContent)
					})
					request := httptest.NewRequest(http.MethodGet, "/conflict", nil)
					first.valid(request)
					second.valid(request)
					response := httptest.NewRecorder()
					engine.ServeHTTP(response, request)
					if response.Code != http.StatusUnauthorized || controllerCalls != 0 {
						t.Fatalf("status %d controller calls %d", response.Code, controllerCalls)
					}
					if first.calls() >= 0 && first.calls() != 1 {
						t.Fatalf("first resolver calls = %d", first.calls())
					}
					if second.calls() >= 0 && second.calls() != 0 {
						t.Fatalf("second resolver calls = %d", second.calls())
					}
				})
			}
		}
	}
}

func TestHTTPAuthenticationFailurePrecedesRoutePolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	policy := httpTestPolicy{http: map[string]authorizer.RoutePolicy{
		"GET /public":   {Authorization: authorizer.AuthorizationPublic},
		"GET /disabled": {Authorization: authorizer.AuthorizationDisabled},
	}}
	for _, path := range []string{"/public", "/disabled", "/unknown"} {
		path := path
		t.Run(path, func(t *testing.T) {
			source := &httpTestProjectSource{}
			projectResolver := resolver.NewProjectPrincipalResolver(source)
			controllerCalls := 0
			engine := gin.New()
			engine.Use(NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger), NewPolicyHTTPMiddleware(policy, authenticationMiddlewareTestLogger))
			engine.GET(path, func(ctx *gin.Context) {
				controllerCalls++
				ctx.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set(types.PROJECT_SCOPE_KEY, " ")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || controllerCalls != 0 || source.calls != 0 {
				t.Fatalf("status %d controller %d resolver %d", response.Code, controllerCalls, source.calls)
			}
		})
	}
}

func TestHTTPFactoriesRejectTypedNilDependencies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var projectResolver *resolver.ProjectPrincipalResolver
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	NewProjectHTTPMiddleware(projectResolver, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusUnauthorized || !ctx.IsAborted() {
		t.Fatalf("typed nil resolver = status %d aborted %v", response.Code, ctx.IsAborted())
	}

	var policy *httpTestPolicy
	response = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	NewPolicyHTTPMiddleware(policy, authenticationMiddlewareTestLogger)(ctx)
	if response.Code != http.StatusForbidden || !ctx.IsAborted() {
		t.Fatalf("typed nil policy = status %d aborted %v", response.Code, ctx.IsAborted())
	}
}

func TestPolicyHTTPCancellationPrecedesRouteDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	policy := httpTestPolicy{http: map[string]authorizer.RoutePolicy{"GET /public": {Authorization: authorizer.AuthorizationPublic}}}
	for _, requestContext := range []context.Context{
		func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
		func() context.Context {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			return ctx
		}(),
	} {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/public", nil).WithContext(requestContext)
		NewPolicyHTTPMiddleware(policy, authenticationMiddlewareTestLogger)(ctx)
		if !ctx.IsAborted() || response.Body.Len() != 0 {
			t.Fatalf("aborted %v body %q", ctx.IsAborted(), response.Body.String())
		}
	}
}
