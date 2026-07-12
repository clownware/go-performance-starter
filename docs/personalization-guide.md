# Personalization Guide

You cloned this template and want it to be *your* app. This is the checklist of everything that carries the template's own identity, split into **required** (the app is misnamed or won't deploy until you change it) and **optional** (works fine as-is, but you'll probably want it eventually).

**Time to complete: ~30 minutes for the required items** (most of it the module rename), assuming the [Quick Start](../README.md#quick-start) already runs.

Nothing in this guide needs to happen before local development works — `task dev` runs fine on a fresh clone.

## Required

### 1. Go module path (~10 min — do this first)

The big one for a Go template: the module path `github.com/clownware/go-performance-starter` appears in `go.mod` and in the import blocks of ~67 Go files, plus the `.templ` sources. Everything else imports through it.

```bash
OLD=github.com/clownware/go-performance-starter
NEW=github.com/you/your-app        # your real repo path

# 1. Rename the module
go mod edit -module "$NEW"

# 2. Rewrite every import (Go + templ sources; skips node_modules)
grep -rl --include='*.go' --include='*.templ' "$OLD" . \
  | grep -v node_modules \
  | xargs sed -i '' "s|$OLD|$NEW|g"     # GNU sed (Linux): sed -i without ''

# 3. Update the goimports grouping prefix
sed -i '' "s|$OLD|$NEW|g" .golangci.yml

# 4. Regenerate and verify
task templ:generate
task ci
```

Notes:

- Step 2 rewrites generated files (`*_templ.go`, `internal/database/`) along with their sources — harmless, and step 4's regeneration confirms sources and output agree.
- `.golangci.yml` (`goimports.local-prefixes`) must match the new path or `task fmt` regroups your imports wrongly.

### 2. Deployment identity — `fly.toml`

| Key | Why |
|---|---|
| `app` | Fly app names are globally unique; `go-performance-starter` is taken (by our demo). The deploy workflows read the name from this file — nothing else to change. |
| `primary_region` | `ewr` (New Jersey) suits us; pick yours. |
| `GUEST_MODE_ENABLED`, `GUEST_TTL` | Public-demo tuning (ADR-024/031). For a real product you likely want guest mode off and the TTL default. |

Not deploying to Fly? Delete `fly.toml` — the Docker image is the portable contract ([ADR-025](adr/ADR-025-Deployment-Target.md)); `deploy.yml` will simply never activate without a `FLY_API_TOKEN` secret.

### 3. Environment — `.env`

`cp .env.example .env` and fill in your own `DATABASE_URL` and Supabase keys (the example file contains no template-specific values, just documented placeholders). Reminder from hard experience: URL-encode any special characters in the database password or pgx fails at boot.

### 4. README

The README sells *this template*. Replace the title, the clone URL (`github.com/clownware/go-performance-starter`), and the pitch sections with your product's. Keep the Available Tasks and Agentic Discipline sections if you keep those workflows.

### 5. `versions.json`

The manifest is this template's public consumption contract ([ADR-030](adr/ADR-030-Versions-Manifest-Contract.md)). As an adopter either:

- **Keep it** (it costs nothing and `task versions:check` keeps it honest) and reset `template` to your own `v0.1.0` before your first tag, or
- **Delete it** along with the `versions:check` line in `Taskfile.yml`'s `ci` task and the `stamp` job in `release.yml`, if you're not publishing a template.

## Optional

### Site branding (`internal/view/`)

"Go Performance Starter" renders in `internal/view/layouts/base.templ` (page title, meta description, header, footer) and the logo mark lives in `internal/view/components/brand.templ`. Change them, then `task templ:generate`.

### Brand colors (one file)

The palette is a role-based token system ([design-system.md](design-system.md)): swap the five base color values in `input.css`'s `@theme` block and every component follows — no view-layer edits, and `task ci` catches anything that bypassed the tokens.

### Demo surfaces

`/patterns`, the explainer, and the quiz/flashcards exist to prove the stack ([ADR-024](adr/ADR-024-Demo-Application-Direction.md)); the README's [load-bearing table](../README.md#whats-load-bearing-vs-removable) marks them **replaceable**. The quiz/flashcard handlers are the reference implementation for RLS-scoped CRUD — read them before deleting them.

### Demo operations (only if you want your own public demo)

The deploy/reset workflows are inert until you opt in ([ADR-031](adr/ADR-031-Public-Demo-Operations.md)):

| Where | What |
|---|---|
| Repo secret `FLY_API_TOKEN` | Enables deploy-on-merge and release deploys |
| Repo secret `SUPABASE_DATABASE_URL` | Enables release migrations and the nightly reset |
| Repo variable `DEMO_MODE=1` | Arms the nightly data reset (double-gated; see ADR-031) |

### Local database name

`docker-compose.yml` (`alpine_saas`) and `ci.yml` (`alpine_saas_test`) carry a legacy database name. Purely cosmetic — rename in both plus `.env` if it bothers you.

### Housekeeping

- **`CHANGELOG.md`** — this template's history; start your own.
- **`LICENSE`** — MIT; update the copyright holder.
- **`docs/adr/`** — the ADRs document why the architecture is the way it is; we recommend keeping them and appending your own from ADR-032.
- **`.claude/` + `AGENTS.md`** — the AI constitution is removable if you don't develop with agents (README table); if you keep it, `AGENTS.md` regenerates via `task agents:build`.

## Verify

```bash
task ci           # full quality gate: fmt, lint, race tests, drift checks, budgets, vuln scan
task dev          # app runs at http://localhost:4000
```

If `task ci` passes after the module rename, the personalization is structurally complete.
