package clients

import (
	"context"
	"errors"
	"net/http"
	"strconv"
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

func TestInternalClientWithTokenPreservesUint64UserID(t *testing.T) {
	client := &internalClient{}
	userID := ^uint64(0)
	authContext := client.WithToken(context.Background(), "token", userID)
	outgoingMetadata, ok := metadata.FromOutgoingContext(authContext)
	if !ok {
		t.Fatal("WithToken() did not attach outgoing metadata")
	}
	if authID := outgoingMetadata.Get(types.AUTH_KEY); len(authID) != 1 || authID[0] != strconv.FormatUint(userID, 10) {
		t.Fatalf("auth id = %v, want %s", authID, strconv.FormatUint(userID, 10))
	}
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
	client := actorAwareInternalClient(t)
	actor := types.ActorIdentity{Type: types.ActorTypeUser, ID: 5}
	token, err := client.createServiceScopeToken(&types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &actor,
		UserValue:         &types.UserContext{UserID: 5},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 7, ProjectID: 11},
	})
	if err != nil {
		t.Fatalf("createServiceScopeToken() error = %v", err)
	}
	scope, err := types.ExtractServiceScope(token, client.cfg.Secret)
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	context, ok := scope.DelegatedContext()
	if !ok || context.OrganizationID != 7 || context.UserID != nil || context.ProjectID == nil || *context.ProjectID != 11 {
		t.Fatalf("DelegatedContext() = %+v, %v", context, ok)
	}
	serviceActor, err := types.ResolveAuditActor(scope)
	if err != nil || serviceActor != (types.ActorIdentity{Type: types.ActorTypeService, ID: 41}) {
		t.Fatalf("ResolveAuditActor() = %+v, %v", serviceActor, err)
	}
	auth, err := scope.Authentication()
	if err != nil {
		t.Fatalf("Authentication() error = %v", err)
	}
	if effectiveActor, err := auth.Actor(); err != nil || effectiveActor != (types.ActorIdentity{Type: types.ActorTypeUser, ID: 5}) {
		t.Fatalf("Actor() = %+v, %v", effectiveActor, err)
	}
}

func TestInternalClientCreateServiceScopeTokenRequiresOrganization(t *testing.T) {
	client := actorAwareInternalClient(t)
	_, err := client.createServiceScopeToken(&types.Authentication{
		AuthType:  types.AuthTypeUser,
		UserValue: &types.UserContext{UserID: 5},
	})
	if !errors.Is(err, types.ErrOrganizationContextUnavailable) {
		t.Fatalf("createServiceScopeToken() error = %v", err)
	}
}

func TestInternalClientCreateServiceScopeTokenRejectsInvalidSourceAuthentication(t *testing.T) {
	client := actorAwareInternalClient(t)
	userActor := types.ActorIdentity{Type: types.ActorTypeUser, ID: 5}
	serviceActor := types.ActorIdentity{Type: types.ActorTypeService, ID: 6}
	tests := []struct {
		name string
		auth *types.Authentication
	}{
		{
			name: "missing user context",
			auth: &types.Authentication{AuthType: types.AuthTypeUser, ActorValue: &userActor, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}},
		},
		{
			name: "mismatched user context",
			auth: &types.Authentication{AuthType: types.AuthTypeUser, ActorValue: &userActor, UserValue: &types.UserContext{UserID: 8}, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}},
		},
		{
			name: "service actor type mismatch",
			auth: &types.Authentication{AuthType: types.AuthTypeService, ActorValue: &userActor, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}},
		},
		{
			name: "system actor type mismatch",
			auth: &types.Authentication{AuthType: types.AuthTypeSystem, ActorValue: &serviceActor, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}},
		},
		{
			name: "system actor missing",
			auth: &types.Authentication{AuthType: types.AuthTypeSystem, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := client.createServiceScopeToken(test.auth); err == nil {
				t.Fatal("createServiceScopeToken() error = nil")
			}
		})
	}
}

func TestInternalClientCreateServiceScopeTokenRejectsMismatchedProjectOrganization(t *testing.T) {
	client := actorAwareInternalClient(t)
	_, err := client.createServiceScopeToken(&types.Authentication{
		AuthType:          types.AuthTypeProject,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &types.ProjectContext{OrganizationID: 8, ProjectID: 11},
	})
	if err == nil {
		t.Fatal("createServiceScopeToken() error = nil")
	}
}

