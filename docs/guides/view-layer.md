# View Layer

Server-rendered HTML in this starter is [templ](https://templ.guide) ([ADR-017](../adr/ADR-017-Templ-Adoption.md), which superseded the `html/template` design in [ADR-008](../adr/ADR-008-UI-Composition-and-Templates.md)). Styling is Tailwind v4 over a role-based token layer ([ADR-029](../adr/ADR-029-Role-Based-Design-Tokens.md)), and every page must meet the accessibility and responsive bar in [ADR-009](../adr/ADR-009-Accessibility-and-Responsive-Design.md). This guide is for a developer adding a page, an HTMX fragment, or a reusable component. The HTMX/Alpine interaction patterns themselves are covered in [htmx-and-alpine.md](htmx-and-alpine.md); the palette and its rules are in [`docs/design-system.md`](../design-system.md).

## Layout of `internal/view`

| Path | What lives there |
|---|---|
| [`layouts/base.templ`](../../internal/view/layouts/base.templ) | `Base(props view.BaseProps)`: `<head>` (meta, canonical, one stylesheet, three deferred scripts), skip link, header with nav and dark-mode toggle, `<main id="main-content">`, footer, toast region |
| [`pages/`](../../internal/view/pages/) | Full routes that wrap `@layouts.Base` — `HomePage`, `PatternsPage`, `QuizPage`, `FlashcardsPage`, `DashboardPage`, `AuthPage`, `RecoverPage`/`ResetPage`, `ProfilePage`, `UpgradePage`, `LogoutPage`, `TermsPage`, `PrivacyPage` |
| [`partials/`](../../internal/view/partials/) | HTMX fragments with no layout wrapper — quiz, flashcard, pattern-showcase, dashboard-widget, auth-message, profile/password/upgrade forms, empty and first-run states |
| [`components/`](../../internal/view/components/) | Reusable elements composed by pages and partials; their props structs live in [`components/props.go`](../../internal/view/components/props.go) |
| [`props.go`](../../internal/view/props.go), [`render.go`](../../internal/view/render.go), [`helpers.go`](../../internal/view/helpers.go), [`seo.go`](../../internal/view/seo.go), [`models.go`](../../internal/view/models.go) | `BaseProps`/`NewBaseProps`, `Render`, HTMX header helpers, canonical/OG URL helpers, shared view models (`QuizScore`, `PatternSection`) |

Every `.templ` file has a sibling `*_templ.go` produced by `task templ:generate` (`templ generate`). Never hand-edit the generated file: `task check:generated` ([`scripts/checkgenerated`](../../scripts/checkgenerated/main.go)) regenerates into a scratch directory and diffs against the committed output, and `task ci` runs it. `task dev` runs air, whose build command in [`.air.toml`](../../.air.toml) is `templ generate && go build`, so edits to `.templ` files regenerate on save. Air does not rebuild CSS — run `task css:watch` alongside it when you touch `input.css` or add new utility classes.

## Props

Every page declares a props struct at the top of its own `.templ` file and embeds `view.BaseProps`; partials declare theirs the same way; components add theirs to `components/props.go`. `map[string]interface{}` is forbidden in `internal/view` (`adr017-typed-view-props` in [`scripts/adrcheck`](../../scripts/adrcheck/main.go)).

```go
// internal/view/props.go
type BaseProps struct {
	Title       string
	CurrentYear int
	UserName    string // display name shown in the authenticated user menu; empty for guest pages
	// Description is the page-specific meta/og description; blank falls
	// back to DefaultDescription (see MetaDescription).
	Description string
}
```

`view.NewBaseProps(title)` fills `CurrentYear`; the layout reads `Title`, `UserName`, and `MetaDescription()`. A page props struct composes it with concrete fields, including pointers to partial props when the page hosts a fragment on direct navigation:

```go
// internal/view/pages/quiz.templ
type QuizPageProps struct {
	view.BaseProps
	// Teaser marks a signed-out visit: preview the quiz and sell the sign-in.
	Teaser bool
	// GuestBanner marks an anonymous identity: offer the upgrade (#68).
	GuestBanner bool
	// Empty marks a database with no seeded questions.
	Empty bool
	// Question is the card to answer; nil when Empty or when Result is shown.
	Question *partials.QuizQuestionProps
	// Result is the outcome of a non-HTMX answer submit (full-page fallback).
	Result *partials.QuizResultProps
	Score  view.QuizScore
}
```

## Rendering a page vs an HTMX fragment

There is one render path. `view.Render` sets the content type, writes the status (headers cannot change once templ starts streaming), and renders the component with the request path stored in the context so the layout can emit a canonical URL without reading the `Host` header:

```go
// internal/view/render.go
func Render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	return component.Render(webutil.WithRequestPath(r.Context(), r.URL.Path), w)
}
```

Handlers pick the component with `view.IsHTMXRequest(r)` (`HX-Request: true`). A partial renders standalone; the same data wrapped in the page props renders inside the layout. The quiz handler does this at every exit, including the validation-error path, which keeps `422` and the same card:

```go
// internal/handler/quiz_handlers.go
qProps := quizQuestionProps(question, choices, topic)
if view.IsHTMXRequest(r) {
	renderQuiz(w, r, http.StatusOK, partials.QuizQuestionCard(qProps))
	return
}
props := pages.QuizPageProps{
	BaseProps:   view.NewBaseProps("Architecture Quiz"),
	GuestBanner: user.IsAnonymous,
	Question:    &qProps,
	Score:       score,
}
renderQuiz(w, r, http.StatusOK, pages.QuizPage(props))
```

`renderQuiz` is a small wrapper that logs a `Render` error with `slog`; copy that shape rather than ignoring the error. For responses that need HTMX headers, `helpers.go` has `SetHXRedirect` and `SetHXTrigger` (the base layout turns an `HX-Trigger` header into a toast).

## Components you already have

| Component | File | Purpose |
|---|---|---|
| `Button(ButtonProps)` | `components/button.templ` | `primary`/`secondary`/`danger` variants mapped to `.btn-*`; `Attrs` carries HTMX/ARIA attributes |
| `Input(InputProps)` | `components/input.templ` | Labelled input with required marker, helper text, error message, and `aria-describedby` wiring |
| `Form(FormProps)` / `FormValidation(FormValidationProps)` | `components/form.templ` | Form wrapper defaulting to `post`; `FormValidation` adds Alpine client-side checks on top of a server error, with prop strings JSON-escaped before entering the `x-data` context |
| `CSRFField()` | `components/csrf.templ` | Hidden token input for POST forms; the non-JS twin of the `hx-headers` set on `<body>` |
| `Card(CardProps)` | `components/card.templ` | Titled surface with an optional `Footer templ.Component` slot; body via `{ children... }` |
| `Alert(AlertProps)` | `components/alert.templ` | `role="alert"` box for `success`/`warning`/`error`/`info`, optionally dismissible with Alpine |
| `BrandMark(class string)` | `components/brand.templ` | The bolt-in-brackets mark in `currentColor`; brand placements only, never functional controls |
| `SkipLink`, `AriaLiveRegion`, `LoadingIndicator`, `FocusableElement` | `components/accessibility.templ` | Skip link, screen-reader live region, HTMX spinner, keyboard-operable container |

## Styling: role tokens

Components never name colors; they name roles (`bg-surface`, `text-muted-foreground`, `border-border`, `bg-danger/10 text-danger`). The roles are CSS variables in the `@theme` block of [`web/static/css/input.css`](../../web/static/css/input.css), and the `.dark` block in the same file overrides them, so a component never writes a `dark:` color variant. Tailwind v4 has no JavaScript config: `input.css` is the only token source, and `@source` there points the scanner at `internal/view/**/*.templ`. `task css:build` compiles it to `app.css` with `--minify`, the same recipe the Dockerfile uses. The `.btn`, `.btn-primary`, `.btn-secondary`, `.btn-danger`, and `.input` classes are defined in its `@layer components`.

Two tests pin this and run in `task ci`:

- [`tokens_test.go`](../../internal/view/tokens_test.go) walks every `.templ` file and fails on `dark:` color variants, raw palette utilities from any Tailwind family (`gray-500`, `green-100`, `blue-600` ...), and the deleted `-dark` token twins.
- [`tokens_contrast_test.go`](../../internal/view/tokens_contrast_test.go) parses `input.css`, resolves the role pairs in both modes, and computes WCAG 2.1 contrast: 4.5:1 for text pairs (foreground, muted-foreground, link, danger, success, warning on surface/background; white on primary), 3:1 for large text and the `border-input` form boundary.

The full role table and the two judgement rules the tests cannot check (solid buttons ride brand constants; status feedback is tint plus role text) are in [`docs/design-system.md`](../design-system.md). Do not restate them in components; read them.

## Accessibility and progressive enhancement

ADR-009 asks for WCAG 2.1 AA, semantic HTML with ARIA only where semantics fall short, mobile-first Tailwind breakpoints, and keyboard/screen-reader testing. It is not machine-checked beyond the contrast test, so the conventions below are what reviewers look for, cited from the templates:

- **Forms work without JavaScript.** `QuizQuestionCard` in `partials/quiz_question.templ` gives the `<form>` both `method="post" action=...` and `hx-post`/`hx-target="#quiz-card"`; the handler serves a full page or a fragment from the same POST. `FlashcardCard` in `partials/flashcard.templ` does the same for "Mark known" and "Delete", each with `@components.CSRFField()`.
- **Native disclosure over scripted toggles.** The home explainer (`pages/home.templ`) and the RLS proof panel (`partials/isolation.templ`) use `<details>`/`<summary>` for source peeks; `open:bg-surface-hover` styles the open state.
- **Grouped inputs get a legend.** The quiz choices are radios inside `<fieldset>` with `<legend class="sr-only">Choose an answer</legend>`.
- **Focus is visible and reachable.** The layout's first element is `<a href="#main-content" class="sr-only focus:not-sr-only ...">Skip to content</a>`; interactive controls use `focus:ring-2 focus:ring-primary`. Icon-only buttons carry `aria-label` (the nav toggle, dark-mode toggle, and delete button), and decorative SVGs carry `aria-hidden="true"`.
- **State is announced.** Toggles bind `:aria-pressed` and disclosure buttons bind `:aria-expanded` with `aria-controls`; the toast container is `aria-live="polite" aria-atomic="true"`; validation messages render with `role="alert"`; `Input` emits `aria-describedby` for helper and error text.
- **Responsive by default.** The layout and header use mobile-first classes (`hidden md:flex`, `hidden sm:inline`, `px-4 sm:px-6 lg:px-8`); the mobile nav is a disclosure panel, not a separate template.

## CSP

[ADR-028](../adr/ADR-028-CSP-Alpine-Compatibility.md): the policy is `script-src 'self' 'unsafe-eval'` — `'unsafe-eval'` exists only because Alpine's expression engine compiles `x-data`/`x-show`/`x-init` with the `Function` constructor, and `'unsafe-inline'` is never added. Templates may ship behavior only as Alpine attributes or by loading a file under `web/static/js/`; the base layout's own behavior lives in `app.js`, loaded before Alpine so `alpine:init` registrations are in place. Two tests enforce this in `task ci`: `internal/server/server_test.go` fails any rendered page with a `<script>` tag lacking `src`, and `internal/middleware/security_test.go` pins the directive. A `<style>` block is not affected (the toast keyframes in `base.templ` are one), but prefer `input.css`.

## Scaffolding a new view

The [`templ-component-scaffold`](../../.claude/skills/templ-component-scaffold/SKILL.md) skill walks the kind decision (page wraps `@layouts.Base`; partial and component do not), writes the typed props struct where the existing views keep theirs, runs `task templ:generate`, and wires the handler through `view.Render` with an `IsHTMXRequest` branch. Whether you use the skill or do it by hand, the checklist before claiming done is the same: generated output committed, a table-driven test for any non-trivial handler logic ([ADR-023](../adr/ADR-023-Testing-Philosophy.md)), and `task ci` green — which is where `tokens_test.go`, the inline-script test, and `check:generated` run.
