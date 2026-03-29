package errors

import (
	"github.com/gin-gonic/gin"
)

// APIError is the standard JSON error response struct used across all services.
type APIError struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// RespondError writes a JSON APIError response. It reads the request_id from
// the Gin context (set by the RequestID middleware) and falls back to an empty
// string when absent.
func RespondError(c *gin.Context, status int, err, message string) {
	requestID, _ := c.Get("request_id")
	rid, _ := requestID.(string)

	c.JSON(status, APIError{
		Error:     err,
		Message:   message,
		RequestID: rid,
	})
}
