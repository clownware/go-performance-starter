# ADR-024: Demo Application Direction

**Date**: 2026-06-11

## Status

Accepted (2026-07-05)

## Context

The starter needs a demo application that (a) markets the project in under 60 seconds, (b) demonstrates every HTMX/Alpine pattern it supports, and (c) proves the full stack works end-to-end — persisted Postgres data, Supabase auth, RLS ownership, and the performance budgets. The current demo uses an abstract "Items" resource backed by an in-memory `map` (`internal/handler/items_handlers.go`) — it persists nothing and exercises none of sqlc/repository/RLS/auth, so it under-sells exactly what the starter exists to provide.

The prior plan ([`docs/updates/ux-overhaul-spec.md`](../updates/ux-overhaul-spec.md)) proposed a per-user **bookmarks** CRUD plus a `/patterns` showcase. The patterns showcase is strong and retained. Bookmarks is replaced: it is a generic domain unrelated to the product, and per-user scoping skips the multi-tenant story the repo already ships.

A second constraint surfaced: to host this demo publicly, a forker should be able to *use* it (take the quiz, save flashcards, see progress) **without a signup wall** — but stripping auth would gut the starter's core proof.

## Decision

Build the demo as a **self-documenting architecture explainer** with three coherent surfaces, replacing the abstract Items domain:

1. **Architecture explainer (narrative spine).** Interactive, server-rendered walkthrough of this system's own architecture — request lifecycle, middleware stack, templ rendering, sqlc/repository, RLS, and the performance budgets (rendered live from `internal/performance/` constants). The medium proves the message: a minimal-JS page teaching minimal-JS architecture.
2. **`/patterns` showcase.** The interactive HTMX/Alpine pattern gallery from the prior spec (live demo + templ source + handler source per pattern). Stub data, no DB. Retained as-is.
3. **Quiz + flashcards (the concrete, persisted, authenticated CRUD domain).** A quiz over the explainer content; wrong answers offer to save a **flashcard**; users review, mark-known, and delete flashcards, with progress persisted. This is the real full-stack demo: sqlc + repository + RLS + auth + forms.

**Identity: anonymous-auth guest mode, issued server-side.** Visitors get a real but anonymous Supabase identity on arrival (no signup), so RLS and per-user CRUD are genuinely exercised — the anonymous user is just a `users` row whose `auth_id` is the anonymous `auth.uid()`, and the existing `auth_id = auth.uid()::text` self-access policy applies unchanged. Because the app is server-rendered (not a SPA), the Go backend performs the anonymous sign-in against GoTrue and manages an httpOnly session cookie. An **"upgrade to keep your progress"** flow links an email/password identity to the same account (data-preserving). This is one identity model — no parallel guest path.

**Anonymous sign-in mechanism (decided at acceptance, 2026-07-05).** `gotrue-go v1.2.1` / `supabase-go v0.0.4` expose no anonymous sign-in method, so the Go backend calls the GoTrue REST endpoint directly (credential-less `POST /auth/v1/signup`, the same call `supabase-js` `signInAnonymously()` makes) with "anonymous sign-ins" enabled in Supabase. No new dependency; the exact request shape is verified against a live Supabase project during implementation. Bumping the client library was rejected until a released version supports anonymous sign-in; a custom server-signed guest session was rejected because it forks the identity model this ADR exists to avoid.

### Accompanying constraints (best practice for anonymous auth)

- Gate expensive/destructive operations on the `is_anonymous` claim.
- A TTL **reaper** background job deletes inactive anonymous users (demonstrates the background-jobs guide).
- **Rate-limit** public write endpoints (demonstrates the existing rate-limiter middleware).

## Consequences

- One coherent story replaces three disconnected pieces (marketing + abstract CRUD + gallery); the demo exercises auth, RLS, CRUD, background jobs, and rate limiting with zero signup friction.
- The abstract `items` handlers and in-memory `itemStore` are retired (some logic may be reused inside the `/patterns` showcase as stub demos).
- New tables (`quiz_questions`, `quiz_attempts`, `flashcards`) with RLS mirroring the existing `users_self_access` pattern; new sqlc queries and repositories.
- New surface area to secure: a public, anonymous-writable endpoint set — must ship with rate limiting and `is_anonymous` gating from day one.
- Requires Supabase anonymous sign-in enabled; the server-side path is the direct GoTrue REST call decided above — still a real anonymous user.
- Explainer content is a writing cost, but it doubles as user-facing documentation.

## Alternatives Considered

- **Per-user or org-scoped bookmarks.** Rejected — generic, unmotivated domain; bookmarks doesn't tie to the learning experience the way quiz-driven flashcards do.
- **No auth (cookie-only guest session, no Supabase user).** Rejected — builds a parallel non-auth code path that isn't what the starter is for, and skips the RLS/auth proof.
- **Seeded read-only demo + login-to-write.** Rejected — reintroduces signup friction for the interactive part that matters.
- **Client-only flashcards (Alpine + localStorage).** Rejected — demonstrates none of the backend.

## References

- Supersedes the bookmarks/landing scope of [`docs/updates/ux-overhaul-spec.md`](../updates/ux-overhaul-spec.md); design brief in [`docs/updates/demo-design-brief.md`](../updates/demo-design-brief.md).
- Related: [ADR-004](ADR-004-Authorization-Strategy-RLS.md) (RLS), [ADR-007](ADR-007-Frontend-Stack-Selection.md) (frontend), [ADR-014](ADR-014-Security-Patterns-and-Threat-Model.md) (security), [ADR-017](ADR-017-Templ-Adoption.md) (templ).
