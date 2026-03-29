# Requirements Document

## Introduction

DataPilot is an enterprise microservices platform designed for a solo entrepreneur. It provides a
file management service, a cron/scheduler service, a shared common utility library, and a React
admin UI — all backed by Go (Gin + GORM) microservices and MySQL. The platform is designed to be
simple to run locally or on a single server, while remaining extensible for future DataPilot
services.

---

## Glossary

- **Platform**: The full DataPilot system comprising all microservices, shared libraries, and the
  admin frontend.
- **API_Gateway**: The single entry-point reverse proxy that routes HTTP requests to the correct
  microservice.
- **File_Service**: The microservice responsible for file upload, download, listing, and deletion.
- **Scheduler_Service**: The microservice responsible for creating, updating, listing, and
  executing cron jobs.
- **Common_Library**: The shared Go module containing utilities (config loader, logger, DB helper,
  JWT middleware, error types, pagination) used by all microservices.
- **Admin_UI**: The React single-page application that provides a colorful dashboard for managing
  files and cron jobs.
- **MySQL**: The relational database used by all microservices for persistent storage.
- **Job**: A scheduled task registered in the Scheduler_Service, defined by a cron expression and
  a target action.
- **File_Record**: A MySQL row that stores metadata (name, size, MIME type, storage path, uploader,
  timestamps) for an uploaded file.
- **JWT**: JSON Web Token used for authenticating requests between services and from the Admin_UI.
- **Inter_Service_Client**: The HTTP client in the Common_Library used by any microservice to call
  another microservice.

---

## Requirements

### Requirement 1: Common Library — Configuration

**User Story:** As a developer, I want a shared configuration loader, so that every microservice
reads its settings from a single, consistent source without duplicating code.

#### Acceptance Criteria

1. THE Common_Library SHALL expose a `LoadConfig` function that reads environment variables and an
   optional `.env` file and returns a typed configuration struct.
2. WHEN a required environment variable is missing, THE Common_Library SHALL return a descriptive
   error identifying the missing variable.
3. THE Common_Library SHALL support configuration keys for: service name, HTTP port, MySQL DSN,
   JWT secret, file storage path, and log level.

---

### Requirement 2: Common Library — Structured Logging

**User Story:** As a developer, I want structured JSON logging, so that I can trace requests
across microservices in production.

#### Acceptance Criteria

1. THE Common_Library SHALL provide a logger that emits structured JSON log lines to stdout.
2. WHEN a log entry is created, THE Common_Library SHALL include fields: `timestamp`, `level`,
   `service`, `trace_id`, and `message`.
3. THE Common_Library SHALL support log levels: DEBUG, INFO, WARN, ERROR.
4. WHEN the configured log level is INFO, THE Common_Library SHALL suppress DEBUG log entries.

---

### Requirement 3: Common Library — MySQL Connection Helper

**User Story:** As a developer, I want a shared database initialiser, so that every microservice
connects to MySQL with consistent settings and connection pooling.

#### Acceptance Criteria

1. THE Common_Library SHALL provide an `InitDB` function that opens a GORM MySQL connection using
   the supplied DSN.
2. WHEN `InitDB` is called, THE Common_Library SHALL set a maximum of 25 open connections and 10
   idle connections.
3. WHEN the MySQL server is unreachable, THE Common_Library SHALL return an error within 10
   seconds.
4. THE Common_Library SHALL run `AutoMigrate` for all registered model structs passed to `InitDB`.

---

### Requirement 4: Common Library — JWT Middleware

**User Story:** As a developer, I want a reusable JWT authentication middleware, so that every
microservice can protect its endpoints without duplicating auth logic.

#### Acceptance Criteria

1. THE Common_Library SHALL provide a Gin middleware function `JWTAuth` that validates a Bearer
   token in the `Authorization` header.
2. WHEN a request carries a valid JWT, THE Common_Library SHALL inject the parsed claims into the
   Gin context under the key `claims`.
3. WHEN a request carries an expired or invalid JWT, THE Common_Library SHALL respond with HTTP
   401 and a JSON error body.
4. WHEN a request has no `Authorization` header, THE Common_Library SHALL respond with HTTP 401.

---

### Requirement 5: Common Library — Pagination Helper

