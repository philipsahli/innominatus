# Multi-stage Dockerfile for innominatus
# Builds Next.js web-ui FIRST, then embeds it into Go binary

# Stage 1: Build Next.js web-ui (MUST be first for embedding)
FROM node:20-alpine AS web-builder

WORKDIR /web-ui

# Copy package files
COPY web-ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy web-ui source
COPY web-ui/ ./

# Copy docs directory to parent level (required by Next.js docs lib)
COPY docs/ /docs/

# Build Next.js for production with static export (for embedding)
ENV DOCKER_BUILD=true
RUN npm run build

# Stage 2: Build Go binaries with embedded web-ui
FROM golang:1.25-alpine AS go-builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY examples/ ./examples/

# Prepare embedded files (will be embedded into Go binary)
COPY migrations/ ./cmd/server/migrations/
COPY swagger-admin.yaml ./cmd/server/swagger-admin.yaml
COPY swagger-user.yaml ./cmd/server/swagger-user.yaml

# Copy built web-ui from web-builder stage (will be embedded)
COPY --from=web-builder /web-ui/out ./cmd/server/web-ui-out

# Build server binary with ALL files embedded (migrations, swagger, web-ui)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o innominatus cmd/server/main.go

# Build CLI binary (useful for debugging inside container)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o innominatus-ctl cmd/cli/main.go

# Stage 3: Runtime image (minimal - only binary and config)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy Go binaries from go-builder (migrations, swagger, web-ui are EMBEDDED)
COPY --from=go-builder /build/innominatus /app/innominatus
COPY --from=go-builder /build/innominatus-ctl /app/innominatus-ctl

# Copy configuration files from examples directory
COPY examples/admin-config.yaml /app/admin-config.yaml
# NOTE: goldenpaths.yaml removed - golden paths now defined in provider manifests
COPY workflows/ /app/workflows/
COPY providers/ /app/providers/

# Expose server port
EXPOSE 8081

# Set environment variables
ENV PORT=8081
# NOTE: WEB_UI_PATH not needed - web-ui is embedded in binary

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/innominatus", "--help"]

# Run server (migrations, swagger, and web-ui served from embedded files)
ENTRYPOINT ["/app/innominatus"]
CMD ["--port", "8081"]
