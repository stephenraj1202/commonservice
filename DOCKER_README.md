# DataPilot Docker Images

Complete file management and job scheduling platform with REST API.

## Images

### 1. Gateway Service
**Image:** `stephenraj1202/datapilot-gateway:latest`

API Gateway with authentication, JWT token management, and reverse proxy to backend services.

**Features:**
- User authentication (login/register)
- JWT token generation and validation
- Reverse proxy to file and scheduler services
- CORS support
- Health check endpoint

**Environment Variables:**
```
HTTP_PORT=8080
MYSQL_DSN=root:password@tcp(mysql:3306)/datapilot_db
JWT_SECRET=your-secret-key
ALLOWED_ORIGINS=https://yourdomain.com
FILE_SERVICE_URL=http://file-service:8081
SCHEDULER_SERVICE_URL=http://scheduler-service:8082
```

---

### 2. File Service
**Image:** `stephenraj1202/datapilot-file-service:latest`

File upload, download, and management service with metadata storage.

**Features:**
- Multi-file upload support (up to 100MB per file)
- File download with original filename
- File listing with pagination
- File deletion
- Metadata tracking (size, mime type, upload date)

**Environment Variables:**
```
HTTP_PORT=8081
MYSQL_DSN=root:password@tcp(mysql:3306)/datapilot_db
FILE_STORAGE_PATH=/data/files
JWT_SECRET=your-secret-key
```

**Volumes:**
```
/data/files - File storage directory
```

---

### 3. Scheduler Service
**Image:** `stephenraj1202/datapilot-scheduler-service:latest`

Cron-based job scheduler with execution logging and management.

**Features:**
- Create scheduled jobs with cron expressions (6-field format)
- Pause/resume jobs
- Job execution logging
- Active job monitoring
- Automatic job execution

**Environment Variables:**
```
HTTP_PORT=8082
MYSQL_DSN=root:password@tcp(mysql:3306)/datapilot_db
JWT_SECRET=your-secret-key
```

**Cron Format:** `second minute hour day month day-of-week`

---

### 4. Admin UI
**Image:** `stephenraj1202/datapilot-admin-ui:latest`

Modern React-based admin interface with Ant Design components.

**Features:**
- Dashboard with statistics
- File management (grid/list view, multi-select, bulk delete)
- Job scheduler management
- API documentation with copyable curl commands
- File type icons and visual indicators
- Real-time upload progress bars

**Environment Variables:**
```
VITE_API_BASE_URL=https://yourdomain.com
```

---

## Quick Start

### Using Docker Compose

```yaml
version: "3.9"

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: datapilot_db
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"

  gateway:
    image: stephenraj1202/datapilot-gateway:latest
    ports:
      - "8080:8080"
    environment:
      HTTP_PORT: "8080"
      MYSQL_DSN: "root:rootpassword@tcp(mysql:3306)/datapilot_db"
      JWT_SECRET: "change-this-secret"
      ALLOWED_ORIGINS: "http://localhost:3000"
      FILE_SERVICE_URL: "http://file-service:8081"
      SCHEDULER_SERVICE_URL: "http://scheduler-service:8082"
    depends_on:
      - mysql

  file-service:
    image: stephenraj1202/datapilot-file-service:latest
    ports:
      - "8081:8081"
    environment:
      HTTP_PORT: "8081"
      MYSQL_DSN: "root:rootpassword@tcp(mysql:3306)/datapilot_db"
      FILE_STORAGE_PATH: "/data/files"
      JWT_SECRET: "change-this-secret"
    volumes:
      - file_storage:/data/files
    depends_on:
      - mysql

  scheduler-service:
    image: stephenraj1202/datapilot-scheduler-service:latest
    ports:
      - "8082:8082"
    environment:
      HTTP_PORT: "8082"
      MYSQL_DSN: "root:rootpassword@tcp(mysql:3306)/datapilot_db"
      JWT_SECRET: "change-this-secret"
    depends_on:
      - mysql

  admin-ui:
    image: stephenraj1202/datapilot-admin-ui:latest
    ports:
      - "3000:80"
    environment:
      VITE_API_BASE_URL: "http://localhost:8080"

volumes:
  mysql_data:
  file_storage:
```

### Run with Docker Compose

```bash
docker-compose up -d
```

Access the UI at: http://localhost:3000

---

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get JWT token

### Files
- `POST /api/v1/files/upload` - Upload file
- `GET /api/v1/files` - List files
- `GET /api/v1/files/:id/download` - Download file
- `DELETE /api/v1/files/:id` - Delete file

### Scheduler
- `POST /api/v1/scheduler/jobs` - Create job
- `GET /api/v1/scheduler/jobs` - List jobs
- `PUT /api/v1/scheduler/jobs/:id` - Update job
- `POST /api/v1/scheduler/jobs/:id/pause` - Pause job
- `POST /api/v1/scheduler/jobs/:id/resume` - Resume job
- `DELETE /api/v1/scheduler/jobs/:id` - Delete job
- `GET /api/v1/scheduler/jobs/:id/logs` - Get job logs

### Health
- `GET /health` - System health check

---

## Example Usage

### Register and Login
```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')
```

### Upload and Download File
```bash
# Upload
curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@document.pdf"

# Download
curl -X GET "http://localhost:8080/api/v1/files/1/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o downloaded.pdf
```

### Create Scheduled Job
```bash
curl -X POST http://localhost:8080/api/v1/scheduler/jobs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Backup",
    "cron_expression": "0 0 2 * * *",
    "command": "backup.sh",
    "enabled": true
  }'
```

---

## Architecture

```
┌─────────────┐
│  Admin UI   │ (Port 3000)
│  (React)    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Gateway   │ (Port 8080)
│   (Auth +   │
│   Proxy)    │
└──────┬──────┘
       │
       ├──────────────┬──────────────┐
       ▼              ▼              ▼
┌──────────┐   ┌──────────┐   ┌──────────┐
│   File   │   │Scheduler │   │  MySQL   │
│ Service  │   │ Service  │   │    DB    │
│  (8081)  │   │  (8082)  │   │  (3306)  │
└──────────┘   └──────────┘   └──────────┘
```

---

## Tech Stack

- **Backend:** Go 1.25+ with Gin framework
- **Frontend:** React 18 with TypeScript and Ant Design
- **Database:** MySQL 8.0
- **Authentication:** JWT tokens with bcrypt password hashing
- **Scheduler:** Cron-based job execution
- **Storage:** Local filesystem for files

---

## Security Notes

- Change default JWT_SECRET in production
- Use strong MySQL passwords
- Configure ALLOWED_ORIGINS for CORS
- Run behind reverse proxy (nginx) with SSL
- Limit file upload sizes as needed

---

## Support

- GitHub: https://github.com/stephenraj1202/commonservice
- Issues: https://github.com/stephenraj1202/commonservice/issues

---

## License

MIT License
