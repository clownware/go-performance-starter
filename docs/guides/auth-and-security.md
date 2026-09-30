# Auth and Security

What protects this app, where each control lives, and how to add a route that inherits all of it. The decisions are in [ADR-014](../adr/ADR-014-Security-Patterns-and-Threat-Model.md) (security patterns and the OWASP table), [ADR-004](../adr/ADR-004-Authorization-Strategy-RLS.md) (Row Level Security), [ADR-024](../adr/ADR-024-Demo-Application-Direction.md) (anonymous guests), [ADR-027](../adr/ADR-027-Trusted-Proxy-Client-IP.md) (client IP behind a proxy), [ADR-028](../adr/ADR-028-CSP-Alpine-Compatibility.md) (CSP), [ADR-031](../adr/ADR-031-Public-Demo-Operations.md) (public-demo abuse surface), [ADR-015](../adr/ADR-015-Configuration-Management-Strategy.md) (config and secrets) and [ADR-034](../adr/ADR-034-Live-Proof-Surfaces.md) (the RLS isolation check and `Retry-After`). The middleware order itself is in [routing-and-middleware.md](routing-and-middleware.md).

There is no RBAC layer, no server-side session store and no audit log. Identity is a Supabase JWT in a cookie; authorization is Postgres RLS keyed on that JWT's `sub`.

## Identity: Supabase JWTs in cookies

