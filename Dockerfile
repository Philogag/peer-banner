# ============================================
# Build Stage
# ============================================
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /app/qbittorrent-banner .

# ============================================
# Runtime Stage
# ============================================
FROM alpine:3.19 AS runtime

# Install ca-certificates for HTTPS support
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/qbittorrent-banner .

# Create data directory
RUN mkdir -p /data && chown -R appuser:appgroup /data

# Use non-root user
USER appuser

# Default command
ENTRYPOINT ["/app/qbittorrent-banner"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080 || exit 1

# Expose (no ports needed, daemon only)
# EXPOSE 8080

# ============================================
# Alternative: Debug Stage (for troubleshooting)
# ============================================
FROM runtime AS debug

USER root
RUN apk --no-cache add strace ltrace gdb
USER appuser
