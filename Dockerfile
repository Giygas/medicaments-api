# syntax=docker/dockerfile:1

ARG VERSION=latest
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Stage 1: Builder
FROM golang:1.26.0-alpine3.23 AS builder

# Install build dependencies
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./

# Cache mounts for downloads
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build binary with static linking and cache
# CGO_ENABLED=0: Disable CGO for static binary (required for scratch)
# GOOS=linux: Target Linux OS
# GOARCH=$TARGETARCH: Target architecture (amd64/arm64) - auto-detected by BuildKit
# -ldflags="-s -w -extldflags '-static'": Strip debug info and force static linking
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build \
    -ldflags="-s -w -extldflags '-static' \
    -X main.Version=${VERSION} \
    -X main.GitCommit=${GIT_COMMIT} \
    -X main.BuildDate=${BUILD_DATE}" \
    -trimpath \
    -o /app/medicaments-api .

# Stage 2: Runtime (scratch - empty filesystem)
FROM scratch

# Metadata labels for security scanning and documentation
LABEL org.opencontainers.image.source="https://github.com/giygas/medicaments-api" \
      org.opencontainers.image.description="French medicaments API - High-performance JSON API for BDPM data" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.title="medicaments-api" \
      org.opencontainers.image.authors="giygas@example.com" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}"

# Copy CA certificates for HTTPS requests (required for BDPM downloads)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy app directory from builder (includes binary and pre-created directories)
# Use --chown to set ownership for scratch container
COPY --from=builder --chown=65534:65534 /app /app

# Copy HTML documentation and assets
COPY --from=builder --chown=65534:65534 /build/html /app/html

# Use non-root user (nobody user with UID 65534)
USER 65534:65534

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 8000

# Add health check using binary's built-in healthcheck subcommand
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/medicaments-api", "healthcheck"]

# Run application
ENTRYPOINT ["/app/medicaments-api"]
