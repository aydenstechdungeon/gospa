# Build stage
# Usage: docker build -t gospa-website -f website/Dockerfile .
# RENDER NOTE: Set "Docker Build Context" to "." in Render settings.
FROM golang:1.24-alpine AS builder

# 1. Install system dependencies
# Bun on Alpine requires libstdc++ and gcompat for glibc compatibility
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    bash \
    unzip \
    libstdc++ \
    gcompat \
    build-base

# 2. Install Bun (using the recommended Alpine-friendly method)
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

# Verify Bun installation immediately to catch errors early
RUN bun --version

WORKDIR /build

# 3. Copy source code
# We copy everything because the website depends on the core gospa modules
COPY . .

# 4. Build the gospa CLI tool
# This tool coordinates the entire build process (templ, css, js, go)
RUN go mod download
RUN go build -o /usr/local/bin/gospa ./cmd/gospa

# 5. Build the website
WORKDIR /build/website

# Ensure all JS dependencies are installed for the build plugins
RUN bun install

# Run the gospa build command
# This will:
# - Generate templ files
# - Generate TypeScript types
# - Build the client runtime using Bun
# - Build the Go server binary
# - Copy and compress static assets
RUN gospa build -o dist

# 6. Prepare runtime artifacts
# We collect everything needed for runtime into a single 'out' folder
RUN mkdir -p /build/out && \
    cp dist/server /build/out/website && \
    cp -r dist/static /build/out/static && \
    cp -r routes /build/out/routes && \
    cp -r components /build/out/components && \
    cp -r generated /build/out/generated

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the prepared artifacts
COPY --from=builder /build/out/website ./website
COPY --from=builder /build/out/static ./static
COPY --from=builder /build/out/routes ./routes
COPY --from=builder /build/out/components ./components
COPY --from=builder /build/out/generated ./generated

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Standard production environment variables
EXPOSE 8080
ENV PORT=8080
ENV GOSPA_DEV=false
ENV GOSPA_ENV=production

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Start the application
CMD ["./website"]
