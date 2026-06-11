# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-04-05

### Added
- Type-safe templ component library: buttons, cards, forms, inputs, alerts,
  accessibility helpers — all with typed props structs (ADR-017)
- ADR-017 documenting templ adoption rationale and migration strategy
- `UserName` field on `BaseProps` — user menu now shows the authenticated user's
  display name (falls back to email, then "User"); guest pages show "Sign In" link
- `lint:install` Taskfile task — auto-installs golangci-lint for local dev parity
  with CI
- HTMX response helpers (`SetHXTrigger`, `SetHXRedirect`) consolidated into `view`
  package alongside `IsHTMXRequest`

### Changed
- **Complete templ migration** — all pages and partials converted from `html/template`
  to [templ](https://templ.guide) type-safe templates
  - Pages: home, dashboard, terms, privacy, logout, auth (login/signup), profile, items
  - Partials: profile_form, items_list, item (optimistic UI), first_run onboarding,
    empty_state
  - All template data is now typed props structs instead of `map[string]interface{}`
  - HTMX partial responses render fragments directly (no layout wrapper)
- **Tailwind CSS 4 upgrade** — removed `postcss.config.js` and `tailwind.config.js`
  in favor of CSS-based `@import "tailwindcss"` configuration
- Semantic color system updated to Tailwind 4 `@theme` syntax
- CI workflow and Dockerfile updated for Tailwind 4 build process
- `lint` task now auto-installs golangci-lint if missing

### Removed
- `web/templates/` directory — all legacy `.html` template files
- `webutil.RenderTemplate`, `RenderTemplateWithErrors`, `FormErrors`,
  `view.TemplateFuncs()` — no longer needed with templ
- Unused component templates that were defined but never invoked
- `postcss.config.js` and `tailwind.config.js` (replaced by CSS-based config)
- `webutil/htmx_helpers.go` — HTMX helpers consolidated into `view` package
- Unused HTMX helpers: `GetHTMXRequest`, `AddHTMXResponseHeaders`,
  `SetHXTriggerAfterSwap`, `SetHXTriggerAfterSettle`

### Fixed
- Hardcoded "John Doe" in user menu, profile page, and API stub replaced with
  dynamic user data from auth context (closes #3)

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

[Unreleased]: https://github.com/clownware/alpine-go-performance-starter/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/clownware/alpine-go-performance-starter/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/clownware/alpine-go-performance-starter/releases/tag/v0.1.0
