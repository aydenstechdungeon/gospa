# Build stage
# Usage: docker build -t gospa-website -f website/Dockerfile .
# RENDER NOTE: While this Dockerfile is now "context-agnostic", for best performance,
# set "Docker Build Context" to "." in Render settings.
FROM golang:1.26-alpine AS builder

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

# 2. Install Bun
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

WORKDIR /build

# 3. Copy source and clean previous build artifacts to ensure fresh build
COPY . .
RUN rm -rf dist generated node_modules && \
    if [ -d "website" ]; then rm -rf website/dist website/generated website/node_modules; fi

# 4. Handle GoSPA CLI
# Detect context: if we have the full repo, build locally. Otherwise, install from remote.
# IMPORTANT: For best results on Render, set 'Docker Build Context' to '.' in Settings.
RUN if [ -f "cmd/gospa/main.go" ]; then \
    echo "Found local GoSPA source. Building CLI..." && \
    go mod download && \
    go build -v -o /usr/local/bin/gospa ./cmd/gospa; \
elif [ -f "go.mod" ] && grep -q "module github.com/aydenstechdungeon/gospa/website" go.mod; then \
    echo "Context appears to be 'website/' subdirectory. Installing GoSPA CLI from remote..." && \
    echo "Note: To build from local source, set Docker Build Context to '.' in Render settings." && \
    GOPROXY=direct go install github.com/aydenstechdungeon/gospa/cmd/gospa@main && \
    cp /go/bin/gospa /usr/local/bin/gospa; \
else \
    echo "Unknown context. Attempting remote install..." && \
    GOPROXY=direct go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest && \
    cp /go/bin/gospa /usr/local/bin/gospa; \
fi

# 5. Build the website
# Detect if we are in the repository root or the website subdirectory
RUN if [ -d "website" ]; then \
    echo "Context: Repository Root. Building website subdirectory..." && \
    cd website && \
    go mod edit -replace github.com/aydenstechdungeon/gospa=../ && \
    bun install && \
    /usr/local/bin/gospa build -o dist; \
    else \
    echo "Context: Website Directory. Building current directory..." && \
    go mod edit -dropreplace github.com/aydenstechdungeon/gospa && \
    bun install && \
    /usr/local/bin/gospa build -o dist; \
    fi

# 6. Normalize artifacts into /build/out for the runtime stage
# Optional dirs (static, components, generated) are copied only when present.
# All destination dirs are pre-created so COPY --from=builder in the runtime stage never fails.
RUN mkdir -p /build/out/static /build/out/routes /build/out/components /build/out/generated && \
    if [ -d "website/dist" ]; then \
      cp website/dist/server /build/out/website && \
      { [ -d website/dist/static ]    && cp -r website/dist/static/.    /build/out/static/    || true; } && \
      { [ -d website/routes ]         && cp -r website/routes           /build/out/routes     || true; } && \
      { [ -d website/components ]     && cp -r website/components       /build/out/components || true; } && \
      { [ -d website/generated ]      && cp -r website/generated        /build/out/generated  || true; }; \
    else \
      cp dist/server /build/out/website && \
      { [ -d dist/static ]    && cp -r dist/static/.    /build/out/static/    || true; } && \
      { [ -d routes ]         && cp -r routes           /build/out/routes     || true; } && \
      { [ -d components ]     && cp -r components       /build/out/components || true; } && \
      { [ -d generated ]      && cp -r generated        /build/out/generated  || true; }; \
    fi

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the normalized artifacts
COPY --from=builder /build/out/website ./website
COPY --from=builder /build/out/static ./static
COPY --from=builder /build/out/routes ./routes
COPY --from=builder /build/out/components ./components
COPY --from=builder /build/out/generated ./generated

RUN chmod +x ./website && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080
ENV PORT=8080
ENV GOSPA_DEV=false
ENV GOSPA_ENV=production

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

CMD ["./website"]
