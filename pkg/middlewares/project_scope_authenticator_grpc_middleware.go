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

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

// NewProjectAuthenticatorUnaryServerMiddleware authenticates only project credentials.
// Deprecated: use NewAuthenticationBoundaryUnaryServerMiddleware.
func NewProjectAuthenticatorUnaryServerMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		apiKey := metadata.ExtractIncoming(ctx).Get(types.PROJECT_SCOPE_KEY)
		if strings.TrimSpace(apiKey) == "" {
			return handler(ctx, req)
		}
		apiKey = strings.TrimPrefix(strings.TrimSpace(apiKey), types.PROJECT_KEY_PREFIX)
		if apiKey == "" {
			logAuthenticationFailure(logger, "project credential is empty")
			return nil, grpcAuthenticationError()
		}
		auth, err := resolver.Claim(ctx, apiKey)
		if err != nil {
			logAuthenticationFailure(logger, "project credential was rejected")
			return nil, grpcAuthenticationError()
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// NewProjectAuthenticatorStreamServerMiddleware authenticates only project credentials.
// Deprecated: use NewAuthenticationBoundaryStreamServerMiddleware.
func NewProjectAuthenticatorStreamServerMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		apiKey := metadata.ExtractIncoming(ctx).Get(types.PROJECT_SCOPE_KEY)
		if strings.TrimSpace(apiKey) == "" {
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}

		// mutating api keys
		apiKey = strings.TrimPrefix(strings.TrimSpace(apiKey), types.PROJECT_KEY_PREFIX)
		if apiKey == "" {
			logAuthenticationFailure(logger, "project credential is empty")
			return grpcAuthenticationError()
		}
		auth, err := resolver.Claim(ctx, apiKey)
		if err != nil {
			logAuthenticationFailure(logger, "project credential was rejected")
			return grpcAuthenticationError()
		}

		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
