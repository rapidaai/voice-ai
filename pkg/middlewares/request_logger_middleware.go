package middlewares

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"google.golang.org/grpc"
)

const (
	REQUEST_ID_KEY = "x-request-id"
)

// Request logger middleware
func NewRequestLoggerMiddleware(serviceName string, logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("%s %s [status:%v request:%dms]", c.Request.Method, c.Request.URL, c.Writer.Status(), time.Since(start).Milliseconds())
	}

}

func NewRequestLoggerUnaryServerMiddleware(serviceName string, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		rqId := metadata.ExtractIncoming(ctx).Get(types.REQUEST_ID_KEY)
		if strings.TrimSpace(rqId) == "" {
			rqId = uuid.New().String()
		}
		a, err := handler(context.WithValue(ctx, types.REQUEST_ID_KEY, rqId), req)
		duration := time.Since(start)
		if err != nil {
			logger.Errorf("[Unary] %s %s [requestID:%s status:%v request:%dms]", serviceName, info.FullMethod, rqId, "error", duration.Milliseconds())
			return a, err

		}
		logger.Infof("[Unary] %s %s [requestID:%s status:%v request:%dms]", serviceName, info.FullMethod, rqId, "success", duration.Milliseconds())
		return a, err

	}
}

func NewRequestLoggerStreamServerMiddleware(serviceName string, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := stream.Context()
		rqId := metadata.ExtractIncoming(ctx).Get(types.REQUEST_ID_KEY)
		if strings.TrimSpace(rqId) == "" {
			rqId = uuid.New().String()
		}
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = context.WithValue(ctx, types.REQUEST_ID_KEY, rqId)
		err := handler(srv, wrapped)
		duration := time.Since(start)
		if err != nil {
			logger.Errorf("[stream] %s %s [requestID:%s status:%v request:%dms]", serviceName, info.FullMethod, rqId, "error", duration.Milliseconds())
			return err

		}
		logger.Infof("[stream] %s %s [requestID:%s status:%v request:%dms]", serviceName, info.FullMethod, rqId, "success", duration.Milliseconds())
		return err

	}
}
