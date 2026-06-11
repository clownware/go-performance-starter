# Demo Design Brief

> Input for the design/mockup phase (Claude Design). Defines the screens to mock and the constraints that make mockups buildable against the existing templ + Tailwind stack. Decision context: [ADR-024](../adr/ADR-024-Demo-Application-Direction.md).

## Concept in one line

A self-documenting, server-rendered tour of this starter's own architecture — explainer → interactive pattern showcase → quiz that spawns saveable flashcards — usable instantly as an anonymous guest, upgradeable to a real account.

## Design principles

1. **The medium is the message.** Minimal JS, fast, server-rendered. The page should *feel* like what it teaches. No heavy client framework, no spinner-heavy SPA feel.
2. **Progressive disclosure.** Value prop in 10s, patterns in 5 min, a working forkable scaffold in 15.
3. **Guest-first.** Nothing the visitor wants to try is behind a signup wall. "Upgrade to keep progress" is a reward, never a gate.
4. **Accessibility is non-negotiable** (ADR-009): WCAG 2.1 AA, semantic HTML, visible focus, keyboard-operable, labelled controls, contrast-checked tokens.
5. **Dark mode is first-class** — every screen must be mocked in both light and dark (the app already switches via the `.dark` class).

## Design tokens (source of truth: `web/static/css/semantic-colors.css`)

Mock against these *roles*, not raw hex — they flip in dark mode.

| Role | Light | Dark | Tailwind/CSS var |
|---|---|---|---|
| Primary | `#468189` (teal) | teal | `--color-primary` |
| Danger | `#bf4342` (bittersweet) | bittersweet | `--color-danger` |
| Background | `#f0ffce` (nyanza) | `#0c0c0c` (night) | `--color-background` |
| Surface | `#ffffff` | `#1a1a1a` | `--color-surface` |
| Text | `#0c0c0c` (night) | `#f0ffce` (nyanza) | `--color-text` |
| Accent / Muted | `#d2cca1` (sage) | sage / teal | `--color-accent`, `--color-muted` |

Base palette: **teal `#468189`, bittersweet `#bf4342`, night `#0c0c0c`, nyanza `#f0ffce`, sage `#d2cca1`.** Type: Inter (body) + Oxanium (display/headings) — both already bundled. Styling is Tailwind v4 utilities; **no hardcoded colors, no manual `dark:` overrides** — use the role tokens so dark mode is automatic.

> Gap to close in parallel: these roles should be promoted into a formal token doc + Tailwind `@theme` mapping (mirroring the astro starter's role-based token ADR). Flagged for an engineering follow-up; mockups can proceed against the table above now.

## Components to reuse (don't redesign these)

Existing templ components in `internal/view/components/`: **button, input, form, card, alert, accessibility** (skip-links/sr-only helpers). Layout shell: `internal/view/layouts/base.templ` (nav, footer, dark-mode toggle, toast region). Mock new screens by *composing* these; only design genuinely new elements (diagram nodes, quiz card, flashcard, request-tracer).

## Screen inventory to mock

**Mock in both light + dark. Priority 1 = highest design value.**

| Priority | Screen | Route | Notes |
|---|---|---|---|
| **1** | Landing / explainer | `/` | Hero, stack overview, **live perf stats** (from budget constants below), architecture diagram entry, footer CTA. The marketing surface. |
| **1** | Patterns showcase | `/patterns` | Vertical sections; each = demo panel + tabbed (templ \| handler) source. The centerpiece. |
| **1** | Quiz flow | `/learn/quiz` | Question card, choices, submit → result + explanation swap, running score, "save as flashcard" on wrong answers. |
| **1** | Flashcards | `/learn/flashcards` | Grid/list of saved cards; flip interaction; mark-known; delete; empty state. |
| 2 | Guest banner + upgrade | (global) | Persistent "You're browsing as a guest — save your progress" affordance; upgrade modal (email/password) that links the account. |
| 2 | Dashboard | `/dashboard` | Progress widgets (quiz score history, cards-to-review), skeleton loaders. |
| 3 | Auth (sign in / create) | `/auth` | Tabbed card (Alpine `x-show`); only reached via upgrade or direct nav. |
| 3 | Profile | `/profile` | Reuse existing; minor polish. |

## Explainer content map (the teachable spine)

Order the narrative as a request's journey through the stack, each a diagram node + short prose + a source peek:

1. Request in → **Chi router + middleware stack** (security headers, request ID, rate limit, metrics, recover, timeout, auth).
2. **Handler** resolves typed input.
3. **Repository → sqlc → Postgres**, with **RLS** scoping rows to `auth.uid()`.
4. **templ** renders typed components; **HTMX** swaps fragments; **Alpine** handles light interactivity.
5. **Performance budgets** enforced in CI + observed via Prometheus.

Each node links to its ADR. The quiz draws questions from these five topics.

## Live performance stats (render from constants, never hardcode)

Source: `internal/performance/budgets.go`. Display on the landing page via a Go handler that reads these — itself a demo of server-rendered dynamic content.

- P50 < 50ms · P95 < 100ms · P99 < 200ms
- Binary < 20MB · Memory < 128MB (peak 256MB) · Startup < 500ms
- JS < 50KB · CSS < 30KB · Total page < 500KB

## Data model (for the quiz/flashcard surfaces)

New tables, RLS mirroring the existing `users_self_access` policy (`auth_id = auth.uid()::text`) so **anonymous guests are scoped identically**. UUID PKs, `updated_at` trigger, FORCE RLS + service-role bypass — same conventions as `migrations/000001`/`000002`.

```
quiz_questions     -- seeded reference content (permissive SELECT; not user-owned)
  id uuid pk, slug text unique, topic text, prompt text,
  choices jsonb, correct_index int, explanation text, created_at

quiz_attempts      -- user-owned (RLS via users.auth_id)
  id uuid pk, user_id uuid fk users, question_id uuid fk quiz_questions,
  selected_index int, is_correct bool, created_at

flashcards         -- user-owned (RLS via users.auth_id); created from wrong answers
  id uuid pk, user_id uuid fk users, question_id uuid fk quiz_questions null,
  front text, back text, is_known bool default false, created_at, updated_at
```

Progress (score, streak, cards-to-review) is computed from `quiz_attempts` + `flashcards` — no extra table needed for v1.

## HTMX/Alpine patterns this demo must exercise

Quiz answer submit (`hx-post` → result swap), save-flashcard (`hx-post`, optimistic), flashcard delete (`hx-delete`, `hx-swap="outerHTML swap:200ms"`), mark-known toggle, live perf stats, skeleton loaders on dashboard widgets, dark-mode toggle, toast on save/delete via `HX-Trigger`. (The full pattern catalogue lives in the retained `/patterns` section of [`ux-overhaul-spec.md`](ux-overhaul-spec.md).)

## What NOT to mock

CRUD edit forms beyond the flashcard/quiz cards, settings pages, and anything that's a conventional reuse of the existing component set — build those directly from components.

## Deliverables expected back from the design phase

1. Light + dark mockups for the Priority-1 screens.
2. Any *new* component specs (diagram node, quiz card, flashcard, request-tracer) expressed as composable pieces with the role tokens above.
3. Confirmation that contrast passes AA for the token pairings in both modes.
