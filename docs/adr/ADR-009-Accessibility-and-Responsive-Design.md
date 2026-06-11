# ADR-009: Accessibility and Responsive Design

**Date**: 2025-05-01

## Status
Accepted

## Context

Accessibility (a11y) and responsive design are core requirements for SaaS targeting a wide audience. The team must define standards for both.

## Decision

- Follow WCAG 2.1 AA guidelines for accessibility.
- Use semantic HTML and ARIA attributes where needed.
- All layouts and components must be mobile-first and responsive, using Tailwind CSS breakpoints.
- Test UI with screen readers and keyboard navigation.

## Consequences

- Product is usable by more people.
- Fewer a11y bugs in production.
- Some extra effort required in design and QA.
