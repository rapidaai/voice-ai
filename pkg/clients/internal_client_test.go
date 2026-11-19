package clients

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/metadata"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
)

type recordingRedisConnector struct {
	command string
	args    []string
}

func TestNewInternalClientPreservesIssuerValidationError(t *testing.T) {
	cfg := &config.AppConfig{Secret: "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="}
	for _, destination := range []string{"", " web-api", "web-api\x00"} {
		client := NewInternalClient(cfg, destination, nil, nil)
		serviceContext, serviceErr := client.WithService(context.Background())
		var invalidIssuer pkg_errors.JWTIssuerInvalidError
		if serviceContext != nil || !errors.As(serviceErr, &invalidIssuer) {
			t.Fatalf("WithService(%q) = %v, %v", destination, serviceContext, serviceErr)
		}
		delegatedContext, delegatedErr := client.WithAuth(context.Background(), authentication.Principal{})
		if delegatedContext != nil || !errors.As(delegatedErr, &invalidIssuer) {
			t.Fatalf("WithAuth(%q) = %v, %v", destination, delegatedContext, delegatedErr)
		}
	}
}

func TestNewInternalClientIssuesDestinationBoundServiceAssertion(t *testing.T) {
	cfg := &config.AppConfig{Secret: "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="}
	client := NewInternalClient(cfg, "web-api", nil, nil)
	authContext, err := client.WithService(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	outgoingMetadata, ok := metadata.FromOutgoingContext(authContext)
	assertions := outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)
	if !ok || len(assertions) != 1 {
		t.Fatalf("service assertion metadata = %v", outgoingMetadata)
	}
	verified, err := authentication_jwt.NewVerifier("web-api", cfg.ServiceAuthenticationKey()).Verify(assertions[0])
	if err != nil {
		t.Fatal(err)
	}
	if verified.Subject() != (authentication.Identity{Type: authentication.IdentityService, ID: 1, Issuer: "internal"}) {
		t.Fatalf("verified subject = %+v", verified.Subject())
	}
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

func TestInternalClientCacheRejectsUnsupportedValue(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	redisConnector := &recordingRedisConnector{}
	client := &internalClient{logger: logger, redis: redisConnector}

	if response := client.Cache(context.Background(), "cache-key", make(chan int)); response != nil {
		t.Fatalf("Cache() response = %+v, want nil", response)
	}
	if redisConnector.command != "" {
		t.Fatalf("Cache() command = %q, want no Redis call", redisConnector.command)
	}

	response := client.CacheWithTTL(context.Background(), "cache-key", make(chan int), time.Minute)
	if response == nil || !response.HasError() {
		t.Fatalf("CacheWithTTL() response = %+v, want marshal error", response)
	}
	if redisConnector.command != "" {
		t.Fatalf("CacheWithTTL() command = %q, want no Redis call", redisConnector.command)
	}
}

func TestInternalClientWithAuthAddsServiceAssertion(t *testing.T) {
	client, verifier := serviceAssertionClient(t, "web-api")
	principal := userPrincipal(t, authentication.Tenant{OrganizationID: 7, ProjectID: 11})
	authContext, err := client.WithAuth(context.Background(), principal)
	if err != nil {
		t.Fatalf("WithAuth() error = %v", err)
	}
	outgoingMetadata, ok := metadata.FromOutgoingContext(authContext)
	if !ok || len(outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)) != 1 {
		t.Fatalf("WithAuth() metadata = %v, %v", outgoingMetadata, ok)
	}
	verified, err := verifier.Verify(outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)[0])
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if verified.Subject() != principal.Subject() || verified.Tenant() != principal.Tenant() {
		t.Fatalf("verified subject/tenant = %+v/%+v", verified.Subject(), verified.Tenant())
	}
	if verified.Actor() != (authentication.ServiceActor{ID: 1, Issuer: "internal"}) {
		t.Fatalf("Actor() = %+v", verified.Actor())
	}
}

