# ADR-027: Trusted-Proxy Client-IP Resolution

**Date**: 2026-07-07

## Status

Accepted (amends ADR-014 §4; refines ADR-025 §2). **Amended 2026-07-12:** X-Forwarded-For resolution changed from left-most to right-most-untrusted — see the dated note in the Decision section.

## Context

The rate limiter (ADR-014 §4) keys on the client IP taken from `r.RemoteAddr`, which the `RealIP` middleware overwrites from the `X-Forwarded-For` / `X-Real-IP` / `True-Client-IP` request headers. That middleware was `chimiddleware.RealIP`, which honors those headers **unconditionally** — it never checks who the direct TCP peer is.

The 2026-07-06 security audit flagged this: ADR-025 places the app behind Cloudflare, but on Fly.io the container is also reachable directly at its `.fly.dev` address unless network-restricted. An attacker hitting the app directly can send a fresh `X-Forwarded-For` value on every request, landing each in its own rate-limit bucket and defeating both the global (50 req/10s) and per-credential (5/min) limiters. The same forged value poisons the client IP recorded in structured logs (ADR-013).

Unconditionally trusting a client-controlled header is the root cause. A forwarded header is only meaningful when the connection actually came *through* a proxy we trust to set it.

## Decision

Replace the unconditional `RealIP` with a **trusted-proxy-gated** resolver:

- A new middleware `RealIP(trustedProxyCIDRs []string)` normalizes `r.RemoteAddr` to a bare IP (stripping the port so the rate-limiter key is per-IP, not per-connection), and honors the forwarded client-IP headers **only when the direct peer's address falls inside one of the configured trusted-proxy CIDRs.**
- Configuration is `TRUSTED_PROXY_CIDRS` (ADR-015): a comma-separated CIDR list, **default empty**. Empty means *trust no forwarded headers* — the direct peer IP is used verbatim. This is the safe default: a misconfigured deployment under-counts distinct clients (fails closed toward more limiting), never over-trusts a spoofed header.
- Deployments that terminate at a trusted proxy set `TRUSTED_PROXY_CIDRS` to that proxy's egress ranges (Cloudflare's published ranges, or the Fly private `fdaa::/16` / edge range when fronted by Fly). Documented in `.env.example` and the deployment guide.
- ~~We assume a **single** trusted hop and read the left-most forwarded entry. Multi-hop proxy chains are out of scope; revisit if a second trusted layer is introduced.~~
  **Amendment (2026-07-12, issue #72):** live verification on Fly falsified the left-most assumption. Edge proxies (Fly, Cloudflare) **append** the peer they saw to any client-supplied `X-Forwarded-For` rather than stripping it, so the left-most entry is attacker-controlled the moment a trusted CIDR is configured — a client sending `X-Forwarded-For: 6.6.6.6` would land every request in a fresh rate-limit bucket, recreating the exact spoofing vector this ADR exists to close. The resolver now walks XFF **right-to-left, skipping trusted-proxy addresses and invalid entries, and takes the first remaining IP** (the nearest hop that is not our own infrastructure). An XFF consisting only of trusted entries falls back to the direct peer (fails closed). This also handles multi-hop chains (e.g. Cloudflare → Fly with both ranges trusted). The trusted-peer gate is unchanged: with `TRUSTED_PROXY_CIDRS` empty, no forwarded header is ever honored.
  Verified values for Fly: the app's direct peer is fly-proxy at a private `172.16.0.0/12` address (observed `172.19.9.185`), so the demo sets `TRUSTED_PROXY_CIDRS=172.16.0.0/12` in `fly.toml`.
  **Second finding from the same live verification:** with the CIDRs trusted, XFF resolved to Fly's *edge* IP (`66.241.125.15`), not the visitor — Fly appends its own anycast edge hop, and those ranges are not documented as trustable CIDRs. XFF alone therefore cannot recover the client on Fly. This revisits the "rejected" per-platform-header alternative below with new evidence: an **operator-configured `CLIENT_IP_HEADER`** (default empty; `Fly-Client-IP` on Fly, `CF-Connecting-IP` behind Cloudflare) is now consulted first — still only when the direct peer is inside a trusted CIDR, so the header remains unspoofable from outside. The rejection stands *as a default*; as opt-in config gated on the trusted peer, it is the only correct resolution on edges with unenumerable hop ranges.

## Consequences

- The spoofing vector closes without coupling the app to one platform's proprietary header: the CIDR list is the single knob, and it degrades safely when unset.
- Operators **must** set `TRUSTED_PROXY_CIDRS` in production for per-client rate limiting to work as intended; with it empty behind a proxy, every request appears to originate from the proxy IP and shares one bucket. This trade-off (fail-closed) is deliberate — the alternative fails open to spoofing.
- The resolver is non-trivial (CIDR membership, header parsing), so it ships with a table-driven test per ADR-023: trusted peer honors the header, untrusted peer ignores it, empty config never trusts, malformed input falls back to the peer.

## Alternatives Considered

- **Key the limiter on `CF-Connecting-IP` / `Fly-Client-IP` directly.** Rejected as the default — hard-couples the app to one edge provider and still trusts a header that must be stripped-and-reset at the edge to be safe. A deployment may still point `TRUSTED_PROXY_CIDRS` at the edge and rely on it setting `X-Forwarded-For`, which every proxy does.
- **Document Cloudflare-as-sole-ingress and change no code.** Rejected — it leaves a spoofable default in the template that every downstream user inherits; the constraint is easy to violate (a forgotten public Fly route) and invisible when violated.
- **Keep `chimiddleware.RealIP`.** Rejected — the unconditional trust is the defect.

## References

- [ADR-014](ADR-014-Security-Patterns-and-Threat-Model.md) §4 (rate limiting), [ADR-015](ADR-015-Configuration-Management-Strategy.md) (env config), [ADR-025](ADR-025-Deployment-Target.md) §2 (edge TLS), [ADR-023](ADR-023-Testing-Philosophy.md) (test-first)
- 2026-07-06 security audit

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: The table-driven trusted-proxy tests (trusted honours headers, untrusted ignores, empty config never trusts, malformed falls back to peer) pass.
- **Checks:**
  - TC-1 → `go test` in `task ci` (status: **block**, pre-existing)
- **Not machine-checkable:** That `TRUSTED_PROXY_CIDRS` is configured correctly for a given deployment — ops configuration, outside the repo.
- **Graduation log:** _(empty)_
