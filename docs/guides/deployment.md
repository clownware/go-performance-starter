# Deployment

The unit of deployment is the Docker image built by the repo `Dockerfile`: one stateless container that any container host can run, fronted by Cloudflare-proxied DNS for TLS and CDN. Fly.io is the worked example, not a requirement; Cloudflare Workers is explicitly not an application runtime ([ADR-025](../adr/ADR-025-Deployment-Target.md)). Sessions are JWT cookies, backups are Supabase's, migrations are forward-only and run before the new image serves.

## The image

Three stages: `node:20-alpine` builds the Tailwind CSS, `golang:1.26-alpine` runs `templ generate` (the templ CLI version is derived from `go.mod`) and a stripped build, and `alpine:3.21` runs the binary as `appuser` on port 4000 with a `HEALTHCHECK` against `/healthz`. `task docker:build` builds it locally; CI's docker job and `release.yml` enforce the 30MB image budget ([ADR-000](../adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md)). The binary reports its version from `-X main.version`, stamped by `git describe` in `task build` and in the deploy workflows (#140/#141).

## Configuration on the host

Non-secret tuning is versioned in `fly.toml`'s `[env]` (`ENV=production`, `HTTP_PORT`, `PUBLIC_BASE_URL`, `DB_MAX_CONNS`, the proxy settings, guest mode). Secrets — `DATABASE_URL`, `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`, `METRICS_TOKEN` — go through `fly secrets set` and never into a file ([ADR-015](../adr/ADR-015-Configuration-Management-Strategy.md)). The full variable list is `.env.example`; `internal/config/config.go` validates it at boot and refuses to start on an invalid combination (for example guest mode without Supabase keys).

Behind a proxy you must set `TRUSTED_PROXY_CIDRS` (and `CLIENT_IP_HEADER` where the edge appends hops you cannot enumerate) or every visitor shares one rate-limit bucket. On Fly the direct peer is fly-proxy (`172.16.0.0/12`) and `Fly-Client-IP` is authoritative; behind Cloudflare use its published ranges and `CF-Connecting-IP`. The default is empty and fails closed ([ADR-027](../adr/ADR-027-Trusted-Proxy-Client-IP.md)).

## Cloudflare's role

DNS proxied in Full (strict) mode: TLS terminates at the edge and the app serves plain HTTP inside the private network, so HSTS is emitted only when `ENV=production`. The edge caches `/static/*` (the app sends a one-year `Cache-Control`; asset URLs carry the build version so a release busts the cache) and absorbs DDoS. Nothing runs at the edge.

## Workflows

| Workflow | Trigger | What it does |
|---|---|---|
| `ci.yml` | push / PR to master | `task ci` against a Postgres service, then the docker image budget |
| `deploy.yml` | merge to master | Continuous demo deploy: `flyctl deploy --remote-only` with the git-describe version. Gated on the `FLY_API_TOKEN` secret through a gate job, so a clone sees a skipped job, not a failure ([ADR-031](../adr/ADR-031-Public-Demo-Operations.md)) |
| `db-migrate.yml` | push to master | Validates the migrations on a fresh Postgres, then applies them to production; gated on `SUPABASE_DATABASE_URL` |
| `release.yml` | `v*` tag | Builds and pushes the ghcr.io image (30MB gate), stamps `versions.json`'s `template` field ([ADR-030](../adr/ADR-030-Versions-Manifest-Contract.md)), runs migrations, then deploys — migrations always precede the deploy step |
| `demo-reset.yml` | nightly 07:00 UTC | Resets guest demo content; double-gated on the secret and `DEMO_MODE=1` |
| `vuln-scan.yml` | schedule | govulncheck against the current module graph |

Rollback is redeploying the previous image tag; migrations are not rolled back in production ([ADR-025](../adr/ADR-025-Deployment-Target.md) §5).

## Health, shutdown, observability

- `/healthz` is liveness (used by the Dockerfile and `fly.toml` checks); `/health` is readiness and pings the database through a `handler.Pinger` seam ([ADR-013](../adr/ADR-013-Error-Handling-and-Observability.md)).
- `cmd/api/main.go` listens for `SIGINT`/`SIGTERM`, stops the reaper via context, and drains the server with a 5-second shutdown timeout. The `http.Server` timeouts are `ReadHeaderTimeout` 5s, `ReadTimeout` 15s, `WriteTimeout` 45s, `IdleTimeout` 120s; the 30-second request timeout middleware fires first by design.
- Logs are JSON `slog` lines with request IDs ([ADR-026](../adr/ADR-026-Logging-Standardization.md)); `/metrics` is Prometheus behind `METRICS_TOKEN`; every response carries `Server-Timing`. There is no OpenTelemetry and no alerting in the template — those are host-side concerns.

## Deploying somewhere else

Run the same image on Railway, Render, a VPS with Docker, or Kubernetes; delete `fly.toml` and `deploy.yml`, keep `db-migrate.yml`'s validate-then-apply shape, and put the equivalent of `fly secrets` in front of the container. The scope boundary ([ADR-019](../adr/ADR-019-Template-Scope-Boundary.md)) permits exactly one worked-example deploy config in the repo, so a second host's config is a fork-local decision.
