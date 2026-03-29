# Stage 1: Builder
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy gateway module files and download dependencies
COPY gateway/go.mod gateway/go.sum ./
# Copy common module (needed for replace directive)
COPY common/ /common/

RUN go mod download

# Copy gateway source
COPY gateway/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway .

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/gateway /gateway

EXPOSE 8080

ENTRYPOINT ["/gateway"]
