package middlewares

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/types"
	"google.golang.org/grpc"
	// "github.com/lexatic/web-backend/pkg/models"
)

func AuthenticationMiddleware(resolver types.Authenticator, logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the token from the request header
		authToken := c.GetHeader("Authorization")
		authId := c.GetHeader("X-Auth-Id")
		if authToken == "" {
			c.Next() // Continue processing the request without authentication
			return
		}
		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil {
			logger.Errorf("auth id is not int.")
			c.Next()
			return
		}
		auth, err := resolver.Authorize(c, authToken, id)
		if err != nil {
			logger.Errorf("unable to resolve auth token and id")
			c.Next()
			return
		}
		// Attach the user information to the context
		c.Set(types.CTX_, auth)
		// Continue processing the request
		c.Next()
	}
}

func UnaryServerInterceptor(resolver types.Authenticator, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		authToken := metadata.ExtractIncoming(ctx).Get("Authorization")
		authId := metadata.ExtractIncoming(ctx).Get("X-Auth-Id")
		logger.Debugf("recieved authentication information %v and %v", authId, authToken)
		if authToken == "" {
			return handler(ctx, req)
		}
		id, err := strconv.ParseUint(authId, 0, 64)
		if err != nil {
			logger.Errorf("auth id is not int.")
			return handler(ctx, req)
		}
		auth, err := resolver.Authorize(ctx, authToken, id)
		if err != nil {
			logger.Errorf("unable to resolve auth token and id")
			return handler(ctx, req)
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}

// StreamServerInterceptor returns a new unary server interceptors that performs per-request auth.
// NOTE(bwplotka): For more complex auth interceptor see https://github.com/grpc/grpc-go/blob/master/authz/grpc_authz_server_interceptors.go.
func StreamServerInterceptor(resolver types.Authenticator, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()

		authToken := metadata.ExtractIncoming(ctx).Get("Authorization")
		authId := metadata.ExtractIncoming(ctx).Get("X-Auth-Id")
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

		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.CTX_, auth)
		return handler(srv, wrapped)
	}
}
