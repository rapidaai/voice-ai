// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

// NewAuthenticationMiddleware authenticates user credentials.
func NewAuthenticationMiddleware(resolver types.Authenticator, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if validator.NonNil(ctx.Request.Context().Value(types.CTX_)) {
			ctx.Next()
			return
		}
		credential := userCredential{
			token:     ctx.Param(types.AUTHORIZATION_KEY),
			authID:    ctx.GetHeader(types.AUTH_KEY),
			projectID: ctx.GetHeader(types.PROJECT_KEY),
		}
		if !validator.NonZero(credential.token) {
			credential.token = ctx.GetHeader(types.AUTHORIZATION_KEY)
		}
		if !validator.NonZero(credential.token) {
			credential.token = ctx.Query(types.AUTHORIZATION_KEY)
		}
		if !validator.NonZero(credential.authID) {
			credential.authID = ctx.Param(types.AUTH_KEY)
		}
		if !validator.NonZero(credential.authID) {
			credential.authID = ctx.Query(types.AUTH_KEY)
		}
		if !validator.NonZero(credential.projectID) {
			credential.projectID = ctx.Param(types.PROJECT_KEY)
		}
		if !validator.NonZero(credential.projectID) {
			credential.projectID = ctx.Query(types.PROJECT_KEY)
		}
		credential.token = strings.TrimSpace(credential.token)
		credential.authID = strings.TrimSpace(credential.authID)
		credential.projectID = strings.TrimSpace(credential.projectID)
		if credential.isEmpty() {
			ctx.Next()
			return
		}
		auth, err := credential.authenticate(ctx.Request.Context(), resolver)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf("%s", err)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		requestContext := context.WithValue(ctx.Request.Context(), types.CTX_, auth)
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}
