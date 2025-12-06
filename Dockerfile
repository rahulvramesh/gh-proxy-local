# Multi-stage build for gh-proxy-local
# Stage 1: Build stage
FROM golang:1.24.5-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Stage 2: Runtime stage
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /build/bin/gh-proxy-local /app/gh-proxy-local

# Create a non-root user for security
RUN addgroup -g 1000 copilot && \
    adduser -D -u 1000 -G copilot copilot && \
    chown -R copilot:copilot /app

USER copilot

# Expose default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["/app/gh-proxy-local"]
