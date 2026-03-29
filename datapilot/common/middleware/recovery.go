package middleware

import (
	"net/http"
	"runtime/debug"

	customerrors "datapilot/common/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery returns a Gin middleware that catches panics, logs the stack trace
// using the provided zap logger, and responds with HTTP 500 using the standard
// APIError struct.
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Error("panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(stack)),
				)
				customerrors.RespondError(c, http.StatusInternalServerError, "internal_server_error", "an unexpected error occurred")
				c.Abort()
			}
		}()
		c.Next()
	}
}
