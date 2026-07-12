// Main application JavaScript

// Register Alpine.js components and stores when available
document.addEventListener('alpine:init', () => {
  // Example Alpine.js component
  Alpine.data('dropdown', () => ({
    open: false,
    toggle() {
      this.open = !this.open;
    }
  }));
});

// HTMX event listeners
document.addEventListener('htmx:configRequest', (event) => {
  // Add any global request handling here
});

document.addEventListener('htmx:beforeSwap', (event) => {
  // HTMX skips swapping 4xx responses by default, but this app's handlers
  // return meaningful fragments on validation and auth failures (400/401/
  // 409/422): inline form errors, re-rendered question cards. Swap those so
  // the user sees the feedback; other statuses keep the default behavior.
  const status = event.detail.xhr.status;
  if ([400, 401, 409, 422].includes(status)) {
    event.detail.shouldSwap = true;
    event.detail.isError = false;
  }
});

document.addEventListener('htmx:afterSwap', (event) => {
  // Reinitialize Alpine components after HTMX content swap
  if (window.Alpine) {
    window.Alpine.initTree(event.detail.target);
  }
});

// Log when the app is fully loaded
document.addEventListener('DOMContentLoaded', () => {
  console.log('Go Performance Starter loaded');
});

// Global HTMX loading indicator — the overlay markup lives in the base
// layout; the listeners live here because inline <script> is CSP-forbidden
// (ADR-028). Events bubble to document, so this covers swapped content too.
document.addEventListener('htmx:beforeRequest', () => {
  const el = document.getElementById('global-loading');
  if (el) el.classList.remove('hidden');
});
document.addEventListener('htmx:afterRequest', () => {
  const el = document.getElementById('global-loading');
  if (el) el.classList.add('hidden');
});
