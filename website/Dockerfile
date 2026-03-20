# syntax=docker/dockerfile:1.7-labs
# ==============================================================================
# GOSPA Website Production Dockerfile
# Redesigned for maximum build speed, layer caching, and minimal runtime footprint.
#
# Context: Repository Root (REQUIRED)
# Command: docker build -t gospa-website -f website/Dockerfile .
# ==============================================================================

# Common base for builder stages
FROM golang:1.26-alpine AS builder-base
RUN apk add --no-cache \
    curl bash unzip libstdc++ gcompat build-base git nodejs npm

# Install Bun directly into the builder base
RUN curl -fsSL https://bun.sh/install | bash -s -- bun-v1.2.4
ENV PATH="/root/.bun/bin:${PATH}"

# ------------------------------------------------------------------------------
# Stage 1: CLI Builder
# Compiles the gospa binary independently to avoid rebuilding it unnecessarily.
# ------------------------------------------------------------------------------
FROM builder-base AS cli-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -v -o /usr/local/bin/gospa ./cmd/gospa

# ------------------------------------------------------------------------------
# Stage 2: Website Assets & Server Build
# ------------------------------------------------------------------------------
FROM builder-base AS website-builder
WORKDIR /build

# 1. External dependencies caching
# Order matters: Go modules first as they change less often than Bun modules.
COPY go.mod go.sum ./
COPY website/go.mod website/go.sum ./website/
RUN go mod download

# 2. Bun dependencies caching
COPY website/package.json website/bun.lock ./website/
RUN cd website && bun install --frozen-lockfile

# 3. Pull in the GoSPA CLI from the previous stage
COPY --from=cli-builder /usr/local/bin/gospa /usr/local/bin/gospa

# 4. Copy entire repository source
# This is required because 'website/' depends on the parent framework via 'replace'
COPY . .

# 5. Build the website for production
# This generates 'dist/server' and builds Tailwind v4 / client JS assets
WORKDIR /build/website
RUN gospa build -o dist

# 6. Normalize artifact structure for runtime
RUN mkdir -p /out/static /out/routes /out/components /out/generated && \
    cp dist/server /out/gospa-server && \
    [ -d dist/static ] && cp -r dist/static/. /out/static/ || true && \
    cp -r routes/. /out/routes/ || true && \
    cp -r components/. /out/components/ || true && \
    cp -r generated/. /out/generated/ || true

# ------------------------------------------------------------------------------
# Stage 3: Runtime
# Minimal Alpine image for the smallest attack surface and fastest deployments.
# ------------------------------------------------------------------------------
FROM alpine:3.21 AS runtime

# Install essentials and setup a non-root environment
RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -g '' appuser

WORKDIR /app

# Copy artifacts with correct ownership
COPY --from=website-builder --chown=appuser:appuser /out/ /app/

# Runtime Configuration
USER appuser
EXPOSE 8080
ENV PORT=8080 \
    GOSPA_DEV=false \
    GOSPA_ENV=production \
    TZ=UTC

# Proactive Healthcheck using standard wget
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/ || exit 1

# Start the optimized Go server
ENTRYPOINT ["./gospa-server"]
