# DataPilot Platform

A microservices platform for file management and cron job scheduling, built with Go (Gin + GORM), MySQL, and a React admin UI. Designed to run on a single server with a one-command startup.

## Architecture

```
Browser / Admin UI (:3000)
        │
        ▼
  API Gateway (:8080)          ← single public entry point
  /api/v1/auth/*  (local)
  /api/v1/files/* ──────────▶  File Service (:8081)
  /api/v1/scheduler/* ──────▶  Scheduler Service (:8082)
        │                              │
        └──────────────┬───────────────┘
                       ▼
                  MySQL (:3306)
                  datapilot_db
```

### Services

| Service           | Port | Description                                      |
|-------------------|------|--------------------------------------------------|
| API Gateway       | 8080 | Auth, CORS, JWT validation, reverse proxy        |
| File Service      | 8081 | File upload / download / list / delete           |
| Scheduler Service | 8082 | Cron job CRUD, execution runner, execution logs  |
| Admin UI          | 3000 | React SPA (served by nginx in Docker)            |
| MySQL             | 3306 | Persistent storage for all services              |

### Monorepo Layout

```
datapilot/
├── docker-compose.yml
├── docker-compose.test.yml
├── .env.example
├── common/                  # Shared Go module (datapilot/common)
│   ├── config/              # Config loader (env vars + .env file)
│   ├── logger/              # Structured JSON logger (zap)
│   ├── database/            # GORM MySQL helper
│   ├── middleware/          # JWT, CORS, Recovery, RequestID
│   ├── pagination/          # Paginate helper + PagedResponse
│   ├── client/              # Inter-service HTTP client
│   └── errors/              # Standard APIError type
├── gateway/                 # API Gateway (datapilot/gateway)
│   ├── handlers/auth.go     # Login + Register
│   └── proxy/proxy.go       # httputil.ReverseProxy
├── file-service/            # File Service (datapilot/file-service)
│   ├── handlers/files.go    # Upload, Download, List, Delete
│   ├── models/              # FileRecord GORM model
│   └── storage/local.go     # Local filesystem storage
├── scheduler-service/       # Scheduler Service (datapilot/scheduler-service)
│   ├── handlers/jobs.go     # Job CRUD + GetLogs
│   ├── models/              # Job + JobExecutionLog GORM models
│   └── runner/cron.go       # robfig/cron runner
├── admin-ui/                # React SPA (Vite + TypeScript + Ant Design)
│   └── src/
│       ├── api/             # Axios API layer (auth, files, scheduler)
│       ├── components/      # Reusable UI components
│       ├── pages/           # Login, Dashboard, Files, Scheduler
│       └── store/auth.ts    # Zustand auth store
└── integration/             # End-to-end integration tests
```

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/) v2+

### 1. Clone and configure

```bash
git clone https://github.com/stephenraj1202/commonservicecommit.git
cd commonservicecommit/datapilot
cp .env.example .env
```

Edit `.env` and set secure values:

```dotenv
MYSQL_ROOT_PASSWORD=your-strong-password
JWT_SECRET=your-jwt-secret-here
ALLOWED_ORIGINS=http://localhost:3000
```

### 2. Start the platform

```bash
docker compose up --build
```

All services start in dependency order: MySQL → backend services → Admin UI.

### 3. Open the Admin UI

