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
)

// NewProjectAuthenticatorMiddleware authenticates project credentials.
func NewProjectAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		apiKey := ctx.GetHeader(types.PROJECT_SCOPE_KEY)
		if strings.TrimSpace(apiKey) == "" {
			apiKey = ctx.Query(types.PROJECT_SCOPE_KEY)
		}
		if strings.TrimSpace(apiKey) == "" {
			apiKey = ctx.Param(types.PROJECT_SCOPE_KEY)
		}
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			ctx.Next()
			return
		}
		if ctx.Request.Context().Value(types.CTX_) != nil {
			if logger != nil {
				logger.Errorf(authenticationConflictMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		if resolver == nil {
			if logger != nil {
				logger.Errorf(projectAuthNotSupportedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		apiKey = strings.TrimPrefix(apiKey, types.PROJECT_KEY_PREFIX)
		if apiKey == "" {
			if logger != nil {
				logger.Errorf(projectAuthEmptyMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		principle, err := resolver.Claim(ctx.Request.Context(), apiKey)
		if err != nil || principle == nil || principle.Info == nil || !principle.Info.IsAuthenticated() {
			if logger != nil {
				logger.Errorf(projectAuthRejectedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if logger != nil {
				logger.Errorf(projectAuthInvalidAuditActorMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
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
