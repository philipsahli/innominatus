# Multi-stage Dockerfile for innominatus
# Builds Go binaries and Next.js web-ui in one optimized container

# Stage 1: Build Go binaries
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
COPY migrations/ ./cmd/server/migrations/
COPY swagger-admin.yaml ./cmd/server/swagger-admin.yaml
COPY swagger-user.yaml ./cmd/server/swagger-user.yaml

# Build server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o innominatus cmd/server/main.go

# Build CLI binary (useful for debugging inside container)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o innominatus-ctl cmd/cli/main.go

# Stage 2: Build Next.js web-ui
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

# Build Next.js for production with standalone output (for Docker)
ENV DOCKER_BUILD=true
RUN npm run build

# Stage 3: Runtime image
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy Go binaries from go-builder
COPY --from=go-builder /build/innominatus /app/innominatus
COPY --from=go-builder /build/innominatus-ctl /app/innominatus-ctl

# Copy Next.js static export from web-builder (uses 'export' mode)
COPY --from=web-builder /web-ui/out /app/web-ui/out

# Copy configuration files from examples directory
COPY examples/admin-config.yaml /app/admin-config.yaml
COPY examples/goldenpaths.yaml /app/goldenpaths.yaml
COPY workflows/ /app/workflows/
COPY providers/ /app/providers/

# Expose server port
EXPOSE 8081

# Set environment variables
ENV PORT=8081
ENV WEB_UI_PATH=/app/web-ui/out

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/innominatus", "--help"]

# Run server
ENTRYPOINT ["/app/innominatus"]
CMD ["--port", "8081"]
