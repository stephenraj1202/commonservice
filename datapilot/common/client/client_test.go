package client_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"datapilot/common/client"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

func TestNewClient_Timeout(t *testing.T) {
	c := client.NewClient("http://example.com")
	if c.HTTPClient.Timeout.Seconds() != 5 {
		t.Fatalf("expected 5s timeout, got %v", c.HTTPClient.Timeout)
	}
}

func TestDo_ForwardsToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL)
	ctx := client.WithToken(context.Background(), "test-jwt-token")
	resp, err := c.Do(ctx, http.MethodGet, "/ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer test-jwt-token" {
		t.Fatalf("expected 'Bearer test-jwt-token', got %q", gotAuth)
	}
}

func TestDo_NoToken_NoAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL)
	resp, err := c.Do(context.Background(), http.MethodGet, "/ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "" {
		t.Fatalf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestDo_ErrUpstream_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL)
	_, err := c.Do(context.Background(), http.MethodGet, "/missing", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	upErr, ok := err.(client.ErrUpstream)
	if !ok {
		t.Fatalf("expected ErrUpstream, got %T: %v", err, err)
	}
	if upErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", upErr.StatusCode)
	}
	if upErr.Body != "not found" {
		t.Fatalf("expected body 'not found', got %q", upErr.Body)
	}
}

func TestDo_ErrUpstream_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL)
	_, err := c.Do(context.Background(), http.MethodGet, "/boom", nil)
	upErr, ok := err.(client.ErrUpstream)
	if !ok {
		t.Fatalf("expected ErrUpstream, got %T: %v", err, err)
	}
	if upErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", upErr.StatusCode)
	}
}

func TestDo_ErrTimeout(t *testing.T) {
	// Use an already-cancelled context to trigger deadline exceeded immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := client.NewClient("http://192.0.2.1") // TEST-NET, unreachable
	_, err := c.Do(ctx, http.MethodGet, "/", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should be ErrTimeout or a context error.
	if err != client.ErrTimeout && !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected ErrTimeout or context canceled, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Property 9: Inter-service client forwards JWT header
// Feature: datapilot-platform, Property 9: Inter-service client forwards JWT header
// Validates: Requirements 6.2
// ---------------------------------------------------------------------------

func TestProperty9_JWTForwarding(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)

	properties.Property("outbound request carries identical Authorization header", prop.ForAll(
		func(token string) bool {
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			c := client.NewClient(srv.URL)
			ctx := client.WithToken(context.Background(), token)
			resp, err := c.Do(ctx, http.MethodGet, "/", nil)
			if err != nil {
				return false
			}
			resp.Body.Close()

			return gotAuth == "Bearer "+token
		},
		// Generate non-empty token strings (printable ASCII, no spaces for simplicity)
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ---------------------------------------------------------------------------
// Property 10: Inter-service client returns typed error on 4xx/5xx
// Feature: datapilot-platform, Property 10: Inter-service client returns typed error on 4xx/5xx
// Validates: Requirements 6.4
// ---------------------------------------------------------------------------

func TestProperty10_ErrUpstreamTyping(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)

	// Generate status codes in the 4xx and 5xx range.
	statusGen := gen.IntRange(400, 599)

	properties.Property("returns ErrUpstream with exact status code and body", prop.ForAll(
		func(statusCode int, body string) bool {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
				fmt.Fprint(w, body)
			}))
			defer srv.Close()

			c := client.NewClient(srv.URL)
			_, err := c.Do(context.Background(), http.MethodGet, "/", nil)
			if err == nil {
				return false
			}
			upErr, ok := err.(client.ErrUpstream)
			if !ok {
				return false
			}
			// Read body from the error (body may be empty string — that's fine).
			_ = io.Discard
			return upErr.StatusCode == statusCode && upErr.Body == body
		},
		statusGen,
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
