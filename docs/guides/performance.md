# Performance budgets

The budgets are numbers in [ADR-000](../adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md); the constants that encode them live in `internal/performance/budgets.go`, and [ADR-034](../adr/ADR-034-Live-Proof-Surfaces.md) added the runtime observer that measures the process against them. This guide is for keeping a change inside budget and knowing which numbers are gates, which are readings, and which are still targets on paper.

## The budgets

ADR-000 §1 classes every budget as **Enforced** (`task ci` fails), **Monitored** (measured, informs, does not gate) or **Aspirational** (no measurement exists).

| Budget | Value | Class | Constant | Where it is checked or read |
|---|---|---|---|---|
| Binary size (stripped linux) | < 20 MB | Enforced | `MaxBinarySize` | `task test:binary-size`; PR comment in `ci.yml` |
| Docker image | < 30 MB | Enforced | — (shell in workflows) | `docker` job in `ci.yml`; `release.yml` |
| JavaScript, gzipped | < 50 KB | Enforced | `MaxJavaScriptSize` | `task test:asset-budgets` |
| CSS, gzipped | < 30 KB | Enforced | `MaxCSSSize` | `task test:asset-budgets` |
| P50 / P95 / P99 response | < 50 / 100 / 200 ms | Monitored | `MaxP50/P95/P99ResponseTime` | metrics middleware, landing-page grid |
| Memory (steady) | < 128 MB | Monitored | `MaxMemoryUsage` | `runtime.MemStats`, grid, `/metrics` |
| Memory (peak) | < 256 MB | Monitored | `MaxPeakMemory` | high-water mark since boot, grid |
| Startup | < 500 ms | Monitored | `MaxStartupTime` | `cmd/api/main.go`, grid |
| Total page weight | < 500 KB | Aspirational | `MaxTotalPageSize` | nothing — renders as unmeasured |
| DB query time (P95) | < 10 ms | Monitored per ADR-000 | none | not measured anywhere in code |

The last row is honest: ADR-000 lists a query-time budget under the Monitored heading, but no constant, histogram or grid row exists for it. Core Web Vitals, critical-path weight, the Lighthouse thresholds and the scalability targets are all Aspirational and unmeasured.

## What fails the build

Four gates run inside `task ci` or the CI workflows. Each one is a hard fail.

