// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package endpoint_router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	endpoint_config "github.com/rapidaai/api/endpoint-api/config"
	"github.com/rapidaai/pkg/authentication/policy"
	"github.com/rapidaai/pkg/commons"
)

func TestEndpointAuthorizationPolicyExactlyMatchesRegisteredRoutes(t *testing.T) {
	server := grpc.NewServer()
	engine := gin.New()
	configuration := &endpoint_config.EndpointConfig{}
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("endpoint-policy-test"))
	if err != nil {
		t.Fatal(err)
	}
	HealthCheckRoutes(configuration, engine, logger, nil)
	EndpointReaderApiRoute(configuration, server, logger, nil, nil)
	InvokeApiRoute(configuration, server, logger, nil, nil)

	authorizationPolicy, err := policy.NewEndpointAuthorizationPolicy()
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
