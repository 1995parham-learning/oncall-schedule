# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /build/oncall-schedule \
    .

# Runtime stage
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 oncall && \
    adduser -D -u 1000 -G oncall oncall

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/oncall-schedule /app/oncall-schedule

# Copy migrations
COPY --from=builder /build/migrations /app/migrations

# Copy default config (can be overridden with volume mount)
COPY --from=builder /build/config.yaml /app/config.yaml

# Change ownership
RUN chown -R oncall:oncall /app

# Switch to non-root user
USER oncall

# Expose port
EXPOSE 1373

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:1373/health || exit 1

# Run the application
ENTRYPOINT ["/app/oncall-schedule"]