**User Story:** As a developer, I want a shared pagination utility, so that all list endpoints
return consistent page/limit/total metadata.

#### Acceptance Criteria

1. THE Common_Library SHALL provide a `Paginate` function that accepts a GORM `*gorm.DB`, a page
   number, and a page size, and returns a scoped `*gorm.DB` with LIMIT and OFFSET applied.
2. THE Common_Library SHALL cap the maximum page size at 100 records per page.
3. WHEN a page number less than 1 is supplied, THE Common_Library SHALL default to page 1.

---

### Requirement 6: Common Library — Inter-Service HTTP Client

**User Story:** As a developer, I want a shared HTTP client, so that any microservice can call
another microservice with automatic JWT forwarding and timeout handling.

#### Acceptance Criteria

1. THE Inter_Service_Client SHALL accept a base URL, an endpoint path, an HTTP method, and an
   optional request body.
2. WHEN making an outbound request, THE Inter_Service_Client SHALL attach the caller's JWT from
   context as a Bearer token.
3. WHEN the target service does not respond within 5 seconds, THE Inter_Service_Client SHALL
   return a timeout error.
4. WHEN the target service returns HTTP 4xx or 5xx, THE Inter_Service_Client SHALL return a typed
   error containing the status code and response body.

---

### Requirement 7: API Gateway — Routing

**User Story:** As a client, I want a single base URL, so that I do not need to know the internal
addresses of individual microservices.

#### Acceptance Criteria

1. THE API_Gateway SHALL expose all microservice routes under a versioned prefix `/api/v1/`.
2. WHEN a request path matches `/api/v1/files/*`, THE API_Gateway SHALL proxy the request to the
   File_Service.
3. WHEN a request path matches `/api/v1/scheduler/*`, THE API_Gateway SHALL proxy the request to
   the Scheduler_Service.
4. WHEN a request path does not match any registered route, THE API_Gateway SHALL respond with
   HTTP 404 and a JSON error body.
5. THE API_Gateway SHALL forward the original `Authorization` header to the upstream service
   unchanged.

---

### Requirement 8: API Gateway — Health Check

**User Story:** As an operator, I want a health endpoint, so that I can verify the platform is
running without authentication.

#### Acceptance Criteria

1. THE API_Gateway SHALL expose `GET /health` that returns HTTP 200 and a JSON body
   `{"status":"ok"}` without requiring authentication.
2. WHEN any registered upstream service is unreachable, THE API_Gateway SHALL include that
   service's name and status in the health response body.

---

### Requirement 9: File Service — Upload

**User Story:** As a user, I want to upload files through the platform, so that I can store and
retrieve them later.

#### Acceptance Criteria

1. WHEN a `POST /api/v1/files/upload` request is received with a valid multipart file, THE
   File_Service SHALL store the file on the configured storage path and save a File_Record to
   MySQL.
2. WHEN a file is stored, THE File_Service SHALL generate a unique UUID-based filename to prevent
   collisions.
3. WHEN a file is stored, THE File_Service SHALL record: original filename, stored filename, MIME
   type, file size in bytes, uploader identity from JWT claims, and upload timestamp.
4. WHEN the uploaded file exceeds 100 MB, THE File_Service SHALL reject the request with HTTP 413
   and a JSON error body.
5. WHEN the storage path is not writable, THE File_Service SHALL return HTTP 500 and log the
   error.
6. WHEN a file is successfully uploaded, THE File_Service SHALL respond with HTTP 201 and the
   created File_Record as JSON.

---

### Requirement 10: File Service — Download

**User Story:** As a user, I want to download a stored file by its ID, so that I can retrieve
previously uploaded content.

#### Acceptance Criteria

1. WHEN a `GET /api/v1/files/:id/download` request is received with a valid file ID, THE
   File_Service SHALL stream the file bytes to the client with the correct `Content-Type` and
   `Content-Disposition` headers.
2. WHEN the requested file ID does not exist in MySQL, THE File_Service SHALL respond with HTTP
   404 and a JSON error body.
3. WHEN the file exists in MySQL but the physical file is missing from storage, THE File_Service
   SHALL respond with HTTP 500, log the inconsistency, and include a descriptive error message.

---

### Requirement 11: File Service — List Files

