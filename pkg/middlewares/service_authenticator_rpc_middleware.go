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

// NewServiceAuthenticatorMiddleware authenticates service credentials.
func NewServiceAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ServiceScope], logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		assertion := ctx.GetHeader(types.SERVICE_SCOPE_KEY)
		if !validator.NotBlank(assertion) {
			assertion = ctx.Query(types.SERVICE_SCOPE_KEY)
		}
		if !validator.NotBlank(assertion) {
			assertion = ctx.Param(types.SERVICE_SCOPE_KEY)
		}
		assertion = strings.TrimSpace(assertion)
		if !validator.NotBlank(assertion) {
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
				logger.Errorf(serviceAuthNotSupportedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		principle, err := resolver.Claim(ctx.Request.Context(), assertion)
		if err != nil || !validator.NonNil(principle) || !validator.NonNil(principle.Info) || !principle.Info.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(serviceAuthRejectedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(serviceAuthInvalidAuditActorMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		delegatedContext, _ := principle.Info.DelegatedContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeService,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: delegatedContext.OrganizationID},
		}
		if validator.NonNil(delegatedContext.ProjectID) {
			auth.ProjectValue = &types.ProjectContext{OrganizationID: delegatedContext.OrganizationID, ProjectID: *delegatedContext.ProjectID}
		}
		requestContext := context.WithValue(ctx.Request.Context(), types.CTX_, auth)
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}
