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

  // Cross-component state for the /patterns global-store demo: unrelated
  // x-data islands read and write this via $store.demo.
  Alpine.store('demo', {
    count: 0,
    inc() {
      this.count++;
    }
  });
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

// No global loading overlay: it flashed the whole screen on every fragment
// swap. In-flight feedback is per-element — htmx puts .htmx-request on the
// triggering element (styled in input.css); hx-indicator and hx-disabled-elt
// handle explicit spinners and disabled submit buttons.
