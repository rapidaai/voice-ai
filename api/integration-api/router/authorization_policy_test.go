// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_routers

import (
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	integration_config "github.com/rapidaai/api/integration-api/config"
	"github.com/rapidaai/pkg/authentication/policy"
	"github.com/rapidaai/pkg/commons"
)

func TestIntegrationAuthorizationPolicyExactlyMatchesRegisteredRoutes(t *testing.T) {
	server := grpc.NewServer()
	engine := gin.New()
	configuration := &integration_config.IntegrationConfig{}
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("integration-policy-test"))
	if err != nil {
		t.Fatal(err)
	}
	HealthCheckRoutes(configuration, engine, logger, nil)
	ProviderApiRoute(configuration, server, logger, nil)
	AuditLoggingApiRoute(configuration, server, logger, nil)

	authorizationPolicy, err := policy.NewIntegrationAuthorizationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	registeredGRPC := map[string]struct{}{}
	for service, information := range server.GetServiceInfo() {
		for _, method := range information.Methods {
			registeredGRPC["/"+service+"/"+method.Name] = struct{}{}
		}
	}
	if len(authorizationPolicy.GRPCRoutes()) != len(registeredGRPC) {
		t.Errorf("gRPC policy routes = %d, registered routes = %d", len(authorizationPolicy.GRPCRoutes()), len(registeredGRPC))
	}
	for _, route := range authorizationPolicy.GRPCRoutes() {
		if _, exists := registeredGRPC[route]; !exists {
			t.Errorf("gRPC policy route %q is not registered", route)
		}
		delete(registeredGRPC, route)
	}
	for route := range registeredGRPC {
		t.Errorf("registered gRPC route %q has no policy", route)
	}

	registeredHTTP := map[string]struct{}{}
	for _, route := range engine.Routes() {
		registeredHTTP[route.Method+" "+route.Path] = struct{}{}
	}
	if len(authorizationPolicy.HTTPRoutes()) != len(registeredHTTP) {
		t.Errorf("HTTP policy routes = %d, registered routes = %d", len(authorizationPolicy.HTTPRoutes()), len(registeredHTTP))
	}
	for _, route := range authorizationPolicy.HTTPRoutes() {
		if _, exists := registeredHTTP[route]; !exists {
			t.Errorf("HTTP policy route %q is not registered", route)
		}
		delete(registeredHTTP, route)
	}
	for route := range registeredHTTP {
		t.Errorf("registered HTTP route %q has no policy", route)
	}
}
