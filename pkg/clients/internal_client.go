package clients

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	"google.golang.org/grpc/metadata"
)

type InternalClient interface {
	WithAuth(ctx context.Context, auth types.SimplePrinciple) context.Context
	Cache(c context.Context, key string, value interface{}) *connectors.RedisResponse
	Retrieve(c context.Context, key string) *connectors.RedisResponse
	CacheKey(c context.Context, funcName string, key ...uint64) string
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

func (ic *internalClient) WithAuth(c context.Context, auth types.SimplePrinciple) context.Context {
	token, err := types.CreateServiceScopeToken(auth, ic.cfg.Secret)

	if err != nil {
		ic.logger.Errorf("Unable to create jwt token for internal service communication %v", err)
		return c
	}
	md := metadata.New(map[string]string{types.SERVICE_SCOPE_KEY: token})
	return metadata.NewOutgoingContext(c, md)
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

func (client *internalClient) CacheKey(c context.Context, funcName string, key ...uint64) string {
	var builder strings.Builder
	builder.WriteString("INTERNAL::")
	builder.WriteString(funcName)
	builder.WriteString("_")
	for i, k := range key {
		if i > 0 {
			builder.WriteString("_")
		}
		builder.WriteString(strconv.FormatUint(k, 10))
	}

	return builder.String()
}
