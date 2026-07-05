# ADR-025: Deployment Target and Production Topology

**Date**: 2026-07-05

## Status

Accepted (supersedes the deployment considerations in ADR-001 §5)

## Context

The starter has a production-grade Dockerfile (multi-stage, non-root, < 30MB budget) but no decision about where and how that container runs. ADR-001 §5 gestures at Cloudflare Workers, which cannot host this application: the starter is a persistent Go HTTP server with a pgx connection pool, background goroutines (ADR-024's reaper), and in-memory rate-limiter state — none of which fit an isolate/WASM runtime or its 1MB limit. `.claude/stack.md` says "Cloudflare edge / container host", which is not a decision.

The 2026-07-05 deployment-readiness audit also found no documented answers for TLS termination, session topology, database backups, or migration/deploy ordering. Each is currently an implicit assumption; this ADR makes them explicit.

Two constraints shape the decision:

1. **This is a template.** The deployment story must be portable — the contract is the Docker image, not a vendor. Any host that runs a container with env vars and a health check must work.
2. **A public demo instance must actually ship** (ADR-024), so one concrete, worked example is needed alongside the portable contract.

## Decision

### 1. Deployment model: single stateless container behind the Cloudflare proxy

- The unit of deployment is the Docker image built by the repo `Dockerfile`. The app is **stateless** (no server-side session store, no local disk state), so instances scale horizontally and roll without draining state.
- **Worked example: Fly.io.** The demo instance runs as a single Fly machine; Cloudflare-proxied DNS sits in front for TLS, CDN caching of static assets (per ADR-016), and DDoS absorption. Railway/Render/a VPS remain fully supported by the same image — the template documents Fly as an example, not a requirement.
- **Cloudflare Workers is explicitly ruled out** as an application runtime. Cloudflare's role is edge proxy/CDN only.

### 2. TLS: terminated at the edge, plain HTTP inside

- TLS terminates at Cloudflare (edge) and the platform proxy (Fly), configured in Full (strict) mode between them. The Go process serves plain HTTP on `HTTP_PORT` inside the private network and does **not** embed certificates.
- The app emits `Strict-Transport-Security` only when `ENV=production` (implementation tracked in the hardening phase; ADR-014 §6 already specifies the header).

### 3. Sessions: stateless cookie-carried JWTs

- Sessions are the Supabase JWTs themselves, carried in httpOnly cookies (see ADR-014 §1, amended alongside this ADR). There is **no server-side session store** — no Redis, no session table. Logout is cookie clearing plus GoTrue sign-out. This is what makes the container stateless and multi-instance-safe with zero sticky-session configuration.

### 4. Database: Supabase managed Postgres; backups delegated

- Production Postgres is Supabase-managed. Backups, point-in-time recovery, and storage-layer encryption are **delegated to Supabase**; deployments must use a plan tier that includes scheduled backups, and this delegation is the documented backup strategy. The template does not ship its own backup tooling.
- Pool sizing is configured via environment (see the hardening phase), not code defaults.

### 5. Migrations: forward-only in production, applied before deploy

- Migrations run via the existing `db-migrate` workflow **before** the new image is released, never concurrently with it.
- Production migrations are **forward-only**: `.down.sql` files exist for development, but production rollback is "roll forward with a new migration". Breaking schema changes follow expand → migrate → contract so that migration N is always compatible with app versions N-1 and N.
- Application rollback is redeploying the previous image tag; images are tagged with version and git SHA by CI.

### 6. Scope boundary amendment (ADR-019)

ADR-019's "don't create deployment infra" boundary is amended to permit **one worked-example config file (`fly.toml`) at the repo root**, maintained as documentation-grade illustration of the contract above. All other deployment infrastructure (Terraform, k8s manifests, multi-environment pipelines) remains out of scope for the template.

## Consequences

- Every implicit production question (where does it run, who does TLS, where do sessions live, who backs up the DB, what order do migrate/deploy happen in) now has one documented answer that hardening work can implement against.
- The stateless-container contract keeps the template host-agnostic while still letting the ADR-024 demo ship on a real URL.
- Choosing edge TLS termination means the binary stays small (no cert management) but makes the Cloudflare/platform proxy a hard dependency for production traffic; running the raw container on the open internet is explicitly unsupported.
- Delegating backups to Supabase trades operational control for zero maintenance; self-hosted Postgres users must supply their own backup strategy (documented limitation).
- Forward-only migrations remove the temptation of risky down-migrations in production but require discipline (expand/contract) for breaking changes.
- ADR-001 §5 is superseded; stack.md's deployment section is updated to match.

## Alternatives Considered

- **Cloudflare Workers/Durable Objects** (ADR-001's sketch). Rejected: no persistent Go server, no pgxpool, 1MB limit; would require rewriting the entire starter around an isolate runtime.
- **VPS + Caddy/nginx (self-managed TLS)**. Rejected as the default: more operational surface (cert renewal, OS patching) than a starter should impose; remains possible since the image is portable.
- **Kubernetes**. Rejected: operational overkill for a template whose demo is one container; nothing in the image prevents it for downstream users.
- **Railway / Render as the worked example**. Equivalent capability; Fly chosen for its first-class Dockerfile deploys, cheap single-machine tier, and proxy that pairs cleanly with Cloudflare. Not an endorsement the template depends on.
- **Server-side session store (Redis/Postgres sessions)**. Rejected: adds a stateful dependency the JWT model doesn't need; would break the stateless-container property.

## References

- [ADR-000](ADR-000-Performance-Budgets-and-Quality-Attributes.md) (image/memory budgets), [ADR-001](ADR-001-Foundation.md) (superseded §5), [ADR-014](ADR-014-Security-Patterns-and-Threat-Model.md) (HSTS, sessions), [ADR-015](ADR-015-Configuration-Management-Strategy.md) (env config), [ADR-016](ADR-016-Caching-Strategy.md) (CDN layer), [ADR-019](ADR-019-Template-Scope-Boundary.md) (amended), [ADR-024](ADR-024-Demo-Application-Direction.md) (demo instance)
- 2026-07-05 deployment-readiness audit (session transcript)
