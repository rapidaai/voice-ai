package clients

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/metadata"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type recordingRedisConnector struct {
	command string
	args    []string
}

func (r *recordingRedisConnector) Connect(context.Context) error    { return nil }
func (r *recordingRedisConnector) Name() string                     { return "recording" }
func (r *recordingRedisConnector) IsConnected(context.Context) bool { return true }
func (r *recordingRedisConnector) Disconnect(context.Context) error { return nil }
func (r *recordingRedisConnector) GetConnection() *redis.Client     { return nil }
func (r *recordingRedisConnector) Cmd(_ context.Context, command string, args []string) *connectors.RedisResponse {
	r.command = command
	r.args = append([]string(nil), args...)
	return &connectors.RedisResponse{}
}
func (r *recordingRedisConnector) Cmds(context.Context, string, *[]string) *connectors.RedisResponse {
	return &connectors.RedisResponse{}
}

func TestInternalClientCacheWithTTL(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	redisConnector := &recordingRedisConnector{}
	client := &internalClient{logger: logger, redis: redisConnector}

	response := client.CacheWithTTL(context.Background(), "key", map[string]string{"value": "data"}, 5*time.Minute)
	if response == nil || response.HasError() {
		t.Fatalf("CacheWithTTL() response = %+v", response)
	}
	if redisConnector.command != "SET" {
		t.Fatalf("command = %q, want SET", redisConnector.command)
	}
	if len(redisConnector.args) != 4 || redisConnector.args[0] != "key" || redisConnector.args[2] != "EX" || redisConnector.args[3] != "300" {
		t.Fatalf("args = %#v", redisConnector.args)
	}
}

func TestInternalClientCacheWithTTLRejectsNonPositiveTTL(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &internalClient{logger: logger, redis: &recordingRedisConnector{}}
	if response := client.CacheWithTTL(context.Background(), "key", "value", 0); response == nil || !response.HasError() {
		t.Fatalf("CacheWithTTL() response = %+v, want error", response)
	}
}

func TestInternalClientCreateServiceScopeToken(t *testing.T) {
	client := &internalClient{cfg: &config.AppConfig{Secret: "secret"}}
	token, err := client.createServiceScopeToken(&types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: 5},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 7, ProjectID: 11},
	})
	if err != nil {
		t.Fatalf("createServiceScopeToken() error = %v", err)
	}
	scope, err := types.ExtractServiceScope(token, "secret")
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	context, ok := scope.DelegatedContext()
	if !ok || context.OrganizationID != 7 || context.UserID == nil || *context.UserID != 5 || context.ProjectID == nil || *context.ProjectID != 11 {
		t.Fatalf("DelegatedContext() = %+v, %v", context, ok)
	}
}

func TestInternalClientCreateServiceScopeTokenRequiresOrganization(t *testing.T) {
	client := &internalClient{cfg: &config.AppConfig{Secret: "secret"}}
	_, err := client.createServiceScopeToken(&types.Authentication{
		AuthType:  types.AuthTypeUser,
		UserValue: &types.UserContext{UserID: 5},
	})
	if !errors.Is(err, types.ErrOrganizationContextUnavailable) {
		t.Fatalf("createServiceScopeToken() error = %v", err)
	}
}

func TestInternalClientCreateServiceScopeTokenRejectsMismatchedProjectOrganization(t *testing.T) {
	client := &internalClient{cfg: &config.AppConfig{Secret: "secret"}}
	_, err := client.createServiceScopeToken(&types.Authentication{
		AuthType:          types.AuthTypeProject,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 8, ProjectID: 11},
	})
	if err == nil {
		t.Fatal("createServiceScopeToken() error = nil")
	}
}

func TestInternalClientWithAuthReturnsErrorWithoutContext(t *testing.T) {
	client := &internalClient{cfg: &config.AppConfig{Secret: "secret"}}
	authContext, err := client.WithAuth(context.Background(), &types.Authentication{})
	if !errors.Is(err, types.ErrOrganizationContextUnavailable) {
		t.Fatalf("WithAuth() error = %v", err)
	}
	if authContext != nil {
		t.Fatalf("WithAuth() context = %v, want nil", authContext)
	}
}

func TestInternalClientWithAuthAddsServiceAssertion(t *testing.T) {
	client := &internalClient{cfg: &config.AppConfig{Secret: "secret"}}
	authContext, err := client.WithAuth(context.Background(), &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	})
	if err != nil {
		t.Fatalf("WithAuth() error = %v", err)
	}
	outgoingMetadata, ok := metadata.FromOutgoingContext(authContext)
	if !ok || len(outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)) != 1 {
		t.Fatalf("WithAuth() metadata = %v, %v", outgoingMetadata, ok)
	}
}

func TestInternalClientWithPlatformReturnsAuthError(t *testing.T) {
	client := &internalClient{cfg: &config.AppConfig{Secret: "secret"}}
	authContext, err := client.WithPlatform(context.Background(), &types.Authentication{})
	if !errors.Is(err, types.ErrOrganizationContextUnavailable) {
		t.Fatalf("WithPlatform() error = %v", err)
	}
	if authContext != nil {
		t.Fatalf("WithPlatform() context = %v, want nil", authContext)
	}
}

func TestInternalClientWithHttpAuthDoesNotMutateRequestOnAuthError(t *testing.T) {
	client := &internalClient{cfg: &config.AppConfig{Secret: "secret"}}
	request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	authenticatedRequest, err := client.WithHttpAuth(context.Background(), &types.Authentication{}, request)
	if !errors.Is(err, types.ErrOrganizationContextUnavailable) {
		t.Fatalf("WithHttpAuth() error = %v", err)
	}
	if authenticatedRequest != nil {
		t.Fatalf("WithHttpAuth() request = %v, want nil", authenticatedRequest)
	}
	if authorization := request.Header.Get("Authorization"); authorization != "" {
		t.Fatalf("Authorization header = %q, want empty", authorization)
	}
}
