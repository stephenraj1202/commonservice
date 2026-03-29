# Implementation Plan: DataPilot Platform

## Overview

Incremental implementation of the DataPilot monorepo: Common Library → API Gateway → File Service → Scheduler Service → Admin UI → Docker Compose wiring → Integration tests. Each task builds on the previous, ending with a fully wired platform launchable via `docker compose up`.

## Tasks

- [x] 1. Scaffold monorepo root structure
  - Create `datapilot/` root directory with `docker-compose.yml` stub, `.env.example`, and `.gitignore`
  - `.env.example` must include: `MYSQL_ROOT_PASSWORD`, `JWT_SECRET`, `FILE_STORAGE_PATH`, `LOG_LEVEL`, `ALLOWED_ORIGINS`, `FILE_SERVICE_URL`, `SCHEDULER_SERVICE_URL`
  - Create empty subdirectories: `common/`, `gateway/`, `file-service/`, `scheduler-service/`, `admin-ui/`
  - _Requirements: 27.1, 27.4_


- [x] 2. Implement Common Library — Go module init and config
  - Create `common/go.mod` with module name `datapilot/common` and add dependencies: `godotenv`, `go.uber.org/zap`, `gorm.io/gorm`, `gorm.io/driver/mysql`, `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`, `github.com/gin-gonic/gin`
  - Implement `common/config/config.go`: `Config` struct with all fields, `LoadConfig()` reading env vars via `os.Getenv` with `godotenv` fallback, returning descriptive error naming any missing required key (`SERVICE_NAME`, `HTTP_PORT`, `MYSQL_DSN`, `JWT_SECRET`)
  - _Requirements: 1.1, 1.2, 1.3_

  - [ ]* 2.1 Write property tests for config loading
    - **Property 1: Config loading round-trip** — for any complete set of valid env vars, `LoadConfig` returns a `Config` whose fields exactly match
    - **Property 2: Missing config key produces named error** — for any absent required key, error message contains that key's name
    - **Validates: Requirements 1.1, 1.2**

- [x] 3. Implement Common Library — structured logger
  - Implement `common/logger/logger.go`: `NewLogger(serviceName, level string) *zap.Logger` using `zap.NewProduction` JSON mode
  - Every log entry must include `timestamp`, `level`, `service`, `trace_id`, `message` fields; suppress DEBUG when level is INFO or higher
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [ ]* 3.1 Write property test for logger output
    - **Property 3: Log entries are valid JSON with required fields** — for any message at any level, output is valid JSON containing all five required fields
    - **Validates: Requirements 2.1, 2.2**

- [x] 4. Implement Common Library — database helper
  - Implement `common/database/database.go`: `InitDB(dsn string, models ...interface{}) (*gorm.DB, error)` opening GORM MySQL connection with `charset=utf8mb4&parseTime=True&loc=Local`
  - Set `SetMaxOpenConns(25)` and `SetMaxIdleConns(10)`; ping with 10-second context deadline; call `db.AutoMigrate(models...)`
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 5. Implement Common Library — error types and middleware
  - Implement `common/errors/errors.go`: `APIError` struct with `error`, `message`, `request_id` JSON fields; `RespondError(c *gin.Context, status int, err, message string)` helper
  - Implement `common/middleware/request_id.go`: `RequestID()` gin.HandlerFunc reading `X-Request-ID` header or generating UUID v4, storing in context and echoing in response header
  - Implement `common/middleware/recovery.go`: `Recovery(logger *zap.Logger) gin.HandlerFunc` catching panics, logging stack trace, responding HTTP 500 with `APIError`
  - Implement `common/middleware/cors.go`: `CORS(allowedOrigins string) gin.HandlerFunc` allowing configured origins, handling OPTIONS with HTTP 204, allowing `Authorization`, `Content-Type`, `X-Request-ID` headers
  - _Requirements: 28.1, 28.2, 28.3, 29.1, 29.2, 29.3_

  - [ ]* 5.1 Write property test for error response structure
    - **Property 33: All error responses contain required JSON fields** — for any error response, body contains `error`, `message`, `request_id` with non-empty `request_id`
    - **Validates: Requirements 28.1, 28.3**

  - [ ]* 5.2 Write property test for CORS headers
    - **Property 34: CORS headers are present for configured origins** — for any request from a domain in `ALLOWED_ORIGINS`, response includes matching `Access-Control-Allow-Origin` and correct `Access-Control-Allow-Headers`
    - **Validates: Requirements 29.1, 29.3**


