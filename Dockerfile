# Stage 1: Builder
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# -ldflags="-s -w" strips debug info for smaller binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o medicaments-api .

# Stage 2: Runtime
FROM alpine:3.20

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create app user and directories
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    mkdir -p /app/logs && \
    chown -R appuser:appuser /app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/medicaments-api .

# Copy html directory for documentation
COPY --from=builder /build/html ./html

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8000

# Run the application
ENTRYPOINT ["./medicaments-api"]
