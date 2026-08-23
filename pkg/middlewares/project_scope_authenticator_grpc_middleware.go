// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"context"
	"strings"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

// NewProjectAuthenticatorUnaryServerMiddleware authenticates project credentials.
func NewProjectAuthenticatorUnaryServerMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		apiKey := strings.TrimSpace(metadata.ExtractIncoming(ctx).Get(types.PROJECT_SCOPE_KEY))
		if apiKey == "" {
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
				logger.Errorf(projectAuthNotSupportedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		apiKey = strings.TrimPrefix(apiKey, types.PROJECT_KEY_PREFIX)
		if apiKey == "" {
			if logger != nil {
				logger.Errorf(projectAuthEmptyMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		principle, err := resolver.Claim(ctx, apiKey)
		if err != nil || principle == nil || principle.Info == nil || !principle.Info.IsAuthenticated() {
			if logger != nil {
				logger.Errorf(projectAuthRejectedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if logger != nil {
				logger.Errorf(projectAuthInvalidAuditActorMessage)
			}
			return nil, status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		projectContext, _ := principle.Info.ProjectContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeProject,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: projectContext.OrganizationID},
			ProjectValue:      &projectContext,
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// NewProjectAuthenticatorStreamServerMiddleware authenticates project credentials.
func NewProjectAuthenticatorStreamServerMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		apiKey := strings.TrimSpace(metadata.ExtractIncoming(ctx).Get(types.PROJECT_SCOPE_KEY))
		if apiKey == "" {
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
				logger.Errorf(projectAuthNotSupportedMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		apiKey = strings.TrimPrefix(apiKey, types.PROJECT_KEY_PREFIX)
		if apiKey == "" {
			if logger != nil {
				logger.Errorf(projectAuthEmptyMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		principle, err := resolver.Claim(ctx, apiKey)
		if err != nil || principle == nil || principle.Info == nil || !principle.Info.IsAuthenticated() {
			if logger != nil {
				logger.Errorf(projectAuthRejectedMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if logger != nil {
				logger.Errorf(projectAuthInvalidAuditActorMessage)
			}
			return status.Error(codes.Unauthenticated, authenticationFailureMessage)
		}
		projectContext, _ := principle.Info.ProjectContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeProject,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: projectContext.OrganizationID},
			ProjectValue:      &projectContext,
		}
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
