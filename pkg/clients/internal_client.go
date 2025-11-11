package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"google.golang.org/grpc/metadata"
)

type InternalClient interface {
	WithPlatform(ctx context.Context, auth types.SimplePrinciple) context.Context
	WithAuth(ctx context.Context, auth types.SimplePrinciple) context.Context
	WithHttpAuth(c context.Context, auth types.SimplePrinciple, req *http.Request) *http.Request

	WithToken(ctx context.Context, token string, userId uint64) context.Context
	WithScopeToken(c context.Context, token string, scope string) context.Context

	Cache(c context.Context, key string, value interface{}) *connectors.RedisResponse
	Retrieve(c context.Context, key string) *connectors.RedisResponse
	CacheKey(c context.Context, funcName string, key ...string) string
}

type internalClient struct {
	cfg    *config.AppConfig
	logger commons.Logger
	redis  connectors.RedisConnector
}

func NewInternalClient(cfg *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) InternalClient {
	return &internalClient{
		cfg:    cfg,
		logger: logger,
		redis:  redis,
	}
}

func (ic *internalClient) WithToken(c context.Context, token string, userId uint64) context.Context {
	md := metadata.New(map[string]string{
		types.AUTHORIZATION_KEY: token,
		types.AUTH_KEY:          strconv.Itoa(int(userId)),
	})
	return metadata.NewOutgoingContext(c, md)
}

func (ic *internalClient) WithScopeToken(c context.Context, token string, scope string) context.Context {
	if scope == "project" {
		md := metadata.New(map[string]string{
			types.PROJECT_SCOPE_KEY: token,
		})
		return metadata.NewOutgoingContext(c, md)
	}
	md := metadata.New(map[string]string{
		types.ORG_SCOPE_KEY: token,
	})
	return metadata.NewOutgoingContext(c, md)
}

func (ic *internalClient) WithAuth(c context.Context, auth types.SimplePrinciple) context.Context {
	token, err := types.CreateServiceScopeToken(auth, ic.cfg.Secret)
	if err != nil {
		ic.logger.Errorf("Unable to create jwt token for internal service communication %v", err)
		return c
	}
	md := metadata.New(map[string]string{types.SERVICE_SCOPE_KEY: token})
	return metadata.NewOutgoingContext(c, md)
}

func (ic *internalClient) WithPlatform(c context.Context, auth types.SimplePrinciple) context.Context {
	token, err := types.CreateServiceScopeToken(auth, ic.cfg.Secret)
	if err != nil {
		ic.logger.Errorf("Unable to create jwt token for internal service communication %v", err)
		return c
	}
	_platform := map[string]string{
		types.SERVICE_SCOPE_KEY: token,
	}
	source, ok := utils.GetClientSource(c)
	if ok {
		_platform[utils.HEADER_SOURCE_KEY] = source.Get()
	}

	env, ok := utils.GetClientEnvironment(c)
	if ok {
		_platform[utils.HEADER_ENVIRONMENT_KEY] = env.Get()
	}

	// HEADER_REGION_KEY
	region, ok := utils.GetClientRegion(c)
	if ok {
		_platform[utils.HEADER_REGION_KEY] = region.Get()
	}

	return metadata.NewOutgoingContext(c, metadata.New(_platform))
}

func (ic *internalClient) WithHttpAuth(c context.Context, auth types.SimplePrinciple, req *http.Request) *http.Request {
	// Create the token using the provided auth and the client's secret
	token, err := types.CreateServiceScopeToken(auth, ic.cfg.Secret)
	if err != nil {
		ic.logger.Errorf("Unable to create JWT token for internal service communication: %v", err)
		return req.WithContext(c) // Return the original request with context if token generation fails
	}

	// Add the token to the request header (assuming SERVICE_SCOPE_KEY is the header name)
	req.Header.Set("Authorization", token)

	// Return the modified request with the new token in the header
	return req.WithContext(c)
}

func (client *internalClient) Cache(c context.Context, key string, value interface{}) *connectors.RedisResponse {
	data, err := json.Marshal(value)
	if err != nil {
		client.logger.Errorf("Unable to cache the record as value is not marshalable %s", err, key)
		return nil
	}
	put := client.redis.Cmd(c, "SET", []string{key, string(data)})
	if put != nil && put.Err != nil {
		client.logger.Errorf("unable to set cache value with err %v for key %s", put, key)
	}
	return put
}

func (client *internalClient) Retrieve(c context.Context, key string) *connectors.RedisResponse {
	return client.redis.Cmd(c, "GET", []string{key})
}

func (client *internalClient) CacheKey(c context.Context, funcName string, key ...string) string {
	var builder strings.Builder
	builder.WriteString("INTERNAL::")
	builder.WriteString(funcName)
	builder.WriteString("_")
	builder.WriteString(strings.Join(key[:], "__"))
	return builder.String()
}