- **Binary size** — `task test:binary-size` builds `./dist/app` with `-ldflags="-s -w -X main.version=..."` and runs `scripts/check-binary-size.go`, which calls `performance.CheckBinarySize`. The `ci` job also posts the size on every PR (`Comment PR with binary size` step, `continue-on-error: true`).
- **Docker image** — the `docker` job in `.github/workflows/ci.yml` builds the image on every PR (no push) and runs `docker image inspect --format='{{.Size}}'` against `30 * 1024 * 1024`. `release.yml` repeats the same check before pushing to ghcr.io. The `Dockerfile` builds with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`, `-ldflags="-s -w"` and `-trimpath`, and runs on `alpine:3.21` with only `ca-certificates` installed; the comments there record two things that blew the budget before (a separate `RUN chown -R` layer, and `tzdata`).
- **JS and CSS gzipped** — `task test:asset-budgets` rebuilds `app.css` (`css:build`, `--minify`) and runs `scripts/check-asset-budgets`, which sums `performance.GzippedTotal` over `ShippedJS` and `ShippedCSS`:

```go
// internal/performance/budgets.go
var (
	ShippedJS  = []string{"js/htmx.min.js", "js/alpine.min.js", "js/app.js"}
	ShippedCSS = []string{"css/app.css"}
)
```

If you add a script or stylesheet to `internal/view/layouts/base.templ`, add it to these lists or the gate measures the wrong thing. The runtime observer reads the same lists, so the landing page and CI can never disagree about what "shipped" means (ADR-034 TC-2).

When a gate trips, do not raise the constant, the workflow threshold, or skip the task. Constitution rule 7 says halt and open an ADR; ADR-000 §3 says any budget increase requires an explicit ADR update. Trim the change first — a new Go dependency is the usual binary-size culprit, a new vendored library the usual JS one.

## What is measured but not gated

`internal/performance/observed.go` holds an `Observer` (`performance.Default`, static root `web/static`) that the middleware and `main` feed:

- **Latency** — `LatencyRecorder` is a ring of the last `DefaultLatencyWindow = 4096` request durations; `Percentiles()` sorts a copy and returns nearest-rank p50/p95/p99. `Metrics` in `internal/middleware/metrics.go` calls `performance.Default.RecordLatency(duration)` after every request, alongside the Prometheus `http_request_duration_seconds` histogram. Any request over `MaxP95ResponseTime` is logged as `slow request detected`, and `performance_budget_violations_total{budget_type,path}` increments for each of the three thresholds a single request crosses.
- **Memory** — `ObserveMemory` reads `runtime.MemStats` and advances a since-boot high-water mark of `Sys`. `StartMemoryMetricsCollector(30 * time.Second)` in `main.go` calls it every 30 s so the peak is tracked even when nobody loads the page, and exports `go_memory_usage_bytes{type="alloc"|"sys"|"stack"}`.
- **Startup** — `main.go` binds the listener with `net.Listen` before `Serve`, so "startup" is process start to a listening socket exactly as ADR-000 defines it. The value goes to `RecordStartup` and, if over 500 ms, a `startup budget exceeded` warning.
- **Binary and asset size** — `Snapshot()` stats the running executable and gzips the shipped lists once, then caches them.

Every zero in a `Snapshot` means "not measured", and `handler.PerfBudgetStats` in `internal/handler/home_explainer.go` renders it as unmeasured — never as a passing zero. Three places show the readings:

1. The landing page grid (`/`): budget, observed, pass/fail and ADR-000 class per row. A dev binary is unstripped, so the binary row may read over budget locally; the note says CI gates the linux build.
2. `/metrics`, behind `MetricsGuard`: with `METRICS_TOKEN` set, `Authorization: Bearer <token>` is required; with no token it is open in development and returns 404 in production.
3. `Server-Timing: app;dur=<ms>` on every response, stamped by the metrics middleware's response writer at the first `WriteHeader` or `Write` — handler time to first byte, visible in browser devtools.

These are per-process readings since boot on whichever instance served the request, not SLOs. ADR-000's graduation rule stands: a Monitored budget becomes Enforced only once a production baseline exists to calibrate it, and no load-test harness is wired.

## Server tuning that exists

```go
// cmd/api/main.go
func newHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}
```

`WriteTimeout` is deliberately longer than the per-request `mw.Timeout(30 * time.Second)` in `internal/server/server.go` (chi's `middleware.Timeout`), so a handler's context deadline fires before the connection is cut. Graceful shutdown waits 5 s.

Responses are compressed by `mw.Compress(5)` (chi's `middleware.Compress`, gzip/deflate at level 5), mounted before `Metrics` so the recorded duration includes compression.

The pgx pool is configured in `internal/config/config.go` from `DB_MAX_CONNS` (default 25), `DB_MIN_CONNS` (2), `DB_MAX_CONN_LIFETIME` (30m) and `DB_MAX_CONN_IDLE_TIME` (5m); `Validate` rejects `DB_MAX_CONNS < 1` and a min above the max.

## Frontend weight

The base layout ships three vendored scripts and one Tailwind build; nothing is bundled. On this branch (`versions.json`: htmx 1.9.10, Alpine 3.13.3, Tailwind 4.3.3) `task test:asset-budgets` reports:

| Asset | On disk | Gzipped total | Budget |
|---|---|---|---|
| `htmx.min.js` + `alpine.min.js` + `app.js` | 47,755 + 43,441 + 3,427 B | **32,895 B** | 51,200 B |
| `app.css` (Tailwind `--minify`) | 39,432 B | **7,768 B** | 30,720 B |

Roughly 18 KB of JS headroom is what a new library must fit inside, gzipped. `app.css` is gitignored and built by `task css:build` with the same `npx @tailwindcss/cli ... --minify` recipe the `Dockerfile` uses, so the gate measures what production ships. [ADR-007](../adr/ADR-007-Frontend-Stack-Selection.md) forbids heavy client frameworks outright; its TC-1 check fails on `react`, `vue`, `svelte`, `angular`, `jquery`, `next` or `nuxt` in `package.json`.

## Caching

[ADR-016](../adr/ADR-016-Caching-Strategy.md) is Accepted but mostly unbuilt, and its Implementation Checklist is entirely unchecked. What actually exists is in `cacheControlWrapper` in `internal/server/server.go`:

- `/static/*` CSS, JS, images and fonts get `Cache-Control: public, max-age=31536000`; anything else under `/static` gets `max-age=3600`.
- An error status downgrades the header to `no-store` (`errorUncachedWriter`), so a 404 for a missing asset is never cached for a year.
- Cache busting is `view.AssetURL`, which appends `?v=<build version>` (`SetAssetVersion` from `-X main.version`; `dev` keeps a process-start stamp).

What does not exist: `ETag`/`If-None-Match`, the `immutable` directive, `CDN-Cache-Control`/`Cache-Tag`, any in-memory cache (`internal/cache` was removed in #119 with zero call sites) and cache hit/miss metrics. Issue [#135](https://github.com/clownware/go-performance-starter/issues/135) holds the decision: implement the cheap subset and log a graduation entry, or supersede ADR-016 with the current reality. Do not cite ADR-016 as describing behaviour the app has.

## Aspirational tier

Total page weight, critical-path weight, LCP/FID/CLS/TTFB and the Lighthouse scores have no measurement. The grid's "Total page" row stays unmeasured by design. Issue [#136](https://github.com/clownware/go-performance-starter/issues/136) proposes a Lighthouse CI run against the deployed demo, feeding the total-page number into the grid; ADR-000 keeps it Monitored until a baseline exists, and it needs a small ADR first.

## Checking locally

```sh
task test:performance   # go test ./internal/performance/... + test:binary-size + test:asset-budgets
task build && ls -la dist/app
```

`task build` on macOS produced a 14,684,754-byte binary; a `GOOS=linux GOARCH=amd64` build with the Dockerfile's flags came to 15,536,290 bytes — about 5 MB under the gate. `task test:memory-profile` runs the package benchmarks under `-memprofile` and is manual only; nothing in `task ci` profiles memory. The docker image gate has no task — run `task docker:build` and `docker image inspect go-performance-starter --format='{{.Size}}'` to check it before pushing.
