# ADR-029: Role-Based Design Tokens

**Date**: 2026-07-12

## Status

Accepted

## Context

The view layer had two competing color systems: a static `@theme` block in
`input.css` with `-dark`-suffixed twins (`--color-background-dark`) driving
manual `dark:` variants in every component, and a runtime-flipping
`semantic-colors.css` loaded twice (once via `@import`, once as a separate
`<link>`). On top of both, components hardcoded Tailwind grays
(`text-gray-500 dark:text-gray-400`, ~120 usages), the `primary` scale was
fake (500 and 600 were the same hex), and the demo design brief explicitly
flagged promoting the palette roles into a formal token layer mirroring the
astro starter's ADR-047.

## Decision

Adopt role-based semantic tokens as the single styling vocabulary, mirroring
astro-performance-starter ADR-047 adapted to this palette (teal, bittersweet,
night, nyanza, sage):

| Role | Light | Dark |
|---|---|---|
| `background` | nyanza | night |
| `surface` | white | `#1a1a1a` |
| `surface-hover` | gray-100 | `#262626` |
| `foreground` | night | nyanza |
| `muted-foreground` | gray-600 | gray-400 |
| `border` | gray-200 | `#2e2e2e` |
| `primary` / `primary-strong` | teal / dark teal | (constant) |
| `accent` | sage | sage |
| `link` | dark teal | soft teal |
| `danger` | bittersweet | soft bittersweet |
| `success` | green-700 | green-400 |
| `warning` | amber-700 | amber-500 |

Rules with constitutional force (ADR-018 layering):

1. **Dark mode flips tokens, not utilities.** The `.dark` block in
   `input.css` overrides the role variables; components never write `dark:`
   color variants. (`dark:` remains legal for non-color concerns, e.g.
   `x-cloak` interplay, but no color role may fork per mode in a component.)
2. **No raw grays in components.** `text-muted-foreground`, `border-border`,
   `bg-surface-hover`, `bg-border` (skeletons) replace the gray-N00 family.
3. **Solid buttons ride brand constants** (`.btn-primary`/`.btn-danger` keep
   teal/bittersweet with white text in both modes); **text and tints ride
   roles** (`text-danger`, `bg-success/10`) and flip for contrast.
4. **Status feedback is tint + role text** (`bg-danger/10 text-danger`), not
   solid status backgrounds with white text — the astro v2.1 lesson.
5. `semantic-colors.css` is deleted; `input.css` is the single token source.

A `tokens_test.go` scan enforces rules 1–2 over `internal/view/**/*.templ`
so drift fails `task ci` (ADR-021).

## Consequences

- One source of truth; the double-loaded stylesheet and the fake primary
  scale are gone; components read as intent (`text-muted-foreground`), not
  as palette trivia.
- One-time sweep of ~15 templ files (~230 class edits) — mechanical, guarded
  by the visual pass and the existing markup tests.
- The old `text` role is renamed `foreground` (astro parity); `-dark` token
  twins are deleted.
- Contrast obligations move into the token layer: dark-mode `link`,
  `danger`, `success` values are lightened variants chosen for the night
  background.

## Alternatives Considered

- **Keep both systems, fix values only.** Rejected — the duplication is the
  bug; every new component would keep choosing between vocabularies.
- **Content-hashed design-token pipeline (Style Dictionary) like astro.**
  Rejected for now — this starter has no token build step and doesn't need
  one at this scale; the astro mapping is captured directly in CSS.

## References

- astro-performance-starter ADR-047 (role-based naming, v2.1 status-color
  amendment) — the sibling-template precedent this adapts.
- [ADR-007](ADR-007-Frontend-Stack-Selection.md), [ADR-017](ADR-017-Templ-Adoption.md),
  [ADR-024](ADR-024-Demo-Application-Direction.md) (design brief flagged the gap).
