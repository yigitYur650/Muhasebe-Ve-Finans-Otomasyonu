# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy source code into WORKDIR
COPY . .

# Build static binary (handles both root context and backend context)
RUN if [ -d "backend" ]; then cd backend; fi && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-w -s" \
        -o /app/api \
        ./cmd/api

# Stage 2: Production Runner stage (minimal & secure)
FROM alpine:3.20 AS runner

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user for security
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/api /app/api

# Set ownership to non-root user
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/api"]
