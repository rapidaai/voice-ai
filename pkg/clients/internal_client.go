// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package clients

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

type InternalClient interface {
	WithPlatform(ctx context.Context, auth *types.Authentication) (context.Context, error)
	WithAuth(ctx context.Context, auth *types.Authentication) (context.Context, error)
	WithHttpAuth(c context.Context, auth *types.Authentication, req *http.Request) (*http.Request, error)

	WithToken(ctx context.Context, token string, userId uint64) context.Context
	WithScopeToken(c context.Context, token string, scope string) context.Context

	Cache(c context.Context, key string, value interface{}) *connectors.RedisResponse
	CacheWithTTL(c context.Context, key string, value interface{}, ttl time.Duration) *connectors.RedisResponse
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

func (ic *internalClient) WithAuth(c context.Context, auth *types.Authentication) (context.Context, error) {
	token, err := ic.createServiceScopeToken(auth)
	if err != nil {
		return nil, err
	}
	md := metadata.New(map[string]string{types.SERVICE_SCOPE_KEY: token})
	return metadata.NewOutgoingContext(c, md), nil
}

func (ic *internalClient) WithPlatform(c context.Context, auth *types.Authentication) (context.Context, error) {
	token, err := ic.createServiceScopeToken(auth)
	if err != nil {
		return nil, err
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

	return metadata.NewOutgoingContext(c, metadata.New(_platform)), nil
}

func (ic *internalClient) WithHttpAuth(c context.Context, auth *types.Authentication, req *http.Request) (*http.Request, error) {
	token, err := ic.createServiceScopeToken(auth)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	return req.WithContext(c), nil
}

func (ic *internalClient) createServiceScopeToken(auth *types.Authentication) (string, error) {
	organizationContext, err := auth.OrganizationContext()
	if err != nil {
		return "", err
	}
	delegatedContext := types.DelegatedContext{OrganizationID: organizationContext.OrganizationID}
	if userContext, userErr := auth.UserContext(); userErr == nil {
		delegatedContext.UserID = &userContext.UserID
	} else if !errors.Is(userErr, types.ErrUserContextUnavailable) {
		return "", userErr
	}
	if projectContext, projectErr := auth.ProjectContext(); projectErr == nil {
		if projectContext.OrganizationID != organizationContext.OrganizationID {
			return "", errors.New("project context organization does not match authentication organization")
		}
		delegatedContext.ProjectID = &projectContext.ProjectID
	} else if !errors.Is(projectErr, types.ErrProjectContextUnavailable) {
		return "", projectErr
	}
	return types.CreateServiceScopeToken(delegatedContext, ic.cfg.Secret)
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

func (client *internalClient) CacheWithTTL(c context.Context, key string, value interface{}, ttl time.Duration) *connectors.RedisResponse {
	if ttl <= 0 {
		return &connectors.RedisResponse{Err: errors.New("cache TTL must be positive")}
	}
	data, err := json.Marshal(value)
	if err != nil {
		client.logger.Errorf("Unable to cache the record as value is not marshalable %s", err, key)
		return &connectors.RedisResponse{Err: err}
	}
	put := client.redis.Cmd(c, "SET", []string{key, string(data), "EX", strconv.FormatInt(int64(ttl/time.Second), 10)})
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
	builder.WriteString(strings.Join(key, "__"))
	return builder.String()
}