func TestInternalClientCreateServiceScopeTokenRequiresActorAndSecret(t *testing.T) {
	client := actorAwareInternalClient(t)
	authentication := &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	}
	client.cfg.ServiceID = 0
	if _, err := client.createServiceScopeToken(authentication); !errors.Is(err, types.ErrServiceActorUnavailable) {
		t.Fatalf("createServiceScopeToken() error = %v, want %v", err, types.ErrServiceActorUnavailable)
	}
	client.cfg.ServiceID = 41
	client.cfg.Secret = ""
	if _, err := client.createServiceScopeToken(authentication); !errors.Is(err, types.ErrServiceSecretUnavailable) {
		t.Fatalf("createServiceScopeToken() error = %v, want %v", err, types.ErrServiceSecretUnavailable)
	}
}

func TestInternalClientWithAuthReturnsErrorWithoutContext(t *testing.T) {
	client := actorAwareInternalClient(t)
	authContext, err := client.WithAuth(context.Background(), &types.Authentication{})
	if !errors.Is(err, types.ErrOrganizationContextUnavailable) {
		t.Fatalf("WithAuth() error = %v", err)
	}
	if authContext != nil {
		t.Fatalf("WithAuth() context = %v, want nil", authContext)
	}
}

func TestInternalClientWithAuthAddsServiceAssertion(t *testing.T) {
	client := actorAwareInternalClient(t)
	actor := types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 9}
	authContext, err := client.WithAuth(context.Background(), &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	})
	if err != nil {
		t.Fatalf("WithAuth() error = %v", err)
	}
	outgoingMetadata, ok := metadata.FromOutgoingContext(authContext)
	if !ok || len(outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)) != 1 {
		t.Fatalf("WithAuth() metadata = %v, %v", outgoingMetadata, ok)
	}
	scope, err := types.ExtractServiceScope(outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)[0], client.cfg.Secret)
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	forwardedAuth, err := scope.Authentication()
	if err != nil {
		t.Fatalf("Authentication() error = %v", err)
	}
	if forwardedActor, err := forwardedAuth.Actor(); err != nil || forwardedActor != actor {
		t.Fatalf("Actor() = %+v, %v; want %+v", forwardedActor, err, actor)
	}
}

func TestInternalClientAllAuthenticationPathsDelegateActor(t *testing.T) {
	client := actorAwareInternalClient(t)
	actor := types.ActorIdentity{Type: types.ActorTypeUser, ID: 5}
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &actor,
		UserValue:         &types.UserContext{UserID: 5},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	}
	platformContext, err := client.WithPlatform(context.Background(), auth)
	if err != nil {
		t.Fatalf("WithPlatform() error = %v", err)
	}
	platformMetadata, _ := metadata.FromOutgoingContext(platformContext)
	assertDelegatedActor(t, platformMetadata.Get(types.SERVICE_SCOPE_KEY)[0], client.cfg.Secret, actor)

	request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	request, err = client.WithHttpAuth(context.Background(), auth, request)
	if err != nil {
		t.Fatalf("WithHttpAuth() error = %v", err)
	}
	assertDelegatedActor(t, request.Header.Get("Authorization"), client.cfg.Secret, actor)
}

func TestInternalClientMultiHopAuthentication(t *testing.T) {
	projectID := uint64(11)
	tests := []struct {
		name  string
		actor types.ActorIdentity
		auth  *types.Authentication
	}{
		{
			name:  "user",
			actor: types.ActorIdentity{Type: types.ActorTypeUser, ID: 5},
			auth:  &types.Authentication{AuthType: types.AuthTypeUser, UserValue: &types.UserContext{UserID: 5}, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}, ProjectValue: &types.ProjectContext{OrganizationID: 7, ProjectID: projectID}},
		},
		{
			name:  "project",
			actor: types.ActorIdentity{Type: types.ActorTypeProject, ID: 6},
			auth:  &types.Authentication{AuthType: types.AuthTypeProject, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}, ProjectValue: &types.ProjectContext{OrganizationID: 7, ProjectID: projectID}},
		},
		{
			name:  "organization",
			actor: types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 8},
			auth:  &types.Authentication{AuthType: types.AuthTypeOrg, OrganizationValue: &types.OrganizationContext{OrganizationID: 7}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.auth.ActorValue = &test.actor
			firstClient := actorAwareInternalClient(t)
			firstToken, err := firstClient.createServiceScopeToken(test.auth)
			if err != nil {
				t.Fatal(err)
			}
			firstScope, err := types.ExtractServiceScope(firstToken, firstClient.cfg.Secret)
			if err != nil {
				t.Fatal(err)
			}
			forwardedAuth, err := firstScope.Authentication()
			if err != nil {
				t.Fatal(err)
			}
			secondClient := &internalClient{cfg: &config.AppConfig{Name: "integration-api", ServiceID: 42, Secret: "secret"}}
			secondToken, err := secondClient.createServiceScopeToken(forwardedAuth)
			if err != nil {
				t.Fatal(err)
			}
			secondScope, err := types.ExtractServiceScope(secondToken, secondClient.cfg.Secret)
			if err != nil {
				t.Fatal(err)
			}
			secondAuth, err := secondScope.Authentication()
			if err != nil {
				t.Fatal(err)
			}
			if actor, err := secondAuth.Actor(); err != nil || actor != test.actor {
				t.Fatalf("Actor() = %+v, %v; want %+v", actor, err, test.actor)
			}
			if caller, err := secondAuth.Caller(); err != nil || caller != (types.ActorIdentity{Type: types.ActorTypeService, ID: 42}) {
				t.Fatalf("Caller() = %+v, %v", caller, err)
			}
		})
	}
}

