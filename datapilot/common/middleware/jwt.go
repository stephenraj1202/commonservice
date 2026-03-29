package middleware

import (
	"net/http"
	"strings"

	"datapilot/common/errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth returns a Gin middleware that validates a Bearer JWT in the
// Authorization header. On success the parsed jwt.MapClaims are injected into
// the Gin context under the key "claims". On failure the request is aborted
// with HTTP 401 and a standard JSON error body.
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.RespondError(c, http.StatusUnauthorized, "unauthorized", "missing Authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			errors.RespondError(c, http.StatusUnauthorized, "unauthorized", "invalid Authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithExpirationRequired())

		if err != nil || !token.Valid {
			errors.RespondError(c, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			errors.RespondError(c, http.StatusUnauthorized, "unauthorized", "invalid token claims")
			c.Abort()
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}
