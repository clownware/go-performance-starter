# UX Overhaul: Product Spec

> **Status: Active / planned.** The Tailwind v4 and templ migrations this spec depended on are now complete. This is the next planned body of work for the demo application; it has not yet been built.

Target state for the Alpine Go Performance Starter demo application. This spec defines the page inventory, routes, data models, and showcase content that will be built after the Tailwind v4 and templ migrations are complete.

Related decisions: [ADR-017](../adr/ADR-017-Templ-Adoption.md), [ADR-007](../adr/ADR-007-Frontend-Stack-Selection.md)

## Goals

1. **Show, don't tell.** Every HTMX and Alpine.js pattern the starter supports should be visible and interactive, not described in docs.
2. **Progressive disclosure.** A visitor should understand the value prop in 10 seconds, explore patterns in 5 minutes, and have a working authenticated scaffold to fork in 15 minutes.
3. **Real enough to fork.** The authenticated app uses a concrete domain (bookmarks) with 3+ fields, not abstract "items" with a name field.

## Navigation structure

```
Public (no auth required)
  /                     Landing page (value prop, stack overview, perf stats)
  /patterns             Interactive pattern showcase
  /patterns/{slug}      Individual pattern deep-dive (anchored sections or routes)
  /auth                 Login / signup (tabbed, single page)
  /terms                Terms of service
  /privacy              Privacy policy

Authenticated
  /dashboard            Dashboard with widget layout + skeleton loading
  /bookmarks            Bookmark list (CRUD, search, pagination)
  /bookmarks/new        Create bookmark form
  /bookmarks/{id}/edit  Edit bookmark (inline or full page)
  /profile              User profile
  /auth/logout          Logout confirmation
```

## Page specifications

### Landing page (`/`)

**Purpose:** Convince a developer to clone the repo in under 60 seconds.

**Sections (top to bottom):**

1. **Hero.** Project name, one-liner ("Production-ready Go + HTMX starter for indie builders"), two CTAs: "Explore patterns" → `/patterns`, "Clone repo" → GitHub.

2. **Stack overview.** Visual grid showing the five core technologies (Go, HTMX, Alpine.js, Tailwind CSS, Supabase) with one-sentence role descriptions. Not a logo wall — each card explains *why* this piece is in the stack.

3. **Performance stats.** Live-rendered stats from the actual performance budget definitions in `internal/performance/`. Display: P95 response time, binary size, Docker image size, memory usage, startup time. These should be served by a Go handler that reads the budget constants, not hardcoded in the template. This is itself a demo of server-rendered dynamic content.

4. **Feature highlights.** 3-4 cards summarizing what ships out of the box: auth, RLS, observability, CI/CD. Brief, linking to relevant ADRs or docs.

5. **Footer CTA.** "Ready to build?" with clone command and link to quick start docs.

**Data requirements:** Performance budget values from `internal/performance/`.
**HTMX patterns demonstrated:** None needed — this is a static marketing page. Server-rendered, fast, no JS required.

### Patterns showcase (`/patterns`)

**Purpose:** Interactive gallery demonstrating every HTMX + Alpine.js pattern the starter supports. This is the centerpiece of the demo — what makes this a showcase, not just a boilerplate.

**Layout:** Vertical sections, each containing a pattern. Each section has three elements:

- **Demo panel** — the working pattern, interactive. A user can click, type, scroll, toggle.
- **Template source** — the templ component code that produces the HTML. Syntax highlighted, read-only.
- **Handler source** — the Go handler code that serves the response. Syntax highlighted, read-only.

Source panels can be tabbed (Template | Handler) to save horizontal space.

**Patterns to include:**

