// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
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
	// "github.com/rapidaai/pkg/models"
)

// NewAuthenticationUnaryServerMiddleware authenticates only user credentials.
// Deprecated: use NewAuthenticationBoundaryUnaryServerMiddleware.
func NewAuthenticationUnaryServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		authToken := metadata.ExtractIncoming(ctx).Get(types.AUTHORIZATION_KEY)
		authId := metadata.ExtractIncoming(ctx).Get(types.AUTH_KEY)
		projectId := metadata.ExtractIncoming(ctx).Get(types.PROJECT_KEY)
		if strings.TrimSpace(authToken) == "" && strings.TrimSpace(authId) == "" && strings.TrimSpace(projectId) == "" {
			return handler(ctx, req)
		}
		if strings.TrimSpace(authToken) == "" || strings.TrimSpace(authId) == "" {
			logAuthenticationFailure(logger, "user credential is incomplete")
			return nil, grpcAuthenticationError()
		}
		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil || id == 0 {
			logAuthenticationFailure(logger, "user credential has invalid auth id")
			return nil, grpcAuthenticationError()
		}
		auth, err := resolver.Authorize(ctx, authToken, id)
		if err != nil {
			logAuthenticationFailure(logger, "user credential was rejected")
			return nil, grpcAuthenticationError()
		}

		if strings.TrimSpace(projectId) == "" {
			return handler(context.WithValue(ctx, types.CTX_, auth), req)
		}
		pId, err := strconv.ParseUint(projectId, 0, 64)
		if err != nil || pId == 0 {
			logAuthenticationFailure(logger, "user credential has invalid project id")
			return nil, grpcAuthenticationError()
		}

		err = auth.SwitchProject(pId)
		if err != nil {
			logAuthenticationFailure(logger, "user credential project selection was rejected")
			return nil, grpcAuthenticationError()
		}

		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// StreamServerInterceptor returns a new unary server interceptors that performs per-request auth.
// NOTE(bwplotka): For more complex auth interceptor see https://github.com/grpc/grpc-go/blob/master/authz/grpc_authz_server_interceptors.go.
// NewAuthenticationStreamServerMiddleware authenticates only user credentials.
// Deprecated: use NewAuthenticationBoundaryStreamServerMiddleware.
func NewAuthenticationStreamServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		authToken := metadata.ExtractIncoming(ctx).Get(types.AUTHORIZATION_KEY)
		authId := metadata.ExtractIncoming(ctx).Get(types.AUTH_KEY)
		projectId := metadata.ExtractIncoming(ctx).Get(types.PROJECT_KEY)
		if strings.TrimSpace(authToken) == "" && strings.TrimSpace(authId) == "" && strings.TrimSpace(projectId) == "" {
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}
		if strings.TrimSpace(authToken) == "" || strings.TrimSpace(authId) == "" {
			logAuthenticationFailure(logger, "user credential is incomplete")
			return grpcAuthenticationError()
		}

		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil || id == 0 {
			logAuthenticationFailure(logger, "user credential has invalid auth id")
			return grpcAuthenticationError()
		}

		auth, err := resolver.Authorize(ctx, authToken, id)
		if err != nil {
			logAuthenticationFailure(logger, "user credential was rejected")
			return grpcAuthenticationError()
		}

		if strings.TrimSpace(projectId) == "" {
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
			return handler(srv, wrapped)
		}
		pId, err := strconv.ParseUint(projectId, 0, 64)
		if err != nil || pId == 0 {
			logAuthenticationFailure(logger, "user credential has invalid project id")
			return grpcAuthenticationError()
		}

		err = auth.SwitchProject(pId)
		if err != nil {
			logAuthenticationFailure(logger, "user credential project selection was rejected")
			return grpcAuthenticationError()
		}
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
