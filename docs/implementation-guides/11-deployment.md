# Phase 10 — Deployment & Monitoring with Fly.io, Cloudflare, and Supabase

Set up production environment with appropriate monitoring. The unit of deployment is the Docker image built by the repo `Dockerfile` — a single stateless container that any container host can run (ADR-025). Fly.io is the worked example; Cloudflare's role is edge proxy/CDN only, and Cloudflare Workers is explicitly ruled out as an application runtime.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 10.01 | Build the Docker image (`task docker:build`) | The image is the portable deployment contract (ADR-025) |
| 10.02 | Configure the Fly.io app via `fly.toml` | Worked-example host; any container host works |
| 10.03 | Configure Supabase production | Database and Auth settings |
| 10.04 | Implement essential telemetry | Machine-readable logs |
| 10.05 | Create health check endpoints | Status verification (/healthz, /health) |
| 10.06 | Set up basic monitoring | Performance tracking |
| 10.07 | Implement graceful shutdown | Clean termination |
| 10.08 | Configure database backups | Data protection |
| 10.09 | Set up CI/CD pipeline | Deployment automation (`deploy.yml`) |

## Core Principles

- Deploy the app as a single stateless container; the Docker image is the contract (ADR-025)
- Front the container with Cloudflare-proxied DNS for TLS, CDN caching of static assets, and DDoS absorption — not Workers
- Configure Supabase for production with appropriate settings
- Set `TRUSTED_PROXY_CIDRS` so client-IP resolution sees real visitor IPs (ADR-027)
- Use structured logging with appropriate context
- Create comprehensive health check endpoints
- Implement graceful shutdown procedures
- Configure automated database backups
- Set up CI/CD pipeline for automated deployment

## Fly.io Deployment (Worked Example)

### fly.toml Configuration

- One worked-example config at the repo root (ADR-025 §6): builds from `Dockerfile`, sets `ENV=production` and `HTTP_PORT`, and registers an HTTP health check on `/healthz`
- Non-secret tuning (pool size, proxy CIDRs, feature toggles) lives in `[env]`, versioned
- Secrets (`DATABASE_URL`, `SUPABASE_*`, `METRICS_TOKEN`) go through `fly secrets set` — never in the file (ADR-015)
- Fly is an example, not a requirement: Railway/Render/a VPS run the same image

### CI/CD Pipeline (deploy.yml)

- `.github/workflows/deploy.yml` deploys every merge to the default branch via `flyctl deploy --remote-only`
- The whole workflow is gated on the `FLY_API_TOKEN` secret: a clone without the secret sees a skipped job, not a failure — adding your own token enables your own deploys
- Migrations run **before** the deploy step (ADR-025 §5): forward-only in production; rollback is redeploying the previous image tag

### Cloudflare's Role: Edge Proxy Only

- Cloudflare-proxied DNS sits in front of the container in Full (strict) mode; TLS terminates at the edge and the app serves plain HTTP inside the private network (ADR-025 §2)
- The edge provides CDN caching of static assets (ADR-016) and DDoS absorption
- Cloudflare Workers is explicitly ruled out as an application runtime — the persistent Go server, pgx pool, and background goroutines do not fit an isolate/WASM runtime

### Trusted Proxy Configuration (ADR-027)

Behind an edge proxy, forwarded client-IP headers are only honored when the direct peer is inside `TRUSTED_PROXY_CIDRS`:

- Default is empty: no forwarded headers are trusted and the direct peer IP is used — this fails closed against IP spoofing
- In production, set it to your proxy's ranges: on Fly the direct peer is fly-proxy (`172.16.0.0/12`); behind Cloudflare use its published ranges
- When the edge appends hop IPs you can't enumerate as CIDRs, also set `CLIENT_IP_HEADER` (`Fly-Client-IP` on Fly, `CF-Connecting-IP` behind Cloudflare)
- Leaving it unset behind a proxy makes every request share one rate-limit bucket — safe, but not what you want

## Supabase Production Configuration

- Configure production environment variables
- Verify RLS policies against production roles
- Enable database backups
- Configure correct CORS settings
- Set up appropriate JWT expiration times

## Health Check Endpoints

The starter registers two endpoints (ADR-013):

- `/healthz`: Simple liveness check — used by the Dockerfile `HEALTHCHECK` and the `fly.toml` HTTP service check
- `/health`: Detailed readiness check that the service is ready to accept traffic (DB connections, etc.)

## Feature Flag Integration

If you plan to use feature flags in Phase 12:
- Define flag naming conventions in CI/CD configuration
- Consider implementing a simple feature flag mechanism in your deployment
- Prepare for blue/green deployment strategy if needed
- Enable Dependabot auto-PRs; block merge until CI passes

## Essential Telemetry

- Implement structured logging in JSON format
- Include request ID, timestamp, log level, source
- Create health check endpoints for monitoring
- Set up basic performance monitoring
- Configure alerts for critical issues
- Expose an OTLP endpoint now if you intend to adopt OpenTelemetry in Phase 12

## Graceful Shutdown for HTTP Servers

Implement proper shutdown so the platform's stop signal drains cleanly:

```go
// Create server with appropriate timeout values
server := &http.Server{
    Addr:         ":8080",
    Handler:      router,
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
}

// Start server in goroutine
go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Server error: %v", err)
    }
}()

// Wait for interrupt signal
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
<-stop

// Create shutdown context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Attempt graceful shutdown
if err := server.Shutdown(ctx); err != nil {
    log.Fatalf("Server forced to shutdown: %v", err)
}
```

## Common Pitfalls

- **Missing environment variables**: Document all required variables
- **Insufficient logging**: Include context in logs
- **Poor health checks**: Keep liveness (/healthz) and readiness (/health) separate
- **No graceful shutdown**: Handle termination signals
- **Missing backups**: Configure Supabase backups
- **Unset TRUSTED_PROXY_CIDRS**: Behind a proxy, every request shares one rate-limit bucket (ADR-027)

## Implementation Strategy

- Build and verify the Docker image locally with `task docker:build`
- Configure `fly.toml` and deploy the worked example (or run the same image on another host)
- Put Cloudflare-proxied DNS in front for TLS and CDN caching
- Set up Supabase production environment
- Implement structured logging and monitoring
- Configure health checks and alerting
- Establish CI/CD pipeline for reliable deployments
- Verify backup and restore procedures

## Exit Criteria

- Container deployment configured (Fly.io worked example or equivalent host)
- Cloudflare edge proxy in front with Full (strict) TLS
- `TRUSTED_PROXY_CIDRS` (and `CLIENT_IP_HEADER` where needed) set for the production topology
- Supabase production environment set up
- Structured logging implemented
- Health check endpoints operational
- Basic monitoring configured
- Graceful shutdown implemented
- Database backups configured
- Deployment automation working
