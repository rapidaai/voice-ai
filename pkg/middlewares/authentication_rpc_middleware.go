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
)

// NewAuthenticationMiddleware authenticates user credentials.
func NewAuthenticationMiddleware(resolver types.Authenticator, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authToken := ctx.Param(types.AUTHORIZATION_KEY)
		if authToken == "" {
			authToken = ctx.GetHeader(types.AUTHORIZATION_KEY)
		}
		if authToken == "" {
			authToken = ctx.Query(types.AUTHORIZATION_KEY)
		}
		authID := ctx.GetHeader(types.AUTH_KEY)
		if authID == "" {
			authID = ctx.Param(types.AUTH_KEY)
		}
		if authID == "" {
			authID = ctx.Query(types.AUTH_KEY)
		}
		projectID := ctx.GetHeader(types.PROJECT_KEY)
		if projectID == "" {
			projectID = ctx.Param(types.PROJECT_KEY)
		}
		if projectID == "" {
			projectID = ctx.Query(types.PROJECT_KEY)
		}
		authToken = strings.TrimSpace(authToken)
		authID = strings.TrimSpace(authID)
		projectID = strings.TrimSpace(projectID)
		if authToken == "" && authID == "" && projectID == "" {
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
				logger.Errorf(userAuthNotSupportedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		if authToken == "" || authID == "" {
			if logger != nil {
				logger.Errorf(userAuthIncompleteMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		userID, err := strconv.ParseUint(authID, 0, 64)
		if err != nil || userID == 0 || userID > math.MaxInt64 {
			if logger != nil {
				logger.Errorf(userAuthInvalidIDMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		principle, err := resolver.Authorize(ctx.Request.Context(), authToken, userID)
		if err != nil || principle == nil || !principle.IsAuthenticated() {
			if logger != nil {
				logger.Errorf(userAuthRejectedMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		if projectID != "" {
			selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
			if err != nil || selectedProjectID == 0 {
				if logger != nil {
					logger.Errorf(userAuthInvalidProjectIDMessage)
				}
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
				return
			}
			if err := principle.SwitchProject(selectedProjectID); err != nil {
				if logger != nil {
					logger.Errorf(userAuthProjectSelectionRejectedMessage)
				}
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
				return
			}
		}
		actor, err := types.ResolveAuditActor(principle)
		if err != nil {
			if logger != nil {
				logger.Errorf(userAuthInvalidAuditActorMessage)
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
			return
		}
		organizationID := principle.GetOrganizationRole().OrganizationId
		auth := &types.Authentication{
			AuthType:          types.AuthTypeUser,
			ActorValue:        &actor,
			UserValue:         &types.UserContext{UserID: principle.GetUserInfo().GetId()},
			OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		}
		if projectRole := principle.GetCurrentProjectRole(); projectRole != nil && projectRole.ProjectId > 0 {
			auth.ProjectValue = &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectRole.ProjectId}
		}
		requestContext := context.WithValue(ctx.Request.Context(), types.CTX_, auth)
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}
