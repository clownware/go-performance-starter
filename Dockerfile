# syntax=docker/dockerfile:1

# Stage 1: Build frontend assets
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy package files
COPY package*.json ./

# Install dependencies. The Tailwind toolchain lives in devDependencies and
# this is a builder stage, so install everything — `--omit=dev` would skip
# the CSS compiler this stage exists to run (and the removed `--only` flag
# errors on npm 10+).
RUN npm ci

# Copy frontend source and templ files (needed for @source directive)
COPY web ./web
COPY internal/view ./internal/view

# Build CSS with Tailwind
RUN npx @tailwindcss/cli -i ./web/static/css/input.css -o ./web/static/css/app.css --minify

# Stage 2: Build Go application
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built CSS from frontend stage
COPY --from=frontend-builder /build/web/static/css/app.css ./web/static/css/app.css

# Install templ CLI and generate Go code from .templ files
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1001
RUN templ generate

# Build the application with optimization flags
# -s: strip symbol table
# -w: strip DWARF debug info
# -trimpath: remove file system paths from executable
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -trimpath \
    -o app \
    ./cmd/api

# Stage 3: Create minimal runtime image
FROM alpine:3.21

# Install runtime dependencies only. tzdata is intentionally absent: nothing
# calls time.LoadLocation; if that changes, build with -tags timetzdata
# rather than reinstalling the ~3.5MB package (30MB image budget, ADR-000).
RUN apk add --no-cache \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder. Ownership is set at COPY time: a separate
# `RUN chown -R` would rewrite every file into an extra image layer,
# duplicating ~37MB (this alone blew the 30MB budget).
COPY --from=go-builder --chown=appuser:appuser /build/app ./app

# Copy static assets
COPY --from=go-builder --chown=appuser:appuser /build/web ./web

# Copy migrations (if needed at runtime)
COPY --from=go-builder --chown=appuser:appuser /build/migrations ./migrations

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4000/healthz || exit 1

# Run the application
ENTRYPOINT ["./app"]
