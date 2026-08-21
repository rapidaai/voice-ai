package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

const authenticationFailureMessage = "invalid authentication credentials"

type credentialPresence struct {
	user    bool
	project bool
	service bool
}

func (presence credentialPresence) count() int {
	count := 0
	if presence.user {
		count++
	}
	if presence.project {
		count++
	}
	if presence.service {
		count++
	}
	return count
}

func (presence credentialPresence) classes() string {
	classes := make([]string, 0, presence.count())
	if presence.user {
		classes = append(classes, "user")
	}
	if presence.project {
		classes = append(classes, "project")
	}
	if presence.service {
		classes = append(classes, "service")
	}
	return strings.Join(classes, "+")
}

func grpcCredentialPresence(ctx context.Context) credentialPresence {
	incoming := metadata.ExtractIncoming(ctx)
	return credentialPresence{
		user: strings.TrimSpace(incoming.Get(types.AUTHORIZATION_KEY)) != "" ||
			strings.TrimSpace(incoming.Get(types.AUTH_KEY)) != "" ||
			strings.TrimSpace(incoming.Get(types.PROJECT_KEY)) != "",
		project: strings.TrimSpace(incoming.Get(types.PROJECT_SCOPE_KEY)) != "",
		service: strings.TrimSpace(incoming.Get(types.SERVICE_SCOPE_KEY)) != "",
	}
}

func ginCredentialPresence(ctx *gin.Context) credentialPresence {
	userToken, userID, projectID := ginUserCredentials(ctx)
	return credentialPresence{
		user:    strings.TrimSpace(userToken) != "" || strings.TrimSpace(userID) != "" || strings.TrimSpace(projectID) != "",
		project: strings.TrimSpace(ginProjectCredential(ctx)) != "",
	}
}

func ginUserCredentials(ctx *gin.Context) (string, string, string) {
	authToken := ctx.Param(types.AUTHORIZATION_KEY)
	if authToken == "" {
		authToken = ctx.GetHeader(types.AUTHORIZATION_KEY)
	}
	if authToken == "" {
		authToken = ctx.Query(types.AUTHORIZATION_KEY)
	}

	authID := ctx.GetHeader(types.AUTH_KEY)
	if authID == "" {
		authID = ctx.Param(types.AUTH_KEY)
	}
	if authID == "" {
		authID = ctx.Query(types.AUTH_KEY)
	}

	projectID := ctx.GetHeader(types.PROJECT_KEY)
	if projectID == "" {
		projectID = ctx.Param(types.PROJECT_KEY)
	}
	if projectID == "" {
		projectID = ctx.Query(types.PROJECT_KEY)
	}
	return authToken, authID, projectID
}

func ginProjectCredential(ctx *gin.Context) string {
	authToken := ctx.GetHeader(types.PROJECT_SCOPE_KEY)
	if strings.TrimSpace(authToken) == "" {
		authToken = ctx.Query(types.PROJECT_SCOPE_KEY)
	}
	if strings.TrimSpace(authToken) == "" {
		authToken = ctx.Param(types.PROJECT_SCOPE_KEY)
	}
	return authToken
}

func grpcAuthenticationError() error {
	return status.Error(codes.Unauthenticated, authenticationFailureMessage)
}

func logAuthenticationFailure(logger commons.Logger, format string, args ...interface{}) {
	if logger != nil {
		logger.Errorf(format, args...)
	}
}

func abortGinAuthentication(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": authenticationFailureMessage})
}

func NewCredentialConflictUnaryServerMiddleware(logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		presence := grpcCredentialPresence(ctx)
		if presence.count() > 1 {
			logAuthenticationFailure(logger, "authentication credential conflict: classes=%s", presence.classes())
			return nil, grpcAuthenticationError()
		}
		return handler(ctx, req)
	}
}

func NewCredentialConflictStreamServerMiddleware(logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		presence := grpcCredentialPresence(stream.Context())
		if presence.count() > 1 {
			logAuthenticationFailure(logger, "authentication credential conflict: classes=%s", presence.classes())
			return grpcAuthenticationError()
		}
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = stream.Context()
		return handler(srv, wrapped)
	}
}

func NewCredentialConflictMiddleware(logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		presence := ginCredentialPresence(ctx)
		if presence.count() > 1 {
			logAuthenticationFailure(logger, "authentication credential conflict: classes=%s", presence.classes())
			abortGinAuthentication(ctx)
			return
		}
		ctx.Next()
	}
}
