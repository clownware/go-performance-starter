# syntax=docker/dockerfile:1

# Stage 1: Build frontend assets
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy frontend source
COPY web ./web
COPY tailwind.config.js ./
COPY postcss.config.js ./

# Build CSS with Tailwind
RUN npx tailwindcss -c ./tailwind.config.js -i ./web/static/css/input.css -o ./web/static/css/app.css --minify

# Stage 2: Build Go application
FROM golang:1.24-alpine AS go-builder

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

# Install runtime dependencies only
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/app ./app

# Copy static assets
COPY --from=go-builder /build/web ./web

# Copy migrations (if needed at runtime)
COPY --from=go-builder /build/migrations ./migrations

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4000/healthz || exit 1

# Run the application
ENTRYPOINT ["./app"]
