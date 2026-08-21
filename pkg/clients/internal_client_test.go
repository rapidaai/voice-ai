package clients

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type internalClientProjectPrinciple struct {
	organizationID uint64
	projectID      uint64
}

func (p internalClientProjectPrinciple) IsAuthenticated() bool   { return true }
func (p internalClientProjectPrinciple) GetCurrentToken() string { return "" }
func (p internalClientProjectPrinciple) Type() types.AuthType    { return types.AuthTypeProject }
func (p internalClientProjectPrinciple) OrganizationContext() (uint64, bool) {
	return p.organizationID, p.organizationID != 0
}
func (p internalClientProjectPrinciple) ProjectContext() (types.ProjectContext, bool) {
	return types.ProjectContext{OrganizationID: p.organizationID, ProjectID: p.projectID}, p.organizationID != 0 && p.projectID != 0
}

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
	token, err := client.createServiceScopeToken(internalClientProjectPrinciple{organizationID: 7, projectID: 11})
	if err != nil {
		t.Fatalf("createServiceScopeToken() error = %v", err)
	}
	scope, err := types.ExtractServiceScope(token, "secret")
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	context, ok := scope.DelegatedContext()
	if !ok || context.OrganizationID != 7 || context.ProjectID == nil || *context.ProjectID != 11 {
		t.Fatalf("DelegatedContext() = %+v, %v", context, ok)
	}
}
