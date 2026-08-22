package middlewares

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func NewAuthenticationBoundaryMiddleware(
	user types.Authenticator,
	project types.ClaimAuthenticator[*types.ProjectScope],
	organization types.ClaimAuthenticator[*types.OrganizationScope],
	service types.ClaimAuthenticator[*types.ServiceScope],
	logger commons.Logger,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		presence := ginCredentialPresence(ctx)
		if presence.count() == 0 {
			ctx.Next()
			return
		}
		if presence.count() > 1 {
			logAuthenticationFailure(logger, "authentication credential conflict: classes=%s", presence.classes())
			abortGinAuthentication(ctx)
			return
		}

		requestContext := ctx.Request.Context()
		var auth *types.Authentication

		if presence.user {
			if user == nil {
				logAuthenticationFailure(logger, "user credential is not supported")
				abortGinAuthentication(ctx)
				return
			}
			authToken, authID, projectID := ginUserCredentials(ctx)
			if strings.TrimSpace(authToken) == "" || strings.TrimSpace(authID) == "" {
				logAuthenticationFailure(logger, "user credential is incomplete")
				abortGinAuthentication(ctx)
				return
			}
			userID, err := strconv.ParseUint(authID, 0, 64)
			if err != nil || userID == 0 || userID > math.MaxInt64 {
				logAuthenticationFailure(logger, "user credential has invalid auth id")
				abortGinAuthentication(ctx)
				return
			}
			principle, err := user.Authorize(requestContext, authToken, userID)
			if err != nil || principle == nil || !principle.IsAuthenticated() {
				logAuthenticationFailure(logger, "user credential was rejected")
				abortGinAuthentication(ctx)
				return
			}
			if strings.TrimSpace(projectID) != "" {
				selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
				if err != nil || selectedProjectID == 0 || principle.SwitchProject(selectedProjectID) != nil {
					logAuthenticationFailure(logger, "user credential project selection was rejected")
					abortGinAuthentication(ctx)
					return
				}
			}
			authenticatedUserID := principle.GetUserInfo().GetId()
			organizationID := principle.GetOrganizationRole().OrganizationId
			actor, err := types.ResolveAuditActor(principle)
			if err != nil {
				logAuthenticationFailure(logger, "user credential has invalid audit actor")
				abortGinAuthentication(ctx)
				return
			}
			auth = &types.Authentication{
				AuthType:          types.AuthTypeUser,
				ActorValue:        &actor,
				UserValue:         &types.UserContext{UserID: authenticatedUserID},
				OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
			}
			if projectRole := principle.GetCurrentProjectRole(); projectRole != nil && projectRole.ProjectId > 0 {
				projectContext := types.ProjectContext{OrganizationID: organizationID, ProjectID: projectRole.ProjectId}
				auth.ProjectValue = &projectContext
			}
		}

		if presence.project {
			if project == nil {
				logAuthenticationFailure(logger, "project credential is not supported")
				abortGinAuthentication(ctx)
				return
			}
			apiKey := strings.TrimPrefix(strings.TrimSpace(ginProjectCredential(ctx)), types.PROJECT_KEY_PREFIX)
			if apiKey == "" {
				logAuthenticationFailure(logger, "project credential is empty")
				abortGinAuthentication(ctx)
				return
			}
			principle, err := project.Claim(requestContext, apiKey)
			if err != nil || principle == nil || principle.Info == nil || !principle.Info.IsAuthenticated() {
				logAuthenticationFailure(logger, "project credential was rejected")
				abortGinAuthentication(ctx)
				return
			}
			actor, err := types.ResolveAuditActor(principle.Info)
			if err != nil {
				logAuthenticationFailure(logger, "project credential has invalid audit actor")
				abortGinAuthentication(ctx)
				return
			}
			projectContext, _ := principle.Info.ProjectContext()
			auth = &types.Authentication{
				AuthType:          types.AuthTypeProject,
				ActorValue:        &actor,
				OrganizationValue: &types.OrganizationContext{OrganizationID: projectContext.OrganizationID},
				ProjectValue:      &projectContext,
			}
		}

		if presence.organization {
			if organization == nil {
				logAuthenticationFailure(logger, "organization credential is not supported")
				abortGinAuthentication(ctx)
				return
			}
			principle, err := organization.Claim(requestContext, strings.TrimSpace(ginOrganizationCredential(ctx)))
			if err != nil || principle == nil || principle.Info == nil || !principle.Info.IsAuthenticated() {
				logAuthenticationFailure(logger, "organization credential was rejected")
				abortGinAuthentication(ctx)
				return
			}
			actor, err := types.ResolveAuditActor(principle.Info)
			if err != nil {
				logAuthenticationFailure(logger, "organization credential has invalid audit actor")
				abortGinAuthentication(ctx)
				return
			}
			organizationID, _ := principle.Info.OrganizationContext()
			auth = &types.Authentication{
				AuthType:          types.AuthTypeOrg,
				ActorValue:        &actor,
				OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
			}
		}

		if presence.service {
			if service == nil {
				logAuthenticationFailure(logger, "service credential is not supported")
				abortGinAuthentication(ctx)
				return
			}
			principle, err := service.Claim(requestContext, strings.TrimSpace(ginServiceCredential(ctx)))
			if err != nil || principle == nil || principle.Info == nil || !principle.Info.IsAuthenticated() {
				logAuthenticationFailure(logger, "service credential was rejected")
				abortGinAuthentication(ctx)
				return
			}
			delegatedContext, _ := principle.Info.DelegatedContext()
			actor, err := types.ResolveAuditActor(principle.Info)
			if err != nil {
				logAuthenticationFailure(logger, "service credential has invalid audit actor")
				abortGinAuthentication(ctx)
				return
			}
			auth = &types.Authentication{
				AuthType:          types.AuthTypeService,
				ActorValue:        &actor,
				OrganizationValue: &types.OrganizationContext{OrganizationID: delegatedContext.OrganizationID},
			}
			if delegatedContext.ProjectID != nil {
				auth.ProjectValue = &types.ProjectContext{
					OrganizationID: delegatedContext.OrganizationID,
					ProjectID:      *delegatedContext.ProjectID,
				}
			}
		}

		ctx.Request = ctx.Request.WithContext(context.WithValue(requestContext, types.CTX_, auth))
		ctx.Next()
	}
}
