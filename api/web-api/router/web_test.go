// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	web_config "github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/authentication/policy"
	"github.com/rapidaai/pkg/commons"
)

func TestWebAPIRouteRegistersProductUsageService(t *testing.T) {
	server := grpc.NewServer()
	WebApiRoute(&web_config.WebAppConfig{}, make([]byte, 32), gin.New(), server, nil, nil, nil)

	if _, ok := server.GetServiceInfo()["billing_api.ProductUsageService"]; !ok {
		t.Fatal("ProductUsageService is not registered")
	}
	if _, ok := server.GetServiceInfo()["dashboard_api.DashboardService"]; !ok {
		t.Fatal("DashboardService is not registered")
	}
	if _, ok := server.GetServiceInfo()["web_api.AccessManagementService"]; !ok {
		t.Fatal("AccessManagementService is not registered")
	}
	if _, ok := server.GetServiceInfo()["web_api.AuthenticationService"]; !ok {
		t.Fatal("AuthenticationService is not registered")
	}
	if _, ok := server.GetServiceInfo()["authentication.AuthenticationResolverService"]; ok {
		t.Fatal("removed AuthenticationResolverService is registered")
	}
}

func TestWebAPIPolicyExactlyMatchesRegisteredRoutes(t *testing.T) {
	server := grpc.NewServer()
	engine := gin.New()
	configuration := &web_config.WebAppConfig{}
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("route-inventory-test"))
	if err != nil {
		t.Fatal(err)
	}
	HealthCheckRoutes(configuration, engine, logger, nil)
	WebApiRoute(configuration, make([]byte, 32), engine, server, logger, nil, nil)
	ProxyApiRoute(configuration, server, logger, nil, nil)

	servicePolicy, err := policy.NewWebAuthorizationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	registeredGRPC := map[string]struct{}{}
	for service, information := range server.GetServiceInfo() {
		for _, method := range information.Methods {
			registeredGRPC["/"+service+"/"+method.Name] = struct{}{}
		}
	}
	assertExactRouteInventory(t, "gRPC", servicePolicy.GRPCRoutes(), registeredGRPC)

	registeredHTTP := map[string]struct{}{}
	for _, route := range engine.Routes() {
		registeredHTTP[route.Method+" "+route.Path] = struct{}{}
	}
	assertExactRouteInventory(t, "HTTP", servicePolicy.HTTPRoutes(), registeredHTTP)
}

func assertExactRouteInventory(t *testing.T, transport string, policies []string, registered map[string]struct{}) {
	t.Helper()
	if len(policies) != len(registered) {
		t.Errorf("%s policy routes = %d, registered routes = %d", transport, len(policies), len(registered))
	}
	for _, route := range policies {
		if _, exists := registered[route]; !exists {
			t.Errorf("%s policy route %q is not registered", transport, route)
		}
		delete(registered, route)
	}
	for route := range registered {
		t.Errorf("registered %s route %q has no policy", transport, route)
	}
}
