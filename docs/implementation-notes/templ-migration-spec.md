# Templ Migration Spec

> **Status: Completed (2026-04).** The migration this spec describes has shipped — see [ADR-017](../adr/ADR-017-Templ-Adoption.md). `web/templates/` and the `html/template` helpers no longer exist; all server-rendered HTML uses templ in `internal/view/`. This document is retained as an implementation note for historical context. The "Current state" section below describes the *pre-migration* codebase.

Reference document for the incremental migration from `html/template` to `templ`. This document is the source of truth for file locations, naming conventions, and the hybrid rendering approach during the transition period.

Related decision: [ADR-017](../adr/ADR-017-Templ-Adoption.md)

## Current state

All server-rendered HTML uses Go's `html/template` via the `webutil.RenderTemplate()` helper. Templates live in `web/templates/` and are loaded at startup. Data is passed as `map[string]interface{}`.

```
web/templates/
  layouts/base.html           ← full page wrapper (head, nav, footer, Alpine/HTMX setup)
  pages/                      ← full page content blocks
    home.html, dashboard.html, items.html, profile.html,
    terms.html, privacy.html, logout.html
  auth/login_signup.html      ← login + signup forms
  partials/                   ← HTMX fragment responses
    items_list.html, item.html, profile_form.html,
    empty_state.html, first_run_experience.html
  components/                 ← reusable UI elements
    button.html, input.html, form.html, card.html,
    alert.html, accessibility.html
```

Rendering is done via:
```go
webutil.RenderTemplate(w, r, http.StatusOK, "pages/home.html", data)
webutil.RenderTemplateWithErrors(w, r, http.StatusOK, "pages/home.html", data, errors)
```

HTMX partial rendering is gated on:
```go
if webutil.IsHTMXRequest(r) {
    // render partial
} else {
    // render full page
}
```

## Target state

All rendering uses templ components in `internal/view/`. The `web/templates/` directory is deleted. The `webutil` package retains HTMX helper functions (`IsHTMXRequest`, `FormErrors` type) but template rendering functions are removed.

```
internal/view/
  layouts/
    base.templ              ← templ Base(props BaseProps) with {children...}
  pages/
    home.templ              ← templ HomePage(props HomePageProps)
    dashboard.templ
    items.templ
    profile.templ
    auth.templ
    terms.templ
    privacy.templ
    logout.templ
  partials/
    items_list.templ        ← templ ItemsList(props ItemsListProps)
    item.templ
    profile_form.templ
    empty_state.templ
    first_run.templ
  components/
    button.templ            ← templ Button(props ButtonProps)
    input.templ
    form.templ
    card.templ
    alert.templ
  render.go                 ← Render() helper for HTTP handlers
  props.go                  ← shared prop types
```

## File mapping (old → new)

| Old path | New path | Notes |
|----------|----------|-------|
| `web/templates/layouts/base.html` | `internal/view/layouts/base.templ` | Uses `{children...}` for content slot |
| `web/templates/pages/home.html` | `internal/view/pages/home.templ` | |
| `web/templates/pages/dashboard.html` | `internal/view/pages/dashboard.templ` | |
| `web/templates/pages/items.html` | `internal/view/pages/items.templ` | |
| `web/templates/pages/profile.html` | `internal/view/pages/profile.templ` | |
| `web/templates/auth/login_signup.html` | `internal/view/pages/auth.templ` | |
| `web/templates/pages/terms.html` | `internal/view/pages/terms.templ` | |
| `web/templates/pages/privacy.html` | `internal/view/pages/privacy.templ` | |
| `web/templates/pages/logout.html` | `internal/view/pages/logout.templ` | |
| `web/templates/partials/items_list.html` | `internal/view/partials/items_list.templ` | Returned as HTMX fragment |
| `web/templates/partials/item.html` | `internal/view/partials/item.templ` | Returned as HTMX fragment |
| `web/templates/partials/profile_form.html` | `internal/view/partials/profile_form.templ` | Returned as HTMX fragment |
| `web/templates/partials/empty_state.html` | `internal/view/partials/empty_state.templ` | |
| `web/templates/partials/first_run_experience.html` | `internal/view/partials/first_run.templ` | |
| `web/templates/components/button.html` | `internal/view/components/button.templ` | |
| `web/templates/components/input.html` | `internal/view/components/input.templ` | |
| `web/templates/components/form.html` | `internal/view/components/form.templ` | Includes form-validation |
| `web/templates/components/card.html` | `internal/view/components/card.templ` | |
| `web/templates/components/alert.html` | `internal/view/components/alert.templ` | |

## Naming conventions

**Package:** `internal/view` (all subpackages share the `view` import path prefix)

**Component functions:** PascalCase matching the page/partial name. Examples:
- `layouts.Base(props layouts.BaseProps)` 
- `pages.HomePage(props pages.HomePageProps)`
- `partials.ItemsList(props partials.ItemsListProps)`
- `components.Button(props components.ButtonProps)`

**Props structs:** `<ComponentName>Props` in the same file as the component. Example:
```go
type HomePageProps struct {
    TestFieldError string
}

templ HomePage(props HomePageProps) {
    @layouts.Base(layouts.BaseProps{Title: "Home Page"}) {
        // page content using props.TestFieldError
    }
}
```

