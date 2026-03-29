# Stage 1: Builder
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy file-service module files and download dependencies
COPY file-service/go.mod file-service/go.sum ./
# Copy common module (needed for replace directive)
COPY common/ /common/

RUN go mod download

# Copy file-service source
COPY file-service/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o file-service .

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/file-service /file-service

EXPOSE 8081

ENTRYPOINT ["/file-service"]
