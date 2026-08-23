# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Bağımlılıkları çek
COPY go.mod go.sum ./
RUN go mod download

# Backend kaynak kodlarını kopyala
COPY . ./

# Binary'yi derle
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/api \
    ./cmd/api

# Stage 2: Production Runner stage (minimal & secure)
FROM alpine:3.20 AS runner

RUN apk add --no-cache ca-certificates tzdata wget

RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/api /app/api
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/api"]
