package middleware_test

// Feature: datapilot-platform, Property 4: JWT middleware passes valid tokens and injects claims
// Feature: datapilot-platform, Property 5: JWT middleware rejects invalid or absent tokens with HTTP 401

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"datapilot/common/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// buildToken creates a signed JWT with the given secret, subject, and expiry offset.
func buildToken(secret string, sub string, expiresIn time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(expiresIn).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// newTestRouter creates a Gin engine with JWTAuth middleware and a test handler
// that echoes the injected claims as JSON.
func newTestRouter(secret string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.JWTAuth(secret))
	r.GET("/protected", func(c *gin.Context) {
		claims, _ := c.Get("claims")
		c.JSON(http.StatusOK, claims)
	})
	return r
}

// --- Unit tests ---

func TestJWTAuth_ValidToken_Passes(t *testing.T) {
	secret := "test-secret"
	tokenStr, err := buildToken(secret, "user1", time.Hour)
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	r := newTestRouter(secret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuth_MissingHeader_Returns401(t *testing.T) {
	r := newTestRouter("secret")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_ExpiredToken_Returns401(t *testing.T) {
	secret := "test-secret"
	tokenStr, err := buildToken(secret, "user1", -time.Hour)
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	r := newTestRouter(secret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_WrongSecret_Returns401(t *testing.T) {
	tokenStr, err := buildToken("correct-secret", "user1", time.Hour)
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	r := newTestRouter("wrong-secret")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_MalformedToken_Returns401(t *testing.T) {
	r := newTestRouter("secret")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_ClaimsInjected(t *testing.T) {
	secret := "test-secret"
	sub := "user42"
	tokenStr, err := buildToken(secret, sub, time.Hour)
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	r := newTestRouter(secret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if body["sub"] != sub {
		t.Errorf("claims[sub] = %v, want %q", body["sub"], sub)
	}
}

// --- Property tests ---

// nonEmptyAlpha generates non-empty alphanumeric strings.
func nonEmptyAlpha() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 })
}

// Property 4: JWT middleware passes valid tokens and injects claims
// Validates: Requirements 4.1, 4.2
func TestProperty4_ValidTokenPassesAndInjectsClaims(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)

	properties.Property("valid JWT allows request and injects matching claims", prop.ForAll(
		func(secret, sub string) bool {
			tokenStr, err := buildToken(secret, sub, time.Hour)
			if err != nil {
				return false
			}

			r := newTestRouter(secret)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenStr)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				return false
			}
			// Claims must be injected and sub must match
			return body["sub"] == sub
		},
		nonEmptyAlpha(),
		nonEmptyAlpha(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Property 5: JWT middleware rejects invalid or absent tokens with HTTP 401
// Validates: Requirements 4.3, 4.4
func TestProperty5_InvalidOrAbsentTokenReturns401(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)

	// Sub-property 5a: expired tokens are rejected
	properties.Property("expired token returns HTTP 401 with JSON error body", prop.ForAll(
		func(secret, sub string) bool {
			tokenStr, err := buildToken(secret, sub, -time.Hour)
			if err != nil {
				return false
			}

			r := newTestRouter(secret)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenStr)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				return false
			}
			return isJSONErrorBody(w.Body.Bytes())
		},
		nonEmptyAlpha(),
		nonEmptyAlpha(),
	))

	// Sub-property 5b: wrong-secret tokens are rejected
	properties.Property("wrong-secret token returns HTTP 401 with JSON error body", prop.ForAll(
		func(correctSecret, wrongSecret, sub string) bool {
			if correctSecret == wrongSecret {
				return true // skip trivially equal case
			}
			tokenStr, err := buildToken(correctSecret, sub, time.Hour)
			if err != nil {
				return false
			}

			r := newTestRouter(wrongSecret)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenStr)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				return false
			}
			return isJSONErrorBody(w.Body.Bytes())
		},
		nonEmptyAlpha(),
		nonEmptyAlpha(),
		nonEmptyAlpha(),
	))

	// Sub-property 5c: missing Authorization header is rejected
	properties.Property("missing Authorization header returns HTTP 401 with JSON error body", prop.ForAll(
		func(secret string) bool {
			r := newTestRouter(secret)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				return false
			}
			return isJSONErrorBody(w.Body.Bytes())
		},
		nonEmptyAlpha(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// isJSONErrorBody checks that the body is valid JSON containing "error" and "message" fields.
func isJSONErrorBody(body []byte) bool {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	_, hasError := m["error"]
	_, hasMessage := m["message"]
	return hasError && hasMessage
}
