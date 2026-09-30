# HTMX and Alpine.js

Server-rendered HTMX first, Alpine only for client-only interactivity, and every page must work as progressive enhancement (CLAUDE.md rule 4; [ADR-007](../adr/ADR-007-Frontend-Stack-Selection.md), [ADR-012](../adr/ADR-012-Routing-and-UI-Patterns.md)). The JS budget is 50KB gzipped ([ADR-000](../adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md)); vendored HTMX and Alpine core plus `web/static/js/app.js` sit around 33KB today, so extensions and Alpine plugins are a budget decision, not a free add.

## The living catalogue: `/patterns`

Every pattern the starter supports is demonstrated live with its templ and handler source at [`/patterns`](https://go-performance-starter.fly.dev/patterns) ([ADR-024](../adr/ADR-024-Demo-Application-Direction.md) surface 2). The catalogue is `internal/handler/patterns_handlers.go` (twenty patterns in five sections: fetch & swap, search & lists, forms & actions, server-driven UX, Alpine islands) rendered by `internal/view/pages/patterns.templ`; the demo endpoints live under `/patterns/api/*` on stub data and need no database or Supabase credentials. Copy from there rather than from memory — the source shown on the page is the code that runs.

| Section | Patterns |
|---|---|
| Fetch & swap | `partial-swap`, `polling`, `oob-swap`, `view-transitions` |
| Search & lists | `live-search`, `typeahead`, `infinite-scroll`, `skeleton-loading` |
| Forms & actions | `inline-validation`, `click-to-edit`, `optimistic-ui`, `confirm-delete`, `loading-states`, `bulk-operations` |
| Server-driven UX | `toasts`, `tabs`, `rate-limit` |
| Alpine islands | `dark-mode`, `modal`, `global-store` |

Deliberately not included, with the reasons recorded in [#73](https://github.com/clownware/go-performance-starter/issues/73): `hx-boost` (the starter favours real navigations), the SSE/WebSocket extensions (JS weight vs the polling pattern), `hx-delete`/`hx-patch` (see below), Alpine plugins (core-only footprint).

## Conventions the real handlers follow

- **One handler, two renders.** A handler decides page vs fragment with `view.IsHTMXRequest(r)` and renders through `view.Render`; partials render without the layout ([view-layer.md](view-layer.md)). The quiz and flashcard handlers are the reference: the same `POST` serves a full page to a plain form and a fragment to HTMX.
- **Forms are `POST`, with `action` and `hx-post` both set.** No JavaScript means the browser submits the form; with HTMX the response swaps in place. State-changing routes do not use `hx-delete`/`hx-put` because a plain form cannot send them — a delete is `POST /learn/flashcards/{id}/delete`.
- **CSRF rides `hx-headers`.** `internal/view/layouts/base.templ` puts `hx-headers='{"X-CSRF-Token": …}'` on `<body>`, inherited by every HTMX request; plain forms include `@components.CSRFField()`. The middleware accepts either ([auth-and-security.md](auth-and-security.md)).
- **Server-driven events.** `view.SetHXTrigger` and `view.SetHXRedirect` set `HX-Trigger` / `HX-Redirect`; the base layout listens for the toast event and Alpine renders it. Redirect-after-POST on an HTMX request must use `HX-Redirect`, not a 303.
- **Error statuses still swap.** `web/static/js/app.js` tells HTMX to swap 400/401/409/422 responses (validation fragments) and a 429 whose body is HTML (the rate-limit demo), so the server can render the refusal instead of the client guessing.
- **Progressive enhancement is tested.** `internal/server/server_test.go` and `wiring_test.go` exercise routes both as plain requests and with `HX-Request: true`; the RLS isolation check on the flashcards page works with `?check=1` and no JS ([ADR-034](../adr/ADR-034-Live-Proof-Surfaces.md)).

## Alpine

Alpine core only, loaded after HTMX from `web/static/js/alpine.min.js`. Use it for state that never needs the server: the dark-mode toggle (`localStorage` + class on `<html>`), modals (`x-teleport`), tab visibility, a shared `Alpine.store`. The CSP allows `'unsafe-eval'` for Alpine's expression compiler and nothing else — no inline `<script>`, ever ([ADR-028](../adr/ADR-028-CSP-Alpine-Compatibility.md)). If a component needs data from the server, that is an HTMX fragment, not an Alpine `fetch`.

## Adding a pattern to the showcase

Add the entry to the catalogue slice in `patterns_handlers.go` (slug, section, HTMX/Alpine features, the templ and handler source strings), the demo endpoint under `/patterns/api`, and the section in `patterns.templ`. The source strings are hand-maintained; keep them byte-equal to the code they show (generating them at build time is the open idea in #73). `task test:asset-budgets` tells you if a new extension broke the JS budget.
