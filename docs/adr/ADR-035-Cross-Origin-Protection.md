# ADR-035: Cross-Origin Protection (`http.CrossOriginProtection`) alongside the CSRF Token

**Date**: 2026-10-20

## Status

Proposed

<!-- If accepted, this ADR amends ADR-014 §3 (CSRF Protection); a dated note is then appended to ADR-014's Status. -->

## Context

ADR-014 §3 requires CSRF protection on every state-changing request. As its 2026-09-30 amendment records, the implementation is a hand-rolled **double-submit cookie** in `internal/middleware/csrf.go`:

- Every response to a visitor without a valid cookie mints a 32-byte random token from `crypto/rand` and sets it in an HttpOnly, `SameSite=Lax` cookie with a 12h lifetime.
- The token goes into the request context and from there into templ. It comes back on unsafe requests as the `X-CSRF-Token` header (`hx-headers` on `<body>`) or the `csrf_token` form field (`components.CSRFField`), and is compared in constant time.
- It has no per-session binding and no rotation.

Go 1.25 added [`http.CrossOriginProtection`](https://pkg.go.dev/net/http#CrossOriginProtection), a stateless CSRF defence in the standard library. It allows the safe methods GET, HEAD and OPTIONS. For any other method it reads the browser-set `Sec-Fetch-Site` header (Fetch Metadata, in every major browser since 2023) and rejects anything other than `same-origin` or `none`. Without that header it falls back to comparing the `Origin` host with `Host`. A request with neither header is treated as a same-origin or non-browser request and is allowed. Trusted origins (`AddTrustedOrigin`) and bypass patterns (`AddInsecureBypassPattern`) are explicit opt-ins, and `SetDenyHandler` customises the 403.

The toolchain is go1.27.2 and the `go` directive is 1.26, so the API is available today without any version change. Issue #178 asked whether it should complement or replace the token. That is a change to a security control, so per CLAUDE.md rule 1 it is decided here first.

### How the two compare

| | Double-submit token (today) | `CrossOriginProtection` |
|---|---|---|
| Mechanism | secret the attacker's page cannot read, echoed back | browser-asserted request provenance (`Sec-Fetch-Site`, `Origin`) |
| State | cookie per visitor, token in every rendered page | none |
| Template coupling | `hx-headers` on `<body>`, `CSRFField` in every form (19 call sites in `internal/view`) | none |
| Responses | `Set-Cookie` on every first-contact response, so shared caches (Cloudflare, ADR-025/ADR-016) cannot store those pages | no headers added |
| Same-site, cross-origin attacker (a sibling subdomain) | **weak**: without session binding, a sibling subdomain or a plain-HTTP MITM can plant its own `csrf_token` cookie for the parent domain and submit the matching value ("cookie tossing") | **rejected**: `Sec-Fetch-Site: same-site` is not `same-origin` |
| Browser with neither `Sec-Fetch-Site` nor `Origin` on a cross-origin POST (pre-2020 engines) | protected | **allowed** (fails open by design) |
| Non-browser clients (curl, server-to-server) | must fetch a cookie and echo it | pass, with no headers needed (CSRF is a browser-only attack) |
| State-changing GET | not protected (method allowlist) | not protected (method allowlist) — same rule: GET must stay safe |
| Cost per request (M4 Pro, go1.27.2, `b.Loop`, recorder overhead ~80 ns included) | returning visitor ~260–340 ns / 10 allocs; first visit ~740–1240 ns / 14 allocs (`BenchmarkCSRF`) | GET ~81 ns / 4 allocs; same-origin POST ~165 ns / 4 allocs; rejected ~490 ns / 11 allocs (scratch benchmark) |
| Maintenance | ~90 lines of security code we own, plus its tests | standard library, maintained and reviewed by the Go security team |

Neither control dominates the other. The token covers old browsers that send no Fetch Metadata. `CrossOriginProtection` covers same-site attackers and token-injection, and it removes per-page state.

## Decision

**Proposed: adopt `http.CrossOriginProtection` as defence-in-depth now, and treat removal of the token as a separate, later decision.**

1. **Phase 1 (this ADR, if accepted).** Wire `http.NewCrossOriginProtection()` into the global chain in `internal/server/server.go`, immediately **before** `mw.CSRF(isProd)`. Both checks then run, and an unsafe request must pass both. No trusted origins, no bypass patterns. A `SetDenyHandler` logs the rejection through `slog` (method, path, `Sec-Fetch-Site`, `Origin`, never cookies) and answers 403, so rejections can be observed on the demo. `internal/middleware/csrf.go` is unchanged.
2. **Phase 2 (not decided here).** After Phase 1 has run on the public demo (ADR-031) long enough to show that the token rejects nothing `CrossOriginProtection` lets through, a superseding ADR may propose removing the double-submit token. That would remove its cookie, the `hx-headers`/`CSRFField` plumbing and the `Set-Cookie` on first contact. The open questions for that ADR are the fail-open case for headerless browsers and whether ADR-016's edge caching needs pages to be free of cookies.

The maintainer may instead choose **replace now** or **reject** (see Alternatives). Accepting this ADR as written means Phase 1 only.

## Consequences

- **Easier:** same-site and cookie-tossing attacks are closed with no template change, and the server gets a standard, reviewed control instead of relying on code we maintain alone.
- **Harder:** two CSRF layers to explain. `/patterns` and the explainer must say which layer rejected a request (the deny handler's distinct body makes that visible). Tests of unsafe requests must set `Sec-Fetch-Site: same-origin` or leave both headers off. `httptest.NewRequest` sends neither, so existing tests keep passing.
- **Cost:** ~85 ns and 0 extra allocations on an allowed POST, and effectively nothing on GET (measured above). This is noise against the P95 < 100 ms budget (ADR-000).
- **Forbidden:** `AddInsecureBypassPattern` and `AddTrustedOrigin` must not be used without amending this ADR, and state-changing handlers on safe methods remain forbidden. Both controls depend on that rule.
- **Attack surface:** none added. It is a stricter filter in front of existing routes, with no new endpoint, form or upload.

## Alternatives Considered

- **Replace the token with `CrossOriginProtection` now.** This is the simplest end state, and the Go team presents the API as sufficient on its own for modern browsers. It is not chosen yet because it removes protection for headerless browsers before there is any evidence from the demo, and because deleting a working security control should be a deliberate, separately reviewed step (Phase 2).
- **Keep the token only (reject this ADR).** No change, but the cookie-tossing gap and the per-visitor `Set-Cookie` stay.
- **Bind the token to the session (HMAC of the session ID) instead.** This closes cookie tossing within the token design. It adds key management and couples CSRF to auth state, which guests (ADR-024) complicate. It costs more than a stdlib filter for the same gain.
- **Rely on `SameSite=Lax` alone.** Lax does not stop same-site subdomains, and it gives the CSRF cookie no protection against being set by an attacker on a sibling subdomain. Insufficient.

## References

- ADR-014 §3 (CSRF Protection) and its 2026-09-30 amendment. This ADR would amend §3.
- ADR-016 (Caching Strategy) and ADR-025 (Cloudflare edge): the `Set-Cookie` and cacheability interaction.
- ADR-024 (guest auth), ADR-031 (public demo operations): where Phase 1 would be observed.
- ADR-000: the latency budget the measured cost is compared against.
- Issue #178 (modern Go showcase), PR #181 (`BenchmarkCSRF`).
- Go 1.25 release notes and `go doc net/http.CrossOriginProtection`. Filippo Valsorda, "Cross-Site Request Forgery" (the design note behind the API).
- MDN: `Sec-Fetch-Site`, `Origin`.

## Enforcement
<!-- added 2026-10-20, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences** (these apply once the ADR is Accepted and Phase 1 lands; while it is Proposed nothing is enforced):
  - TC-1: the server's global chain rejects an unsafe request carrying `Sec-Fetch-Site: cross-site` with 403 **even when it carries a valid CSRF cookie and matching token** (proving the layer runs in front of `mw.CSRF`).
  - TC-2: the server's global chain still rejects an unsafe same-origin request without a token (the token layer is not removed by Phase 1).
  - TC-3: no call to `AddInsecureBypassPattern` or `AddTrustedOrigin` appears under `internal/` or `cmd/`.
- **Checks:**
  - TC-1, TC-2 → table-driven wiring test in `internal/server` (status: **warn** until graduated; it is a Go test, so it fails `task ci` once written).
  - TC-3 → an `adrcheck` grep check `adr035-no-cop-bypass`, to be added with Phase 1 (status: **warn**).
- **Not machine-checkable:** that no handler changes state on a safe method (both layers depend on it), and the Phase 2 judgement of whether the demo's observed traffic justifies removing the token.
- **Graduation log:** _(empty)_
