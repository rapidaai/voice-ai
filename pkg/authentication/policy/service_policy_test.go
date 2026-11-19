// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package policy

import (
	"errors"
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/authorizer"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

func TestNewWebAuthorizationPolicy(t *testing.T) {
	servicePolicy, err := NewWebAuthorizationPolicy()
	if err != nil {
		t.Fatalf("NewWebAuthorizationPolicy() error = %v", err)
	}
	if servicePolicy.Version() != 1 || servicePolicy.Deployable() != "web-api" {
		t.Fatalf("service policy identity = (%d, %q)", servicePolicy.Version(), servicePolicy.Deployable())
	}
	policy, exists := servicePolicy.GRPCPolicy("/vault_api.VaultService/GetAllOrganizationCredential")
	if !exists || policy.Authorization != authorizer.AuthorizationAnyOf || len(policy.AnyOf) != 2 {
		t.Fatalf("Vault policy = %#v, exists = %v", policy, exists)
	}

	policy.AnyOf[0].Actors[0] = authentication.IdentityService
	again, _ := servicePolicy.GRPCPolicy("/vault_api.VaultService/GetAllOrganizationCredential")
	if again.AnyOf[0].Actors[0] == authentication.IdentityService {
		t.Fatal("GRPC() exposed mutable service policy state")
	}
}

func TestNewWebAuthorizationPolicyRouteAlternatives(t *testing.T) {
	servicePolicy, err := NewWebAuthorizationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"/vault_api.VaultService/GetAllOrganizationCredential": 2,
		"/web_api.AccessManagementService/ListPermissions":     2,
		"/web_api.AccessManagementService/ListRoles":           2,
		"/web_api.AccessManagementService/GetRole":             2,
		"/web_api.ProjectService/RevokeProjectCredential":      2,
	}
	for route, alternatives := range want {
		routePolicy, exists := servicePolicy.GRPCPolicy(route)
		if !exists {
			t.Fatalf("missing route %s", route)
		}
		if routePolicy.Authorization != authorizer.AuthorizationAnyOf || len(routePolicy.AnyOf) != alternatives {
			t.Fatalf("%s policy = %#v, want %d alternatives", route, routePolicy, alternatives)
		}
	}
	roles, _ := servicePolicy.GRPCPolicy("/web_api.AccessManagementService/ListRoles")
	for _, alternative := range roles.AnyOf {
		if alternative.Authenticated {
			t.Fatal("ListRoles retained authenticated-only selector alternative")
		}
	}
	create, exists := servicePolicy.GRPCPolicy("/web_api.ProjectService/CreateProjectCredential")
	if !exists || create.Authorization != authorizer.AuthorizationDisabled {
		t.Fatalf("CreateProjectCredential policy = %+v, exists = %v", create, exists)
	}
}

func TestNewWebAuthorizationPolicyUsesGenericInternalServiceCaller(t *testing.T) {
	servicePolicy, err := NewWebAuthorizationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{
		"/billing_api.ProductUsageService/CreateProductUsage",
		"/web_api.AuthenticationService/VerifyToken",
	} {
		routePolicy, exists := servicePolicy.GRPCPolicy(route)
		if !exists || len(routePolicy.CallerServices) != 1 ||
			routePolicy.CallerServices[0] != (authorizer.CallerServicePolicy{Issuer: "internal", ActorID: 1}) {
			t.Fatalf("%s caller services = %#v, exists = %v", route, routePolicy.CallerServices, exists)
		}
	}
	for _, route := range []string{
		"/authentication.AuthenticationResolverService/ResolveOrganizationCredential",
		"/authentication.AuthenticationResolverService/ResolveProjectCredential",
		"/authentication.AuthenticationResolverService/ResolveUserCredential",
	} {
		if _, exists := servicePolicy.GRPCPolicy(route); exists {
			t.Fatalf("removed resolver route %s remains in policy", route)
		}
	}
}

func TestNewAuthorizationPoliciesOwnTheirDeployable(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() (*ServicePolicy, error)
		deployable  string
		grpcRoutes  int
		httpRoutes  int
	}{
		{name: "web", constructor: NewWebAuthorizationPolicy, deployable: "web-api", grpcRoutes: 138, httpRoutes: 18},
		{name: "assistant", constructor: NewAssistantAuthorizationPolicy, deployable: "assistant-api", grpcRoutes: 78, httpRoutes: 32},
		{name: "endpoint", constructor: NewEndpointAuthorizationPolicy, deployable: "endpoint-api", grpcRoutes: 16, httpRoutes: 2},
		{name: "integration", constructor: NewIntegrationAuthorizationPolicy, deployable: "integration-api", grpcRoutes: 8, httpRoutes: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			servicePolicy, err := test.constructor()
			if err != nil {
				t.Fatal(err)
			}
			if servicePolicy.Version() != 1 || servicePolicy.Deployable() != test.deployable {
				t.Fatalf("policy identity = (%d, %q)", servicePolicy.Version(), servicePolicy.Deployable())
			}
			if routes := servicePolicy.GRPCRoutes(); len(routes) != test.grpcRoutes {
				t.Fatalf("gRPC routes = %d, want %d", len(routes), test.grpcRoutes)
			}
			if routes := servicePolicy.HTTPRoutes(); len(routes) != test.httpRoutes {
				t.Fatalf("HTTP routes = %d, want %d", len(routes), test.httpRoutes)
			}
		})
	}
}

func TestServicePolicyExposesNoMutableState(t *testing.T) {
	policyType := reflect.TypeOf(ServicePolicy{})
	for index := 0; index < policyType.NumField(); index++ {
		field := policyType.Field(index)
		if field.IsExported() {
			t.Errorf("ServicePolicy field %q is exported", field.Name)
		}
	}
}

