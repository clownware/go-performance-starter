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

Implement CSRF protection for all state-changing HTMX requests to prevent cross-site request forgery attacks.

### Token Generation and Storage

Generate CSRF tokens server-side for each session and include them in your templates:

```html
<!-- Include CSRF token in meta tag for global access -->
<head>
    <meta name="csrf-token" content="{{ .CSRFToken }}">
</head>

<!-- Or include in hidden form field for form submissions -->
<form hx-post="/api/items" hx-target="#result">
    <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
    <input type="text" name="item_name" placeholder="Item name">
    <button type="submit">Create Item</button>
</form>
```

### HTMX Configuration

Configure HTMX to send the CSRF token with all requests:

```html
<!-- Option 1: Global configuration using hx-headers on body -->
<body hx-headers='{"X-CSRF-Token": "{{ .CSRFToken }}"}'>
    <!-- All HTMX requests will include the token -->
</body>

<!-- Option 2: Per-request using hx-headers attribute -->
<button 
    hx-post="/api/items/delete/123"
    hx-headers='{"X-CSRF-Token": "{{ .CSRFToken }}"}'
    hx-target="#item-123"
    hx-swap="outerHTML">
    Delete
</button>

<!-- Option 3: Using JavaScript to read from meta tag -->
<script>
    document.body.addEventListener('htmx:configRequest', (event) => {
        // Get token from meta tag
        const token = document.querySelector('meta[name="csrf-token"]').getAttribute('content');
        // Add to request headers for non-GET requests
        if (event.detail.verb !== 'get') {
            event.detail.headers['X-CSRF-Token'] = token;
        }
    });
</script>
```

### Server-Side Validation

Validate the CSRF token on the server for all state-changing operations (implemented in Phase 6):

```go
// Example validation in middleware
func ValidateCSRF(r *http.Request) bool {
    // Get token from header (sent by HTMX)
    requestToken := r.Header.Get("X-CSRF-Token")
    if requestToken == "" {
        // Fallback to form field
        requestToken = r.FormValue("csrf_token")
    }
    
    // Validate against session token
    sessionToken := getSessionCSRFToken(r)
    return requestToken == sessionToken && requestToken != ""
}
```

### Best Practices

- **Token rotation**: Regenerate tokens on login/logout
- **Skip safe methods**: Only validate non-GET requests
- **Consistent approach**: Use one method (headers or form fields) consistently
- **Error handling**: Return clear 403 Forbidden for invalid tokens
- **Token expiry**: Consider time-based token expiration for sensitive operations

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