func TestInternalClientAuthenticationPathsUseDestinationBoundAssertion(t *testing.T) {
	client, verifier := serviceAssertionClient(t, "document-api")
	principal := organizationPrincipal(t)
	platformContext, err := client.WithPlatform(context.Background(), principal)
	if err != nil {
		t.Fatalf("WithPlatform() error = %v", err)
	}
	platformMetadata, _ := metadata.FromOutgoingContext(platformContext)
	if _, err := verifier.Verify(platformMetadata.Get(types.SERVICE_SCOPE_KEY)[0]); err != nil {
		t.Fatalf("platform assertion error = %v", err)
	}

	request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	request, err = client.WithHttpAuth(context.Background(), principal, request)
	if err != nil {
		t.Fatalf("WithHttpAuth() error = %v", err)
	}
	assertion := request.Header.Get(types.SERVICE_SCOPE_KEY)
	if request.Header.Get("Authorization") != "" || assertion == "" {
		t.Fatalf("service/authorization headers = %q/%q", assertion, request.Header.Get("Authorization"))
	}
	if _, err := verifier.Verify(assertion); err != nil {
		t.Fatalf("HTTP assertion error = %v", err)
	}
}

func TestInternalClientRejectsInvalidPrincipalWithoutMutatingRequest(t *testing.T) {
	client, _ := serviceAssertionClient(t, "document-api")
	if authContext, err := client.WithPlatform(context.Background(), authentication.Principal{}); err == nil || authContext != nil {
		t.Fatalf("WithPlatform() = %v, %v", authContext, err)
	}
	request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	authenticatedRequest, err := client.WithHttpAuth(context.Background(), authentication.Principal{}, request)
	if err == nil || authenticatedRequest != nil {
		t.Fatalf("WithHttpAuth() = %v, %v", authenticatedRequest, err)
	}
	if request.Header.Get(types.SERVICE_SCOPE_KEY) != "" {
		t.Fatal("WithHttpAuth() mutated request on failure")
	}
}

func TestInternalClientAllowsTenantlessDirectService(t *testing.T) {
	client, verifier := serviceAssertionClient(t, "web-api")
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: 41, Issuer: "assistant-api"},
		authentication.Tenant{},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "local-key"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	token, err := client.createServiceAssertion(principal)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := verifier.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Subject() != (authentication.Identity{Type: authentication.IdentityService, ID: 1, Issuer: "internal"}) ||
		verified.Tenant() != (authentication.Tenant{}) {
		t.Fatalf("verified = %+v, %+v", verified.Subject(), verified.Tenant())
	}
}

func serviceAssertionClient(t *testing.T, destination string) (*internalClient, authentication_jwt.Verifier) {
	t.Helper()
	key := bytes.Repeat([]byte{1}, sha256.Size)
	issuer, err := authentication_jwt.NewIssuer(destination, key)
	if err != nil {
		t.Fatal(err)
	}
	verifier := authentication_jwt.NewVerifier(destination, key)
	return NewInternalClientWithIssuer(issuer, nil, nil).(*internalClient), verifier
}

func TestInternalClientWithPlatformReturnsAuthError(t *testing.T) {
	client, _ := serviceAssertionClient(t, "web-api")
	authContext, err := client.WithPlatform(context.Background(), authentication.Principal{})
	if err == nil || authContext != nil {
		t.Fatalf("WithPlatform() = %v, %v", authContext, err)
	}
}

func TestInternalClientWithServiceIssuesDirectTenantlessAssertion(t *testing.T) {
	client, verifier := serviceAssertionClient(t, "web-api")
	ctx, err := client.WithService(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	metadataValue, ok := metadata.FromOutgoingContext(ctx)
	if !ok || len(metadataValue.Get(types.SERVICE_SCOPE_KEY)) != 1 {
		t.Fatalf("service assertion metadata = %v", metadataValue)
	}
	verified, err := verifier.Verify(metadataValue.Get(types.SERVICE_SCOPE_KEY)[0])
	if err != nil {
		t.Fatal(err)
	}
	if verified.Subject() != (authentication.Identity{Type: authentication.IdentityService, ID: 1, Issuer: "internal"}) ||
		verified.Tenant() != (authentication.Tenant{}) {
		t.Fatalf("verified service assertion = %+v", verified)
	}
}

func userPrincipal(t *testing.T, tenant authentication.Tenant) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 5},
		tenant,
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "session-1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func projectPrincipal(t *testing.T) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 6},
		authentication.Tenant{OrganizationID: 7, ProjectID: 11},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "6"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func organizationPrincipal(t *testing.T) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 9},
		authentication.Tenant{OrganizationID: 7},
		authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "9"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func servicePrincipal(t *testing.T, id uint64, issuer string) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: id, Issuer: issuer},
		authentication.Tenant{OrganizationID: 7},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "key-1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