func TestInternalClientMultiHopServiceAuthenticationRotatesCaller(t *testing.T) {
	firstClient := actorAwareInternalClient(t)
	serviceActor := types.ActorIdentity{Type: types.ActorTypeService, ID: 40}
	serviceAuth := &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &serviceActor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	}
	firstToken, err := firstClient.createServiceScopeToken(serviceAuth)
	if err != nil {
		t.Fatal(err)
	}
	firstScope, err := types.ExtractServiceScope(firstToken, firstClient.cfg.Secret)
	if err != nil {
		t.Fatal(err)
	}
	reconstructedAuth, err := firstScope.Authentication()
	if err != nil {
		t.Fatal(err)
	}
	if actor, err := reconstructedAuth.Actor(); err != nil || actor != serviceActor {
		t.Fatalf("first Actor() = %+v, %v", actor, err)
	}
	if caller, err := reconstructedAuth.Caller(); err != nil || caller != (types.ActorIdentity{Type: types.ActorTypeService, ID: 41}) {
		t.Fatalf("first Caller() = %+v, %v", caller, err)
	}
	secondClient := &internalClient{cfg: &config.AppConfig{Name: "integration-api", ServiceID: 42, Secret: "secret"}}
	secondToken, err := secondClient.createServiceScopeToken(reconstructedAuth)
	if err != nil {
		t.Fatal(err)
	}
	secondScope, err := types.ExtractServiceScope(secondToken, secondClient.cfg.Secret)
	if err != nil {
		t.Fatal(err)
	}
	secondAuth, err := secondScope.Authentication()
	if err != nil {
		t.Fatal(err)
	}
	if actor, err := secondAuth.Actor(); err != nil || actor != serviceActor {
		t.Fatalf("second Actor() = %+v, %v", actor, err)
	}
	if caller, err := secondAuth.Caller(); err != nil || caller != (types.ActorIdentity{Type: types.ActorTypeService, ID: 42}) {
		t.Fatalf("second Caller() = %+v, %v", caller, err)
	}
}

func TestInternalClientSystemAuthenticationIsDelegated(t *testing.T) {
	client := actorAwareInternalClient(t)
	actor := types.ActorIdentity{Type: types.ActorTypeSystem, ID: 5}
	token, err := client.createServiceScopeToken(&types.Authentication{
		AuthType:          types.AuthTypeSystem,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertDelegatedActor(t, token, client.cfg.Secret, actor)
}

func assertDelegatedActor(t *testing.T, token, secret string, expected types.ActorIdentity) {
	t.Helper()
	scope, err := types.ExtractServiceScope(token, secret)
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	auth, err := scope.Authentication()
	if err != nil {
		t.Fatalf("Authentication() error = %v", err)
	}
	actor, err := auth.Actor()
	if err != nil || actor != expected {
		t.Fatalf("Actor() = %+v, %v; want %+v", actor, err, expected)
	}
}

func TestInternalClientWithPlatformReturnsAuthError(t *testing.T) {
	client := actorAwareInternalClient(t)
	authContext, err := client.WithPlatform(context.Background(), &types.Authentication{})
	if !errors.Is(err, types.ErrOrganizationContextUnavailable) {
		t.Fatalf("WithPlatform() error = %v", err)
	}
	if authContext != nil {
		t.Fatalf("WithPlatform() context = %v, want nil", authContext)
	}
}

func TestInternalClientWithHttpAuthDoesNotMutateRequestOnAuthError(t *testing.T) {
	client := actorAwareInternalClient(t)
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

func actorAwareInternalClient(t *testing.T) *internalClient {
	t.Helper()
	return &internalClient{cfg: &config.AppConfig{Name: "assistant-api", ServiceID: 41, Secret: "secret"}}
}
