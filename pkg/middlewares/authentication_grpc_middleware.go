// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"context"
	"strconv"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"google.golang.org/grpc"
	// "github.com/rapidaai/pkg/models"
)

func NewAuthenticationUnaryServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		authToken := metadata.ExtractIncoming(ctx).Get(types.AUTHORIZATION_KEY)
		authId := metadata.ExtractIncoming(ctx).Get(types.AUTH_KEY)
		projectId := metadata.ExtractIncoming(ctx).Get(types.PROJECT_KEY)
		if authToken == "" {
			return handler(ctx, req)
		}
		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil {
			logger.Errorf("auth id is not int. passed auth id %s", authId)
			return handler(ctx, req)
		}
		auth, err := resolver.Authorize(ctx, authToken, id)
		if err != nil {
			logger.Errorf("unable to resolve auth token and id with error %v", err)
			return handler(ctx, req)
		}

		if projectId == "" {
			return handler(context.WithValue(ctx, types.CTX_, auth), req)
		}
		pId, err := strconv.ParseUint(projectId, 0, 64)
		if err != nil {
			logger.Errorf("there is project id but not able to resolve with err %v and project id %s", err, projectId)
			return handler(context.WithValue(ctx, types.CTX_, auth), req)
		}

		err = auth.SwitchProject(pId)
		if err != nil {
			logger.Errorf("there is project id but not found in the list of user project")
			return handler(context.WithValue(ctx, types.CTX_, auth), req)
		}

		return handler(context.WithValue(ctx, types.CTX_, auth), req)

	}
}

// StreamServerInterceptor returns a new unary server interceptors that performs per-request auth.
// NOTE(bwplotka): For more complex auth interceptor see https://github.com/grpc/grpc-go/blob/master/authz/grpc_authz_server_interceptors.go.
func NewAuthenticationStreamServerMiddleware(resolver types.Authenticator, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()

		authToken := metadata.ExtractIncoming(ctx).Get(types.AUTHORIZATION_KEY)
		authId := metadata.ExtractIncoming(ctx).Get(types.AUTH_KEY)
		projectId := metadata.ExtractIncoming(ctx).Get(types.PROJECT_KEY)
		logger.Debugf("recieved authentication information %v and %v", authId, authToken)
		if authToken == "" {
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}

		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil {
			logger.Errorf("auth id is not int.")
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}

		auth, err := resolver.Authorize(ctx, authToken, id)
		if err != nil {
			logger.Errorf("unable to resolve auth token and id")
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}

		pId, err := strconv.ParseUint(projectId, 0, 64)
		if err != nil {
			logger.Errorf("there is project id but not able to resolve")
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}

		err = auth.SwitchProject(pId)
		if err != nil {
			logger.Errorf("there is project id but not able to resolve")
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