| Pattern | Slug | HTMX features | Alpine features | Description |
|---------|------|---------------|-----------------|-------------|
| Partial swap | `partial-swap` | `hx-get`, `hx-target`, `hx-swap` | — | Click a button, replace a fragment. Simplest HTMX pattern. |
| Live search | `live-search` | `hx-get`, `hx-trigger="keyup changed delay:300ms"`, `hx-target` | — | Type in a search box, results filter server-side with debounce. |
| Click-to-edit | `click-to-edit` | `hx-get` (fetch edit form), `hx-put` (save), `hx-swap` | — | Click text to turn it into an edit form. Save returns the display view. |
| Inline validation | `inline-validation` | `hx-post`, `hx-target` (error container), `hx-swap` | `x-data`, `x-model`, computed getters | Server + client validation on a form. Server errors merge with Alpine's client-side checks. |
| Optimistic UI | `optimistic-ui` | `hx-post`, `hx-swap="outerHTML"` | `x-data`, `x-transition` | Favorite toggle with instant visual feedback. Includes artificial delay to show the pattern working. |
| Infinite scroll | `infinite-scroll` | `hx-get`, `hx-trigger="revealed"`, `hx-swap="beforeend"` | — | Scroll to bottom, next page loads automatically. Fallback "Load more" button. |
| Active search (typeahead) | `typeahead` | `hx-get`, `hx-trigger="keyup changed delay:200ms"`, `hx-indicator` | — | Typeahead search with loading indicator. |
| Toast notifications | `toasts` | `HX-Trigger` response header, `HX-Toast-Type` custom header | `x-data`, `$watch`, `x-transition`, `x-for` | Trigger success/error/warning toasts from server responses. Stacks, auto-dismisses. |
| Dark mode | `dark-mode` | — | `x-data`, `$watch`, `localStorage` | Toggle with system preference detection and persistence. |
| Skeleton loading | `skeleton-loading` | `hx-get`, `hx-trigger="load"` | `x-data`, `x-show`, `x-transition` | Show placeholder skeleton while HTMX fetches real content. |
| Tabs | `tabs` | `hx-get`, `hx-target`, `hx-push-url` | `x-data`, `x-show` | Tab navigation — Alpine for client-side tabs, HTMX for server-loaded tab content. Both shown. |
| Bulk operations | `bulk-operations` | `hx-post`, `hx-vals` | `x-data`, checkbox state management | Select multiple items, perform batch action. Shows Alpine managing selection state + HTMX submitting. |

**Data requirements:** Each pattern has its own handler(s) under a `/patterns/api/` route group. These are self-contained demos with stub data — they don't touch the real database. Example:

```
GET  /patterns                          → full showcase page
GET  /patterns/api/swap                 → partial swap fragment
GET  /patterns/api/search?q=...         → live search results
GET  /patterns/api/edit/{id}            → click-to-edit form
PUT  /patterns/api/edit/{id}            → save edit, return display
POST /patterns/api/favorite/{id}        → toggle favorite
GET  /patterns/api/scroll?page=N        → infinite scroll page
GET  /patterns/api/typeahead?q=...      → typeahead results
POST /patterns/api/toast                → trigger toast via headers
GET  /patterns/api/skeleton             → delayed content for skeleton demo
GET  /patterns/api/tab/{name}           → server-loaded tab content
POST /patterns/api/bulk                 → bulk operation result
```

**Source display approach:** Source code is embedded in the template as pre-formatted text blocks, not loaded dynamically. This keeps the showcase functional as a static page and avoids file-reading complexity. The displayed code should be the *actual* code from the handlers and templates, manually kept in sync (or generated from source in a future iteration).

### Auth page (`/auth`)

**Purpose:** Login and signup in a clean tabbed interface.

**Layout:** Single centered card, max-width ~md. Two tabs at the top: "Sign in" and "Create account". Alpine.js manages which form is visible via `x-show`. Active tab has a bottom border indicator.

**Tab: Sign in**
- Email input (autocomplete="email")
- Password input (autocomplete="current-password")
- "Forgot password?" link (can be placeholder)
- Submit button: "Sign in"
- Flash message area for errors (rendered via HTMX swap)

**Tab: Create account**
- Email input (autocomplete="email")
- Password input (autocomplete="new-password")
- Confirm password input (autocomplete="new-password")
- Submit button: "Create account"
- Flash message area for errors

**HTMX behavior:** Forms submit via `hx-post` to `/auth/login` and `/auth/signup`. On success, server returns `HX-Redirect` header. On failure, server returns the error partial swapped into the flash message area.

**Alpine behavior:** Tab switching is purely client-side (`x-show`). No server round-trip to change tabs.

### Dashboard (`/dashboard`)

**Purpose:** Demonstrate HTMX partial loading with skeleton states in a realistic layout. This is what a developer sees after signing up — it should look like a real app, not a placeholder.

**Layout:** 2-column grid on desktop, stacked on mobile.

**Widgets (all loaded via HTMX after page render):**

| Widget | Route | Content |
|--------|-------|---------|
| Recent bookmarks | `GET /dashboard/recent` | 5 most recent bookmarks with title + URL. Links to `/bookmarks`. |
| Stats summary | `GET /dashboard/stats` | Bookmark count, tag count, most-used tag. Simple number cards. |
| Quick add | — | Inline form to add a bookmark without leaving dashboard. `hx-post="/bookmarks"`, swaps success into recent list. |

Each widget container shows a skeleton loader (pulsing gray bars) until the HTMX response arrives. This demonstrates the `x-data="{ loading: true }"` + `@htmx:afterSwap` pattern already partially built in the current dashboard.

**Data requirements:** Needs bookmark data in the database. Seed data or a first-run seeder that creates 5-10 example bookmarks for new users.

### Bookmarks (`/bookmarks`)

**Purpose:** Replace the abstract "Items" CRUD with a concrete domain that demonstrates all CRUD patterns with enough fields to be realistic.

