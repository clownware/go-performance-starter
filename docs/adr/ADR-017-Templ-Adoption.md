# ADR-017: Templ Adoption for Type-Safe Templates

**Date**: 2026-04-04

## Status
Accepted (supersedes ADR-008)

## Context

ADR-008 chose Go's `html/template` for UI rendering. While functional, this approach had
significant drawbacks at scale:

- **No compile-time safety.** Template errors (missing fields, wrong types) only surface at
  runtime, often in production.
- **Stringly-typed data passing.** Handlers pass `map[string]interface{}` — no IDE
  autocompletion, no refactoring support, no type checking.
- **No component model.** Partials are included via string names (`{{template "name" .}}`),
  making dependency tracking and dead-code detection impossible.
- **Fragment rendering bug.** The old `RenderTemplate` always executed the base layout, even
  for HTMX partial responses, sending full-page HTML where only a fragment was needed.

## Decision

Replace `html/template` with [templ](https://templ.guide) for all server-rendered HTML.

### Architecture

```
internal/view/
├── layouts/base.templ       # Base layout (header, footer, dark mode, toasts)
├── pages/*.templ             # Full-page components (wrap in @layouts.Base)
├── partials/*.templ          # HTMX fragment components (no layout wrapper)
├── render.go                 # view.Render(w, r, status, component)
├── props.go                  # BaseProps + NewBaseProps helper
├── helpers.go                # HTMX request/response utilities
└── models.go                 # Shared view models (Item, UserProfile, etc.)
```

### Key patterns

- **Typed props:** Every page defines a props struct embedding `view.BaseProps`. Handlers
  construct props with concrete types — no `map[string]interface{}`.
- **Layout composition:** Pages call `@layouts.Base(props.BaseProps) { children }`. Partials
  render standalone (no layout wrapper) for correct HTMX fragment responses.
- **Single render path:** `view.Render(w, r, status, component)` handles all responses.
  Handlers choose between full-page and partial components based on
  `view.IsHTMXRequest(r)`.
- **Compile-time validation:** `templ generate` produces Go code. Type mismatches and
  missing fields are caught by `go build`, not by users in production.

### What was removed

- `web/templates/` directory (all `.html` files)
- `webutil.RenderTemplate` and `RenderTemplateWithErrors` functions
- `webutil.FormErrors` type
- `view.TemplateFuncs()` function map

### What was kept

- `webutil` HTMX helpers (`SetHXTrigger`, `SetHXRedirect`, etc.)
- `webutil` context helpers (`GetUserFromContext`, `GetUserRepoFromContext`)
- HTMX + Alpine.js frontend architecture (unchanged)

## Consequences

- All template errors are now compile-time errors.
- IDE support (autocompletion, go-to-definition, rename) works across templates.
- HTMX partial responses are correct — fragments render without the base layout.
- `templ generate` adds a build step, integrated into CI and Taskfile.
- Developers must learn templ syntax (minimal — it's Go with HTML).

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: No file imports `html/template`.
  - TC-2: No `map[string]interface{}` inside `internal/view` — props are typed structs.
  - TC-3: Generated templ output compiles.
- **Checks:**
  - TC-1 → `adr017-no-html-template` in `scripts/adrcheck` (status: **warn**)
  - TC-2 → `adr017-typed-view-props` in `scripts/adrcheck` (status: **warn**)
  - TC-3 → `templ generate` + `go build` in `task ci` (status: **block**, pre-existing)
- **Not machine-checkable:** Single-render-path discipline (`view.Render` everywhere) and partial/page composition judgment. templ-regeneration drift has no wired check — TODO in ADR-033.
- **Graduation log:** _(empty)_