A session is two `HttpOnly`, `SameSite=Lax` cookies, `Secure` when `ENV=production` (`Config.IsProduction()`; TLS terminates at the edge, so `r.TLS` is never consulted): `sb-access-token` (max-age = GoTrue's `expires_in`) and `sb-refresh-token` (30 days). Login (`internal/handler/auth_handlers.go`), guest sign-in, the upgrade flow and the password-reset flow all issue the same pair with the same attributes.

`AuthMiddleware` validates the cookie by asking GoTrue — there is no local signature check and no JWT secret in the app:

```go
// internal/middleware/auth_middleware.go
	cookie, err := r.Cookie("sb-access-token") // Default Supabase cookie name
	if err != nil || cookie.Value == "" {
		slog.Debug("Unauthenticated: no access token cookie")
		return r.Context(), false
	}
	accessToken := cookie.Value

	// GetUser implicitly validates the token.
	user, err := authClient.Client.Auth.WithToken(accessToken).GetUser()
```

On success the context gains the gotrue user and a `webutil.AuthClaims{Sub, Role: "authenticated", IsAnonymous}` — `IsAnonymous` is read from the (already validated) token payload by `auth.TokenClaims`. On failure both cookies are cleared and the request goes to `/auth/page`: a 303 for browsers, `401` + `HX-Redirect` for HTMX (`toLogin`).

Sessions do not slide. An expired access token means a fresh login; the app never refreshes it on the user's behalf. The one exception is `RefreshSession` in `internal/auth/upgrade.go`, called once right after a guest upgrades so the new token's `is_anonymous` claim is false immediately instead of at expiry.

`UserLoader` runs after `AuthMiddleware` and resolves the claims to a `users` row, provisioning it just-in-time on the first authenticated request (`provisionUser`); the insert runs RLS-scoped, so `users_self_access`'s `WITH CHECK` proves the row belongs to the requester. Logout (`POST /auth/logout`) calls GoTrue's `Logout` and clears both cookies.

## Anonymous guests

With `GUEST_MODE_ENABLED=true` (default `false`; requires "anonymous sign-ins" enabled in the Supabase project), the `/learn` group in `internal/server/server.go` mounts `GuestSession` ahead of `OptionalAuth`. A visitor with no `sb-access-token` gets a real Supabase identity server-side: `AuthClient.SignInAnonymously` (`internal/auth/anon.go`) makes the credential-less `POST /auth/v1/signup` that `supabase-js` `signInAnonymously()` makes, and the middleware sets the same cookie pair as login. If GoTrue is down the page still renders unauthenticated — a warning, not a 500.

From there guests are ordinary users: `OptionalAuth` validates the cookie exactly like `AuthMiddleware` but lets a signed-out GET through, and `OptionalUserLoader` provisions the `users` row with `is_anonymous = true` (migration `000006`) and a placeholder email of `<sub>@guest.invalid`. Handlers guard every mutation themselves (`webutil.GetUserFromContext` nil → redirect to login).

Guests expire. `internal/jobs/reaper.go` runs every `REAPER_INTERVAL` (default `1h`) and deletes anonymous `users` rows older than `GUEST_TTL` (default `720h`; the demo's `fly.toml` sets `48h` to match the nightly content reset, ADR-031), then deletes the GoTrue twin via `AdminDeleteUser`. A second pass lists anonymous GoTrue identities with no `users` row and deletes those orphans. Both auth-side passes need `SUPABASE_SERVICE_ROLE_KEY`; without it only the rows go. The reaper's repository (`postgres.ReaperRepo`) is the one place the app runs as `service_role`. See [background-jobs.md](background-jobs.md).

Upgrading keeps everything. `POST /learn/upgrade` (`internal/handler/upgrade_handlers.go`) calls `UpgradeAnonymousUser`, a `PUT /auth/v1/user` on the current anonymous session that attaches email + password — the `auth.uid()` is unchanged, so every RLS-scoped row survives. The handler then flips `is_anonymous` on the row (`SetAnonymous`), replaces the placeholder email, and refreshes the session. If the row flip is missed, `resolveUser` in `user_loader.go` heals it on the next request; a row is only ever promoted, never demoted, so the reaper cannot take an upgraded account.

## Authorization is RLS, not RBAC

Every repository method runs through `inScope` (`internal/repository/postgres/scope.go`), which opens a transaction and applies the request's identity before running the sqlc query:

```go
// internal/repository/postgres/scope.go
	if claims, ok := webutil.AuthClaimsFromContext(ctx); ok {
		if err := claims.Validate(); err != nil {
			return zero, err
		}
		claimsJSON, err := claims.JSON()
		if err != nil {
			return zero, fmt.Errorf("marshal jwt claims: %w", err)
		}
		// claims.Validate() restricts Role to the authenticated/anon
		// allowlist, so interpolating it is safe (SET ROLE cannot take a
		// bind parameter).
		if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+claims.Role); err != nil {
			return zero, fmt.Errorf("set local role: %w", err)
		}
		// Set both the modern claims JSON and the legacy per-claim setting:
		// Supabase's auth.uid() coalesces request.jwt.claim.sub with
		// request.jwt.claims->>'sub' (the test stub mirrors this).
		if _, err := tx.Exec(ctx,
			"SELECT set_config('request.jwt.claims', $1, true), set_config('request.jwt.claim.sub', $2, true)",
			claimsJSON, claims.Sub,
		); err != nil {
```

`auth.uid()` then resolves to the requester inside the policies. Migration `000002` enables **and forces** RLS on `users`, `organizations` and `organization_members` (`FORCE ROW LEVEL SECURITY` means even the table owner is subject to it), with `users_self_access` as `auth_id = auth.uid()::text`. Migration `000003` does the same for `quiz_attempts` and `flashcards` (`*_self_access`, joined through `users`) and gives `quiz_questions` a public `SELECT`. Because RLS is forced, an unscoped query on Supabase returns nothing — `inScope` is what makes the app work at all, not an optional hardening.

The `service_role_bypass` policies are `FOR ALL TO service_role`. Migration `000005` exists because an earlier version of `000002` created them without the `TO service_role` scope, which OR-overrode the self-access policies and nullified RLS on already-migrated databases; `000005` drops and recreates them scoped. Only `ReaperRepo` runs as `service_role`; request handlers never do.

The flashcards page proves the boundary live (ADR-034). `runIsolationCheck` in `internal/handler/flashcard_isolation.go` runs the visitor's own `ListByUser` twice through the same `repository.FlashcardRepository`: once with the request's claims, once with a freshly minted stranger (`webutil.AuthClaims{Sub: uuid.NewString(), Role: webutil.RoleAuthenticated, IsAnonymous: true}`). Same SQL, same `user_id` parameter; the stranger gets zero rows because Postgres refused, not because of a `WHERE` clause. `GET /learn/flashcards/isolation` serves it as an HTMX fragment; a plain navigation lands on `/learn/flashcards?check=1`.

## CSRF

`internal/middleware/csrf.go` is a stateless double-submit cookie, mounted globally. Every request gets a random 32-byte hex token in the `HttpOnly` `csrf_token` cookie (12h, reissued transparently) and the same token in the request context. Safe methods pass; anything else must echo it back:

```go
// internal/middleware/csrf.go
			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil || !isValidTokenFormat(cookie.Value) {
				http.Error(w, "Forbidden: missing CSRF cookie", http.StatusForbidden)
				return
			}
			sent := r.Header.Get(CSRFHeaderName)
			if sent == "" {
				sent = r.PostFormValue(CSRFFormField)
			}
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(sent)) != 1 {
				http.Error(w, "Forbidden: CSRF token mismatch", http.StatusForbidden)
				return
			}
```

Two transports carry the token back. `internal/view/layouts/base.templ` sets `hx-headers` on `<body>` so every HTMX request sends `X-CSRF-Token`; plain forms include the hidden field:

```templ
// internal/view/components/csrf.templ
templ CSRFField() {
	<input type="hidden" name="csrf_token" value={ webutil.CSRFTokenFromContext(ctx) }/>
}
```

HTMX serialises form fields too, so a form with `@components.CSRFField()` works with and without JavaScript. There is no per-session binding or rotation; the compare is constant-time.

## Rate limiting

`internal/middleware/ratelimit.go` keeps one `golang.org/x/time/rate` token bucket per client IP (`r.RemoteAddr`, already normalised by `RealIP`), evicts buckets idle for 10 minutes, and answers a refused request with `429` plus a whole-second `Retry-After` (never `0`). `RateLimiterWith` takes a custom refusal handler so the `/patterns` demo can render an HTML 429 from the same limiter. The tiers, as wired in `internal/server/server.go` and the handlers:

| Where | Call | Effective |
|---|---|---|
| Global (every route) | `mw.RateLimiter(ctx, 50, 10)` | 50 req/s, burst 10 |
| `/learn` group (guest-writable) | `mw.RateLimiter(ctx, 30.0/60.0, 20)` | 30/min, burst 20 |
| `POST /auth/{login,signup,recover,reset}` | `mw.RateLimiter(ctx, 5.0/60.0, 5)` | 5/min, burst 5 |
| `POST /learn/upgrade`, `POST /profile/password` | same strict tier | 5/min, burst 5 |
| `/patterns` rate-limit demo | `mw.RateLimiterWith(ctx, 1/float64(partials.PatternLimitPerSeconds), partials.PatternLimitBurst, …)` | 1 per 2 s, burst 3 |

Tiers stack: a login attempt spends a token in the global bucket and the strict bucket. Per-client limiting only means anything when the limiter sees the real client address, which is the next section. ADR-014's per-user and per-email tiers are not implemented; the OWASP table there records A07 as partial for that reason.

## Client IP behind a proxy

`RealIP(trustedProxyCIDRs, clientIPHeader)` (`internal/middleware/realip.go`) strips the port from `RemoteAddr` and honours forwarded headers **only when the direct peer is inside `TRUSTED_PROXY_CIDRS`**. The default is empty, which trusts nothing: behind a proxy every request then shares the proxy's bucket (over-limiting), but a client hitting the app directly can never spoof its way into a fresh bucket. `config.Validate` rejects a malformed CIDR at boot.

When the peer is trusted, `CLIENT_IP_HEADER` (e.g. `Fly-Client-IP`, `CF-Connecting-IP`) is consulted first; otherwise `X-Forwarded-For` is walked right-to-left skipping trusted addresses, because edge proxies append rather than replace, so the left-most entry is attacker-controlled. The demo's `fly.toml` sets `TRUSTED_PROXY_CIDRS = "172.16.0.0/12"` and `CLIENT_IP_HEADER = "Fly-Client-IP"` — the values ADR-027's live verification found necessary.

## Security headers and CSP

`SecurityHeaders(isProd)` (`internal/middleware/security.go`) is the outermost middleware, so even errors carry the headers:

```go
// internal/middleware/security.go
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// X-XSS-Protection set to 0 per OWASP recommendation — CSP supersedes it,
			// and non-zero values can introduce XSS vulnerabilities in older browsers.
			w.Header().Set("X-XSS-Protection", "0")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-eval'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; "+
					"font-src 'self'; "+
					"connect-src 'self'; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'")
			if isProd {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
```

HSTS is gated on `ENV=production` rather than `r.TLS` because TLS terminates at the edge. `'unsafe-eval'` is there because Alpine 3 compiles `x-data`/`x-show` expressions with the `Function` constructor; without it every Alpine behaviour fails silently (ADR-028). Inline `<script>` stays forbidden — `internal/server/server_test.go` fails any rendered page whose script tag lacks `src`, and `internal/middleware/security_test.go` pins the exact policy. The primary XSS defence is templ's contextual escaping ([ADR-017](../adr/ADR-017-Templ-Adoption.md)); CSP is the second layer.

## Request body cap, /metrics guard, input validation

- `MaxBodyBytes(MAX_REQUEST_BODY_BYTES)` (`internal/middleware/bodylimit.go`, default `1048576`) wraps the body in `http.MaxBytesReader` before any handler reads it; an over-limit read fails and the server answers `413`. `cmd/api/main.go` also sets `ReadHeaderTimeout` 5s, `ReadTimeout` 15s, `WriteTimeout` 45s, `IdleTimeout` 120s, and the router applies a 30s request timeout.
- `MetricsGuard(METRICS_TOKEN, isProd)` (`internal/middleware/metrics_guard.go`) fronts `/metrics`: with a token set it requires `Authorization: Bearer <token>` (constant-time compare) in every environment; with no token the endpoint is open in development and `404` in production.
- `internal/validate/validate.go` provides `Required`, `MaxLength`, `MinLength`, `Email` (via `net/mail.ParseAddress`, bare addresses only) and `Slug`. Handlers validate server-side before touching a repository — e.g. flashcards cap both fields at `flashcardFieldMax = 500`, and the upgrade form runs `validate.Email` and an 8-character password minimum. No HTML sanitiser is used; user text is escaped by templ on output.
- SQL is never built from strings: sqlc queries only ([ADR-003](../adr/ADR-003-SQL-Code-Generation-and-Data-Access.md)), checked by `adr003-no-sql-in-handlers` in `scripts/adrcheck`.

## Secrets

Configuration is environment-only, parsed by `envconfig` in `internal/config/config.go` and validated at boot (`SUPABASE_URL` and `SUPABASE_ANON_KEY` must be set together; both empty disables auth entirely). `SUPABASE_SERVICE_ROLE_KEY` is optional and only unlocks the reaper's GoTrue-side cleanup. Locally, run tasks under 1Password with nothing on disk: `op run --env-file=.env.tpl -- task <name>` (the header of `Taskfile.yml`). In production, `DATABASE_URL`, `SUPABASE_*` and `METRICS_TOKEN` go through `fly secrets set`; non-secret tuning lives in `fly.toml [env]`. Two `scripts/adrcheck` rules back this: `adr015-env-only-config` (env reads only in `internal/config`) and `adr015-no-hardcoded-secrets` (credential-shaped literals in shipped source).

## Password reset

`GET /auth/recover` renders the form; `POST /auth/recover` calls GoTrue `Recover` and always answers with the same generic message, whether the email exists, is unknown or was throttled — the endpoint cannot enumerate accounts, and the email is never logged. The Supabase *Reset Password* email template must link to `{{ .SiteURL }}/auth/reset?token_hash={{ .TokenHash }}&type=recovery` (see the README): `GET /auth/reset` exchanges that `token_hash` server-side via `VerifyRecovery` (`POST /auth/v1/verify`, `internal/auth/recover.go`), issues the normal cookie pair, and shows the new-password form; an invalid or expired hash renders a "request a new link" state and sets no cookies. `POST /auth/reset` then calls `UpdateUser` with the recovery session's access token. Both POSTs sit on the strict 5/min tier.

## Adding a protected route

1. **Pick the identity chain** in `internal/server/server.go`. Must-be-signed-in pages go in the `protectedRouter` group (`AuthMiddleware` + `UserLoader`; unauthenticated requests are redirected). Browse-first pages that guests may use go in the `learn` group (`GuestSession` when enabled, `OptionalAuth`, `OptionalUserLoader`); there, check `webutil.GetUserFromContext(r.Context()) == nil` on every mutation and redirect to `/auth/page`, as `flashcardCreate` and `UpgradeSubmit` do.
2. **Forms carry the CSRF token.** Add `@components.CSRFField()` inside the `<form>`; HTMX requests already inherit `hx-headers` from `<body>`. Nothing else to configure — the middleware is global.
3. **Data is scoped by RLS, not by your handler.** Add the query to `sql/queries/`, run `task db:generate`, and call it through a repository so it runs inside `inScope`. A new table needs `ENABLE` + `FORCE ROW LEVEL SECURITY`, a `*_self_access` policy on `auth.uid()`, and a `service_role_bypass` policy scoped `TO service_role` — copy the shape from migration `000003`.
4. **Add a tier if the route is credential-bearing or guest-writable:** `.With(mw.RateLimiter(ctx, 5.0/60.0, 5))` on the route (as `/profile/password` does) for the former; the `/learn` group already carries the latter. Pass the server's lifecycle `ctx` so the eviction goroutine stops on `Server.Close`.
5. **Validate input with `internal/validate`** before the repository call and write the table-driven test first ([ADR-023](../adr/ADR-023-Testing-Philosophy.md)); `task ci` runs the header, CSP, rate-limit and RLS tests that pin everything above.
