# ADR-028: CSP Compatibility with Alpine.js

**Date**: 2026-07-12

## Status

Accepted

## Context

The Content-Security-Policy shipped by `internal/middleware/security.go` locks
`script-src` to `'self'`. The comment reasoned that HTMX and Alpine.js are
self-hosted script *files*, so nothing more was needed. That reasoning covered
script **loading** but not script **evaluation**: Alpine 3's expression engine
compiles every `x-data`, `x-show`, `x-on`, and `x-init` attribute with the
`Function` constructor, which CSP treats as `eval`. Without `'unsafe-eval'`,
every Alpine expression throws
`Alpine Expression Error: ... 'unsafe-eval' is not an allowed source`.

The failure was invisible while the demo was hardcoded stubs. The 2026-07-12
design review of the ADR-024 surfaces found the entire Alpine layer dead in
every environment: the dark-mode toggle (and its persistence), the user menu,
toast notifications, the showcase source tabs, inline validation, bulk
selection, and the flashcard flip. Two inline `<script>` blocks in the base
layout (the loading-indicator wiring and an `alpine:init` placeholder) were
also silently blocked, because `script-src` without `'unsafe-inline'` or a
nonce forbids inline script bodies.

This is a direct conflict between two Accepted ADRs: ADR-014 (strict CSP) and
ADR-007/ADR-012 (Alpine.js is the sanctioned client-interactivity layer).

## Decision

1. **Add `'unsafe-eval'` to `script-src`** — the policy becomes
   `script-src 'self' 'unsafe-eval'`. This is Alpine's documented supported
   posture for the standard build.
2. **Keep inline scripts forbidden** — `script-src` does not gain
   `'unsafe-inline'`. The base layout's inline `<script>` blocks move into the
   self-hosted `app.js`; templates may only ship behavior as Alpine attributes
   or external files. A server test pins this (no `<script>` without `src`).
3. All other directives are unchanged (`default-src 'self'`,
   `form-action 'self'`, `frame-ancestors 'none'`, …).

## Consequences

- Alpine works; the ADR-007/012 frontend bet is functional again.
- `'unsafe-eval'` weakens CSP as a defense-in-depth layer: an attacker who can
  already inject markup could evaluate expressions. The primary XSS defense in
  this stack is templ's contextual auto-escaping (ADR-017) plus input
  validation (ADR-014 §2); CSP remains effective against foreign-origin
  script injection, inline handlers, and clickjacking.
- The no-inline-scripts rule becomes testable and tested — future templates
  cannot quietly reintroduce inline `<script>`.

## Alternatives Considered

- **Alpine CSP build (`@alpinejs/csp`).** Eval-free, but it forbids inline
  expressions entirely — every `x-data="{ open: false }"` in the codebase
  (and in the pattern showcase's teaching material) would need to become a
  registered `Alpine.data()` component. That rewrites the exact idiom the
  starter exists to demonstrate, and the showcase would then teach a
  non-standard Alpine dialect. Rejected.
- **Nonce-based script-src.** Nonces authorize inline script *blocks*, not
  `Function`-constructor evaluation — it does not fix Alpine, only the two
  inline scripts, which move to `app.js` anyway. Rejected as insufficient.
- **Drop Alpine for vanilla JS.** Contradicts ADR-007/ADR-012 and the
  product's HTMX+Alpine positioning. Rejected.

## References

- [ADR-007](ADR-007-Frontend-Stack-Selection.md), [ADR-012](ADR-012-Routing-and-UI-Patterns.md) — Alpine's role
- [ADR-014](ADR-014-Security-Patterns-and-Threat-Model.md) — the CSP this amends (amendment note added there)
- [ADR-017](ADR-017-Templ-Adoption.md) — templ escaping as the primary XSS defense
- Alpine.js docs, "Content Security Policy" — the standard build requires `'unsafe-eval'`
