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

// NewOrganizationAuthenticatorUnaryServerMiddleware authenticates organization credentials.
func NewOrganizationAuthenticatorUnaryServerMiddleware(resolver types.ClaimAuthenticator[*types.OrganizationScope], logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		apiKey := strings.TrimSpace(metadata.ExtractIncoming(ctx).Get(types.ORG_SCOPE_KEY))
		if !validator.NotBlank(apiKey) {
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
				logger.Errorf(organizationAuthNotSupportedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		principle, err := resolver.Claim(ctx, apiKey)
		if err != nil || !validator.NonNil(principle) || !validator.NonNil(principle.Info) || !principle.Info.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(organizationAuthRejectedMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(organizationAuthInvalidAuditActorMessage)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		organizationID, _ := principle.Info.OrganizationContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeOrg,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// NewOrganizationAuthenticatorStreamServerMiddleware authenticates organization credentials.
func NewOrganizationAuthenticatorStreamServerMiddleware(resolver types.ClaimAuthenticator[*types.OrganizationScope], logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		apiKey := strings.TrimSpace(metadata.ExtractIncoming(ctx).Get(types.ORG_SCOPE_KEY))
		if !validator.NotBlank(apiKey) {
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
				logger.Errorf(organizationAuthNotSupportedMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		principle, err := resolver.Claim(ctx, apiKey)
		if err != nil || !validator.NonNil(principle) || !validator.NonNil(principle.Info) || !principle.Info.IsAuthenticated() {
			if validator.NonNil(logger) {
				logger.Errorf(organizationAuthRejectedMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		actor, err := types.ResolveAuditActor(principle.Info)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf(organizationAuthInvalidAuditActorMessage)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		organizationID, _ := principle.Info.OrganizationContext()
		auth := &types.Authentication{
			AuthType:          types.AuthTypeOrg,
			ActorValue:        &actor,
			OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		}
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
