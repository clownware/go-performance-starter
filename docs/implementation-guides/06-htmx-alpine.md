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

## CSRF Protection with HTMX

Implement CSRF protection for all state-changing HTMX requests to prevent cross-site request forgery attacks. The starter uses the double-submit-cookie pattern (ADR-014 §3) — there is no server-side token store.

### Token Generation and Storage

The CSRF middleware (`internal/middleware/csrf.go`) issues a random 32-byte token in an HttpOnly `csrf_token` cookie (12h max-age, reissued transparently on expiry) and places the same token in the request context, where templ components read it. Templates are templ components — raw `html/template` syntax is forbidden (ADR-017):

```templ
// Hidden input for form submissions — components.CSRFField()
// (internal/view/components/csrf.templ)
templ CSRFField() {
	<input type="hidden" name="csrf_token" value={ webutil.CSRFTokenFromContext(ctx) }/>
}
```

Use it inside any state-changing form:

```templ
<form hx-post="/api/items" hx-target="#result">
	@components.CSRFField()
	<input type="text" name="item_name" placeholder="Item name"/>
	<button type="submit">Create Item</button>
</form>
```

### HTMX Configuration

The base layout (`internal/view/layouts/base.templ`) sets `hx-headers` on `<body>` so every HTMX request carries the token — `hx-headers` is inherited by all descendants:

```templ
<body hx-headers={ csrfHxHeaders(webutil.CSRFTokenFromContext(ctx)) }>
```

where `csrfHxHeaders` builds the `{"X-CSRF-Token": "<token>"}` JSON. Forms rendered with `@components.CSRFField()` also work without JavaScript: HTMX serializes form fields, and the middleware falls back to the `csrf_token` form field when the header is absent.

### Server-Side Validation

For unsafe methods, the middleware compares the submitted header/form value against the cookie the browser sent, in constant time (implemented in Phase 6):

```go
// internal/middleware/csrf.go (abridged)
cookie, err := r.Cookie(CSRFCookieName)
if err != nil || !isValidTokenFormat(cookie.Value) {
    http.Error(w, "Forbidden: missing CSRF cookie", http.StatusForbidden)
    return
}
sent := r.Header.Get(CSRFHeaderName) // X-CSRF-Token, set via hx-headers
if sent == "" {
    sent = r.PostFormValue(CSRFFormField) // csrf_token hidden input
}
if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(sent)) != 1 {
    http.Error(w, "Forbidden: CSRF token mismatch", http.StatusForbidden)
    return
}
```

### Best Practices

- **Skip safe methods**: Only validate non-GET/HEAD/OPTIONS/TRACE requests
- **HttpOnly cookie**: The token reaches markup via server rendering, never via script access
- **Constant-time comparison**: Use `subtle.ConstantTimeCompare` to avoid timing leaks
- **Error handling**: Return clear 403 Forbidden for missing or mismatched tokens
- **Token expiry**: The cookie's 12h max-age bounds token lifetime; expired tokens are reissued transparently

This CSRF implementation will be fully wired up in Phase 6 (Authentication & Security).

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
