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
RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/gospa ./cmd/gospa

# ------------------------------------------------------------------------------
# Stage 2: Website Assets & Server Build
# ------------------------------------------------------------------------------
FROM builder-base AS website-builder
WORKDIR /build

# 1. External dependencies caching
# Order matters: Go modules first as they change less often than Bun modules.
COPY go.mod go.sum ./
COPY website/go.mod website/go.sum ./website/
RUN go mod download && cd website && go mod download

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
# We copy source assets first, then overlay build artifacts (dist/) to ensure 
# nothing is missed (e.g. fonts/images that gospa build might have skipped).
RUN mkdir -p /out/static /out/routes /out/components /out/generated && \
    cp dist/server /out/gospa-server && \
    # 6a. Copy source static files (original assets)
    cp -r static/. /out/static/ && \
    # 6b. Overlay built static files (overwriting with minified/processed versions)
    if [ -d dist/static ]; then cp -r dist/static/. /out/static/; fi && \
    # 6c. Copy routes and other runtime-required directories
    cp -r routes/. /out/routes/ || true && \
    { [ -d components ] && cp -r components/. /out/components/ || true; } && \
    { [ -d generated ]  && cp -r generated/.  /out/generated/ || true; }

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
