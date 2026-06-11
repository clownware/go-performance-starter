---
name: templ-component-scaffold
description: Scaffolds a new templ page, partial, or component in internal/view/ following the typed-props conventions from ADR-017. Use when asked to create a templ page/partial/component, add a view, build a UI element, or scaffold a screen.
allowed-tools: Bash, Read, Write, Glob, Grep
license: MIT
---

# templ Component Scaffold

Scaffold a new view in `internal/view/` that matches this repo's templ conventions (ADR-017).

## Before scaffolding

1. Read existing examples to match style:
   - Pages: `!ls internal/view/pages/*.templ`
   - Partials: `!ls internal/view/partials/*.templ`
   - Components: `!ls internal/view/components/*.templ`
2. Read `internal/view/props.go` (`BaseProps`, `NewBaseProps`) and `internal/view/render.go` (`Render`, `IsHTMXRequest`).

## Decide the kind

| Kind | Location | Wraps layout? | Use for |
|---|---|---|---|
| **page** | `internal/view/pages/` | Yes — `@layouts.Base(...)` | A full route rendered on direct navigation |
| **partial** | `internal/view/partials/` | No | An HTMX fragment swapped into a page |
| **component** | `internal/view/components/` | No | A reusable element (button, input, card) composed by pages/partials |

## Rules

- Define a props struct with **concrete types**. Pages embed `view.BaseProps`. Never `map[string]interface{}`.
- Pages call `@layouts.Base(props.BaseProps) { ... }`. Partials and components render standalone so HTMX fragments are correct.
- Accessibility: semantic HTML, labelled inputs, ARIA only where semantics fall short (ADR-009). Mobile-first responsive Tailwind.
- Prefer server-rendered HTMX; add Alpine.js only for light client-only interactivity.
- Styling via Tailwind utility classes / existing semantic color tokens — no inline styles.

## After scaffolding

1. Add the props struct to `internal/view/props.go` (or alongside the component) with concrete fields.
2. Run `task templ:generate` to produce the `*_templ.go`. Never hand-edit generated files.
3. If it's a page, wire the handler to call `view.Render(w, r, status, pages.Foo(props))`, branching on `view.IsHTMXRequest(r)` when a partial variant exists.
4. Run `task ci` before claiming done.