**Data model:**

```go
type Bookmark struct {
    ID          uuid.UUID
    UserID      uuid.UUID  // RLS-enforced ownership
    Title       string     // required, max 200 chars
    URL         string     // required, valid URL
    Description string     // optional, max 500 chars
    Tags        []string   // optional, for filtering/search
    IsFavorite  bool       // toggle, optimistic UI
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Routes:**

| Route | Method | Behavior |
|-------|--------|----------|
| `/bookmarks` | GET | List page with search + pagination |
| `/bookmarks/list` | GET | HTMX partial: filtered/paginated list fragment |
| `/bookmarks/new` | GET | Create form (can be modal or inline) |
| `/bookmarks` | POST | Create bookmark, redirect or swap |
| `/bookmarks/{id}` | GET | Single bookmark detail (optional) |
| `/bookmarks/{id}/edit` | GET | Edit form (click-to-edit inline or full page) |
| `/bookmarks/{id}` | PUT | Update bookmark |
| `/bookmarks/{id}` | DELETE | Delete bookmark (with confirmation) |
| `/bookmarks/{id}/toggle` | POST | Toggle favorite (optimistic UI) |
| `/bookmarks/search` | GET | HTMX partial: search results with debounce |

**List features:**
- Search bar with `hx-trigger="keyup changed delay:300ms"` filtering by title/URL/tags
- Infinite scroll or paginated (consistent with patterns showcase)
- Favorite toggle on each row
- Click row to expand/edit inline (click-to-edit pattern)
- Bulk select + delete (checkbox + HTMX batch delete)
- Empty state when no bookmarks exist

**Database:** New migration for `bookmarks` table. sqlc queries for all CRUD operations. RLS policies matching the existing `items` pattern.

### Profile (`/profile`)

**Purpose:** Demonstrate HTMX form submission with server-side validation and partial re-render.

**Fields:** Display name (editable), email (read-only, from Supabase auth), member since date.

**Behavior:** `hx-post="/profile"` submits the form. On success, the form partial re-renders with a success toast (via `HX-Trigger` header). On validation error, the form partial re-renders with field-level error messages.

### First-run experience

**Purpose:** Guide new signups from registration to a populated dashboard.

**Flow:**

```
Sign up (/auth, "Create account" tab)
  → Email verification (Supabase handles this)
  → First login
  → Welcome banner (step 1/3)
  → Profile setup (step 2/3, display name)
  → Seed bookmarks + Go to dashboard (step 3/3)
```

**Implementation:** The first-run state is tracked server-side (user metadata or a `first_run_completed` column). Each step is an HTMX partial swap within a container on the dashboard page. Steps advance via `hx-get="/onboarding/step/{n}"` replacing the onboarding container. On final step, seed 5 example bookmarks and swap in the real dashboard content.

**Routes:**

| Route | Method | Behavior |
|-------|--------|----------|
| `/onboarding/step/1` | GET | Welcome banner partial |
| `/onboarding/step/2` | GET | Profile setup form partial |
| `/onboarding/step/2` | POST | Save profile, return step 3 |
| `/onboarding/step/3` | POST | Seed bookmarks, mark complete, return dashboard |

## Data seeding

For the patterns showcase, all data is in-memory stubs — no database dependency. The patterns page works without Supabase credentials.

For the authenticated experience, a seed script creates example data:

```
task db:seed
```

Creates:
- 10 bookmarks across 5 tags for the demo user
- Covers various states: some favorited, some with descriptions, some without

## Implementation order

This work happens AFTER Tailwind v4 migration and templ migration are complete.

| Phase | Scope | Dependencies |
|-------|-------|-------------|
| 1. Routes + navigation | New route definitions in server.go, updated nav in base layout | Templ migration complete |
| 2. Landing page | Hero, stack overview, perf stats, feature cards | — |
| 3. Auth page redesign | Tabbed login/signup, HTMX form handlers | — |
| 4. Patterns showcase | All 12 pattern demos with source display | Most complex phase |
| 5. Bookmarks data model | Migration, sqlc queries, repository | — |
| 6. Bookmarks CRUD | List, create, edit, delete, search, pagination | Phase 5 |
| 7. Dashboard | Widget layout, skeleton loading, quick add | Phase 6 |
| 8. First-run experience | Onboarding flow, seed data | Phases 6-7 |
| 9. Cleanup | Remove old items code, update README, update ADRs | All phases |

## Out of scope

- Billing/payment integration (remains a stub interface per PRD)
- Multi-tenancy or organization accounts
- Full-text search (basic LIKE/ILIKE filtering is sufficient)
- Real-time features (WebSocket, SSE) — could be a future pattern addition
- Mobile-specific layout beyond responsive Tailwind breakpoints
