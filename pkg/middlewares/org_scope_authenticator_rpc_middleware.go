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

// NewOrganizationAuthenticatorMiddleware authenticates organization credentials.
func NewOrganizationAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.OrganizationScope], logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		apiKey := ctx.GetHeader(types.ORG_SCOPE_KEY)
		if !validator.NotBlank(apiKey) {
			apiKey = ctx.Query(types.ORG_SCOPE_KEY)
		}
		if !validator.NotBlank(apiKey) {
			apiKey = ctx.Param(types.ORG_SCOPE_KEY)
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
				logger.Errorf(organizationAuthNotSupportedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		principle, err := resolver.Claim(ctx.Request.Context(), apiKey)
		if err != nil || !validator.NonNil(principle) || !validator.NonNil(principle.Info) || !principle.Info.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(organizationAuthRejectedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(organizationAuthInvalidAuditActorMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		organizationID, _ := principle.Info.OrganizationContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeOrg,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		}
		requestContext := context.WithValue(ctx.Request.Context(), types.CTX_, auth)
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}
