# ADR-007: Frontend Stack Selection

**Date**: 2025-05-01

## Status

Accepted

## Context

The Alpine Go Performance Starter requires a frontend architecture that complements its Go backend. The goals are to provide a modern user experience, enable rapid development, minimize client-side complexity, and maintain good performance. Traditional Single Page Applications (SPAs) often introduce significant build tooling complexity and state management overhead separate from the backend.

We need a stack that integrates well with server-rendered Go templates while allowing for dynamic UI updates and modern styling without requiring a heavy JavaScript framework.

## Decision

We will adopt the following frontend stack:

1.  **Tailwind CSS:** For utility-first CSS styling.
2.  **HTMX:** For enhancing server-rendered HTML with dynamic behaviors triggered via attributes (AJAX, CSS Transitions, WebSockets, etc.) directly from HTML.
3.  **Alpine.js:** For minimal, declarative client-side interactivity when needed (e.g., dropdowns, modals, simple component state) directly within the HTML markup.

These technologies will be used in conjunction with Go's `html/template` engine for server-side rendering.

## Consequences

### Positive

- **Reduced JavaScript:** Significantly less client-side JavaScript compared to SPA frameworks, leading to smaller bundle sizes and faster initial page loads.
- **Simpler State Management:** Most application state remains on the server, managed by Go handlers.
- **Improved Developer Experience:** Developers can often stay within HTML and Go, reducing context switching.
- **Progressive Enhancement:** Easier to build core functionality that works without JavaScript.
- **Performance:** Leverages server rendering for fast initial loads, with HTMX providing dynamic updates efficiently.
- **Alignment with Go:** Integrates naturally with Go's templating and HTTP handling capabilities.

### Negative

- **Less Suited for Highly Complex UIs:** May require more intricate solutions or fall back to more Alpine.js/custom JS for interfaces with very complex, real-time, stateful client-side interactions compared to what SPAs handle natively.
- **Potential HTML Bloat:** Overuse of utility classes or complex HTMX responses could increase HTML size if not managed.
- **Learning Curve:** Requires understanding HTMX attributes and how Alpine.js complements server-rendered HTML.
- **Server Coupling:** Frontend interactions are more tightly coupled to server endpoints and responses compared to decoupled SPAs.

## Alternatives Considered

- **Full SPA (React, Vue, Svelte) + Go API:**
    - *Rejected because:* Introduces significant build complexity, requires separate frontend routing and state management, increases overall JavaScript footprint, moves away from the simplicity goal.
- **Go Templates + Vanilla JS/jQuery:**
    - *Rejected because:* Requires writing significantly more imperative JavaScript for dynamic updates compared to HTMX's declarative approach.
- **Go Templates + Other CSS Frameworks (Bootstrap, Bulma):**
    - *Rejected because:* Tailwind's utility-first approach offers greater flexibility and avoids 'component-overload', aligning well with custom SaaS designs.
- **Other Server-HTML Libraries (Unpoly):**
    - *Rejected because:* HTMX has gained significant traction, has a simple API, and integrates well with various backends, including Go.

## References

- [Tailwind CSS](https://tailwindcss.com/)
- [HTMX](https://htmx.org/)
- [Alpine.js](https://alpinejs.dev/)
- [Go `html/template` Package](https://pkg.go.dev/html/template)
