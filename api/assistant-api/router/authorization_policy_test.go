// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_router

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/pkg/authentication/policy"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
)

func TestAssistantAuthorizationPolicyExactlyMatchesRegisteredRoutes(t *testing.T) {
	for _, withKnowledge := range []bool{false, true} {
		name := "without OpenSearch"
		if withKnowledge {
			name = "with OpenSearch"
		}
		t.Run(name, func(t *testing.T) {
			server := grpc.NewServer()
			engine := gin.New()
			configuration := &assistant_config.AssistantConfig{}
			logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("assistant-policy-test"))
			if err != nil {
				t.Fatal(err)
			}
			clients := &rapida_client.RapidaClient{}
			HealthCheckRoutes(configuration, engine, logger, nil)
			AssistantApiRoute(configuration, server, engine, logger, nil, nil, nil)
			AssistantDeploymentApiRoute(configuration, server, engine, logger, nil)
			AssistantConversationApiRoute(configuration, server, logger, nil, nil, nil, nil, clients)
			TalkApiRoute(configuration, engine, logger, nil, nil, nil, nil, clients)
			if withKnowledge {
				KnowledgeApiRoute(configuration, server, logger, nil, nil, nil, clients)
				DocumentApiRoute(configuration, server, logger, nil, nil, nil, clients)
			}

			authorizationPolicy, err := policy.NewAssistantAuthorizationPolicy()
			if err != nil {
				t.Fatal(err)
			}
			registeredGRPC := map[string]struct{}{}
			for service, information := range server.GetServiceInfo() {
				for _, method := range information.Methods {
					registeredGRPC["/"+service+"/"+method.Name] = struct{}{}
				}
			}
			expectedGRPC := make([]string, 0, len(authorizationPolicy.GRPCRoutes()))
			for _, route := range authorizationPolicy.GRPCRoutes() {
				isKnowledgeRoute := strings.HasPrefix(route, "/knowledge_api.KnowledgeService/") ||
					strings.HasPrefix(route, "/document_api.DocumentService/")
				if withKnowledge || !isKnowledgeRoute {
					expectedGRPC = append(expectedGRPC, route)
				}
			}
			if len(expectedGRPC) != len(registeredGRPC) {
				t.Errorf("gRPC policy routes = %d, registered routes = %d", len(expectedGRPC), len(registeredGRPC))
			}
			for _, route := range expectedGRPC {
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
		})
	}
}
