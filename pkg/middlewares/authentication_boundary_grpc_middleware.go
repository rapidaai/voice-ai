package middlewares

import (
	"context"
	"strconv"
	"strings"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"google.golang.org/grpc"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func NewAuthenticationBoundaryUnaryServerMiddleware(
	user types.Authenticator,
	project types.ClaimAuthenticator[*types.ProjectScope],
	organization types.ClaimAuthenticator[*types.OrganizationScope],
	service types.ClaimAuthenticator[*types.ServiceScope],
	logger commons.Logger,
) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		auth, err := authenticateGRPCRequest(ctx, user, project, organization, service, logger)
		if err != nil {
			return nil, err
		}
		if auth == nil {
			return handler(ctx, req)
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

func NewAuthenticationBoundaryStreamServerMiddleware(
	user types.Authenticator,
	project types.ClaimAuthenticator[*types.ProjectScope],
	organization types.ClaimAuthenticator[*types.OrganizationScope],
	service types.ClaimAuthenticator[*types.ServiceScope],
	logger commons.Logger,
) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		auth, err := authenticateGRPCRequest(stream.Context(), user, project, organization, service, logger)
		if err != nil {
			return err
		}
		if auth == nil {
			return handler(srv, stream)
		}
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(stream.Context(), types.CTX_, auth)
		return handler(srv, wrapped)
	}
}

func authenticateGRPCRequest(
	ctx context.Context,
	user types.Authenticator,
	project types.ClaimAuthenticator[*types.ProjectScope],
	organization types.ClaimAuthenticator[*types.OrganizationScope],
	service types.ClaimAuthenticator[*types.ServiceScope],
	logger commons.Logger,
) (*types.Authentication, error) {
	presence := grpcCredentialPresence(ctx)
	if presence.count() == 0 {
		return nil, nil
	}
	if presence.count() > 1 {
		logAuthenticationFailure(logger, "authentication credential conflict: classes=%s", presence.classes())
		return nil, grpcAuthenticationError()
	}

	incoming := metadata.ExtractIncoming(ctx)
	if presence.user {
		if user == nil {
			logAuthenticationFailure(logger, "user credential is not supported")
			return nil, grpcAuthenticationError()
		}
		authToken := strings.TrimSpace(incoming.Get(types.AUTHORIZATION_KEY))
		authID := strings.TrimSpace(incoming.Get(types.AUTH_KEY))
		projectID := strings.TrimSpace(incoming.Get(types.PROJECT_KEY))
		if authToken == "" || authID == "" {
			logAuthenticationFailure(logger, "user credential is incomplete")
			return nil, grpcAuthenticationError()
		}

		userID, err := strconv.ParseUint(authID, 0, 64)
		if err != nil || userID == 0 {
			logAuthenticationFailure(logger, "user credential has invalid auth id")
			return nil, grpcAuthenticationError()
		}
		auth, err := user.Authorize(ctx, authToken, userID)
		if err != nil || auth == nil || !auth.IsAuthenticated() {
			logAuthenticationFailure(logger, "user credential was rejected")
			return nil, grpcAuthenticationError()
		}
		if projectID != "" {
			selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
			if err != nil || selectedProjectID == 0 {
				logAuthenticationFailure(logger, "user credential has invalid project id")
				return nil, grpcAuthenticationError()
			}
			if err := auth.SwitchProject(selectedProjectID); err != nil {
				logAuthenticationFailure(logger, "user credential project selection was rejected")
				return nil, grpcAuthenticationError()
			}
		}
		authenticatedUserID := auth.GetUserInfo().GetId()
		organizationID := auth.GetOrganizationRole().OrganizationId
		requestAuth := &types.Authentication{
			AuthType:          types.AuthTypeUser,
			UserValue:         &types.UserContext{UserID: authenticatedUserID},
			OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		}
		actor := types.ActorIdentity{Type: types.ActorTypeUser, ID: strconv.FormatUint(authenticatedUserID, 10)}
		requestAuth.ActorValue = &actor
		if projectRole := auth.GetCurrentProjectRole(); projectRole != nil && projectRole.ProjectId > 0 {
			projectContext := types.ProjectContext{OrganizationID: organizationID, ProjectID: projectRole.ProjectId}
			requestAuth.ProjectValue = &projectContext
		}
		return requestAuth, nil
	}

	if presence.project {
		if project == nil {
			logAuthenticationFailure(logger, "project credential is not supported")
			return nil, grpcAuthenticationError()
		}
		apiKey := strings.TrimPrefix(strings.TrimSpace(incoming.Get(types.PROJECT_SCOPE_KEY)), types.PROJECT_KEY_PREFIX)
		if apiKey == "" {
			logAuthenticationFailure(logger, "project credential is empty")
			return nil, grpcAuthenticationError()
		}
		auth, err := project.Claim(ctx, apiKey)
		if err != nil || auth == nil || auth.Info == nil || !auth.Info.IsAuthenticated() {
			logAuthenticationFailure(logger, "project credential was rejected")
			return nil, grpcAuthenticationError()
		}
		projectContext, _ := auth.Info.ProjectContext()
		requestAuth := &types.Authentication{
			AuthType:          types.AuthTypeProject,
			OrganizationValue: &types.OrganizationContext{OrganizationID: projectContext.OrganizationID},
			ProjectValue:      &projectContext,
		}
		if actor, ok := auth.Info.AuditActor(); ok {
			requestAuth.ActorValue = &actor
		}
		return requestAuth, nil
	}

	if presence.organization {
		if organization == nil {
			logAuthenticationFailure(logger, "organization credential is not supported")
			return nil, grpcAuthenticationError()
		}
		apiKey := strings.TrimSpace(incoming.Get(types.ORG_SCOPE_KEY))
		auth, err := organization.Claim(ctx, apiKey)
		if err != nil || auth == nil || auth.Info == nil || !auth.Info.IsAuthenticated() {
			logAuthenticationFailure(logger, "organization credential was rejected")
			return nil, grpcAuthenticationError()
		}
		organizationID, _ := auth.Info.OrganizationContext()
		return &types.Authentication{
			AuthType:          types.AuthTypeOrg,
			OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		}, nil
	}

	if service == nil {
		logAuthenticationFailure(logger, "service credential is not supported")
		return nil, grpcAuthenticationError()
	}
	apiKey := strings.TrimSpace(incoming.Get(types.SERVICE_SCOPE_KEY))
	auth, err := service.Claim(ctx, apiKey)
	if err != nil || auth == nil || auth.Info == nil || !auth.Info.IsAuthenticated() {
		logAuthenticationFailure(logger, "service credential was rejected")
		return nil, grpcAuthenticationError()
	}
	delegatedContext, _ := auth.Info.DelegatedContext()
	requestAuth := &types.Authentication{
		AuthType:          types.AuthTypeService,
		OrganizationValue: &types.OrganizationContext{OrganizationID: delegatedContext.OrganizationID},
	}
	if delegatedContext.UserID != nil {
		requestAuth.UserValue = &types.UserContext{UserID: *delegatedContext.UserID}
	}
	if delegatedContext.ProjectID != nil {
		requestAuth.ProjectValue = &types.ProjectContext{OrganizationID: delegatedContext.OrganizationID, ProjectID: *delegatedContext.ProjectID}
	}
	return requestAuth, nil
}
