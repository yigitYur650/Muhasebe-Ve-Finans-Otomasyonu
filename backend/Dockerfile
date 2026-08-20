# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Retrieve & cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build static binary for Linux CGO_ENABLED=0
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/api \
    ./cmd/api

# Stage 2: Production Runner stage (minimal & secure)
FROM alpine:3.20 AS runner

# Install runtime dependencies (ca-certificates for HTTPS/TLS to Supabase)
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user for security (rootless execution)
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/api /app/api

# Set ownership to non-root user
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api"]
