// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func NewProjectAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var authToken string
		authToken, ok := c.GetQuery(types.PROJECT_SCOPE_KEY)
		if !ok || authToken == "" {
			authToken = c.Param(types.PROJECT_SCOPE_KEY)
		}
		if authToken == "" {
			c.Next()
			return
		}
		auth, err := resolver.Claim(c, authToken)
		if err != nil {
			c.Next()
			return
		}
		c.Set(string(types.CTX_), auth)
		c.Next()
	}
}
