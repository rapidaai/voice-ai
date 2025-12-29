// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	// "github.com/rapidaai/pkg/models"
)

func NewAuthenticationMiddleware(resolver types.Authenticator, logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the token from the request header
		authToken := c.Param(types.AUTHORIZATION_KEY)
		if authToken == "" {
			authToken = c.GetHeader(types.AUTHORIZATION_KEY)
		}
		authId := c.GetHeader(types.AUTH_KEY)
		if authId == "" {
			authId = c.Param(types.AUTH_KEY)
		}
		projectId := c.GetHeader(types.PROJECT_KEY)
		if projectId == "" {
			projectId = c.Param(types.PROJECT_KEY)
		}
		if authToken == "" {
			c.Next() // Continue processing the request without authentication
			return
		}
		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil {
			logger.Errorf("auth id is not int.")
			c.Next()
			return
		}
		auth, err := resolver.Authorize(c, authToken, id)
		if err != nil {
			logger.Errorf("unable to resolve auth token and id")
			c.Next()
			return
		}
		pid, err := strconv.ParseUint(projectId, 0, 64)
		if err == nil {
			auth.SwitchProject(pid)
		}
		// Attach the user information to the context
		c.Set(string(types.CTX_), auth)
		// Continue processing the request
		c.Next()
	}
}