**User Story:** As an admin, I want to list all uploaded files with pagination, so that I can
browse and manage stored content.

#### Acceptance Criteria

1. WHEN a `GET /api/v1/files` request is received, THE File_Service SHALL return a paginated JSON
   list of File_Records ordered by upload timestamp descending.
2. THE File_Service SHALL accept `page` and `limit` query parameters and apply them using the
   Common_Library Paginate helper.
3. THE File_Service SHALL include `total`, `page`, `limit`, and `data` fields in the response
   body.

---

### Requirement 12: File Service — Delete File

**User Story:** As an admin, I want to delete a file by its ID, so that I can remove unwanted
content and free storage space.

#### Acceptance Criteria

1. WHEN a `DELETE /api/v1/files/:id` request is received with a valid file ID, THE File_Service
   SHALL delete the physical file from storage and remove the File_Record from MySQL.
2. WHEN the file ID does not exist, THE File_Service SHALL respond with HTTP 404.
3. WHEN the physical file cannot be deleted, THE File_Service SHALL still remove the File_Record
   from MySQL, log the storage error, and respond with HTTP 200.

---

### Requirement 13: File Service — Persistence

**User Story:** As an operator, I want all file metadata stored in MySQL, so that file records
survive service restarts.

#### Acceptance Criteria

1. THE File_Service SHALL use GORM to persist File_Records in a MySQL table named `file_records`.
2. THE File_Service SHALL use the Common_Library `InitDB` function to initialise the database
   connection.
3. WHEN the File_Service starts, THE File_Service SHALL run GORM AutoMigrate on the
   `file_records` table.

---

### Requirement 14: Scheduler Service — Create Job

**User Story:** As a developer or admin, I want to create a cron job from any service or the UI,
so that I can schedule recurring tasks without manual intervention.

#### Acceptance Criteria

1. WHEN a `POST /api/v1/scheduler/jobs` request is received with a valid JSON body containing
   `name`, `cron_expression`, `target_url`, and `http_method`, THE Scheduler_Service SHALL
   persist the Job to MySQL and register it with the in-process cron runner.
2. WHEN the supplied `cron_expression` is not a valid 5-field or 6-field cron string, THE
   Scheduler_Service SHALL respond with HTTP 422 and a JSON error body describing the invalid
   expression.
3. WHEN a Job is created, THE Scheduler_Service SHALL set its initial status to `active`.
4. WHEN a Job is created, THE Scheduler_Service SHALL respond with HTTP 201 and the created Job
   record as JSON.

---

### Requirement 15: Scheduler Service — List Jobs

**User Story:** As an admin, I want to list all scheduled jobs with their status, so that I can
monitor what is running.

#### Acceptance Criteria

1. WHEN a `GET /api/v1/scheduler/jobs` request is received, THE Scheduler_Service SHALL return a
   paginated JSON list of Jobs ordered by creation timestamp descending.
2. THE Scheduler_Service SHALL include `total`, `page`, `limit`, and `data` fields in the
   response body.
3. THE Scheduler_Service SHALL accept an optional `status` query parameter to filter jobs by
   `active`, `paused`, or `deleted`.

---

### Requirement 16: Scheduler Service — Update Job

**User Story:** As an admin, I want to update a job's schedule or target, so that I can adjust
automation without recreating it.

#### Acceptance Criteria

1. WHEN a `PUT /api/v1/scheduler/jobs/:id` request is received with valid fields, THE
   Scheduler_Service SHALL update the Job record in MySQL and re-register the job with the new
   cron expression.
2. WHEN the job ID does not exist, THE Scheduler_Service SHALL respond with HTTP 404.
3. WHEN the updated `cron_expression` is invalid, THE Scheduler_Service SHALL respond with HTTP
   422 and leave the existing job unchanged.

---

### Requirement 17: Scheduler Service — Pause and Resume Job

**User Story:** As an admin, I want to pause and resume jobs, so that I can temporarily stop
automation without deleting it.

#### Acceptance Criteria

1. WHEN a `POST /api/v1/scheduler/jobs/:id/pause` request is received, THE Scheduler_Service
   SHALL set the Job status to `paused` and remove it from the active cron runner.
