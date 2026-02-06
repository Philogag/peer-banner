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
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /app/peer-banner .

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
COPY --from=builder /app/peer-banner .

# Create data directory
RUN mkdir -p /data && chown -R appuser:appgroup /data

# Use non-root user
USER appuser

# Environment variable for custom config path
ENV CONFIG_PATH=/data/config.yaml

# Default command
CMD ["/bin/sh", "-c", "/app/peer-banner -config ${CONFIG_PATH}"]
