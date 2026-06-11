# Engineering

Strong defaults for writing code in this repo. These rules apply with the same force as `CLAUDE.md`; the layering exists for organisation, not for softening. Named exceptions require an ADR.

## Package Layout

- `cmd/api/` — entrypoint only: config load, wiring, graceful shutdown. No business logic.
- `internal/` is the application. Keep packages focused: `handler` (HTTP), `server` (router + middleware stack), `repository` (data access interfaces + postgres impls), `database` (sqlc-generated — never hand-edit), `view` (templ UI), `middleware`, `auth`, `cache`, `config`, `performance`, `validate`, `webutil`.
- Dependencies point inward. Handlers depend on repository *interfaces*, not concrete postgres types.

## Go Style

- Idiomatic, concise Go. Functional patterns; avoid needless abstraction. Three similar lines beat a premature helper.
- Create an interface only when a consumer needs it (testing seam, multiple impls) — define it at the consumer, not the producer.
- Handle errors explicitly; wrap with context (`fmt.Errorf("...: %w", err)`). No panics in request paths; `Recoverer` is a backstop, not a strategy.
- Error handling lives at boundaries (HTTP handlers, DB calls, external APIs), not sprinkled through pure logic.
- Descriptive names with auxiliary verbs for booleans: `isActive`, `hasSession`, `canEdit`.
- Use `context.Context` for cancellation, timeouts, and request-scoped values; pass it as the first argument.

## View Layer (templ)

- Every page defines a props struct embedding `view.BaseProps` (`internal/view/props.go`). Construct props with concrete types in the handler.
- Pages compose the layout: `@layouts.Base(props.BaseProps) { ... }`. Partials render standalone (no layout) so HTMX fragments are correct.
- One render path: `view.Render(w, r, status, component)`. Choose page vs. partial with `view.IsHTMXRequest(r)`.
- After editing a `.templ` file, run `task templ:generate` (or `task dev`, which watches). Never hand-edit `*_templ.go`.

## Data Access

- Queries are declared in `sql/queries/`, schema in `sql/schema/`; `task db:generate` regenerates `internal/database/`.
- Handlers and services call repository interfaces (`internal/repository/`); the postgres implementation is an injected dependency.
- Schema changes are migrations in `migrations/` (golang-migrate, `*.up.sql`/`*.down.sql`); never edit an applied migration — add a new one.

## Logging & Errors

- Structured logging only. Log errors with request-scoped context (request ID, route); never log secrets.
- Return correct HTTP status codes and user-safe messages; keep internal detail in logs, not responses.

## Testing

- Table-driven tests with `testing` + `net/http/httptest`. One behaviour per case; name cases for the behaviour, not the implementation.
- Test both success and error paths. Use the repository interfaces with fakes; reserve a real Postgres harness for repository/integration tests.
- Never lower a coverage or performance threshold to make a test pass — fix the code or open an ADR documenting the exception.