2. WHEN a `POST /api/v1/scheduler/jobs/:id/resume` request is received, THE Scheduler_Service
   SHALL set the Job status to `active` and re-register it with the cron runner.
3. WHEN a pause or resume request targets a non-existent job ID, THE Scheduler_Service SHALL
   respond with HTTP 404.

---

### Requirement 18: Scheduler Service — Delete Job

**User Story:** As an admin, I want to delete a job, so that I can permanently remove automation
that is no longer needed.

#### Acceptance Criteria

1. WHEN a `DELETE /api/v1/scheduler/jobs/:id` request is received, THE Scheduler_Service SHALL
   soft-delete the Job record in MySQL (set `deleted_at`) and remove it from the cron runner.
2. WHEN the job ID does not exist, THE Scheduler_Service SHALL respond with HTTP 404.

---

### Requirement 19: Scheduler Service — Job Execution

**User Story:** As an operator, I want the scheduler to execute jobs on time and log results, so
that I can audit what ran and whether it succeeded.

#### Acceptance Criteria

1. WHEN a Job's cron expression fires, THE Scheduler_Service SHALL send an HTTP request to the
   Job's `target_url` using the Job's `http_method`.
2. WHEN the target URL responds with HTTP 2xx, THE Scheduler_Service SHALL record an execution
   log entry with status `success`, response code, and duration in milliseconds.
3. WHEN the target URL responds with HTTP 4xx or 5xx, or does not respond within 10 seconds, THE
   Scheduler_Service SHALL record an execution log entry with status `failed` and the error
   detail.
4. THE Scheduler_Service SHALL persist execution log entries in a MySQL table named
   `job_execution_logs`.

---

### Requirement 20: Scheduler Service — Execution Log Retrieval

**User Story:** As an admin, I want to view execution history for a job, so that I can debug
failures.

#### Acceptance Criteria

1. WHEN a `GET /api/v1/scheduler/jobs/:id/logs` request is received, THE Scheduler_Service SHALL
   return a paginated list of execution log entries for that job ordered by execution timestamp
   descending.
2. WHEN the job ID does not exist, THE Scheduler_Service SHALL respond with HTTP 404.

---

### Requirement 21: Scheduler Service — Persistence

**User Story:** As an operator, I want all job definitions and execution logs stored in MySQL, so
that they survive service restarts.

#### Acceptance Criteria

1. THE Scheduler_Service SHALL use GORM to persist Jobs in a MySQL table named `jobs`.
2. WHEN the Scheduler_Service starts, THE Scheduler_Service SHALL load all Jobs with status
   `active` from MySQL and register them with the cron runner.
3. THE Scheduler_Service SHALL use the Common_Library `InitDB` function to initialise the
   database connection.

---

### Requirement 22: Admin UI — Dashboard

**User Story:** As an admin, I want a colorful dashboard overview, so that I can see platform
health at a glance.

#### Acceptance Criteria

1. THE Admin_UI SHALL display a dashboard page showing: total file count, total storage used in
   MB, total job count, and count of active jobs.
2. THE Admin_UI SHALL use a distinct color theme per section (files section and scheduler section
   use visually differentiated colors).
3. WHEN the Admin_UI loads, THE Admin_UI SHALL fetch summary statistics from the API_Gateway and
   display them within 3 seconds on a standard broadband connection.

---

### Requirement 23: Admin UI — File Management Page

**User Story:** As an admin, I want a file management page, so that I can upload, browse, and
delete files from the browser.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide a file list table showing: filename, MIME type, size, uploader, and
   upload date, with pagination controls.
2. THE Admin_UI SHALL provide a file upload form that accepts drag-and-drop or click-to-browse
   file selection and displays upload progress.
3. WHEN a file upload completes, THE Admin_UI SHALL refresh the file list without a full page
   reload.
4. THE Admin_UI SHALL provide a download button per file row that triggers a browser file
   download.
5. THE Admin_UI SHALL provide a delete button per file row that shows a confirmation dialog before
   sending the delete request.
6. WHEN a delete request succeeds, THE Admin_UI SHALL remove the row from the list without a full
   page reload.

---

### Requirement 24: Admin UI — Scheduler Management Page

**User Story:** As an admin, I want a scheduler management page, so that I can create, monitor,
and control cron jobs from the browser.