- [x] 6. Implement Common Library — JWT middleware
  - Implement `common/middleware/jwt.go`: `JWTAuth(secret string) gin.HandlerFunc` extracting `Authorization: Bearer <token>`, validating with `github.com/golang-jwt/jwt/v5`, injecting `jwt.MapClaims` into context under key `"claims"` on success, aborting HTTP 401 with `APIError` on failure
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [ ]* 6.1 Write property test for JWT middleware — valid tokens
    - **Property 4: JWT middleware passes valid tokens and injects claims** — for any JWT signed with correct secret and not expired, middleware allows request and injected claims equal original payload
    - **Validates: Requirements 4.1, 4.2**

  - [ ]* 6.2 Write property test for JWT middleware — invalid/absent tokens
    - **Property 5: JWT middleware rejects invalid or absent tokens with HTTP 401** — for any expired, malformed, wrong-secret, or missing token, middleware aborts with HTTP 401 and JSON error body
    - **Validates: Requirements 4.3, 4.4**

- [x] 7. Implement Common Library — pagination helper
  - Implement `common/pagination/pagination.go`: `Paginate(db *gorm.DB, page, limit int) *gorm.DB` applying LIMIT/OFFSET; cap limit at 100; default page < 1 to page 1
  - Implement `ParseParams(c *gin.Context) (page, limit int)` reading `page` and `limit` query params with safe defaults
  - Define `PagedResponse` struct with `total`, `page`, `limit`, `data` fields
  - _Requirements: 5.1, 5.2, 5.3_

  - [ ]* 7.1 Write property test for pagination LIMIT/OFFSET
    - **Property 6: Pagination applies correct LIMIT and OFFSET** — for any page ≥ 1 and limit 1–100, SQL contains `LIMIT <limit> OFFSET <(page-1)*limit>`
    - **Validates: Requirements 5.1**

  - [ ]* 7.2 Write property test for pagination cap
    - **Property 7: Pagination caps limit at 100** — for any limit > 100, applied limit is exactly 100
    - **Validates: Requirements 5.2**

  - [ ]* 7.3 Write property test for pagination page default
    - **Property 8: Pagination defaults sub-1 page to page 1** — for any page ≤ 0, OFFSET is 0
    - **Validates: Requirements 5.3**

- [x] 8. Implement Common Library — inter-service HTTP client
  - Implement `common/client/client.go`: `Client` struct with `BaseURL` and `*http.Client` (5-second timeout); `NewClient(baseURL string) *Client`; `Do(ctx context.Context, method, path string, body interface{}) (*http.Response, error)`
  - Extract JWT from Gin context and attach as `Authorization: Bearer <token>`; return `ErrTimeout` on deadline exceeded; return `ErrUpstream{StatusCode, Body}` on 4xx/5xx
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [ ]* 8.1 Write property test for JWT forwarding
    - **Property 9: Inter-service client forwards JWT header** — for any JWT in calling context, outbound request carries identical `Authorization: Bearer <token>` header
    - **Validates: Requirements 6.2**

  - [ ]* 8.2 Write property test for upstream error typing
    - **Property 10: Inter-service client returns typed error on 4xx/5xx** — for any 4xx/5xx response, returns `ErrUpstream` containing exact status code and response body
    - **Validates: Requirements 6.4**

- [x] 9. Checkpoint — Common Library
  - Ensure all Common Library tests pass, ask the user if questions arise.


- [x] 10. Implement API Gateway — Go module and auth handler
  - Create `gateway/go.mod` with module name `datapilot/gateway`; add `replace datapilot/common => ../common` directive and all required dependencies
  - Implement `gateway/handlers/auth.go`: `Login` handler querying `users` table, comparing bcrypt hash, returning signed JWT with 24-hour expiry and claims `sub`, `username`, `exp`, `iat`; `Register` handler hashing password with bcrypt cost 12, inserting user row, returning HTTP 201
  - Define `User` GORM model in `gateway/handlers/auth.go` (or `gateway/models/user.go`) with `id`, `username`, `password`, `created_at`, `updated_at`, `deleted_at`
  - _Requirements: 26.1, 26.2, 26.3, 26.4_

  - [ ]* 10.1 Write property test for login JWT validity
    - **Property 30: Login returns valid JWT with 24-hour expiry** — for any registered user with correct credentials, returned JWT is verifiable and `exp` ≈ `iat` + 24h
    - **Validates: Requirements 26.1**

  - [ ]* 10.2 Write property test for invalid credentials
    - **Property 31: Invalid credentials return HTTP 401** — for any non-existent username or wrong password, response is HTTP 401 with JSON error body
    - **Validates: Requirements 26.2**

  - [ ]* 10.3 Write property test for bcrypt storage
    - **Property 32: Passwords are always stored as bcrypt hashes** — for any registration, stored `password` is a valid bcrypt hash; subsequent login with plaintext succeeds
    - **Validates: Requirements 26.3, 26.4**

