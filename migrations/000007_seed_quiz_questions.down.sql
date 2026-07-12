-- Remove the seeded quiz questions (attempts/flashcards cascade or null out
-- per the FKs in 000003).
DELETE FROM quiz_questions WHERE slug IN (
    'chi-middleware-order',
    'rate-limit-tiers',
    'typed-props',
    'progressive-enhancement',
    'rls-scoping',
    'sqlc-repositories',
    'templ-compilation',
    'htmx-fragments',
    'performance-budgets',
    'guest-identities'
);