Navigate to [http://localhost:3000](http://localhost:3000) and register an account.

## API Reference

All routes are served through the gateway at `http://localhost:8080`.

### Auth

| Method | Path                        | Auth | Description          |
|--------|-----------------------------|------|----------------------|
| POST   | `/api/v1/auth/register`     | No   | Create a new account |
| POST   | `/api/v1/auth/login`        | No   | Get a JWT token      |
| GET    | `/health`                   | No   | Platform health      |

**Login response:**
```json
{ "token": "<jwt>" }
```

### Files

| Method | Path                            | Auth | Description              |
|--------|---------------------------------|------|--------------------------|
| POST   | `/api/v1/files/upload`          | JWT  | Upload a file (multipart)|
| GET    | `/api/v1/files`                 | JWT  | List files (paginated)   |
| GET    | `/api/v1/files/:id/download`    | JWT  | Download a file          |
| DELETE | `/api/v1/files/:id`             | JWT  | Delete a file            |

**Pagination query params:** `?page=1&limit=20`

**Upload limit:** 100 MB per file.

### Scheduler

| Method | Path                                    | Auth | Description                  |
|--------|-----------------------------------------|------|------------------------------|
| POST   | `/api/v1/scheduler/jobs`                | JWT  | Create a job                 |
| GET    | `/api/v1/scheduler/jobs`                | JWT  | List jobs (paginated)        |
| PUT    | `/api/v1/scheduler/jobs/:id`            | JWT  | Update a job                 |
| POST   | `/api/v1/scheduler/jobs/:id/pause`      | JWT  | Pause a job                  |
| POST   | `/api/v1/scheduler/jobs/:id/resume`     | JWT  | Resume a job                 |
| DELETE | `/api/v1/scheduler/jobs/:id`            | JWT  | Soft-delete a job            |
| GET    | `/api/v1/scheduler/jobs/:id/logs`       | JWT  | Get execution logs           |

**Create job body:**
```json
{
  "name": "Daily report",
  "cron_expression": "0 9 * * *",
  "target_url": "https://example.com/webhook",
  "http_method": "POST",
  "description": "Fires every day at 09:00"
}
```

Supports standard 5-field cron expressions and 6-field (with seconds) expressions.

**Status filter:** `GET /api/v1/scheduler/jobs?status=active|paused`

## Environment Variables

| Variable               | Required | Default                  | Description                          |
|------------------------|----------|--------------------------|--------------------------------------|
| `MYSQL_ROOT_PASSWORD`  | Yes      | —                        | MySQL root password                  |
| `JWT_SECRET`           | Yes      | —                        | Secret used to sign JWTs             |
| `ALLOWED_ORIGINS`      | Yes      | —                        | CORS allowed origins (comma-separated)|
| `FILE_STORAGE_PATH`    | No       | `/data/files`            | Path for uploaded file storage       |
| `LOG_LEVEL`            | No       | `info`                   | Log level: debug, info, warn, error  |
| `FILE_SERVICE_URL`     | No       | `http://file-service:8081`     | Internal file service URL      |
| `SCHEDULER_SERVICE_URL`| No       | `http://scheduler-service:8082`| Internal scheduler service URL |

## Development

### Running a single service locally

Each service is an independent Go module. To run the gateway locally:

```bash
cd datapilot/gateway
cp ../.env.example .env   # adjust values
go run .
```

Repeat for `file-service` and `scheduler-service`.

### Running unit tests

```bash
# Common library
cd datapilot/common && go test ./...

# Scheduler service
cd datapilot/scheduler-service && go test ./...
```

### Running integration tests

The integration tests require the full stack to be running:

```bash
# Start the stack + run tests
cd datapilot
docker compose -f docker-compose.test.yml up --build --abort-on-container-exit
```

Or run against a locally running stack:

```bash
cd datapilot/integration
GATEWAY_URL=http://localhost:8080 go test -v -timeout 120s ./...
```

### Admin UI development

```bash
cd datapilot/admin-ui
npm install
npm run dev      # starts Vite dev server on :5173, proxies /api to :8080
```

## Tech Stack

**Backend**
- Go 1.22
- [Gin](https://github.com/gin-gonic/gin) — HTTP framework
- [GORM](https://gorm.io) — ORM with MySQL driver
- [go.uber.org/zap](https://github.com/uber-go/zap) — structured logging
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt) — JWT auth
- [robfig/cron](https://github.com/robfig/cron) — cron runner
- [google/uuid](https://github.com/google/uuid) — UUID generation

**Frontend**
- React 18 + TypeScript
- [Vite](https://vitejs.dev) — build tool
- [Ant Design](https://ant.design) — UI component library
- [Zustand](https://github.com/pmndrs/zustand) — state management
- [Axios](https://axios-http.com) — HTTP client
- [cronstrue](https://github.com/bradymholt/cronstrue) — human-readable cron descriptions
- [React Router v6](https://reactrouter.com)

**Infrastructure**
- MySQL 8.0
- Docker + Docker Compose v2
- nginx (Admin UI runtime)

## Adding a New Microservice

1. Create a new Go module under `datapilot/<service-name>/` with a `replace datapilot/common => ../common` directive.
2. Import and use the common library for config, logger, database, and middleware.
3. Add a new route group in `gateway/main.go`:
   ```go
   engine.Any("/api/v1/<prefix>/*path", middleware.JWTAuth(cfg.JWTSecret), proxy.NewProxy(cfg.NewServiceURL))
   ```
4. Add the service to `docker-compose.yml` with `depends_on: mysql: condition: service_healthy`.

No changes to existing services are required.

## License

MIT
