# ADR-036: Strict CSP via the Alpine CSP Build

**Date**: 2026-10-20

## Status

Accepted (supersedes ADR-028)

Proposed and accepted by the operator 2026-10-20, implemented in the same PR (#190). ADR-028's Status carries the dated supersession note, and ADR-014's Status lists the §6 amendment.

## Context

[ADR-028](ADR-028-CSP-Alpine-Compatibility.md) added `'unsafe-eval'` to `script-src`, because the standard Alpine build compiles every `x-data`, `x-show`, `x-on` and `x-init` expression with the `Function` constructor. It rejected the Alpine CSP build (`@alpinejs/csp`) on one premise: that the build "forbids inline expressions entirely", so every `x-data="{ open: false }"` would have to become an `Alpine.data()` registration, and the showcase would end up teaching a non-standard dialect of Alpine.

That premise no longer holds. Since Alpine 3.15.0 ([alpinejs/alpine#4671](https://github.com/alpinejs/alpine/pull/4671)), the CSP build ships its own expression parser. It evaluates most inline syntax without `eval`: object and array literals, comparisons, `!`/`&&`/`||`, ternaries, string concatenation, assignment to scope variables, `++`, and method calls on scope values and magics. It rejects arrow functions, template literals, object methods and getters, property assignment (`a.b = …`), multi-statement blocks, and browser globals (`window`, `document`, `localStorage`, `setTimeout`, `Date`, `Math`, `RegExp`, …).

The cost was measured rather than estimated (#177, [comment](https://github.com/clownware/go-performance-starter/issues/177#issuecomment-6302849873)). `@alpinejs/csp@3.17.4` (npm integrity verified) was served temporarily in place of the standard build, and `/patterns` and `/auth/page` were loaded. The parser rejected exactly three components:

| Component | Rejected because |
|---|---|
| `layouts/base.templ`: `darkModeData` + `darkModeInit` | globals (`localStorage`, `matchMedia`, `document`), multi-statement block, `$watch` with an arrow callback |
| `layouts/base.templ`: `toastData` + `toastInit` | object method; `Date`, `Math`, `setTimeout`, `addEventListener` |
| `pages/patterns.templ` confirm-delete | `pause(e) { … }` method in `x-data` |

Two more components fail by inspection. Both are on pages behind login, so the experiment did not load them:
- `components/form.templ` `formValidationXData`: getters and `new RegExp`.
- `partials/{password,profile}_form.templ`: `setTimeout(() => show = false, 3000)`.

Every other expression parsed unchanged. That covers about 20 `x-data` islands: the source tabs, `$store.demo.inc()`, the bulk-selection `x-model` with `selected.length`, the inline-validation `email.includes('@')`, the teleported modal, the mobile nav's `navOpen.toString()`, and the auth tabs' `$el.classList.remove(…)`. The idiom ADR-028 wanted to protect survives.

The CSP build is 23,840 B gzipped, against 19,927 B for the standard build, so +3.9 KB. With htmx 2.0.11 and `app.js`, the shipped JS goes from 39,496 B to about 43.4 KB, within the 51,200 B budget ([ADR-000](ADR-000-Performance-Budgets-and-Quality-Attributes.md); about 7.8 KB of headroom remains).

htmx 2 also has eval paths: `hx-trigger` event filters (`click[ctrlKey]`), and `js:`/`javascript:` values in `hx-vals` and `hx-headers`. Neither is used today. Under a strict CSP they would fail at runtime instead of in review.

## Decision

1. **Vendor the CSP build.** `web/static/js/alpine.min.js` is `@alpinejs/csp`'s `dist/cdn.min.js`, checked against the npm registry's integrity hash. The path, the `versions.json` key `alpine` and its value format stay as they are. `@alpinejs/csp` has released in lockstep with `alpinejs` since 3.13.3 and embeds the same `version:"…"` literal, so the [ADR-030](ADR-030-Versions-Manifest-Contract.md) contract does not change. Only the provenance does, and `docs/guides/performance.md` records it.
2. **`script-src 'self'`.** The policy drops `'unsafe-eval'` and still has no `'unsafe-inline'`. Every other directive in `internal/middleware/security.go` is unchanged. Inline `<script>` blocks remain forbidden (carried over from ADR-028 §2).
3. **Inline Alpine stays the default.** An expression the CSP parser accepts stays inline in the `.templ` file. A component moves to an `Alpine.data()` registration in `web/static/js/app.js` when it needs any of the following: object methods or getters, arrow functions or callbacks, multi-statement blocks, property assignment, or browser globals.
4. **Server values ride `data-*` attributes.** A registered component that needs server-supplied values (the form-field rules) reads them from `data-*` attributes in `init()`. Go helpers no longer build JavaScript source for registered components. The JS-context escaping in `formValidationXData`/`jsonString` is then no longer needed, because templ's attribute escaping ([ADR-017](ADR-017-Templ-Adoption.md)) covers it.
5. **The five components above move.** They become `darkMode`, `toasts`, `confirmDialog`, `formField` and `autoDismiss`. The `/patterns` source panes for the moved demos show the registration as well as the markup, so the showcase teaches both halves.
6. **htmx stays eval-free.** The base layout sets `<meta name="htmx-config" content='{"allowEval":false}'>`. `hx-trigger` filters and `js:`/`javascript:` attribute values are forbidden.

## Consequences

- The starter ships `script-src 'self'` with no `unsafe-*` keyword. That is a stronger security showcase, and it closes the defence-in-depth gap ADR-028 accepted.
- Injected markup can no longer evaluate arbitrary JavaScript through Alpine. It can still run the parser's restricted expressions against the scope it lands in, such as calling a registered method or flipping a flag. The blast radius shrinks a lot but is not zero. templ's contextual escaping (ADR-017) remains the primary XSS defence.
- Writing an expression the parser rejects is a runtime failure: an `Alpine Expression Error` in the console, and that behaviour does nothing. Go tests and lint cannot see it. The `adr036-csp-safe-alpine` heuristic and the browser walk on every Alpine change are what catch it.
- More JS: +3.9 KB gzipped for the CSP build, plus about 1.5 KB for the registrations in `app.js`. Measured after implementation: 44,930 B of 51,200 B, so the headroom for a future library drops from about 11.7 KB to about 6.3 KB.
- The parser is young code. 3.17.0 and 3.17.3 fixed parser bugs (Unicode escapes, reserved words as keys). Alpine bumps now have to re-run the browser walk under the strict CSP, not just the standard build.
- Interpreted expressions may cost more per evaluation than compiled `Function`s. This has not been measured. The demo's expressions are tiny, and the implementation PR records a before/after if the difference is observable.
- Five components become two-part (markup plus registration), and readers of `/patterns` see `Alpine.data()` earlier than before.

## Alternatives Considered

- **Keep ADR-028 (`'unsafe-eval'`).** No work and no size cost. But it keeps the weakest directive in the policy, for a reason that no longer applies.
- **Move every component to `Alpine.data()`.** That is ADR-028's own reading of the CSP build. The parser makes it unnecessary, and it would rewrite the inline idiom the showcase exists to teach.
- **Nonce- or hash-based `script-src`.** Nonces and hashes authorise script blocks, not `Function`-constructor evaluation, so they do not remove the need for `'unsafe-eval'` (unchanged from ADR-028).
- **Replace Alpine with vanilla JS in `app.js`.** Strict CSP at zero library cost, but it contradicts [ADR-007](ADR-007-Frontend-Stack-Selection.md) and [ADR-012](ADR-012-Routing-and-UI-Patterns.md) and the HTMX + Alpine positioning.

## References

- [ADR-028](ADR-028-CSP-Alpine-Compatibility.md): the decision this supersedes on acceptance
- [ADR-014](ADR-014-Security-Patterns-and-Threat-Model.md) §6: the CSP this amends
- [ADR-000](ADR-000-Performance-Budgets-and-Quality-Attributes.md): JS budget; [ADR-030](ADR-030-Versions-Manifest-Contract.md): `versions.json` contract; [ADR-017](ADR-017-Templ-Adoption.md): escaping
- [ADR-007](ADR-007-Frontend-Stack-Selection.md), [ADR-012](ADR-012-Routing-and-UI-Patterns.md): Alpine's role
- #177 (evaluation and measurements), #186 (this decision), #184 (Alpine 3.17.4)
- Alpine docs, "CSP (Content-Security Policy) Build"; [alpinejs/alpine#4671](https://github.com/alpinejs/alpine/pull/4671)

## Enforcement
<!-- added 2026-10-20, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: The CSP header's `script-src` is exactly `'self'`: no `'unsafe-eval'`, no `'unsafe-inline'`.
  - TC-2: No rendered page ships an inline `<script>` block (every script tag has `src`).
  - TC-3: The vendored `web/static/js/alpine.min.js` is the CSP build. It contains the parser's `CSP Parser Error` marker, which the standard build lacks.
  - TC-4: No quoted Alpine attribute value (`x-*`, `@…`, `:…`) in `internal/view/**/*.templ` uses a construct the CSP parser rejects: `=>`, `function`, a template literal, `new` with a constructor, or `window`/`document`/`localStorage`/`sessionStorage`/`setTimeout`/`setInterval`/`console`.
  - TC-5: The base layout sets `htmx.config.allowEval` to `false`, and no `.templ` file uses an `hx-trigger` filter or a `js:`/`javascript:` attribute value.
- **Checks:**
  - TC-1 → `internal/middleware/security_test.go`, which replaces ADR-028's `'unsafe-eval'` assertion, via `task ci` (status: **block**; it replaces an existing blocking test)
  - TC-2 → `internal/server/server_test.go` via `task ci` (status: **block**, carried over from ADR-028)
  - TC-3 → `scripts/adrcheck` check `adr036-csp-build-vendored` (status: **warn**, added 2026-10-20)
  - TC-4 → `scripts/adrcheck` check `adr036-csp-safe-alpine` (status: **warn**, added 2026-10-20)
  - TC-5 → `TestServer_StrictCSPLayout` in `internal/server/server_test.go` for the meta tag (status: **block**, a `task ci` test), and `adr036-csp-safe-alpine` for the `hx-trigger`/`hx-vals`/`hx-headers` attributes (status: **warn**, added 2026-10-20)
- **Not machine-checkable:** TC-4 is a heuristic. It reads quoted attributes only, so a value a Go helper builds (`x-data={ authTabData(…) }`) is out of its reach. The parser is the real authority, and CI has no browser to run it in. A rejected expression the heuristic misses shows up only as a console `Alpine Expression Error`, so every Alpine change, and every Alpine bump, needs a browser walk under the strict CSP. Whether a component is clearer inline or registered, beyond what the parser forces, is a review judgement.
- **Graduation log:** _(empty)_