- [x] 11. Implement API Gateway — reverse proxy and route registration
  - Implement `gateway/proxy/proxy.go`: `NewProxy(target string) gin.HandlerFunc` using `httputil.ReverseProxy`, forwarding all headers including `Authorization`, stripping gateway prefix before forwarding
  - Implement `gateway/main.go`: Gin engine with `RequestID`, `Recovery`, `CORS` middleware; register routes: `GET /health` (no auth), `POST /api/v1/auth/login`, `POST /api/v1/auth/register`, `ANY /api/v1/files/*path` (JWTAuth → File Service proxy), `ANY /api/v1/scheduler/*path` (JWTAuth → Scheduler Service proxy)
  - Health handler checks reachability of `FILE_SERVICE_URL` and `SCHEDULER_SERVICE_URL` and includes each service's status in response body
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 8.1, 8.2, 27.3, 29.1_

  - [ ]* 11.1 Write property test for gateway routing by prefix
    - **Property 11: Gateway proxies requests to correct upstream by prefix** — for any path under `/api/v1/files/*` or `/api/v1/scheduler/*`, request is forwarded to correct upstream with prefix stripped
    - **Validates: Requirements 7.2, 7.3**

  - [ ]* 11.2 Write property test for 404 on unregistered routes
    - **Property 12: Gateway returns 404 for unregistered routes** — for any path not matching a registered prefix, response is HTTP 404 with JSON error body
    - **Validates: Requirements 7.4**

  - [ ]* 11.3 Write property test for Authorization header forwarding
    - **Property 13: Gateway forwards Authorization header unchanged** — for any `Authorization` header value, proxied request carries identical value
    - **Validates: Requirements 7.5**

  - [ ]* 11.4 Write property test for health endpoint upstream reporting
    - **Property 14: Health endpoint reports unreachable services** — for any subset of unreachable upstream services, health response body includes each service's name and non-ok status
    - **Validates: Requirements 8.2**

- [x] 12. Add API Gateway Dockerfile
  - Create `gateway/Dockerfile`: multi-stage build (`golang:1.22-alpine` builder → `alpine` runtime), copy binary, expose port 8080, set entrypoint
  - _Requirements: 27.4_

- [x] 13. Checkpoint — API Gateway
  - Ensure all Gateway tests pass, ask the user if questions arise.


- [x] 14. Implement File Service — Go module, model, and local storage
  - Create `file-service/go.mod` with module name `datapilot/file-service`; add `replace datapilot/common => ../common` directive and dependencies
  - Implement `file-service/models/file_record.go`: `FileRecord` GORM struct with all fields matching the schema (`id`, `original_filename`, `stored_filename`, `mime_type`, `size_bytes`, `uploader_identity`, `storage_path`, `created_at`, `updated_at`); table name `file_records`
  - Implement `file-service/storage/local.go`: `Storage` interface with `Save`, `Open`, `Delete` methods; `LocalStorage{BasePath string}` implementation using `os` package
  - _Requirements: 13.1, 13.2, 13.3_

