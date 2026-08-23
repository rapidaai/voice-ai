// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"context"
	"math"
	"strconv"
	"strings"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

// NewAuthenticationUnaryServerMiddleware authenticates user credentials.
func NewAuthenticationUnaryServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		incoming := metadata.ExtractIncoming(ctx)
		authToken := strings.TrimSpace(incoming.Get(types.AUTHORIZATION_KEY))
		authID := strings.TrimSpace(incoming.Get(types.AUTH_KEY))
		projectID := strings.TrimSpace(incoming.Get(types.PROJECT_KEY))
		if authToken == "" && authID == "" && projectID == "" {
			return handler(ctx, req)
		}
		if ctx.Value(types.CTX_) != nil {
			if logger != nil {
				logger.Errorf(authenticationConflictMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		if resolver == nil {
			if logger != nil {
				logger.Errorf(userAuthNotSupportedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		if authToken == "" || authID == "" {
			if logger != nil {
				logger.Errorf(userAuthIncompleteMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		userID, err := strconv.ParseUint(authID, 0, 64)
		if err != nil || userID == 0 || userID > math.MaxInt64 {
			if logger != nil {
				logger.Errorf(userAuthInvalidIDMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		principle, err := resolver.Authorize(ctx, authToken, userID)
		if err != nil || principle == nil || !principle.IsAuthenticated() {
			if logger != nil {
				logger.Errorf(userAuthRejectedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		if projectID != "" {
			selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
			if err != nil || selectedProjectID == 0 {
				if logger != nil {
					logger.Errorf(userAuthInvalidProjectIDMessage)
				}
				return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
			}
			if err := principle.SwitchProject(selectedProjectID); err != nil {
				if logger != nil {
					logger.Errorf(userAuthProjectSelectionRejectedMessage)
				}
				return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
			}
		}
		actor, err := types.ResolveAuditActor(principle)
		if err != nil {
			if logger != nil {
				logger.Errorf(userAuthInvalidAuditActorMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
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
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// NewAuthenticationStreamServerMiddleware authenticates user credentials.
func NewAuthenticationStreamServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		incoming := metadata.ExtractIncoming(ctx)
		authToken := strings.TrimSpace(incoming.Get(types.AUTHORIZATION_KEY))
		authID := strings.TrimSpace(incoming.Get(types.AUTH_KEY))
		projectID := strings.TrimSpace(incoming.Get(types.PROJECT_KEY))
		if authToken == "" && authID == "" && projectID == "" {
			return handler(srv, stream)
		}
		if ctx.Value(types.CTX_) != nil {
			if logger != nil {
				logger.Errorf(authenticationConflictMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		if resolver == nil {
			if logger != nil {
				logger.Errorf(userAuthNotSupportedMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		if authToken == "" || authID == "" {
			if logger != nil {
				logger.Errorf(userAuthIncompleteMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		userID, err := strconv.ParseUint(authID, 0, 64)
		if err != nil || userID == 0 || userID > math.MaxInt64 {
			if logger != nil {
				logger.Errorf(userAuthInvalidIDMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		principle, err := resolver.Authorize(ctx, authToken, userID)
		if err != nil || principle == nil || !principle.IsAuthenticated() {
			if logger != nil {
				logger.Errorf(userAuthRejectedMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		if projectID != "" {
			selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
			if err != nil || selectedProjectID == 0 {
				if logger != nil {
					logger.Errorf(userAuthInvalidProjectIDMessage)
				}
				return status.Error(codes.Unauthenticated, authenticationFailureMessage)
			}
			if err := principle.SwitchProject(selectedProjectID); err != nil {
				if logger != nil {
					logger.Errorf(userAuthProjectSelectionRejectedMessage)
				}
				return status.Error(codes.Unauthenticated, authenticationFailureMessage)
			}
		}
		actor, err := types.ResolveAuditActor(principle)
		if err != nil {
			if logger != nil {
				logger.Errorf(userAuthInvalidAuditActorMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
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
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
