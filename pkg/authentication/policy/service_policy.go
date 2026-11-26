// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package policy

import (
	"bytes"
	_ "embed"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rapidaai/pkg/authentication/authorizer"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed v1/web-api.yaml
var webAPIV1 []byte

//go:embed v1/assistant-api.yaml
var assistantAPIV1 []byte

//go:embed v1/endpoint-api.yaml
var endpointAPIV1 []byte

//go:embed v1/integration-api.yaml
var integrationAPIV1 []byte

// ServicePolicy is one validated authorization policy owned by a service.
type ServicePolicy struct {
	version    uint64
	deployable string
	grpc       map[string]authorizer.RoutePolicy
	http       map[string]authorizer.RoutePolicy
}

// NewWebAuthorizationPolicy loads the embedded Web API V1 authorization policy.
func NewWebAuthorizationPolicy() (*ServicePolicy, error) {
	return newAuthorizationPolicy(webAPIV1, "web-api")
}

// NewAssistantAuthorizationPolicy loads the embedded Assistant API V1 authorization policy.
func NewAssistantAuthorizationPolicy() (*ServicePolicy, error) {
	return newAuthorizationPolicy(assistantAPIV1, "assistant-api")
}

// NewEndpointAuthorizationPolicy loads the embedded Endpoint API V1 authorization policy.
func NewEndpointAuthorizationPolicy() (*ServicePolicy, error) {
	return newAuthorizationPolicy(endpointAPIV1, "endpoint-api")
}

// NewIntegrationAuthorizationPolicy loads the embedded Integration API V1 authorization policy.
func NewIntegrationAuthorizationPolicy() (*ServicePolicy, error) {
	return newAuthorizationPolicy(integrationAPIV1, "integration-api")
}

// newAuthorizationPolicy is the single strict parser for embedded deployable policies.
func newAuthorizationPolicy(source []byte, deployable string) (*ServicePolicy, error) {
	var root yaml.Node
	nodeDecoder := yaml.NewDecoder(bytes.NewReader(source))
	if err := nodeDecoder.Decode(&root); err != nil || len(root.Content) != 1 {
		return nil, pkg_errors.AuthenticationPolicyInvalidError{}
	}
	nodes := []*yaml.Node{root.Content[0]}
	for len(nodes) > 0 {
		index := len(nodes) - 1
		node := nodes[index]
		nodes = nodes[:index]
		if node == nil || node.Kind == yaml.AliasNode || node.Anchor != "" {
			return nil, pkg_errors.AuthenticationPolicyInvalidError{}
		}
		if node.Kind == yaml.MappingNode {
			keys := make(map[string]struct{}, len(node.Content)/2)
			for index := 0; index < len(node.Content); index += 2 {
				if index+1 >= len(node.Content) || node.Content[index].Value == "<<" {
					return nil, pkg_errors.AuthenticationPolicyInvalidError{}
				}
				key := node.Content[index].Value
				if _, exists := keys[key]; exists {
					return nil, pkg_errors.AuthenticationPolicyInvalidError{}
				}
				keys[key] = struct{}{}
			}
		}
		nodes = append(nodes, node.Content...)
	}
	var trailing yaml.Node
	if err := nodeDecoder.Decode(&trailing); err != io.EOF {
		return nil, pkg_errors.AuthenticationPolicyInvalidError{}
	}

	parsed := struct {
		Version    uint64                            `yaml:"version"`
		Deployable string                            `yaml:"deployable"`
		GRPC       map[string]authorizer.RoutePolicy `yaml:"grpc"`
		HTTP       map[string]authorizer.RoutePolicy `yaml:"http"`
	}{}
	decoder := yaml.NewDecoder(bytes.NewReader(source))
	decoder.KnownFields(true)
	if err := decoder.Decode(&parsed); err != nil {
		return nil, pkg_errors.AuthenticationPolicyInvalidError{}
	}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, pkg_errors.AuthenticationPolicyInvalidError{}
	}
	if parsed.Version != 1 || parsed.Deployable != deployable || len(parsed.GRPC) == 0 || len(parsed.HTTP) == 0 {
		return nil, pkg_errors.AuthenticationPolicyInvalidError{}
	}
	servicePolicy := &ServicePolicy{
		version:    parsed.Version,
		deployable: parsed.Deployable,
		grpc:       make(map[string]authorizer.RoutePolicy, len(parsed.GRPC)),
		http:       make(map[string]authorizer.RoutePolicy, len(parsed.HTTP)),
	}
	for route, routePolicy := range parsed.GRPC {
		parts := strings.Split(strings.TrimPrefix(route, "/"), "/")
		if !utf8.ValidString(route) || len(parts) != 2 || parts[0] == "" || parts[1] == "" ||
			strings.TrimSpace(route) != route || !strings.HasPrefix(route, "/") || strings.ContainsRune(route, '\x00') ||
			routePolicy.Validate() != nil {
			return nil, pkg_errors.AuthenticationPolicyInvalidError{}
		}
		servicePolicy.grpc[route] = routePolicy.Snapshot()
	}
	for route, routePolicy := range parsed.HTTP {
		parts := strings.SplitN(route, " ", 2)
		allowedMethod := parts[0] == "DELETE" || parts[0] == "GET" || parts[0] == "HEAD" || parts[0] == "OPTIONS" ||
			parts[0] == "PATCH" || parts[0] == "POST" || parts[0] == "PUT"
		if !utf8.ValidString(route) || len(parts) != 2 || !allowedMethod ||
			!strings.HasPrefix(parts[1], "/") || strings.TrimSpace(route) != route || strings.ContainsRune(route, '\x00') ||
			routePolicy.Validate() != nil {
			return nil, pkg_errors.AuthenticationPolicyInvalidError{}
		}
		servicePolicy.http[route] = routePolicy.Snapshot()
	}
	return servicePolicy, nil
}

// Version identifies the embedded policy schema.
func (policy ServicePolicy) Version() uint64 {
	return policy.version
}

// Deployable identifies the service that owns this route inventory.
func (policy ServicePolicy) Deployable() string {
	return policy.deployable
}

// GRPCPolicy returns a detached policy for one fully qualified gRPC method.
func (policy ServicePolicy) GRPCPolicy(route string) (authorizer.RoutePolicy, bool) {
	routePolicy, exists := policy.grpc[route]
	return routePolicy.Snapshot(), exists
}

// HTTPPolicy returns a detached policy for one HTTP method and path.
func (policy ServicePolicy) HTTPPolicy(route string) (authorizer.RoutePolicy, bool) {
	routePolicy, exists := policy.http[route]
	return routePolicy.Snapshot(), exists
}

// GRPCRoutes returns the sorted gRPC route inventory owned by this service policy.
func (policy ServicePolicy) GRPCRoutes() []string {
	routes := make([]string, 0, len(policy.grpc))
	for route := range policy.grpc {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	return routes
}

// HTTPRoutes returns the sorted HTTP route inventory owned by this service policy.
func (policy ServicePolicy) HTTPRoutes() []string {
	routes := make([]string, 0, len(policy.http))
	for route := range policy.http {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	return routes
}