- [x] 15. Implement File Service — handlers
  - Implement `file-service/handlers/files.go` with four handlers:
    - `Upload`: parse multipart form with 100 MB limit, generate UUID v4 stored filename, write via `LocalStorage.Save`, insert `FileRecord` via GORM, return HTTP 201
    - `Download`: look up `FileRecord` by ID (404 if missing), open file via `LocalStorage.Open` (500 if missing from storage), stream bytes with correct `Content-Type` and `Content-Disposition` headers
    - `List`: use `Paginate` helper, query `file_records` ordered by `created_at DESC`, return `PagedResponse`
    - `Delete`: look up record (404 if missing), delete physical file via `LocalStorage.Delete` (log error but continue), remove DB record, return HTTP 200
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 10.1, 10.2, 10.3, 11.1, 11.2, 11.3, 12.1, 12.2, 12.3_

  - [ ]* 15.1 Write property test for upload metadata persistence
    - **Property 15: File upload persists record with complete metadata** — for any valid multipart upload, resulting `FileRecord` contains correct original filename, MIME type, size, uploader identity, and non-zero timestamp
    - **Validates: Requirements 9.1, 9.3**

  - [ ]* 15.2 Write property test for unique stored filenames
    - **Property 16: Uploaded files receive unique stored filenames** — for any two upload requests (even identical original filenames), `stored_filename` values are distinct
    - **Validates: Requirements 9.2**

  - [ ]* 15.3 Write property test for 100 MB rejection
    - **Property 17: Files larger than 100 MB are rejected with HTTP 413** — for any upload exceeding 100 MB, response is HTTP 413 and no `FileRecord` is created
    - **Validates: Requirements 9.4**

  - [ ]* 15.4 Write property test for download round-trip
    - **Property 18: File download round-trip preserves bytes and headers** — for any successfully uploaded file, download returns exact same bytes with matching `Content-Type` and `Content-Disposition`
    - **Validates: Requirements 10.1**

  - [ ]* 15.5 Write property test for non-existent file ID
    - **Property 19: Non-existent file ID returns HTTP 404** — for any file ID not in MySQL, both download and delete respond HTTP 404 with JSON error body
    - **Validates: Requirements 10.2, 12.2**

  - [ ]* 15.6 Write property test for file list ordering
    - **Property 20: File list is ordered by upload timestamp descending** — for any set of uploaded files, `GET /files` returns records with `created_at` descending and includes `total`, `page`, `limit`, `data` fields
    - **Validates: Requirements 11.1, 11.3**

  - [ ]* 15.7 Write property test for file deletion completeness
    - **Property 21: File deletion removes both storage file and DB record** — for any existing file, after successful DELETE neither physical file nor `FileRecord` row exists
    - **Validates: Requirements 12.1**

- [x] 16. Implement File Service — main.go and Dockerfile
  - Implement `file-service/main.go`: load config, init logger, call `InitDB` with `FileRecord` model, set up Gin with `RequestID`, `Recovery`, `JWTAuth` middleware, register all four file routes, start server
  - Create `file-service/Dockerfile`: multi-stage build, expose port 8081
  - _Requirements: 13.2, 13.3, 27.4_

- [x] 17. Checkpoint — File Service
  - Ensure all File Service tests pass, ask the user if questions arise.


- [x] 18. Implement Scheduler Service — Go module and models
  - Create `scheduler-service/go.mod` with module name `datapilot/scheduler-service`; add `replace datapilot/common => ../common` directive and dependencies including `github.com/robfig/cron/v3`
  - Implement `scheduler-service/models/job.go`: `Job` GORM struct with all fields (`id`, `name`, `cron_expression`, `target_url`, `http_method`, `description`, `status`, `cron_entry_id`, `created_at`, `updated_at`, `deleted_at`); table name `jobs`; indexes on `status` and `deleted_at`
  - Implement `scheduler-service/models/job_execution_log.go`: `JobExecutionLog` GORM struct with all fields (`id`, `job_id`, `status`, `response_code`, `duration_ms`, `error_detail`, `executed_at`); table name `job_execution_logs`; indexes on `job_id` and `executed_at`
  - _Requirements: 19.4, 21.1, 21.3_

