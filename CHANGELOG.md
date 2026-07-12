# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.1] - 2026-07-11

First release deployed against hosted Supabase: security-audit fixes, the
migration bug that blocked production, and the handler/repository test-coverage
gaps closed.

### Security
- Client IP resolution now honors only trusted proxies (ADR-027,
  `TRUSTED_PROXY_CIDRS`; default trusts none) so rate limits can't be bypassed
  with forged `X-Forwarded-For` headers
- Request bodies capped before handlers read them (`MAX_REQUEST_BODY_BYTES`,
  1 MiB default)
- Cookie `Secure` flags made consistent across auth middleware and logout
  (edge-TLS aware per ADR-025 instead of `r.TLS`)
- Form-validation `x-data` JSON-escaped, closing a latent XSS path in the
  shared form component
- 5xx JSON error bodies genericized so internal error strings never reach
  clients
- Go toolchain bumped to 1.26.5 (GO-2026-5856, `crypto/tls`); release workflow
  actions pinned to commit SHAs

### Fixed
- Migration `000002` created its RLS helper function in the Supabase-reserved
  `auth` schema, which hosted Supabase rejects (`permission denied`); moved to
  `public`. CI's vanilla-Postgres auth stub owns that schema, which masked the
  failure until the first production migration run

### Added
- Handler tests for the auth flow (login/signup/logout against an `httptest`
  fake GoTrue), profile, and first-run onboarding; integration tests for the
  organization and organization-member repositories

## [0.3.0] - 2026-07-05

Deployment-readiness release: the 2026-07-05 audit's findings implemented end
to end — decided deployment topology, security hardening, runtime RLS
enforcement, unified logging, guest-mode backend, and a release pipeline.

### Added
- **ADR-025 Deployment Target**: stateless container behind the Cloudflare
  proxy (Fly.io worked example, `fly.toml` at repo root); edge-terminated TLS;
  stateless JWT-cookie sessions; Supabase-delegated backups; forward-only
  pre-deploy migrations
- **ADR-026 Logging Standardization**: stdlib `log/slog` everywhere
- **CSRF protection** (ADR-014 §3): double-submit-cookie middleware, token via
  `hx-headers` on `<body>` plus hidden-field fallback in all POST forms
- **Runtime RLS enforcement** (ADR-004): every repository call carries the
  requester's JWT identity onto the connection (`SET LOCAL ROLE` +
  `request.jwt.claims`); CI integration tests prove cross-user isolation
  through the production path
- **Anonymous guest-mode backend** (ADR-024): server-side GoTrue anonymous
  sign-in, `is_anonymous` claim plumbing, `GuestSession` middleware, TTL
  reaper with GoTrue-side cleanup (`GUEST_MODE_ENABLED`, `GUEST_TTL`,
  `REAPER_INTERVAL`)
- **User provisioning**: `UserLoader` middleware JIT-creates the `users` row
  on first authenticated request (RLS `WITH CHECK` proves ownership); fixes
  the first-run onboarding flow
- Tiered rate limiting: 5/min per IP on login/signup atop the global limiter
- HTTP server timeouts (read/write/idle/read-header); env-tunable pgxpool
  (`DB_MAX_CONNS` etc.); `config.Validate()` fail-fast at boot
- `/metrics` guard: bearer `METRICS_TOKEN`, hidden in production when unset
- HSTS emitted when `ENV=production`; gzip compression middleware
- Release workflow: ghcr.io image publish with a 30MB image-budget gate,
  migrations applied before deploy, secret-gated Fly.io deploy
- DB migration workflow now validates the full chain against a fresh Postgres
  before touching production (and actually triggers — it watched the wrong
  branch)
- Migrations 000005 (rescope `service_role_bypass` for pre-fix deployments)
  and 000006 (`users.is_anonymous`)

### Changed
- ADR-024 accepted (demo direction + anonymous sign-in mechanism); ADR-001/013
  amended per ADR-025/026; ADR-014's OWASP table corrected to audited reality
- Logging: env-driven setup — JSON in production, `LOG_LEVEL` (default info);
  auth logs no longer contain email addresses (ADR-014 §7)
- Go toolchain pinned to a patched release; pgx upgraded (govulncheck clean)

### Removed
- zerolog dependency; dead `profile_handler.go`; stub `user_postgres.go`
  (nil-returning methods and a hand-written SQL string)

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

[Unreleased]: https://github.com/clownware/alpine-go-performance-starter/compare/v0.3.1...HEAD
[0.3.1]: https://github.com/clownware/alpine-go-performance-starter/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/clownware/alpine-go-performance-starter/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/clownware/alpine-go-performance-starter/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/clownware/alpine-go-performance-starter/releases/tag/v0.1.0
