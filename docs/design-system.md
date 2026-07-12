# Design System

The starter ships a role-based token system ([ADR-029](adr/ADR-029-Role-Based-Design-Tokens.md), mirroring astro-performance-starter's ADR-047): components never name colors, they name **roles** — `bg-surface`, `text-muted-foreground`, `border-border` — and the roles resolve to the palette. Dark mode flips the role variables, not the components. The practical payoff for a template: **restyle the entire app by editing one `@theme` block.**

Everything below is enforced in CI, not aspirational: `internal/view/tokens_test.go` scans every `.templ` file and fails `task ci` on raw grays or `dark:` color variants.

## The roles

Defined once in [`web/static/css/input.css`](../web/static/css/input.css); the `.dark` block overrides them at runtime.

| Role utility | Use for | Light | Dark |
|---|---|---|---|
| `background` | Page background | nyanza | night |
| `surface` | Cards, panels, inputs | white | `#1a1a1a` |
| `surface-hover` | Hover states on surfaces | gray-100 | `#262626` |
| `foreground` | Primary text | night | nyanza |
| `muted-foreground` | Secondary text | gray-600 | gray-400 |
| `border` | Borders, dividers, skeletons | gray-200 | `#2e2e2e` |
| `primary` / `primary-strong` | Brand actions | teal / dark teal | (constant) |
| `accent` | Highlights, secondary chips | sage | sage |
| `link` | Hyperlinks | dark teal | soft teal |
| `danger` | Destructive text/tints | bittersweet | soft bittersweet |
| `success` | Positive feedback | green-700 | green-400 |
| `warning` | Cautionary feedback | amber-700 | amber-500 |

## The rules (CI-enforced)

1. **Dark mode flips tokens, not utilities.** Components never write `dark:` color variants; the `.dark` block in `input.css` is the only place color forks per mode.
2. **No raw grays in components.** `text-gray-500` is a violation; `text-muted-foreground` is the vocabulary.
3. **Solid buttons ride brand constants** (`.btn-primary` stays teal with white text in both modes); **text and tints ride roles** and flip for contrast.
4. **Status feedback is tint + role text** (`bg-danger/10 text-danger`), never a solid status background with white text.

## Restyling for your brand

The whole exercise is one file:

1. Replace the five base palette values in the `@theme` block of `input.css` (`--color-teal`, `--color-bittersweet`, `--color-night`, `--color-nyanza`, `--color-sage`) with your palette — keep the `-strong`/`-soft` variants roughly one step darker/lighter.
2. Re-point any role that referenced a swapped color, and mirror your choices in the `.dark` block (lighten `link`/`danger`/`success`/`warning` enough to keep WCAG AA on your dark background).
3. `task css:build` and eyeball both modes (the header toggle flips live).

No component changes are needed — that is the point of the system. `task ci` will tell you if anything in the view layer bypassed it.

## Reference implementation

The [`/patterns`](../internal/view/pages/patterns.templ) showcase and every page in `internal/view/` are written exclusively in role utilities — treat any of them as copy-paste-safe examples. The dark-mode pattern entry documents the toggle mechanism itself.
