// Package client provides an inter-service HTTP client that forwards JWTs
// from the calling context and returns typed errors for upstream failures.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// tokenKey is an unexported context key type to avoid collisions.
type tokenKey struct{}

// WithToken returns a new context carrying the given JWT token string.
// Call this before invoking Do so the client can forward the token.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey{}, token)
}

// tokenFromContext extracts the JWT token string stored by WithToken.
func tokenFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(tokenKey{}).(string)
	return v, ok && v != ""
}

// ErrTimeout is returned when the upstream request exceeds the 5-second deadline.
var ErrTimeout = errors.New("upstream request timed out")

// ErrUpstream is returned when the upstream service responds with a 4xx or 5xx
// status code. It carries the exact status code and response body.
type ErrUpstream struct {
	StatusCode int
	Body       string
}

func (e ErrUpstream) Error() string {
	return fmt.Sprintf("upstream error %d: %s", e.StatusCode, e.Body)
}

// Client is a lightweight HTTP client for inter-service calls.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client // 5-second timeout
}

// NewClient creates a Client targeting baseURL with a 5-second timeout.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Do executes an HTTP request against c.BaseURL+path using the given method.
// If body is non-nil it is JSON-encoded as the request body.
// The JWT stored in ctx via WithToken is forwarded as Authorization: Bearer <token>.
// Returns ErrTimeout on context deadline exceeded, ErrUpstream on 4xx/5xx.
func (c *Client) Do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("client: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("client: build request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token, ok := tokenFromContext(ctx); ok {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		// Distinguish timeout / deadline from other network errors.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ErrTimeout
		}
		// http.Client wraps deadline errors; check the URL error too.
		var urlErr interface{ Timeout() bool }
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return nil, ErrTimeout
		}
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return nil, ErrUpstream{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	return resp, nil
}
