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
	"github.com/rapidaai/pkg/validator"
)

// NewAuthenticationUnaryServerMiddleware authenticates user credentials.
func NewAuthenticationUnaryServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		incoming := metadata.ExtractIncoming(ctx)
		authToken := strings.TrimSpace(incoming.Get(types.AUTHORIZATION_KEY))
		authID := strings.TrimSpace(incoming.Get(types.AUTH_KEY))
		projectID := strings.TrimSpace(incoming.Get(types.PROJECT_KEY))
		if !validator.NotBlank(authToken) && !validator.NotBlank(authID) && !validator.NotBlank(projectID) {
			return handler(ctx, req)
		}
		if validator.NonNil(ctx.Value(types.CTX_)) {
			if validator.NonNil(logger) {
				logger.Errorf(authenticationConflictMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		if !validator.NonNil(resolver) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthNotSupportedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		if !validator.NotBlank(authToken) || !validator.NotBlank(authID) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthIncompleteMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		userID, err := strconv.ParseUint(authID, 0, 64)
		if err != nil || !validator.Between(userID, uint64(1), uint64(math.MaxInt64)) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthInvalidIDMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		principle, err := resolver.Authorize(ctx, authToken, userID)
		if err != nil || !validator.NonNil(principle) || !principle.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthRejectedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		if validator.NotBlank(projectID) {
			selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
			if err != nil || !validator.NonZero(selectedProjectID) {
				if validator.NonNil(logger) {
					logger.Errorf(userAuthInvalidProjectIDMessage)
				}
				return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
			}
			if err := principle.SwitchProject(selectedProjectID); err != nil {
				if validator.NonNil(logger) {
					logger.Errorf(userAuthProjectSelectionRejectedMessage)
				}
				return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
			}
		}
		actor, err := types.ResolveAuditActor(principle)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthInvalidAuditActorMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
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
		if !validator.NotBlank(authToken) && !validator.NotBlank(authID) && !validator.NotBlank(projectID) {
			return handler(srv, stream)
		}
		if validator.NonNil(ctx.Value(types.CTX_)) {
			if validator.NonNil(logger) {
				logger.Errorf(authenticationConflictMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		if !validator.NonNil(resolver) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthNotSupportedMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		if !validator.NotBlank(authToken) || !validator.NotBlank(authID) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthIncompleteMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		userID, err := strconv.ParseUint(authID, 0, 64)
		if err != nil || !validator.Between(userID, uint64(1), uint64(math.MaxInt64)) {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthInvalidIDMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		principle, err := resolver.Authorize(ctx, authToken, userID)
		if err != nil || !validator.NonNil(principle) || !principle.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthRejectedMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		if validator.NotBlank(projectID) {
			selectedProjectID, err := strconv.ParseUint(projectID, 0, 64)
			if err != nil || !validator.NonZero(selectedProjectID) {
				if validator.NonNil(logger) {
					logger.Errorf(userAuthInvalidProjectIDMessage)
				}
				return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
			}
			if err := principle.SwitchProject(selectedProjectID); err != nil {
				if validator.NonNil(logger) {
					logger.Errorf(userAuthProjectSelectionRejectedMessage)
				}
				return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
			}
		}
		actor, err := types.ResolveAuditActor(principle)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(userAuthInvalidAuditActorMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
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
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
