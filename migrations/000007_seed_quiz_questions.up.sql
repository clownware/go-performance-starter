-- Seed the architecture quiz (ADR-024): the questions ARE the explainer's
-- teachable spine — two per topic across the request's journey through the
-- stack (routing → handlers → database/RLS → frontend → performance/auth).
-- Idempotent by slug so re-running (or a pre-existing row) is harmless.

INSERT INTO quiz_questions (slug, topic, prompt, choices, correct_index, explanation) VALUES
(
    'chi-middleware-order',
    'routing',
    'Where is the HTTP middleware stack assembled?',
    '["In internal/server via chi''s Use chain, in a deliberate order", "Each handler wraps itself in what it needs", "In the reverse proxy config", "A code generator wires it at build time"]',
    0,
    'setupMiddleware in internal/server/server.go composes security headers, request ID, client-IP resolution, rate limiting, compression, metrics, logging, recovery, timeout, and CSRF — and the order matters (the rate limiter must see the real client IP, so RealIP runs first).'
),
(
    'rate-limit-tiers',
    'routing',
    'How do sensitive endpoints get stricter rate limits than the rest of the app?',
    '["A second RateLimiter middleware wraps just that route group, on top of the global one", "The global limiter is tuned down for everyone", "They don''t — one limit fits all", "A CDN handles it"]',
    0,
    'The global limiter guards everything; credential endpoints and the anonymous-writable /learn group each add a stricter tier by wrapping their chi route group with another RateLimiter (ADR-014, ADR-024).'
),
(
    'typed-props',
    'handlers',
    'How does a handler pass data into a template?',
    '["A typed props struct compiled by templ", "map[string]interface{}", "A JSON string the template parses", "Package-level template variables"]',
    0,
    'Every view takes a typed props struct — templ compiles templates to Go, so a missing or mistyped field is a build error, and map[string]interface{} is forbidden (ADR-017).'
),
(
    'progressive-enhancement',
    'handlers',
    'What happens to the quiz answer form when JavaScript is disabled?',
    '["It posts normally and the server renders a full result page", "It breaks — HTMX is required", "Nothing happens silently", "A bundled SPA takes over"]',
    0,
    'Every mutation is a plain POST form; HTMX is an enhancement that swaps just the card when available. Pages must work without JS (ADR-007, ADR-012).'
),
(
    'rls-scoping',
    'database',
    'What stops one user from reading another user''s flashcards?',
    '["Postgres Row Level Security, evaluated from the request''s JWT claims", "A WHERE clause every handler must remember to add", "Filtering in Go after the query", "Nothing — it''s a demo"]',
    0,
    'Repository methods run inside a transaction that applies the requester''s JWT claims (SET LOCAL ROLE + request.jwt.claims), so auth.uid()-based policies scope every query at the database layer (ADR-004).'
),
(
    'sqlc-repositories',
    'database',
    'How does SQL enter this codebase?',
    '["Written in sql/queries/ and compiled to typed Go by sqlc, behind repository interfaces", "String concatenation in handlers", "An ORM builds it at runtime", "Stored procedures only"]',
    0,
    'sqlc generates type-checked Go from the queries in sql/queries/; handlers only see repository interfaces, so hand-written SQL strings never appear in request code (ADR-003).'
),
(
    'templ-compilation',
    'frontend',
    'What is a templ template at build time?',
    '["Type-checked Go code", "A text file parsed on every request", "JSX transpiled to JavaScript", "A YAML schema"]',
    0,
    'templ generates Go source from .templ files — rendering is a compiled function call with contextual auto-escaping, not runtime string templating (ADR-017).'
),
(
    'htmx-fragments',
    'frontend',
    'How does the pattern showcase update the page without a reload?',
    '["Handlers return HTML fragments that HTMX swaps into a target element", "JSON APIs plus client-side rendering", "WebSockets push a virtual DOM", "Hidden iframes"]',
    0,
    'The server stays the single source of rendered HTML: an hx-get/hx-post returns a fragment and hx-target/hx-swap places it — no client templating layer (ADR-007, ADR-012).'
),
(
    'performance-budgets',
    'performance',
    'Where do the performance budgets live, and where do they bite?',
    '["Constants in internal/performance/, enforced by task ci", "A wiki page nobody reads", "Only in the README", "They''re aspirational everywhere"]',
    0,
    'ADR-000 classifies every budget as Enforced, Monitored, or Aspirational: binary size, Docker image size, and gzipped JS/CSS fail CI on violation via task ci; response times are observed via Prometheus.'
),
(
    'guest-identities',
    'auth',
    'What identity does an anonymous demo visitor get?',
    '["A real Supabase user, created server-side via GoTrue anonymous sign-in", "A cookie-only fake session", "A shared demo account", "None — reads only"]',
    0,
    'Guests are real anonymous identities so RLS and per-user CRUD are genuinely exercised — the same users_self_access policy covers guests and registered accounts, and a TTL reaper cleans up inactive ones (ADR-024).'
)
ON CONFLICT (slug) DO NOTHING;
