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

// NewAuthenticationUnaryServerMiddleware authenticates user credentials.
func NewAuthenticationUnaryServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if validator.NonNil(ctx.Value(types.CTX_)) {
			return handler(ctx, req)
		}

		incoming := metadata.ExtractIncoming(ctx)
		credential := userCredential{
			token:     strings.TrimSpace(incoming.Get(types.AUTHORIZATION_KEY)),
			authID:    strings.TrimSpace(incoming.Get(types.AUTH_KEY)),
			projectID: strings.TrimSpace(incoming.Get(types.PROJECT_KEY)),
		}
		if credential.isEmpty() {
			return handler(ctx, req)
		}

		auth, err := credential.authenticate(ctx, resolver)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf("%s", err)
			}
			return nil, status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// NewAuthenticationStreamServerMiddleware authenticates user credentials.
func NewAuthenticationStreamServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		if validator.NonNil(ctx.Value(types.CTX_)) {
			return handler(srv, stream)
		}

		incoming := metadata.ExtractIncoming(ctx)
		credential := userCredential{
			token:     strings.TrimSpace(incoming.Get(types.AUTHORIZATION_KEY)),
			authID:    strings.TrimSpace(incoming.Get(types.AUTH_KEY)),
			projectID: strings.TrimSpace(incoming.Get(types.PROJECT_KEY)),
		}
		if credential.isEmpty() {
			return handler(srv, stream)
		}

		auth, err := credential.authenticate(ctx, resolver)
		if err != nil {
			if validator.NonNil(logger) {
				logger.Errorf("%s", err)
			}
			return status.Error(codes.Unauthenticated, AuthenticationFailureMessage)
		}

		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
