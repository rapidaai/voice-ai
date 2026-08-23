package web_client

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	web_api "github.com/rapidaai/protos"
)

type scopeAuthInternalClient struct {
	clients.InternalClient
	retrieved *connectors.RedisResponse
	cacheKey  string
	ttl       time.Duration
}

func (c *scopeAuthInternalClient) CacheKey(_ context.Context, functionName string, values ...string) string {
	c.cacheKey = functionName + ":" + strings.Join(values, ":")
	return c.cacheKey
}
func (c *scopeAuthInternalClient) Retrieve(context.Context, string) *connectors.RedisResponse {
	return c.retrieved
}
func (c *scopeAuthInternalClient) CacheWithTTL(_ context.Context, _ string, _ interface{}, ttl time.Duration) *connectors.RedisResponse {
	c.ttl = ttl
	return &connectors.RedisResponse{}
}
func (c *scopeAuthInternalClient) WithScopeToken(ctx context.Context, _, _ string) context.Context {
	return ctx
}

type scopeAuthGRPCClient struct {
	web_api.AuthenticationServiceClient
	response *web_api.ScopedAuthenticationResponse
	calls    int
}

func (c *scopeAuthGRPCClient) ScopeAuthorize(context.Context, *web_api.ScopeAuthorizeRequest, ...grpc.CallOption) (*web_api.ScopedAuthenticationResponse, error) {
	c.calls++
	return c.response, nil
}

func testLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	return logger
}

func TestScopeAuthorizationCacheKeyUsesVersionedHMAC(t *testing.T) {
	internal := &scopeAuthInternalClient{}
	client := &authServiceClient{InternalClient: internal, cfg: &config.AppConfig{Secret: "cache-secret"}}

	first := client.scopeAuthorizationCacheKey(context.Background(), "raw-project-key", "project")
	second := client.scopeAuthorizationCacheKey(context.Background(), "different-key", "project")
	third := client.scopeAuthorizationCacheKey(context.Background(), "raw-project-key", "organization")

	if strings.Contains(first, "raw-project-key") {
		t.Fatalf("cache key exposes raw token: %q", first)
	}
	if !strings.Contains(first, scopeAuthorizationCacheVersion) {
		t.Fatalf("cache key lacks version: %q", first)
	}
	if first == second || first == third {
		t.Fatalf("cache keys are not domain separated: %q %q %q", first, second, third)
	}
}

func TestScopeAuthorizeTreatsLegacyProjectCacheAsMiss(t *testing.T) {
	legacy := `{"organizationId":1,"projectId":2,"status":"ACTIVE"}`
	internal := &scopeAuthInternalClient{retrieved: &connectors.RedisResponse{Result: legacy}}
	actorType := "project"
	actorID := "42"
	grpcClient := &scopeAuthGRPCClient{response: &web_api.ScopedAuthenticationResponse{
		Success: true,
		Data: &web_api.ScopedAuthentication{
			OrganizationId: 1,
			ProjectId:      2,
			Status:         "ACTIVE",
			ActorType:      &actorType,
			ActorId:        &actorID,
		},
	}}
	client := &authServiceClient{
		InternalClient: internal,
		cfg:            &config.AppConfig{Secret: "cache-secret"},
		logger:         testLogger(t),
		authClient:     grpcClient,
	}

	result, err := client.ScopeAuthorize(context.Background(), "raw-project-key", "project")
	if err != nil {
		t.Fatalf("ScopeAuthorize() error = %v", err)
	}
	if grpcClient.calls != 1 {
		t.Fatalf("ScopeAuthorize() remote calls = %d, want 1", grpcClient.calls)
	}
	if result.GetActorId() != "42" {
		t.Fatalf("ScopeAuthorize() actor ID = %q", result.GetActorId())
	}
	if internal.ttl != scopeAuthorizationCacheTTL {
		t.Fatalf("cache TTL = %v, want %v", internal.ttl, scopeAuthorizationCacheTTL)
	}
	if strings.Contains(internal.cacheKey, "raw-project-key") {
		t.Fatalf("cache key exposes raw token: %q", internal.cacheKey)
	}
}

func TestScopeAuthorizeRejectsRemoteProjectWithoutActor(t *testing.T) {
	internal := &scopeAuthInternalClient{retrieved: &connectors.RedisResponse{Err: context.Canceled}}
	grpcClient := &scopeAuthGRPCClient{response: &web_api.ScopedAuthenticationResponse{
		Success: true,
		Data: &web_api.ScopedAuthentication{
			OrganizationId: 1,
			ProjectId:      2,
			Status:         "ACTIVE",
		},
	}}
	client := &authServiceClient{
		InternalClient: internal,
		cfg:            &config.AppConfig{Secret: "cache-secret"},
		logger:         testLogger(t),
		authClient:     grpcClient,
	}

	if _, err := client.ScopeAuthorize(context.Background(), "raw-project-key", "project"); err == nil {
		t.Fatal("ScopeAuthorize() error = nil, want missing actor error")
	}
	if internal.ttl != 0 {
		t.Fatalf("invalid response was cached with TTL %v", internal.ttl)
	}
}
