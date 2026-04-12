package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorLogger is a central Gin middleware that runs after every handler.
// It collects all errors attached via c.Error() and logs them in one
// structured entry with full request context — method, path, latency,
// status, and the originating error message.
//
// This is the ONLY place in the system that logs request-level errors.
// Individual handlers and helpers must NOT log; they must propagate via c.Error().
func ErrorLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)

		// Only log if errors were attached during the request lifecycle
		if len(c.Errors) == 0 {
			return
		}

		for _, ginErr := range c.Errors {
			logger.Error("request error",
				zap.String("method", c.Request.Method),
				zap.String("path", c.FullPath()),
				zap.Int("status", c.Writer.Status()),
				zap.String("latency", latency.String()),
				zap.String("error", ginErr.Error()),
			)
		}
	}
}
