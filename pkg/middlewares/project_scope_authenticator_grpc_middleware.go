package middlewares

import (
	"context"
	"strings"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"google.golang.org/grpc"
)

func NewProjectAuthenticatorUnaryServerMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		apiKey := metadata.ExtractIncoming(ctx).Get(types.PROJECT_SCOPE_KEY)
		if strings.TrimSpace(apiKey) == "" {
			return handler(ctx, req)
		}
		apiKey = strings.Replace(apiKey, types.PROJECT_KEY_PREFIX, "", 1)
		auth, err := resolver.Claim(ctx, apiKey)
		if err != nil {
			logger.Errorf("unable to resolve given api key for project")
			return handler(ctx, req)
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

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
		apiKey = strings.Replace(apiKey, types.PROJECT_KEY_PREFIX, "", 1)
		auth, err := resolver.Claim(ctx, apiKey)
		if err != nil {
			logger.Errorf("unable to resolve auth token and id")
			wrapped := middleware.WrapServerStream(stream)
			wrapped.WrappedContext = ctx
			return handler(srv, wrapped)
		}

		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)

	}
}
