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
	"github.com/rapidaai/pkg/validator"
)

// NewServiceAuthenticatorUnaryServerMiddleware authenticates service credentials.
func NewServiceAuthenticatorUnaryServerMiddleware(resolver types.ClaimAuthenticator[*types.ServiceScope], logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		assertion := strings.TrimSpace(metadata.ExtractIncoming(ctx).Get(types.SERVICE_SCOPE_KEY))
		if !validator.NotBlank(assertion) {
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
				logger.Errorf(serviceAuthNotSupportedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		principle, err := resolver.Claim(ctx, assertion)
		if err != nil || !validator.NonNil(principle) || !validator.NonNil(principle.Info) || !principle.Info.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(serviceAuthRejectedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(serviceAuthInvalidAuditActorMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
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
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// NewServiceAuthenticatorStreamServerMiddleware authenticates service credentials.
func NewServiceAuthenticatorStreamServerMiddleware(resolver types.ClaimAuthenticator[*types.ServiceScope], logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		assertion := strings.TrimSpace(metadata.ExtractIncoming(ctx).Get(types.SERVICE_SCOPE_KEY))
		if !validator.NotBlank(assertion) {
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
				logger.Errorf(serviceAuthNotSupportedMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		principle, err := resolver.Claim(ctx, assertion)
		if err != nil || !validator.NonNil(principle) || !validator.NonNil(principle.Info) || !principle.Info.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(serviceAuthRejectedMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(serviceAuthInvalidAuditActorMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
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
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