- [x] 19. Implement Scheduler Service — cron runner
  - Implement `scheduler-service/runner/cron.go`: `Runner` struct holding `*cron.Cron` (with `cron.WithSeconds()`), `*gorm.DB`, `*zap.Logger`
  - `NewRunner`: initialise cron instance
  - `Register(job *models.Job) error`: add job to cron, store returned `cron.EntryID` back on the `Job` struct; each tick function sends HTTP request to `job.TargetURL` with 10-second timeout, records `JobExecutionLog` with `status`, `response_code`, `duration_ms`, `error_detail`, `executed_at`
  - `Remove(entryID int)`: remove entry from cron
  - `Start()` / `Stop()`: delegate to underlying `*cron.Cron`
  - On startup (called from `main.go`): load all `active` jobs from MySQL and register each
  - _Requirements: 19.1, 19.2, 19.3, 19.4, 21.2_

  - [-]* 19.1 Write property test for execution log outcome
    - **Property 28: Job execution log reflects HTTP outcome** — for any job execution, log entry has `status="success"` for 2xx, `status="failed"` with non-empty `error_detail` for 4xx/5xx/timeout; always records `duration_ms` and `executed_at`
    - **Validates: Requirements 19.2, 19.3**

  - [ ]* 19.2 Write property test for execution log ordering
    - **Property 29: Execution log list is ordered by execution timestamp descending** — for any job with multiple log entries, `GET /scheduler/jobs/:id/logs` returns entries with `executed_at` descending
    - **Validates: Requirements 20.1**

- [x] 20. Implement Scheduler Service — job handlers
  - Implement `scheduler-service/handlers/jobs.go` with seven handlers:
    - `CreateJob`: validate cron expression (HTTP 422 if invalid), persist `Job` with `status="active"`, register with runner, return HTTP 201
    - `ListJobs`: paginated query ordered by `created_at DESC`, support optional `status` filter query param, return `PagedResponse`
    - `UpdateJob`: look up job (404 if missing), validate new cron expression (422 if invalid, leave unchanged), update DB record, re-register with runner
    - `PauseJob`: look up job (404 if missing), set `status="paused"`, remove from runner
    - `ResumeJob`: look up job (404 if missing), set `status="active"`, re-register with runner
    - `DeleteJob`: look up job (404 if missing), soft-delete (`deleted_at`), remove from runner
    - `GetLogs`: look up job (404 if missing), return paginated `JobExecutionLog` ordered by `executed_at DESC`
  - _Requirements: 14.1, 14.2, 14.3, 14.4, 15.1, 15.2, 15.3, 16.1, 16.2, 16.3, 17.1, 17.2, 17.3, 18.1, 18.2, 20.1, 20.2_

  - [ ]* 20.1 Write property test for job creation with active status
    - **Property 22: Job creation persists with active status** — for any valid job creation request, resulting `Job` row has `status="active"` and is registered in cron runner
    - **Validates: Requirements 14.1, 14.3**

  - [ ]* 20.2 Write property test for invalid cron expression rejection
    - **Property 23: Invalid cron expression is rejected with HTTP 422 and leaves state unchanged** — for any invalid cron string in create or update, response is HTTP 422 and no record is created or modified
    - **Validates: Requirements 14.2, 16.3**

  - [ ]* 20.3 Write property test for job list ordering and status filter
    - **Property 24: Job list is ordered by creation timestamp descending with status filter** — for any set of jobs and any optional status filter, response returns only matching jobs ordered by `created_at` descending
    - **Validates: Requirements 15.1, 15.3**

  - [ ]* 20.4 Write property test for job update persistence
    - **Property 25: Job update persists new values** — for any existing job and valid update payload, after successful PUT the `Job` row reflects updated field values
    - **Validates: Requirements 16.1**

  - [ ]* 20.5 Write property test for pause/resume round-trip
    - **Property 26: Pause then resume restores active status** — for any active job, pause sets `status="paused"`, subsequent resume sets `status="active"`
    - **Validates: Requirements 17.1, 17.2**

  - [ ]* 20.6 Write property test for soft-delete behavior
    - **Property 27: Deleted job is soft-deleted and excluded from default listing** — for any existing job, after DELETE its `deleted_at` is non-null and it does not appear in unfiltered job list
    - **Validates: Requirements 18.1**

- [x] 21. Implement Scheduler Service — main.go and Dockerfile
  - Implement `scheduler-service/main.go`: load config, init logger, call `InitDB` with `Job` and `JobExecutionLog` models, create `Runner`, load and register active jobs, set up Gin with middleware, register all seven job routes, start cron runner and HTTP server
  - Create `scheduler-service/Dockerfile`: multi-stage build, expose port 8082
  - _Requirements: 21.1, 21.2, 21.3, 27.4_

- [x] 22. Checkpoint — Scheduler Service
  - Ensure all Scheduler Service tests pass, ask the user if questions arise.


