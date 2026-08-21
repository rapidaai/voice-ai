// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	// "github.com/rapidaai/pkg/models"
)

func NewAuthenticationMiddleware(resolver types.Authenticator, logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the token from the request header, URL param, or query param
		// Query param is needed for WebSocket connections (browsers can't set custom WS headers)
		authToken, authId, projectId := ginUserCredentials(c)
		if strings.TrimSpace(authToken) == "" && strings.TrimSpace(authId) == "" && strings.TrimSpace(projectId) == "" {
			c.Next() // Continue processing the request without authentication
			return
		}
		if strings.TrimSpace(authToken) == "" || strings.TrimSpace(authId) == "" {
			logAuthenticationFailure(logger, "user credential is incomplete")
			abortGinAuthentication(c)
			return
		}
		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil || id == 0 {
			logAuthenticationFailure(logger, "user credential has invalid auth id")
			abortGinAuthentication(c)
			return
		}
		auth, err := resolver.Authorize(c, authToken, id)
		if err != nil {
			logAuthenticationFailure(logger, "user credential was rejected")
			abortGinAuthentication(c)
			return
		}
		if strings.TrimSpace(projectId) != "" {
			pid, err := strconv.ParseUint(projectId, 0, 64)
			if err != nil || pid == 0 || auth.SwitchProject(pid) != nil {
				logAuthenticationFailure(logger, "user credential project selection was rejected")
				abortGinAuthentication(c)
				return
			}
		}
		// Attach the user information to the context
		c.Set(string(types.CTX_), auth)
		// Continue processing the request
		c.Next()
	}
}