#### Acceptance Criteria

1. THE Admin_UI SHALL provide a job list table showing: job name, cron expression, target URL,
   HTTP method, status, and last execution result, with pagination controls.
2. THE Admin_UI SHALL provide a create job form with fields: name, cron expression, target URL,
   HTTP method, and an optional description.
3. WHEN a cron expression is entered in the create job form, THE Admin_UI SHALL display a
   human-readable description of the schedule (e.g. "Every day at 09:00").
4. THE Admin_UI SHALL provide pause, resume, and delete action buttons per job row.
5. THE Admin_UI SHALL provide a job execution log drawer or modal that shows the last 50
   execution entries for a selected job.
6. WHEN a job status is `failed`, THE Admin_UI SHALL highlight the row in a visually distinct
   warning color.

---

### Requirement 25: Admin UI — Authentication

**User Story:** As an admin, I want to log in with credentials, so that the platform is not
accessible to unauthenticated users.

#### Acceptance Criteria

1. THE Admin_UI SHALL display a login page as the default route when no valid JWT is stored in
   browser local storage.
2. WHEN valid credentials are submitted, THE Admin_UI SHALL store the returned JWT in local
   storage and redirect to the dashboard.
3. WHEN the stored JWT is expired, THE Admin_UI SHALL clear local storage and redirect to the
   login page.
4. THE Admin_UI SHALL send all API requests with the stored JWT as a Bearer token in the
   `Authorization` header.

---

### Requirement 26: Auth Service — Login Endpoint

**User Story:** As an admin, I want a login endpoint, so that I can obtain a JWT to access the
platform.

#### Acceptance Criteria

1. WHEN a `POST /api/v1/auth/login` request is received with valid `username` and `password`
   fields, THE API_Gateway SHALL verify the credentials against the MySQL `users` table and
   return a signed JWT with a 24-hour expiry.
2. WHEN the credentials are invalid, THE API_Gateway SHALL respond with HTTP 401 and a JSON error
   body.
3. THE API_Gateway SHALL hash passwords using bcrypt before comparison.
4. WHEN a `POST /api/v1/auth/register` request is received with `username` and `password`, THE
   API_Gateway SHALL create a new user record with a bcrypt-hashed password and respond with HTTP
   201.

---

### Requirement 27: Platform — Extensibility

**User Story:** As a solo entrepreneur, I want the platform to be easy to extend with new
microservices, so that I can add DataPilot capabilities over time without rewriting existing code.

#### Acceptance Criteria

1. THE Platform SHALL organise each microservice as an independent Go module within a monorepo,
   sharing only the Common_Library module.
2. THE Common_Library SHALL be importable by any new microservice without modification to
   existing services.
3. THE API_Gateway SHALL support adding new upstream route prefixes through configuration without
   requiring code changes to existing route handlers.
4. THE Platform SHALL include a `docker-compose.yml` that starts all services and MySQL with a
   single `docker compose up` command.

---

### Requirement 28: Platform — Error Handling

**User Story:** As a developer, I want consistent error responses across all services, so that the
Admin_UI and API clients can handle errors uniformly.

#### Acceptance Criteria

1. THE Common_Library SHALL define a standard JSON error response struct with fields: `error`,
   `message`, and `request_id`.
2. WHEN any microservice encounters an unhandled panic, THE Common_Library recovery middleware
   SHALL catch it, log the stack trace, and respond with HTTP 500 using the standard error struct.
3. WHEN any microservice returns an error, THE Common_Library SHALL include the `request_id`
   (derived from the `X-Request-ID` header or a generated UUID) in the response.

---

### Requirement 29: Platform — CORS

**User Story:** As a frontend developer, I want CORS configured on the API Gateway, so that the
React Admin_UI can call the API from the browser without cross-origin errors.

#### Acceptance Criteria

1. THE API_Gateway SHALL apply CORS middleware that allows requests from the configured
   `ALLOWED_ORIGINS` environment variable.
2. WHEN an OPTIONS preflight request is received, THE API_Gateway SHALL respond with HTTP 204 and
   the appropriate CORS headers.
3. THE API_Gateway SHALL allow the `Authorization`, `Content-Type`, and `X-Request-ID` headers in
   CORS responses.
