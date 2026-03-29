package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	allowedHeaders = "Authorization, Content-Type, X-Request-ID"
	allowedMethods = "GET, POST, PUT, DELETE, OPTIONS"
)

// CORS returns a Gin middleware that handles Cross-Origin Resource Sharing.
// allowedOrigins is a comma-separated list of allowed origins, or "*" to allow
// all origins. OPTIONS preflight requests are responded to with HTTP 204.
func CORS(allowedOrigins string) gin.HandlerFunc {
	origins := parseOrigins(allowedOrigins)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := resolveOrigin(origin, origins)

		if allowed != "" {
			c.Header("Access-Control-Allow-Origin", allowed)
			c.Header("Access-Control-Allow-Headers", allowedHeaders)
			c.Header("Access-Control-Allow-Methods", allowedMethods)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// parseOrigins splits the comma-separated origins string into a trimmed slice.
func parseOrigins(allowedOrigins string) []string {
	parts := strings.Split(allowedOrigins, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// resolveOrigin returns the value to use for Access-Control-Allow-Origin.
// Returns "*" when the wildcard is configured, the matching origin when found,
// or an empty string when the request origin is not allowed.
func resolveOrigin(requestOrigin string, origins []string) string {
	for _, o := range origins {
		if o == "*" {
			return "*"
		}
		if o == requestOrigin {
			return requestOrigin
		}
	}
	return ""
}
