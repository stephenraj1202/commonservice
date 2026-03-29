# Stage 1: Builder
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy scheduler-service module files and download dependencies
COPY scheduler-service/go.mod scheduler-service/go.sum ./
# Copy common module (needed for replace directive)
COPY common/ /common/

RUN go mod download

# Copy scheduler-service source
COPY scheduler-service/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o scheduler-service .

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/scheduler-service /scheduler-service

EXPOSE 8082

ENTRYPOINT ["/scheduler-service"]
