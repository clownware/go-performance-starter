# Phase 5 — HTMX & Alpine Integration

Implement smooth user interactions with minimal JavaScript.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 5.01 | Create reusable HTMX patterns | Tables, forms, lazy loading |
| 5.02 | Implement Alpine components | Dropdowns, modals, tabs |
| 5.03 | Design smooth transitions | Loading states and swaps |
| 5.04 | Create advanced form handling | Validation and dynamic fields |
| 5.05 | Implement optimistic UI updates | Immediate user feedback |
| 5.06 | Ensure baseline functionality | Core flows work without JS |
| 5.07 | Create advanced HTMX patterns | Infinite scroll, typeahead |
| 5.08 | Implement toast notifications | User feedback system |
| 5.09 | Add animation system | Smooth transitions |

## Core Principles

- Use HTMX for server-driven DOM updates (no API endpoints needed)
- Implement Alpine.js only for client-side interactivity that HTMX can't handle
- Design handlers that respond appropriately to both regular and HTMX requests
- Ensure baseline functionality works without JavaScript for critical paths
- Implement proper loading indicators and transitions

## Common HTMX Patterns

- **Data tables**: Sorting, filtering, pagination with HTMX
- **Forms**: Validation, dynamic fields, conditional sections
- **Infinite scroll**: Load more content as user scrolls
- **Lazy loading**: Load content only when visible
- **Toast notifications**: Server-triggered notifications
- **Typeahead search**: Dynamic search results

## Alpine.js Use Cases

- **Dropdowns**: Toggle visibility of dropdown menus
- **Modals**: Show/hide modal dialogs
- **Tabs**: Switch between tab panels
- **Form validation**: Client-side validation before submission
- **Tooltips**: Show/hide tooltips on hover
- **Accordions**: Expand/collapse content sections

## Common Pitfalls

- **Overusing Alpine.js**: Use HTMX when possible for simpler code
- **Missing loading indicators**: Always show loading state for better UX
- **Race conditions**: Use proper swap strategies (swap:complete)
- **Poor error handling**: Provide clear error feedback
- **Missing progressive enhancement**: Ensure critical paths work without JS

## Exit Criteria

- Reusable HTMX patterns implemented
- Alpine.js components created for interactive elements
- Smooth transitions and loading states working
- Advanced form handling with validation functioning
- Critical paths work without JavaScript
- Toast notification system implemented
- Animation system providing smooth transitions
