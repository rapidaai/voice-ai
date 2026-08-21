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

// NewServiceAuthenticatorUnaryServerMiddleware authenticates only service credentials.
// Deprecated: use NewAuthenticationBoundaryUnaryServerMiddleware.
func NewServiceAuthenticatorUnaryServerMiddleware(resolver types.ClaimAuthenticator[*types.ServiceScope], logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		apiKey := metadata.ExtractIncoming(ctx).Get(types.SERVICE_SCOPE_KEY)
		if strings.TrimSpace(apiKey) == "" {
			return handler(ctx, req)
		}
		auth, err := resolver.Claim(ctx, apiKey)
		if err != nil {
			logAuthenticationFailure(logger, "service credential was rejected")
			return nil, grpcAuthenticationError()
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// NewServiceAuthenticatorStreamServerMiddleware authenticates only service credentials.
// Deprecated: use NewAuthenticationBoundaryStreamServerMiddleware.
func NewServiceAuthenticatorStreamServerMiddleware(resolver types.ClaimAuthenticator[*types.ServiceScope], logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		apiKey := metadata.ExtractIncoming(ctx).Get(types.SERVICE_SCOPE_KEY)
		if strings.TrimSpace(apiKey) == "" {
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}

		auth, err := resolver.Claim(ctx, apiKey)
		if err != nil {
			logAuthenticationFailure(logger, "service credential was rejected")
			return grpcAuthenticationError()
		}

		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