- [x] 23. Scaffold Admin UI — project setup
  - Create `admin-ui/package.json` with dependencies: `react`, `react-dom`, `react-router-dom@6`, `axios`, `zustand`, `antd`, `cronstrue`; devDependencies: `vite`, `@vitejs/plugin-react`, `typescript`, `@types/react`, `@types/react-dom`, `vitest`, `@testing-library/react`, `fast-check`
  - Create `admin-ui/vite.config.ts`: React plugin, server proxy to `http://localhost:8080` for `/api`
  - Create `admin-ui/tsconfig.json` with strict mode, JSX react-jsx
  - Create `admin-ui/src/main.tsx`: mount `<App />` into `#root`
  - Create `admin-ui/src/theme.ts`: Ant Design theme token with `colorPrimary: '#6366f1'`; teal (`#0d9488`) for files section, amber (`#d97706`) for scheduler section
  - _Requirements: 22.2, 27.4_

- [x] 24. Implement Admin UI — auth store and API layer
  - Implement `admin-ui/src/store/auth.ts`: Zustand store with `{ token, user, login(), logout() }`; `login()` stores JWT in localStorage; `logout()` clears localStorage and navigates to `/login`
  - Implement `admin-ui/src/api/auth.ts`: `login(username, password)` and `register(username, password)` using Axios; configure Axios instance with base URL from `VITE_API_BASE_URL`, request interceptor attaching JWT Bearer token, response interceptor calling `logout()` on 401
  - Implement `admin-ui/src/api/files.ts`: `listFiles`, `uploadFile` (with `onProgress` callback for progress bar), `downloadFile` (triggers browser download via blob URL), `deleteFile`
  - Implement `admin-ui/src/api/scheduler.ts`: `listJobs`, `createJob`, `updateJob`, `pauseJob`, `resumeJob`, `deleteJob`, `getJobLogs`
  - _Requirements: 25.1, 25.2, 25.3, 25.4_

- [x] 25. Implement Admin UI — routing and layout
  - Implement `admin-ui/src/App.tsx`: React Router v6 `<BrowserRouter>` with routes: `/login` → `<Login>` (public), `/` → `<Dashboard>` (protected), `/files` → `<Files>` (protected), `/scheduler` → `<Scheduler>` (protected)
  - Protected route wrapper checks for valid non-expired JWT in localStorage; redirects to `/login` if absent or expired
  - Implement `admin-ui/src/components/Layout.tsx`: Ant Design `<Layout>` with sidebar navigation (Dashboard, Files, Scheduler links), top header with logout button
  - _Requirements: 25.1, 25.3_

- [x] 26. Implement Admin UI — Login page
  - Implement `admin-ui/src/pages/Login.tsx`: full-bleed gradient background, Ant Design `<Form>` with username and password fields, submit calls `login()` API, stores JWT via auth store, redirects to `/`; shows error message on HTTP 401
  - _Requirements: 25.1, 25.2_

- [x] 27. Implement Admin UI — Dashboard page
  - Implement `admin-ui/src/components/StatCard.tsx`: reusable card component accepting `title`, `value`, `icon`, `color` props; renders Ant Design `<Card>` with colored icon
  - Implement `admin-ui/src/pages/Dashboard.tsx`: on mount fetch aggregate stats (total file count, total storage MB, total job count, active job count) from API; render four `<StatCard>` components — file stats in teal, scheduler stats in amber
  - _Requirements: 22.1, 22.2, 22.3_

- [x] 28. Implement Admin UI — Files page
  - Implement `admin-ui/src/components/ConfirmDialog.tsx`: reusable Ant Design `<Modal>` confirmation dialog accepting `title`, `message`, `onConfirm`, `onCancel` props
  - Implement `admin-ui/src/components/UploadForm.tsx`: Ant Design `<Upload>` with drag-and-drop, calls `uploadFile` with progress callback, shows `<Progress>` bar, calls `onSuccess` callback on completion
  - Implement `admin-ui/src/components/FileTable.tsx`: Ant Design `<Table>` with columns: filename, MIME type, size, uploader, upload date, actions (download button, delete button); delete triggers `<ConfirmDialog>`; pagination controls
  - Implement `admin-ui/src/pages/Files.tsx`: compose `<UploadForm>` and `<FileTable>`; on upload success refresh file list without full page reload; on delete success remove row without full page reload
  - _Requirements: 23.1, 23.2, 23.3, 23.4, 23.5, 23.6_

