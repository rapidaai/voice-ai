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

// NewProjectAuthenticatorMiddleware authenticates project credentials.
func NewProjectAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		apiKey := ctx.GetHeader(types.PROJECT_SCOPE_KEY)
		if !validator.NotBlank(apiKey) {
			apiKey = ctx.Query(types.PROJECT_SCOPE_KEY)
		}
		if !validator.NotBlank(apiKey) {
			apiKey = ctx.Param(types.PROJECT_SCOPE_KEY)
		}
		apiKey = strings.TrimSpace(apiKey)
		if !validator.NotBlank(apiKey) {
			ctx.Next()
			return
		}
		if validator.NonNil(ctx.Request.Context().Value(types.CTX_)) {
			if validator.NonNil(logger) {
				logger.Errorf(authenticationConflictMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		if !validator.NonNil(resolver) {
			if validator.NonNil(logger) {
				logger.Errorf(projectAuthNotSupportedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		apiKey = strings.TrimPrefix(apiKey, types.PROJECT_KEY_PREFIX)
		if !validator.NotBlank(apiKey) {
			if validator.NonNil(logger) {
				logger.Errorf(projectAuthEmptyMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		principle, err := resolver.Claim(ctx.Request.Context(), apiKey)
		if err != nil || !validator.NonNil(principle) || !validator.NonNil(principle.Info) || !principle.Info.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(projectAuthRejectedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(projectAuthInvalidAuditActorMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		projectContext, _ := principle.Info.ProjectContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeProject,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: projectContext.OrganizationID},
			ProjectValue:      &projectContext,
		}
		requestContext := context.WithValue(ctx.Request.Context(), types.CTX_, auth)
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}
