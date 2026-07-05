# ADR-008: UI Composition and Template Structure

**Date**: 2025-05-01

## Status
Superseded by ADR-017

## Context

The Alpine Go Performance Starter aims for a modern, maintainable, and accessible UI with minimal JS. The team needs a clear architectural approach for composing UI, structuring templates, and rendering dynamic content.

## Decision

- Use Go's `html/template` for all HTML rendering, leveraging its built-in escaping for security.
- Organize templates in `/web/templates`, with partials in `/web/templates/partials`.
- Use HTMX for dynamic UI updates and partial rendering, enabling server-driven UI with minimal client JS.
- Favor small, composable partials (e.g., `_item.html`, `_error.html`) for reuse.
- All UI should be accessible and responsive by default (see ADR-009).

## Consequences

- UI is modular and maintainable.
- Progressive enhancement is easy; works with and without JS.
- Security is improved via template escaping.