- [x] 29. Implement Admin UI — Scheduler page
  - Implement `admin-ui/src/components/JobForm.tsx`: Ant Design `<Form>` with fields: name, cron expression, target URL, HTTP method (select), description; live `cronstrue` description rendered below cron expression input; used for both create and edit
  - Implement `admin-ui/src/components/LogDrawer.tsx`: Ant Design `<Drawer>` showing last 50 execution log entries for a selected job in a table with columns: status, response code, duration, error detail, executed at
  - Implement `admin-ui/src/components/JobTable.tsx`: Ant Design `<Table>` with columns: name, cron expression, target URL, HTTP method, status badge, last execution result, actions (pause/resume/delete buttons, Logs button); failed-status rows highlighted in amber/red; pagination controls
  - Implement `admin-ui/src/pages/Scheduler.tsx`: compose `<JobForm>` (in modal for create), `<JobTable>`, `<LogDrawer>`; wire all CRUD actions; on action success refresh job list without full page reload
  - _Requirements: 24.1, 24.2, 24.3, 24.4, 24.5, 24.6_

- [x] 30. Add Admin UI Dockerfile
  - Create `admin-ui/Dockerfile`: multi-stage build (`node:20-alpine` builder running `vite build` → `nginx:alpine` runtime serving `dist/`); include `nginx.conf` with `try_files $uri /index.html` for SPA routing; expose port 80
  - _Requirements: 27.4_

- [x] 31. Checkpoint — Admin UI
  - Ensure Admin UI builds without TypeScript errors and all component tests pass, ask the user if questions arise.


- [x] 32. Implement full docker-compose.yml
  - Write `datapilot/docker-compose.yml` with services: `mysql` (image `mysql:8.0`, volume `mysql_data`, healthcheck `mysqladmin ping`), `gateway` (build `./gateway`, port 8080, env `MYSQL_DSN`, `FILE_SERVICE_URL`, `SCHEDULER_SERVICE_URL`, `JWT_SECRET`, `ALLOWED_ORIGINS`, depends on mysql healthy), `file-service` (build `./file-service`, port 8081, volume `file_storage:/data/files`, env `MYSQL_DSN`, `FILE_STORAGE_PATH`, `JWT_SECRET`, depends on mysql healthy), `scheduler-service` (build `./scheduler-service`, port 8082, env `MYSQL_DSN`, `JWT_SECRET`, depends on mysql healthy), `admin-ui` (build `./admin-ui`, port 3000:80, env `VITE_API_BASE_URL`)
  - Define named volumes: `mysql_data`, `file_storage`
  - All backend services must declare `depends_on: mysql: condition: service_healthy`
  - _Requirements: 27.4_

- [x] 33. Implement integration test suite
  - Create `datapilot/docker-compose.test.yml` extending `docker-compose.yml` with a `test-runner` service that runs the integration test binary against the live stack
  - Create `datapilot/integration/` Go package with test cases covering:
    - Full auth flow: `POST /api/v1/auth/register` → `POST /api/v1/auth/login` → access protected endpoint with returned JWT
    - File round-trip: upload file → `GET /api/v1/files` (verify in list) → `GET /api/v1/files/:id/download` (verify bytes) → `DELETE /api/v1/files/:id` (verify removed from list)
    - Scheduler lifecycle: create job → `GET /api/v1/scheduler/jobs` (verify active) → pause → verify status paused → resume → verify status active → delete → verify soft-deleted and absent from list
    - Execution log capture: create job with short interval targeting a test HTTP server, wait for execution, verify `GET /api/v1/scheduler/jobs/:id/logs` returns at least one entry with correct status
  - _Requirements: 7.1, 7.2, 7.3, 8.1, 9.1, 10.1, 11.1, 12.1, 14.1, 15.1, 17.1, 17.2, 18.1, 19.2, 20.1_

- [x] 34. Final checkpoint — full platform
  - Ensure all unit tests, property tests, and integration tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for a faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at each service boundary
- Property tests use `github.com/leanovate/gopter` for Go services and `fast-check` for the React Admin UI
- Each property test must include the tag comment: `// Feature: datapilot-platform, Property <N>: <property_text>`
- Each property test runs a minimum of 100 iterations
- Unit tests use `testing` + `testify` for Go; Vitest + React Testing Library for the UI
- New microservices can be added by creating a new Go module and appending a route group in `gateway/main.go` (Requirement 27.3)
