// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	type_enums "github.com/rapidaai/pkg/types/enums"

	"github.com/rapidaai/pkg/types"
)

type projectScopeClaimAuthenticatorStub struct {
	lastToken    string
	callCount    int
	err          error
	credentialID uint64
}

func (s *projectScopeClaimAuthenticatorStub) Claim(
	_ context.Context,
	claimToken string,
) (*types.PlainClaimPrinciple[*types.ProjectScope], error) {
	s.lastToken = claimToken
	s.callCount++
	if s.err != nil {
		return nil, s.err
	}
	projectID := uint64(1)
	orgID := uint64(1)
	credentialID := s.credentialID
	if credentialID == 0 {
		credentialID = 1
	}
	return &types.PlainClaimPrinciple[*types.ProjectScope]{
		Info: &types.ProjectScope{
			CredentialId:   &credentialID,
			ProjectId:      &projectID,
			OrganizationId: &orgID,
			Status:         type_enums.RECORD_ACTIVE.String(),
			CurrentToken:   claimToken,
		},
	}, nil
}

func TestNewProjectAuthenticatorMiddleware_HeaderToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &projectScopeClaimAuthenticatorStub{}
	engine := gin.New()
	engine.Use(NewProjectAuthenticatorMiddleware(resolver, nil))
	engine.GET("/test", func(c *gin.Context) {
		_, err := types.Authorize(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"hasAuth": err == nil})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(types.PROJECT_SCOPE_KEY, "header-token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, resolver.callCount)
	assert.Equal(t, "header-token", resolver.lastToken)
	assert.JSONEq(t, `{"hasAuth":true}`, rec.Body.String())
}

func TestNewProjectAuthenticatorMiddleware_RemovesLeadingPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &projectScopeClaimAuthenticatorStub{}
	engine := gin.New()
	engine.Use(NewProjectAuthenticatorMiddleware(resolver, nil))
	engine.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX+"secret")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "secret", resolver.lastToken)
}

func TestNewProjectAuthenticatorMiddleware_QueryFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &projectScopeClaimAuthenticatorStub{}
	engine := gin.New()
	engine.Use(NewProjectAuthenticatorMiddleware(resolver, nil))
	engine.GET("/test", func(c *gin.Context) {
		_, err := types.Authorize(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"hasAuth": err == nil})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?x-api-key=query-token", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, resolver.callCount)
	assert.Equal(t, "query-token", resolver.lastToken)
	assert.JSONEq(t, `{"hasAuth":true}`, rec.Body.String())
}

func TestNewProjectAuthenticatorMiddleware_HeaderPrecedenceOverQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &projectScopeClaimAuthenticatorStub{}
	engine := gin.New()
	engine.Use(NewProjectAuthenticatorMiddleware(resolver, nil))
	engine.GET("/test", func(c *gin.Context) {
		_, err := types.Authorize(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"hasAuth": err == nil})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?x-api-key=query-token", nil)
	req.Header.Set(types.PROJECT_SCOPE_KEY, "header-token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, resolver.callCount)
	assert.Equal(t, "header-token", resolver.lastToken)
	assert.JSONEq(t, `{"hasAuth":true}`, rec.Body.String())
}

func TestNewProjectAuthenticatorMiddleware_BlankHeaderFallsBackToQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &projectScopeClaimAuthenticatorStub{}
	engine := gin.New()
	engine.Use(NewProjectAuthenticatorMiddleware(resolver, nil))
	engine.GET("/test", func(c *gin.Context) {
		_, err := types.Authorize(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"hasAuth": err == nil})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?x-api-key=query-token", nil)
	req.Header.Set(types.PROJECT_SCOPE_KEY, "   ")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, resolver.callCount)
	assert.Equal(t, "query-token", resolver.lastToken)
	assert.JSONEq(t, `{"hasAuth":true}`, rec.Body.String())
}

func TestNewProjectAuthenticatorMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &projectScopeClaimAuthenticatorStub{}
	engine := gin.New()
	engine.Use(NewProjectAuthenticatorMiddleware(resolver, nil))
	engine.GET("/test", func(c *gin.Context) {
		_, err := types.Authorize(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"hasAuth": err == nil})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, resolver.callCount)
	assert.JSONEq(t, `{"hasAuth":false}`, rec.Body.String())
}
