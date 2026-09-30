# Routing and middleware

The router is Chi ([ADR-001](../adr/ADR-001-Foundation.md)); routes and the middleware stack are wired in `internal/server/server.go`, and handlers live in `internal/handler/` grouped by surface, each exposing a `…Routes(r chi.Router, …)` function the server mounts ([ADR-012](../adr/ADR-012-Routing-and-UI-Patterns.md)). Handlers depend on repository interfaces, build typed props, and render through `view.Render` ([view-layer.md](view-layer.md)).

## Middleware order

`setupMiddleware`, outermost first. Order is load-bearing; the comments in the file say why each one sits where it does.

1. **Security headers** (`mw.SecurityHeaders`) — on every response, including errors; HSTS only in production ([ADR-014](../adr/ADR-014-Security-Patterns-and-Threat-Model.md), [ADR-025](../adr/ADR-025-Deployment-Target.md)).
2. **Request ID** (`mw.RequestID`).
3. **Real IP** (`mw.RealIP`) — honours forwarded headers only from `TRUSTED_PROXY_CIDRS`; must precede the rate limiter ([ADR-027](../adr/ADR-027-Trusted-Proxy-Client-IP.md)).
4. **Body cap** (`mw.MaxBodyBytes`, `MAX_REQUEST_BODY_BYTES`) — before anything reads the body.
5. **Rate limiter** (`mw.RateLimiter`) — global per-IP token bucket; refusals are 429 with `Retry-After` ([ADR-034](../adr/ADR-034-Live-Proof-Surfaces.md)).
6. **Compression** (`mw.Compress(5)`).
7. **Metrics** (`mw.Metrics`) — Prometheus histograms and the `Server-Timing` header.
8. **Request logger** (`mw.RequestLogger`) — slog with request ID and path ([ADR-026](../adr/ADR-026-Logging-Standardization.md)).
9. **Recoverer** — inside Request ID so panic logs carry it, inside the logger so a recovered panic is a logged 500 rather than a dropped connection.
10. **Timeout** (`mw.Timeout(30s)`) — the `http.Server` `WriteTimeout` in `cmd/api/main.go` is deliberately longer (45s) so the request timeout fires first.
11. **CSRF** (`mw.CSRF`) — double-submit cookie, validated on unsafe methods ([auth-and-security.md](auth-and-security.md)).

## Route groups

| Group | Middleware added | What mounts there |
|---|---|---|
| Public | — | `/`, `/terms`, `/privacy`, SEO routes (`/robots.txt`, `/sitemap.xml` when `PUBLIC_BASE_URL` is set), `/auth/logout`, static assets under `/static/` with a one-year cache header, `handler.PatternsRoutes` (`/patterns` + its stub `/patterns/api/*`) |
| Health | — | `/healthz` (liveness, used by the Dockerfile `HEALTHCHECK` and `fly.toml`), `/health` (readiness; pings the database through a `handler.Pinger` seam) |
| Metrics | `mw.MetricsGuard` | `/metrics`, bearer `METRICS_TOKEN` |
| Authenticated | `AuthMiddleware`, `UserLoader` | `/profile` (view and update) |
| `/learn` | `OptionalAuth`, `GuestSession` (mints an anonymous identity on first touch when `GUEST_MODE_ENABLED`), `OptionalUserLoader`, a per-route rate tier | `handler.QuizRoutes`, `FlashcardRoutes`, `DashboardRoutes`, `UpgradeRoutes` |
| `/auth` | strict credential rate tier on the POST group | `/auth/page`, `/auth/recover`, `/auth/reset`, login / signup / logout POSTs |

The Supabase-dependent groups are registered only when the auth client is configured; without credentials the app still serves the public surfaces and a non-functional `/auth/page` ([ADR-024](../adr/ADR-024-Demo-Application-Direction.md)).

## Adding a route

1. Put the handler in the `internal/handler/` file for its surface, taking repository interfaces as parameters, and register it in that surface's `…Routes` function.
2. Mount under the group whose middleware you need. Anything that touches per-user rows goes under `/learn` or the authenticated group so the request carries claims for RLS.
3. State-changing routes are `POST` (the starter standardises progressive-enhancement forms on POST; there is no `hx-delete`) and the form includes `@components.CSRFField()`.
4. Validate input at the boundary with `internal/validate`; return user-safe messages, keep detail in the log ([`.claude/engineering.md`](../../.claude/engineering.md)).
5. Test it with `httptest` against fakes ([testing.md](testing.md)); `internal/server/wiring_test.go` asserts the mounted routes and middleware, so extend it if the route matters to the wiring.
