# Build stage - must be built from project root
# Usage: docker build -t gospa-website -f website/Dockerfile .
# RENDER NOTE: Set "Docker Build Context" to "." (repository root) in your Render service settings.
FROM golang:1.24-alpine AS builder

# Install build dependencies
# We need bun for the client runtime and CSS builds, plus standard tools
RUN apk add --no-cache git ca-certificates tzdata curl bash unzip libc6-compat

# Install Bun
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

WORKDIR /build

# Copy everything for building
COPY . .

# Build the gospa CLI tool from source
# This ensures we use the local version with all its features and plugins
RUN go mod download
RUN go build -o /usr/local/bin/gospa ./cmd/gospa

# Install Node dependencies for client and website
# This is required for Tailwind/PostCSS and Client Runtime building
WORKDIR /build/client
RUN bun install

WORKDIR /build/website
RUN bun install

# Build the website for production
# This command generates templates, builds the client JS, and compiles the Go binary
RUN gospa build -o dist

# Runtime stage
# Using a slim alpine image for the final container
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the build artifacts from the builder stage
# gospa build puts everything in the output directory (dist)
COPY --from=builder /build/website/dist/server ./website
COPY --from=builder /build/website/dist/static ./static

# Copy directories needed by GoSPA at runtime (for scanning)
COPY --from=builder /build/website/routes ./routes
COPY --from=builder /build/website/components ./components
COPY --from=builder /build/website/generated ./generated

# Ensure appuser owns the app directory
RUN chown -R appuser:appuser /app

USER appuser

# Render and other platforms often use the PORT environment variable
EXPOSE 8080
ENV PORT=8080
ENV GOSPA_DEV=false

# Health check to ensure the server is responding
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the GoSPA application
CMD ["./website"]
