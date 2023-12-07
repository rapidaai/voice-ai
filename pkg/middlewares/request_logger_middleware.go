package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	commons "github.com/lexatic/web-backend/pkg/commons"
)

// Request logger middleware
func RequestLoggerMiddleware(serviceName string, logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Infof("%s %s [status:%v request:%dms]", c.Request.Method, c.Request.URL, c.Writer.Status(), time.Since(start).Milliseconds())
	}

}
