# DataPilot API Documentation

## Authentication

### 1. Register a New User
```bash
curl -X POST https://commonservice.datapilot.co.in/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"your_username","password":"your_password"}'
```

### 2. Login and Get Token
```bash
curl -X POST https://commonservice.datapilot.co.in/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"your_username","password":"your_password"}'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Save the token for subsequent requests.

## File Operations

### 3. Upload a File
```bash
TOKEN="your_token_here"

curl -X POST https://commonservice.datapilot.co.in/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/path/to/your/file.pdf"
```

**Response:**
```json
{
  "id": 1,
  "original_filename": "file.pdf",
  "stored_filename": "uuid.pdf",
  "mime_type": "application/pdf",
  "size_bytes": 12345,
  "created_at": "2026-03-29T13:00:00Z"
}
```

### 4. List Files
```bash
curl -X GET "https://commonservice.datapilot.co.in/api/v1/files?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

### 5. Download a File
```bash
FILE_ID=1

curl -X GET "https://commonservice.datapilot.co.in/api/v1/files/$FILE_ID/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o downloaded_file.pdf
```

### 6. Delete a File
```bash
curl -X DELETE "https://commonservice.datapilot.co.in/api/v1/files/$FILE_ID" \
  -H "Authorization: Bearer $TOKEN"
```

## Scheduler Operations

### 7. Create a Scheduled Job
```bash
curl -X POST https://commonservice.datapilot.co.in/api/v1/scheduler/jobs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Backup",
    "cron_expression": "0 0 2 * * *",
    "command": "backup.sh",
    "enabled": true
  }'
```

**Note:** Cron format is `second minute hour day month day-of-week`

Examples:
- Every minute: `0 * * * * *`
- Every 5 minutes: `0 */5 * * * *`
- Daily at 2 AM: `0 0 2 * * *`

### 8. List Jobs
```bash
curl -X GET "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

### 9. Pause a Job
```bash
JOB_ID=1

curl -X POST "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs/$JOB_ID/pause" \
  -H "Authorization: Bearer $TOKEN"
```

### 10. Resume a Job
```bash
curl -X POST "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs/$JOB_ID/resume" \
  -H "Authorization: Bearer $TOKEN"
```

### 11. Get Job Logs
```bash
curl -X GET "https://commonservice.datapilot.co.in/api/v1/scheduler/jobs/$JOB_ID/logs?page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

## Health Check

### 12. Check System Health
```bash
curl https://commonservice.datapilot.co.in/health
```

## Complete Example Workflow

```bash
#!/bin/bash

# 1. Login and get token
TOKEN=$(curl -s -X POST https://commonservice.datapilot.co.in/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin2","password":"admin"}' | jq -r '.token')

echo "Token: $TOKEN"

# 2. Upload a file
UPLOAD_RESPONSE=$(curl -s -X POST https://commonservice.datapilot.co.in/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@document.pdf")

echo "Upload Response: $UPLOAD_RESPONSE"

# 3. Get file ID
FILE_ID=$(echo $UPLOAD_RESPONSE | jq -r '.id')

echo "File ID: $FILE_ID"

# 4. Download the file
curl -X GET "https://commonservice.datapilot.co.in/api/v1/files/$FILE_ID/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o downloaded_document.pdf

echo "File downloaded as downloaded_document.pdf"

# 5. List all files
curl -s -X GET "https://commonservice.datapilot.co.in/api/v1/files?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

## Error Handling

All endpoints return standard HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized (invalid or missing token)
- `404` - Not Found
- `422` - Unprocessable Entity (validation error)
- `500` - Internal Server Error

Error response format:
```json
{
  "error": "error_code",
  "message": "Human readable error message",
  "request_id": "unique-request-id"
}
```
