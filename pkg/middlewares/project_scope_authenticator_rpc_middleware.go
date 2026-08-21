// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

// NewProjectAuthenticatorMiddleware authenticates only project credentials.
// Deprecated: use NewAuthenticationBoundaryMiddleware.
func NewProjectAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := ginProjectCredential(c)
		if strings.TrimSpace(authToken) == "" {
			c.Next()
			return
		}
		authToken = strings.TrimPrefix(strings.TrimSpace(authToken), types.PROJECT_KEY_PREFIX)
		if authToken == "" {
			logAuthenticationFailure(logger, "project credential is empty")
			abortGinAuthentication(c)
			return
		}
		auth, err := resolver.Claim(c, authToken)
		if err != nil {
			logAuthenticationFailure(logger, "project credential was rejected")
			abortGinAuthentication(c)
			return
		}
		c.Set(string(types.CTX_), auth)
		c.Next()
	}
}
