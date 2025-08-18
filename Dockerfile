# Multi-stage build for Arcron
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o arcron ./cmd/arcron

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata sqlite

# Create non-root user
RUN addgroup -g 1001 -S arcron && \
    adduser -u 1001 -S arcron -G arcron

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/arcron .

# Create necessary directories
RUN mkdir -p config data logs models && \
    chown -R arcron:arcron /app

# Switch to non-root user
USER arcron

# Expose port (if web interface is added later)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ps aux | grep arcron || exit 1

# Default command
CMD ["./arcron", "--config", "config/arcron.yaml"]
