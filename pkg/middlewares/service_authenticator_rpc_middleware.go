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

// NewServiceAuthenticatorMiddleware authenticates service credentials.
func NewServiceAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ServiceScope], logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		assertion := ctx.GetHeader(types.SERVICE_SCOPE_KEY)
		if strings.TrimSpace(assertion) == "" {
			assertion = ctx.Query(types.SERVICE_SCOPE_KEY)
		}
		if strings.TrimSpace(assertion) == "" {
			assertion = ctx.Param(types.SERVICE_SCOPE_KEY)
		}
		assertion = strings.TrimSpace(assertion)
		if assertion == "" {
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
				logger.Errorf(serviceAuthNotSupportedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		principle, err := resolver.Claim(ctx.Request.Context(), assertion)
		if err != nil || principle == nil || principle.Info == nil || !principle.Info.IsAuthenticated() {
			if logger != nil {
				logger.Errorf(serviceAuthRejectedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if logger != nil {
				logger.Errorf(serviceAuthInvalidAuditActorMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		delegatedContext, _ := principle.Info.DelegatedContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeService,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: delegatedContext.OrganizationID},
		}
		if delegatedContext.ProjectID != nil {
			auth.ProjectValue = &types.ProjectContext{OrganizationID: delegatedContext.OrganizationID, ProjectID: *delegatedContext.ProjectID}
		}
		requestContext := context.WithValue(ctx.Request.Context(), types.CTX_, auth)
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}
