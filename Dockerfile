# ============================================
# Multi-stage Dockerfile for go_ProFiBus
# ============================================
# This Dockerfile uses multi-stage build to:
# 1. Reduce final image size
# 2. Improve security by not including build tools
# 3. Speed up builds with layer caching
# ============================================

# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /build

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./

# Download dependencies (cached if go.mod/go.sum haven't changed)
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0: Build static binary
# -ldflags: Strip debug info and set version
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s \
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.GitCommit=${GIT_COMMIT}" \
    -o /build/profibus \
    ./cmd/server

# Stage 2: Build frontend (optional, if building frontend in same image)
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# Copy package files
COPY web/dashboard/package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy frontend source
COPY web/dashboard/ ./

# Build frontend
RUN npm run build

# Stage 3: Create minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    postgresql-client

# Create non-root user
RUN addgroup -g 1000 profibus && \
    adduser -D -u 1000 -G profibus profibus

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/profibus /app/profibus

# Copy migrations
COPY --from=builder /build/migrations /app/migrations

# Copy frontend build (optional)
COPY --from=frontend-builder /frontend/dist /app/web/dist

# Copy configuration files
COPY --from=builder /build/configs /app/configs

# Set ownership
RUN chown -R profibus:profibus /app

# Switch to non-root user
USER profibus

# Expose ports
# 8080: REST API
# 8081: Metrics (Prometheus)
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV GO_ENV=production \
    LOG_LEVEL=info \
    LOG_FORMAT=json

# Run the application
ENTRYPOINT ["/app/profibus"]
CMD ["serve"]
