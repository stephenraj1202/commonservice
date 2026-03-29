// Package integration contains end-to-end tests that run against the live
// DataPilot stack (gateway + file-service + scheduler-service + MySQL).
//
// Set GATEWAY_URL to override the default base URL (http://localhost:8080).
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func gatewayURL() string {
	if u := os.Getenv("GATEWAY_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://localhost:8080"
}

// apiURL builds a full URL from a path relative to the gateway.
func apiURL(path string) string {
	return gatewayURL() + path
}

// doJSON sends a JSON request and returns the response.
func doJSON(t *testing.T, method, url string, body interface{}, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// readBody reads and closes the response body, returning the raw bytes.
func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return b
}

// decodeJSON reads the response body and unmarshals it into dst.
func decodeJSON(t *testing.T, resp *http.Response, dst interface{}) {
	t.Helper()
	b := readBody(t, resp)
	require.NoError(t, json.Unmarshal(b, dst), "body: %s", string(b))
}

// uniqueUsername returns a username that is unlikely to collide across test runs.
func uniqueUsername(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// ---------------------------------------------------------------------------
// Auth helpers
// ---------------------------------------------------------------------------

type authResponse struct {
	Token string `json:"token"`
}

// registerAndLogin creates a new user and returns the JWT token.
func registerAndLogin(t *testing.T, username, password string) string {
	t.Helper()

	creds := map[string]string{"username": username, "password": password}

	// Register
	resp := doJSON(t, http.MethodPost, apiURL("/api/v1/auth/register"), creds, "")
	readBody(t, resp) // drain
	require.Equal(t, http.StatusCreated, resp.StatusCode, "register should return 201")

	// Login
	resp = doJSON(t, http.MethodPost, apiURL("/api/v1/auth/login"), creds, "")
	var ar authResponse
	decodeJSON(t, resp, &ar)
	require.Equal(t, http.StatusOK, resp.StatusCode, "login should return 200")
	require.NotEmpty(t, ar.Token, "login should return a non-empty token")

	return ar.Token
}

// ---------------------------------------------------------------------------
// Test: Auth flow
// ---------------------------------------------------------------------------

// TestAuthFlow verifies the full register → login → access protected endpoint flow.
func TestAuthFlow(t *testing.T) {
	username := uniqueUsername("authuser")
	password := "s3cr3tP@ss"

	// 1. Register
	creds := map[string]string{"username": username, "password": password}
	resp := doJSON(t, http.MethodPost, apiURL("/api/v1/auth/register"), creds, "")
	body := readBody(t, resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "register: %s", string(body))

	// 2. Login
	resp = doJSON(t, http.MethodPost, apiURL("/api/v1/auth/login"), creds, "")
	var ar authResponse
	decodeJSON(t, resp, &ar)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, ar.Token)

	// 3. Access a protected endpoint with the returned JWT
	resp = doJSON(t, http.MethodGet, apiURL("/api/v1/files"), nil, ar.Token)
	body = readBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, "protected endpoint with valid JWT: %s", string(body))

	// 4. Access a protected endpoint without a JWT → 401
	resp = doJSON(t, http.MethodGet, apiURL("/api/v1/files"), nil, "")
	readBody(t, resp)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "protected endpoint without JWT should be 401")
}

// ---------------------------------------------------------------------------
// Test: File round-trip
// ---------------------------------------------------------------------------

type fileRecord struct {
	ID               uint   `json:"id"`
	OriginalFilename string `json:"original_filename"`
	MIMEType         string `json:"mime_type"`
	SizeBytes        int64  `json:"size_bytes"`
}

type pagedFiles struct {
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
	Data  []fileRecord `json:"data"`
}

// uploadFile uploads fileContent as a multipart form and returns the created FileRecord.
func uploadFile(t *testing.T, token, filename string, content []byte) fileRecord {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	mw.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		apiURL("/api/v1/files/upload"), &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	var rec fileRecord
	decodeJSON(t, resp, &rec)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotZero(t, rec.ID)
	return rec
}

