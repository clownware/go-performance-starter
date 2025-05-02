# ADR-012: Routing, Handler, and UI Patterns

## Status
Accepted

## Context

Consistent routing, handler structure, and UI update patterns are important for maintainability and developer experience. The team must formalize conventions for Go HTTP routing, handler organization, and dynamic UI updates (HTMX/Alpine.js).

## Decision

- Use chi router with route groups for feature/module organization.
- Follow RESTful conventions for endpoint naming.
- Keep handler functions focused and short; business logic is separated from HTTP layer.
- Use HTMX for partial UI updates, with Go templates rendering HTML fragments.
- Alpine.js is used only for client-side interactivity that cannot be handled server-side.
- Error feedback to users is delivered via HTMX triggers and reusable error partials.

## Consequences

- Routing and handlers are easy to follow and extend.
- UI updates are efficient and minimize client JS.
- Error handling is consistent and user-friendly.
