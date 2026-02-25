# Build stage - Optimized for both root and subdirectory contexts
# Usage: docker build -t gospa-website -f website/Dockerfile .
# RENDER NOTE: For best results, set "Docker Build Context" to "." in Render settings.
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata curl bash unzip libc6-compat

# Install Bun (required for client and CSS builds)
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

WORKDIR /build

# Copy all available source
COPY . .

# 1. Build or Install the gospa CLI tool
# If we have the full repo context, build from local source.
# Otherwise, install the latest version from GitHub.
RUN if [ -d "./cmd/gospa" ]; then \
    echo "Found GoSPA source, building CLI locally..." && \
    go build -o /usr/local/bin/gospa ./cmd/gospa; \
    else \
    echo "GoSPA source not found in current context, installing from remote..." && \
    go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest && \
    mv /go/bin/gospa /usr/local/bin/gospa; \
    fi

# 2. Normalize and Build
# We ensure that regardless of the context, the final artifacts are in /build/out
RUN mkdir -p /build/out && \
    if [ -d "website" ]; then \
    echo "Building from root context..." && \
    cd website && \
    bun install && \
    gospa build -o /build/out && \
    cp -r routes components generated /build/out/ 2>/dev/null || true; \
    else \
    echo "Building from website context..." && \
    bun install && \
    gospa build -o /build/out && \
    cp -r routes components generated /build/out/ 2>/dev/null || true; \
    fi

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy artifacts from the normalized /build/out directory
COPY --from=builder /build/out/server ./website
COPY --from=builder /build/out/static ./static
COPY --from=builder /build/out/routes ./routes
COPY --from=builder /build/out/components ./components
COPY --from=builder /build/out/generated ./generated

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Expose port
EXPOSE 8080
ENV PORT=8080
ENV GOSPA_DEV=false

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the binary
CMD ["./website"]
