# Build stage - must be built from project root
# Usage: docker build -t gospa-website -f website/Dockerfile .
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy entire project first (needed for replace directive to work)
COPY . .

# Download dependencies and build
WORKDIR /build/website
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /website .

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /website .

# Copy routes directory (needed for runtime route scanning)
COPY --from=builder /build/website/routes ./routes

# Copy components directory (contains compiled templates)
COPY --from=builder /build/website/components ./components

# Copy generated directory (contains TypeScript routes/types)
COPY --from=builder /build/website/generated ./generated

# Copy static assets if they exist
COPY --from=builder /build/website/static ./static

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Expose port (Fly.io uses 8080 by default)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Default environment for production
ENV PORT=8080

# Run
CMD ["./website"]