func TestDownstreamAuthorizationPoliciesPreserveDelegatedProjectCallers(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() (*ServicePolicy, error)
		route       string
	}{
		{name: "assistant", constructor: NewAssistantAuthorizationPolicy, route: "/assistant_api.AssistantService/GetAssistant"},
		{name: "endpoint", constructor: NewEndpointAuthorizationPolicy, route: "/endpoint_api.EndpointService/GetEndpoint"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			servicePolicy, err := test.constructor()
			if err != nil {
				t.Fatal(err)
			}
			routePolicy, exists := servicePolicy.GRPCPolicy(test.route)
			if !exists || routePolicy.Authorization != authorizer.AuthorizationAnyOf || len(routePolicy.AnyOf) != 3 {
				t.Fatalf("route policy = %#v, exists = %v", routePolicy, exists)
			}
			directProject := routePolicy.AnyOf[1]
			if directProject.ProjectToken == nil || directProject.ProjectToken.Audience != authorizer.ProjectTokenAudienceManagement || len(directProject.CallerServices) != 0 {
				t.Fatalf("direct Project branch = %#v", directProject)
			}
			delegatedProject := routePolicy.AnyOf[2]
			if delegatedProject.ProjectToken == nil || delegatedProject.ProjectToken.Audience != authorizer.ProjectTokenAudienceRuntime ||
				len(delegatedProject.CallerServices) != 1 || delegatedProject.CallerServices[0] != (authorizer.CallerServicePolicy{Issuer: "internal", ActorID: 1}) {
				t.Fatalf("delegated Project branch = %#v", delegatedProject)
			}
		})
	}
}

func TestAssistantAuthorizationPolicyDisablesUnboundTalkRoutes(t *testing.T) {
	servicePolicy, err := NewAssistantAuthorizationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{
		"/assistant_api.AssistantService/GetAllConversationMessage",
		"/talk_api.TalkService/CreateConversationMetric",
		"/talk_api.TalkService/CreateMessageMetric",
		"/talk_api.TalkService/GetAllConversationMessage",
	} {
		routePolicy, exists := servicePolicy.GRPCPolicy(route)
		if !exists || routePolicy.Authorization != authorizer.AuthorizationDisabled {
			t.Errorf("%s policy = %#v, exists = %v", route, routePolicy, exists)
		}
	}
	for _, route := range []string{
		"GET /v1/talk/:telephony/event/:assistantId",
		"POST /v1/talk/:telephony/event/:assistantId",
		"GET /v1/talk/:telephony/call/:assistantId",
		"POST /v1/talk/:telephony/call/:assistantId",
		"GET /v1/talk/:telephony/ctx/:contextId",
		"GET /v1/talk/:telephony/ctx/:contextId/event",
		"POST /v1/talk/:telephony/ctx/:contextId/event",
	} {
		routePolicy, exists := servicePolicy.HTTPPolicy(route)
		if !exists || routePolicy.Authorization != authorizer.AuthorizationDisabled {
			t.Errorf("%s policy = %#v, exists = %v", route, routePolicy, exists)
		}
	}
}

func TestIntegrationAuthorizationPolicyDisablesProviderExecution(t *testing.T) {
	servicePolicy, err := NewIntegrationAuthorizationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{
		"/integration_api.UnifiedProviderService/Chat",
		"/integration_api.UnifiedProviderService/Embedding",
		"/integration_api.UnifiedProviderService/Reranking",
		"/integration_api.UnifiedProviderService/StreamChat",
		"/integration_api.UnifiedProviderService/VerifyCredential",
	} {
		routePolicy, exists := servicePolicy.GRPCPolicy(route)
		if !exists || routePolicy.Authorization != authorizer.AuthorizationDisabled {
			t.Errorf("%s policy = %#v, exists = %v", route, routePolicy, exists)
		}
	}
	audit, exists := servicePolicy.GRPCPolicy("/integration_api.AuditLoggingService/GetAuditLog")
	if !exists || audit.Authorization != authorizer.AuthorizationPermission || len(audit.CallerServices) != 1 ||
		audit.CallerServices[0] != (authorizer.CallerServicePolicy{Issuer: "internal", ActorID: 1}) {
		t.Fatalf("audit policy = %#v, exists = %v", audit, exists)
	}
}

func TestAuthorizationPolicyRejectsMismatchedDeployable(t *testing.T) {
	original := assistantAPIV1
	t.Cleanup(func() { assistantAPIV1 = original })
	assistantAPIV1 = webAPIV1

	_, err := NewAssistantAuthorizationPolicy()
	var invalid pkg_errors.AuthenticationPolicyInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("NewAssistantAuthorizationPolicy() error = %v", err)
	}
}

func TestNewWebAuthorizationPolicyRejectsUnsafePolicy(t *testing.T) {
	original := webAPIV1
	t.Cleanup(func() { webAPIV1 = original })
	tests := map[string]string{
		"unknown field": "version: 1\ndeployable: web-api\ngrpc: {}\nhttp: {}\nextra: true\n",
		"duplicate key": "version: 1\nversion: 1\ndeployable: web-api\ngrpc: {}\nhttp: {}\n",
		"alias":         "version: 1\ndeployable: web-api\ngrpc: &routes {}\nhttp: *routes\n",
		"trailing":      "version: 1\ndeployable: web-api\ngrpc: {}\nhttp: {}\n---\nversion: 1\n",
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			webAPIV1 = []byte(source)
			_, err := NewWebAuthorizationPolicy()
			var invalid pkg_errors.AuthenticationPolicyInvalidError
			if !errors.As(err, &invalid) {
				t.Fatalf("NewWebAuthorizationPolicy() error = %v", err)
			}
		})
	}
}
