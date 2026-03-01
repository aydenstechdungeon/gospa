# Build stage
# Usage: docker build -t gospa-website -f website/Dockerfile .
# RENDER NOTE: While this Dockerfile is now "context-agnostic", for best performance,
# set "Docker Build Context" to "." in Render settings.
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
RUN if [ -f "cmd/gospa/main.go" ]; then \
    echo "Building GoSPA CLI from local source..." && \
    go mod download && \
    go build -o /usr/local/bin/gospa ./cmd/gospa; \
    else \
    echo "Local GoSPA source not found in context, installing via go install..." && \
    GOPROXY=direct go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest && \
    cp /go/bin/gospa /usr/local/bin/gospa; \
    fi

# 5. Build the website
# Detect if we are in the repository root or the website subdirectory
RUN if [ -d "website" ]; then \
    echo "Context: Repository Root. Building website subdirectory..." && \
    cd website && \
    go mod edit -replace github.com/aydenstechdungeon/gospa=../ && \
    go mod tidy && \
    bun install && \
    /usr/local/bin/gospa build -o dist; \
    else \
    echo "Context: Website Directory. Building current directory..." && \
    go mod edit -dropreplace github.com/aydenstechdungeon/gospa && \
    go mod tidy && \
    bun install && \
    /usr/local/bin/gospa build -o dist; \
    fi

# 6. Normalize artifacts into /build/out for the runtime stage
RUN mkdir -p /build/out && \
    if [ -d "website/dist" ]; then \
    cp website/dist/server /build/out/website && \
    cp -r website/dist/static /build/out/static && \
    cp -r website/routes /build/out/routes && \
    cp -r website/components /build/out/components && \
    cp -r website/generated /build/out/generated; \
    else \
    cp dist/server /build/out/website && \
    cp -r dist/static /build/out/static && \
    cp -r routes /build/out/routes && \
    cp -r components /build/out/components && \
    cp -r generated /build/out/generated; \
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
