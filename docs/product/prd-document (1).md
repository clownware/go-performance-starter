# Micro SaaS Starter Kit – Product Requirements Document

## 1. Executive Summary

The Micro SaaS Starter Kit (MSSK) is an opinionated boilerplate that enables solo developers and small teams to bootstrap a subscription-based SaaS product in minutes. It combines a Go (Chi) backend, HTMX + Alpine.js frontend, Tailwind CSS styling, and Supabase-powered authentication/PostgreSQL—deployed to Cloudflare's edge.

The kit focuses on core plumbing (auth, CRUD pattern, billing stub, testing, CI) so builders can spend their time on application logic, not infrastructure setup. MSSK ships as an MIT-licensed GitHub template with comprehensive docs and example code.

## 2. Problem Statement

Setting up a production-viable SaaS baseline is repetitive and error-prone. Developers must wire authentication, DB migrations, configuration, CI, basic UI, logging, and deployment before shipping any real feature. Existing boilerplates are often language-specific, over-featured, or closed source. Indie hackers need a minimal, modern, OSS starter that demonstrates best practice but stays uncluttered.

## 3. Goals & Non-Goals

### Goals
- Provide secure email/password auth via Supabase with server-side JWT validation
- Demonstrate user-scoped CRUD ("Items") with HTMX forms and progressive enhancement
- Ship batteries-included DX (hot reload, linting, tests, CI, container, Cloudflare deploy guide)
- Offer stubs and interfaces for billing, email, background jobs
- Keep codebase under 2,000 LOC and first-run setup under 15 minutes

### Non-Goals
- Build an end-to-end SaaS application (CRM, PM tool, etc.)
- Offer a full UI component library or design system
- Support non-Postgres databases or non-Supabase auth providers out of the box
- Implement production-ready billing flows or complex RBAC
- Handle multi-tenancy—single tenant only

## 4. Target Audience & Personas

- **Indie Hacker Ian** – part-time builder launching a microproduct for niche audiences
- **Small Team Tina** – 2-3 devs inside a startup validating MVPs quickly

Both value clear docs, fast onboarding, and opinionated defaults that map to best practice.

## 5. Functional Requirements

### Authentication
- Email/password (mandatory) + example Google OAuth (commented)
- Server-side JWT verification middleware, login/logout forms

### Account Management
- Change password, (optional) delete account

### CRUD Resource – Items
- List, create, edit, delete user-owned items
- Validation & flash messages

### Basic UI/Layout
- Guest vs. authenticated layouts, navbar w/ user dropdown

### Billing Stub
- BillingProvider interface, Stripe sample client creating customer on signup

### Background Job Example
- Welcome-email goroutine after signup

### Developer Tooling (DX)
- air, Taskfile, sqlc, golangci-lint, unit + integration tests

### Deployment Story
- Dockerfile, Cloudflare Pages + Workers guide

## 6. Technical Requirements

- **Language/Frameworks**: Go ≥ 1.22, Chi router, HTMX 1.x, Alpine 3.x, Tailwind 3.x
- **Database**: PostgreSQL 15 (Supabase), migration via golang-migrate, type-safe access via sqlc
- **Observability**: zerolog logging, OpenTelemetry hooks (exporter toggle)
- **Security**: CSRF tokens on form posts, rate-limit middleware, secure headers
- **Config**: 12-Factor env var loader with .env.example + direnv

## 7. Developer Experience Requirements

| Area | Requirement |
|------|-------------|
| Setup time | From gh repo clone to first running server ≤ 15 min |
| Docs | README + /docs/ covering local dev, testing, extending CRUD, deploying |
| Code Quality | go vet, golangci-lint run zero errors by default |
| Testing | ≥ 80% coverage on core packages; CI runs unit + integration in < 3 min |
| Container | docker compose up brings backend + Postgres for tests |

## 8. Success Metrics

- GitHub Stars: ≥ 500 within 6 months of launch
- Average Setup Time (surveyed users): ≤ 20 min
- Issue-to-PR turnaround: ≤ 72 h for critical bugs
- CI Pass Rate: ≥ 95% on main

## 9. Milestones & Timeline (indicative)

| Milestone | Deliverables | ETA |
|-----------|--------------|-----|
| M0 – Kickoff | ADRs, repo scaffolding, license | Day 0 |
| M1 – Auth + Layout | Supabase email/pass auth, guest/app templates | Week 1 |
| M2 – Items CRUD | DB schema, handlers, HTMX views, tests | Week 2 |
| M3 – DX Tooling | Taskfile, air, lint, sqlc, CI, Docker | Week 3 |
| M4 – Billing/Email Stubs | Interfaces + Stripe sample, console email | Week 4 |
| M5 – Docs + Release | README, docs site, v0.1.0 tag | Week 5 |

## 10. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Supabase pricing changes | Abstract auth via interface; docs note alternative altruAuth |
| Cloudflare Workers Go SDK breaking changes | Pin SDK version; CI fails on breaking API |
| Scope creep | Strict PR gate: non-goals table must stay intact |

## 11. Out-of-Scope (Reiterated)

- Multi-tenancy or organization accounts
- Granular RBAC
- Full-text search, analytics, advanced payment flows

## 12. Glossary

- **HTMX** – library enabling hypermedia-driven dynamic UIs without SPAs
- **sqlc** – generates type-safe Go code from SQL queries
- **Supabase** – OSS Firebase alternative providing Postgres, Auth, Storage

## Appendix A – File/Folder Layout (high-level)

```
/cmd
  └─ api/main.go        # entrypoint for Go server
/internal
  ├─ auth/middleware.go
  ├─ items/handler.go
  ├─ billing/stripe.go
  └─ …
/migrations             # golang-migrate SQL files
/sql                    # sqlc query files
/web
  ├─ templates/*.html
  └─ static/css/output.css (Tailwind build)
Taskfile.yml            # dev tasks
Dockerfile              # prod image
.github/workflows/ci.yml
```