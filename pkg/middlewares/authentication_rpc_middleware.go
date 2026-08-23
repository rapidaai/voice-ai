// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

// NewAuthenticationMiddleware authenticates user credentials.
func NewAuthenticationMiddleware(resolver types.Authenticator, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authToken := ctx.Param(types.AUTHORIZATION_KEY)
		if !validator.NonZero(authToken) {
			authToken = ctx.GetHeader(types.AUTHORIZATION_KEY)
		}
		if !validator.NonZero(authToken) {
			authToken = ctx.Query(types.AUTHORIZATION_KEY)
		}
		authID := ctx.GetHeader(types.AUTH_KEY)
		if !validator.NonZero(authID) {
			authID = ctx.Param(types.AUTH_KEY)
		}
		if !validator.NonZero(authID) {
			authID = ctx.Query(types.AUTH_KEY)
		}
		projectID := ctx.GetHeader(types.PROJECT_KEY)
		if !validator.NonZero(projectID) {
			projectID = ctx.Param(types.PROJECT_KEY)
		}
		if !validator.NonZero(projectID) {
			projectID = ctx.Query(types.PROJECT_KEY)
		}
		authToken = strings.TrimSpace(authToken)
		authID = strings.TrimSpace(authID)
		projectID = strings.TrimSpace(projectID)
		if !validator.NotBlank(authToken) && !validator.NotBlank(authID) && !validator.NotBlank(projectID) {
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
				logger.Errorf(userAuthNotSupportedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		if !validator.NotBlank(authToken) || !validator.NotBlank(authID) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthIncompleteMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		userID, err := strconv.ParseUint(authID, 0, 64)
		if err != nil || !validator.Between(userID, uint64(1), uint64(math.MaxInt64)) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthInvalidIDMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		principle, err := resolver.Authorize(ctx.Request.Context(), authToken, userID)
		if err != nil || !validator.NonNil(principle) || !principle.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthRejectedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		if validator.NotBlank(projectID) {
			selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
			if err != nil || !validator.NonZero(selectedProjectID) {
				if validator.NonNil(logger) {
					logger.Errorf(userAuthInvalidProjectIDMessage)
				}
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
				return
			}
			if err := principle.SwitchProject(selectedProjectID); err != nil {
				if validator.NonNil(logger) {
					logger.Errorf(userAuthProjectSelectionRejectedMessage)
				}
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
				return
			}
		}
		actor, err := types.ResolveAuditActor(principle)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthInvalidAuditActorMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, AuthenticationError{Error: AuthenticationFailureMessage})
			return
		}
		organizationID := principle.GetOrganizationRole().OrganizationId
		auth := &types.Authentication{
			AuthType:          types.AuthTypeUser,
			ActorValue:        &actor,
			UserValue:         &types.UserContext{UserID: principle.GetUserInfo().GetId()},
			OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		}
		if projectRole := principle.GetCurrentProjectRole(); validator.NonNil(projectRole) && validator.NonZero(projectRole.ProjectId) {
			auth.ProjectValue = &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectRole.ProjectId}
		}
		requestContext := context.WithValue(ctx.Request.Context(), types.CTX_, auth)
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}
