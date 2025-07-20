# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/autocomplete-server cmd/server/main.go

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/autocomplete-server ./
COPY --from=builder /app/web ./web/
COPY --from=builder /app/configs ./configs/

# Create necessary directories
RUN mkdir -p logs data && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Set default environment variables
ENV PORT=8080 \
    LOG_LEVEL=info \
    ENABLE_CORS=true \
    CACHE_ENABLED=true \
    CACHE_TTL=5m \
    REDIS_ENABLED=false

# Run the application
CMD ["./autocomplete-server"] 