// Main application JavaScript

// Register Alpine.js components and stores when available.
//
// Alpine ships as the @alpinejs/csp build (ADR-036): its expression parser
// needs no eval, so script-src stays 'self'. Simple state stays inline in
// the templates; a component moves here when it needs what the parser
// rejects — methods, getters, arrow callbacks, multi-statement blocks,
// property assignment or browser globals.
document.addEventListener('alpine:init', () => {
  // Example Alpine.js component
  Alpine.data('dropdown', () => ({
    open: false,
    toggle() {
      this.open = !this.open;
    }
  }));

  // Dark mode on <body>: an explicit user choice wins; the system preference
  // is only the default. The class lands on <html> so role tokens flip
  // (ADR-029).
  Alpine.data('darkMode', () => ({
    dark: false,
    init() {
      const stored = localStorage.getItem('dark');
      this.dark = stored !== null
        ? stored === 'true'
        : window.matchMedia('(prefers-color-scheme: dark)').matches;
      this.apply(this.dark);
      this.$watch('dark', (val) => {
        this.apply(val);
        localStorage.setItem('dark', val ? 'true' : 'false');
      });
    },
    apply(val) {
      document.documentElement.classList.toggle('dark', val);
    }
  }));

  // Toast stack in the base layout, fed by the window 'toast' event
  // (@toast.window in the markup; the htmx bridge is below).
  Alpine.data('toasts', () => ({
    toasts: [],
    add(type, msg) {
      if (this.toasts.length >= 3) this.toasts.shift();
      this.toasts.push({ type, msg, id: Date.now() + Math.random() });
      setTimeout(() => this.toasts.shift(), 4000);
    }
  }));

  // /patterns confirm-delete: htmx:confirm pauses the request and hands over
  // issueRequest; Delete resumes it, Cancel/Escape/backdrop drop it. pause()
  // stores the function inside a method because Alpine calls a
  // function-valued x-on result (#176).
  Alpine.data('confirmDialog', () => ({
    open: false,
    issue: null,
    pause(e) {
      this.issue = e.detail.issueRequest;
      this.open = true;
    },
    cancel() {
      this.open = false;
      this.issue = null;
    },
    confirm() {
      const issue = this.issue;
      this.cancel();
      if (issue) issue(true);
    }
  }));

  // components.FormValidation: client-side rules mirroring the server's,
  // read from data-* attributes (ADR-036 §4) so prop values arrive through
  // templ's attribute escaping.
  Alpine.data('formField', () => ({
    value: '',
    touched: false,
    required: false,
    pattern: null,
    minLength: 0,
    maxLength: 0,
    serverError: '',
    requiredMessage: '',
    patternMessage: '',
    init() {
      const d = this.$el.dataset;
      this.value = d.value || '';
      this.required = d.required === 'true';
      this.pattern = d.pattern ? new RegExp(d.pattern) : null;
      this.minLength = Number(d.minLength) || 0;
      this.maxLength = Number(d.maxLength) || 0;
      this.serverError = d.serverError || '';
      this.requiredMessage = d.requiredMessage || '';
      this.patternMessage = d.patternMessage || '';
    },
    get displayMessage() {
      if (this.serverError) return this.serverError;
      if (this.required && !this.value) return this.requiredMessage;
      if (this.pattern && !this.pattern.test(this.value)) return this.patternMessage;
      if (this.minLength > 0 && this.value.length < this.minLength) return 'Must be at least ' + this.minLength + ' characters';
      if (this.maxLength > 0 && this.value.length > this.maxLength) return 'Must be at most ' + this.maxLength + ' characters';
      return '';
    },
    get isValid() {
      return this.displayMessage === '';
    }
  }));

  // Success flash on the profile and password forms: visible for 3s.
  Alpine.data('autoDismiss', () => ({
    show: true,
    init() {
      setTimeout(() => { this.show = false; }, 3000);
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

// Server-driven toasts: a response's HX-Trigger header carries the message
// and HX-Toast-Type its kind; the base layout's toasts component listens for
// the window 'toast' event.
document.addEventListener('htmx:afterRequest', (event) => {
  const xhr = event.detail.xhr;
  const msg = xhr.getResponseHeader('HX-Trigger');
  if (!msg) return;
  const type = xhr.getResponseHeader('HX-Toast-Type') || 'success';
  window.dispatchEvent(new CustomEvent('toast', { detail: { type, msg } }));
});

document.addEventListener('htmx:beforeSwap', (event) => {
  // htmx 2's default responseHandling skips swapping 4xx/5xx responses, but
  // this app's handlers return meaningful fragments on validation and auth
  // failures (400/401/409/422): inline form errors, re-rendered question
  // cards. Swap those so the user sees the feedback; other statuses keep the
  // default behavior.
  // A 429 is swapped only when the limiter's refusal is HTML (the /patterns
  // demo, ADR-034) — the plain-text default stays out of the page.
  const status = event.detail.xhr.status;
  const contentType = event.detail.xhr.getResponseHeader('Content-Type') || '';
  if ([400, 401, 409, 422].includes(status) || (status === 429 && contentType.includes('text/html'))) {
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

// Scroll-spy for the /patterns sidebar TOC: highlight the pattern section
// currently in view. Progressive enhancement — without JS the TOC is still
// a working anchor list.
document.addEventListener('DOMContentLoaded', () => {
  const nav = document.querySelector('[data-testid="patterns-nav"]');
  if (!nav || !('IntersectionObserver' in window)) return;

  const links = new Map();
  nav.querySelectorAll('a[href^="#"]').forEach((a) => {
    links.set(a.getAttribute('href').slice(1), a);
  });

  const activate = (id) => {
    links.forEach((a, key) => {
      if (key === id) {
        a.setAttribute('aria-current', 'location');
        a.classList.add('text-link', 'font-medium');
        a.classList.remove('text-muted-foreground');
      } else {
        a.removeAttribute('aria-current');
        a.classList.remove('text-link', 'font-medium');
        a.classList.add('text-muted-foreground');
      }
    });
  };

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((e) => {
        if (e.isIntersecting) activate(e.target.id);
      });
    },
    // Fire when a section crosses the upper-middle band of the viewport.
    { rootMargin: '-15% 0px -70% 0px' }
  );
  links.forEach((_, id) => {
    const section = document.getElementById(id);
    if (section) observer.observe(section);
  });
});