// TestFileRoundTrip uploads a file, verifies it appears in the list, downloads
// it and checks the bytes, then deletes it and verifies it is gone.
func TestFileRoundTrip(t *testing.T) {
	token := registerAndLogin(t, uniqueUsername("fileuser"), "p@ssw0rd!")

	fileContent := []byte("hello datapilot integration test")
	filename := "test_integration.txt"

	// 1. Upload
	rec := uploadFile(t, token, filename, fileContent)
	require.Equal(t, filename, rec.OriginalFilename)

	// 2. Verify in list
	resp := doJSON(t, http.MethodGet, apiURL("/api/v1/files"), nil, token)
	var paged pagedFiles
	decodeJSON(t, resp, &paged)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	found := false
	for _, f := range paged.Data {
		if f.ID == rec.ID {
			found = true
			break
		}
	}
	require.True(t, found, "uploaded file should appear in list")

	// 3. Download and verify bytes
	resp = doJSON(t, http.MethodGet, apiURL(fmt.Sprintf("/api/v1/files/%d/download", rec.ID)), nil, token)
	downloaded := readBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, fileContent, downloaded, "downloaded bytes should match uploaded bytes")

	// 4. Delete
	resp = doJSON(t, http.MethodDelete, apiURL(fmt.Sprintf("/api/v1/files/%d", rec.ID)), nil, token)
	readBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 5. Verify removed from list
	resp = doJSON(t, http.MethodGet, apiURL("/api/v1/files"), nil, token)
	var pagedAfter pagedFiles
	decodeJSON(t, resp, &pagedAfter)
	for _, f := range pagedAfter.Data {
		require.NotEqual(t, rec.ID, f.ID, "deleted file should not appear in list")
	}
}

// ---------------------------------------------------------------------------
// Test: Scheduler lifecycle
// ---------------------------------------------------------------------------

type job struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	CronExpression string `json:"cron_expression"`
	TargetURL      string `json:"target_url"`
	HTTPMethod     string `json:"http_method"`
	Status         string `json:"status"`
}

type pagedJobs struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Data  []job `json:"data"`
}

// createJob creates a scheduler job and returns it.
func createJob(t *testing.T, token string, payload map[string]string) job {
	t.Helper()
	resp := doJSON(t, http.MethodPost, apiURL("/api/v1/scheduler/jobs"), payload, token)
	var j job
	decodeJSON(t, resp, &j)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotZero(t, j.ID)
	return j
}

// findJobInList returns the job with the given ID from the list, or nil.
func findJobInList(t *testing.T, token string, jobID uint) *job {
	t.Helper()
	resp := doJSON(t, http.MethodGet, apiURL("/api/v1/scheduler/jobs"), nil, token)
	var paged pagedJobs
	decodeJSON(t, resp, &paged)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	for _, j := range paged.Data {
		if j.ID == jobID {
			jCopy := j
			return &jCopy
		}
	}
	return nil
}

// TestSchedulerLifecycle creates a job, pauses it, resumes it, and deletes it,
// verifying the status at each step.
func TestSchedulerLifecycle(t *testing.T) {
	token := registerAndLogin(t, uniqueUsername("scheduser"), "p@ssw0rd!")

	payload := map[string]string{
		"name":            "integration-test-job",
		"cron_expression": "0 0 * * *", // daily at midnight — won't fire during test
		"target_url":      "http://example.com",
		"http_method":     "GET",
		"description":     "integration test job",
	}

	// 1. Create job
	j := createJob(t, token, payload)
	require.Equal(t, "active", j.Status)

	// 2. Verify active in list
	found := findJobInList(t, token, j.ID)
	require.NotNil(t, found, "job should appear in list after creation")
	require.Equal(t, "active", found.Status)

	// 3. Pause
	resp := doJSON(t, http.MethodPost, apiURL(fmt.Sprintf("/api/v1/scheduler/jobs/%d/pause", j.ID)), nil, token)
	var paused job
	decodeJSON(t, resp, &paused)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "paused", paused.Status)

	// 4. Verify status paused in list
	found = findJobInList(t, token, j.ID)
	require.NotNil(t, found)
	require.Equal(t, "paused", found.Status)

	// 5. Resume
	resp = doJSON(t, http.MethodPost, apiURL(fmt.Sprintf("/api/v1/scheduler/jobs/%d/resume", j.ID)), nil, token)
	var resumed job
	decodeJSON(t, resp, &resumed)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "active", resumed.Status)

	// 6. Verify status active in list
	found = findJobInList(t, token, j.ID)
	require.NotNil(t, found)
	require.Equal(t, "active", found.Status)

	// 7. Delete
	resp = doJSON(t, http.MethodDelete, apiURL(fmt.Sprintf("/api/v1/scheduler/jobs/%d", j.ID)), nil, token)
	readBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 8. Verify soft-deleted and absent from default list
	found = findJobInList(t, token, j.ID)
	require.Nil(t, found, "deleted job should not appear in default list")
}

