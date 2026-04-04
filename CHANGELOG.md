# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-04-04

### Added
- Go (Chi) server with graceful shutdown and signal handling
- Supabase authentication integration (login, signup, logout)
- HTMX + Alpine.js frontend with dark mode support
- Tailwind CSS with semantic color system
- PostgreSQL database layer via pgx and sqlc
- Prometheus metrics and performance budget tracking
- Optimistic UI for item favorite toggle
- HTMX items list with pagination and typeahead
- Profile page with HTMX form submission
- Health check endpoint (`/healthz`)
- Dockerfile with multi-stage build (frontend + Go + minimal Alpine runtime)
- GitHub Actions workflow for Supabase DB migrations
- Database migrations via golang-migrate
- Hot-reload development with Air
- Taskfile for common operations (build, test, lint, migrate, css)
- golangci-lint configuration
- Performance budget tests and binary size checks

### Fixed
- `envconfig` struct tags corrected (`env:` → `envconfig:`) — config loading
  no longer relies on manual `os.Getenv` fallbacks
- Version variable (`main.version`) now declared in `main.go` so `-ldflags -X`
  injection works correctly
- Build commands in Taskfile now include `-ldflags` version injection

[Unreleased]: https://github.com/yourusername/go-alpine-saas-starter/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/yourusername/go-alpine-saas-starter/releases/tag/v0.1.0