**Generated files:** `*_templ.go` — committed to repo, marked as generated in `.gitattributes`.

## Render helper

`internal/view/render.go` provides an HTTP handler helper:

```go
package view

import (
    "net/http"
    "github.com/a-h/templ"
)

func Render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    return component.Render(r.Context(), w)
}
```

Handlers call it as:
```go
func HomeHandler(w http.ResponseWriter, r *http.Request) {
    props := pages.HomePageProps{
        TestFieldError: "Server: This value is invalid!",
    }
    view.Render(w, r, http.StatusOK, pages.HomePage(props))
}
```

## HTMX dual-render pattern

Handlers that serve both full pages and HTMX partial responses:

```go
func ItemsList(w http.ResponseWriter, r *http.Request) {
    items := fetchItems(page)
    
    if webutil.IsHTMXRequest(r) {
        // HTMX request: return just the list fragment
        view.Render(w, r, http.StatusOK, partials.ItemsList(partials.ItemsListProps{
            Items:    items,
            NextPage: page + 1,
        }))
    } else {
        // Full page request: wrap in layout
        view.Render(w, r, http.StatusOK, pages.ItemsPage(pages.ItemsPageProps{
            Items:    items,
            NextPage: page + 1,
        }))
    }
}
```

## Alpine.js in templ

Alpine directives are standard HTML attributes in templ. Use raw attribute syntax for dynamic values:

```go
templ DarkModeToggle() {
    <button
        @click="dark = !dark"
        :aria-pressed="dark"
        :class="dark ? 'bg-primary text-white' : 'bg-surface text-primary'"
        class="rounded-full p-2 border-2 border-primary"
        title="Toggle dark mode"
    >
        <img x-show="!dark" x-cloak src="/static/img/emblem-black.svg" alt="Enable dark mode" class="h-5 w-5"/>
        <img x-show="dark" x-cloak src="/static/img/emblem-white.svg" alt="Enable light mode" class="h-5 w-5"/>
    </button>
}
```

**Important:** Templ's `@` prefix for event handlers is also Go's templ syntax for calling components. Alpine's `@click` works in templ because templ recognizes it as an HTML attribute, not a component call. No escaping needed.

## HTMX attributes in templ

Standard HTML attributes — no special handling:

```go
templ ItemToggleButton(itemID string, isFavorite bool) {
    <button
        hx-post={ "/items/" + itemID + "/toggle" }
        hx-target="this"
        hx-swap="outerHTML"
        class={ templ.KV("text-yellow-500", isFavorite), templ.KV("text-gray-400", !isFavorite), "hover:text-yellow-500" }
    >
        // star SVG
    </button>
}
```

## Build pipeline changes

### Taskfile.yml additions
```yaml
templ:generate:
  desc: Generate Go code from templ files
  cmds:
    - templ generate

dev:
  desc: Start dev server with hot reload
  deps: [templ:generate, css:build]
  cmds:
    - air
```

### .air.toml changes
```toml
[build]
  cmd = "templ generate && go build -o ./tmp/main ./cmd/api"
  
include_ext = ["go", "templ", "html", "css"]
```

### Dockerfile changes
```dockerfile
# Add templ generate step before go build
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate
RUN go build -o /app ./cmd/api
```

### CI changes
```yaml
- name: Install templ
  run: go install github.com/a-h/templ/cmd/templ@latest
- name: Generate templ
  run: templ generate
- name: Build
  run: go build ./...
```

## Migration phases

The migration proceeds in 5 phases. Each phase produces a working, buildable checkpoint.

| Phase | Scope | Verification |
|-------|-------|-------------|
| 1. Tooling + hello world | Install templ, convert home page only, keep old pipeline for all other routes | `templ generate && go build` passes, `/` renders via templ |
| 2. Component library | Convert all `components/*.html` to templ with typed props | `go build` passes, home page uses new components |
| 3. Layout + partials | Convert base layout, all partials, create render helper | `go build` passes, `/` has full layout via templ |
| 4. Page conversion | Convert all remaining pages, update all handlers | Every route renders correctly, both HTMX and full-page paths |
| 5. Cleanup | Delete `web/templates/`, remove old rendering code, update docs | `grep -r "html/template" internal/` returns nothing |

During phases 1–4, both rendering paths coexist. The old `webutil.RenderTemplate()` serves unconverted routes. New templ routes use `view.Render()`. This is ugly but intentional — it prevents big-bang failures.

## Post-migration doc updates

After phase 5, update these documents to reflect the new state:

- `README.md` — project structure, prerequisites, stack description
- `docs/implementation-guides/00-overview.md` — technology stack table
- `docs/implementation-guides/03-interface-design.md` — component patterns
- `docs/implementation-guides/06-htmx-alpine.md` — HTMX integration examples
- `docs/adr/ADR-007-Frontend-Stack-Selection.md` — add "Superseded by ADR-017" note to status
- `docs/adr/ADR-008-UI-Composition-and-Templates.md` — add "Superseded by ADR-017" note to status