// ---------------------------------------------------------------------------
// Test: Execution log capture
// ---------------------------------------------------------------------------

type execLog struct {
	ID           uint      `json:"id"`
	JobID        uint      `json:"job_id"`
	Status       string    `json:"status"`
	ResponseCode int       `json:"response_code"`
	DurationMS   int64     `json:"duration_ms"`
	ExecutedAt   time.Time `json:"executed_at"`
}

type pagedLogs struct {
	Total int64     `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
	Data  []execLog `json:"data"`
}

// TestExecutionLogCapture creates a job that fires every second against a local
// test HTTP server, waits for at least one execution, and verifies the log entry.
func TestExecutionLogCapture(t *testing.T) {
	// Start a local HTTP server that the scheduler can call.
	// The server must be reachable from inside the Docker network, so we bind
	// on all interfaces and use the host's IP as seen from the container.
	// When running outside Docker (e.g. local dev), localhost works fine.
	targetHit := make(chan struct{}, 10)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case targetHit <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Listen on all interfaces so the scheduler container can reach us.
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	srv.Listener = listener
	srv.Start()
	defer srv.Close()

	// Determine the target URL. Inside Docker the test-runner container needs
	// to use its own IP; outside Docker localhost works.
	targetURL := srv.URL
	if dockerHost := os.Getenv("DOCKER_HOST_IP"); dockerHost != "" {
		_, port, _ := net.SplitHostPort(listener.Addr().String())
		targetURL = fmt.Sprintf("http://%s:%s", dockerHost, port)
	}

	token := registerAndLogin(t, uniqueUsername("loguser"), "p@ssw0rd!")

	payload := map[string]string{
		"name":            "log-capture-job",
		"cron_expression": "*/1 * * * * *", // every second (6-field with seconds)
		"target_url":      targetURL,
		"http_method":     "GET",
		"description":     "execution log integration test",
	}

	j := createJob(t, token, payload)
	defer func() {
		// Clean up: delete the job so it stops firing.
		resp := doJSON(t, http.MethodDelete, apiURL(fmt.Sprintf("/api/v1/scheduler/jobs/%d", j.ID)), nil, token)
		readBody(t, resp)
	}()

	// Wait up to 15 seconds for at least one execution log entry.
	deadline := time.Now().Add(15 * time.Second)
	var logs []execLog
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		resp := doJSON(t, http.MethodGet, apiURL(fmt.Sprintf("/api/v1/scheduler/jobs/%d/logs", j.ID)), nil, token)
		var paged pagedLogs
		decodeJSON(t, resp, &paged)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		if len(paged.Data) > 0 {
			logs = paged.Data
			break
		}
	}

	require.NotEmpty(t, logs, "expected at least one execution log entry within 15 seconds")

	entry := logs[0]
	require.Equal(t, j.ID, entry.JobID)
	require.Equal(t, "success", entry.Status, "log entry status should be 'success' for HTTP 200 target")
	require.Positive(t, entry.DurationMS, "duration_ms should be positive")
	require.False(t, entry.ExecutedAt.IsZero(), "executed_at should be set")
}
