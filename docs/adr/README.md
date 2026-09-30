# Architecture Decision Records

Every ADR is a constraint with halt-on-violation force (CLAUDE.md rule 1) and carries an `## Enforcement` section mapping its rules to checks in `scripts/adrcheck` or naming honestly what no machine can check (ADR-033). Existing ADRs are **append-only**: amend with a dated note, promote a check in the graduation log, or supersede with a new ADR. New ADRs start from [`TEMPLATE.md`](TEMPLATE.md); the next number is one above the last row below.

Numbering is not chronological: ADR-000 was written in 2025-11 as the budgets umbrella after ADR-001 … ADR-012 already existed.

| ADR | Title | Status | Date | Relationships |
|---|---|---|---|---|
| [000](ADR-000-Performance-Budgets-and-Quality-Attributes.md) | Performance Budgets and Quality Attributes | Accepted | 2025-11-15 | Budgets classified Enforced / Monitored / Aspirational; ADR-034 moved the asset lists into `internal/performance` |
| [001](ADR-001-Foundation.md) | Foundation (Go, Chi, module path) | Accepted | 2025-04-30 | §3 logging superseded by ADR-026; §5 deployment superseded by ADR-025 |
| [002](ADR-002-Database-Migration-Tool.md) | Database Migration Tool (golang-migrate) | Accepted | 2025-04-30 | |
| [003](ADR-003-SQL-Code-Generation-and-Data-Access.md) | SQL Code Generation and Data Access (sqlc + repositories) | Accepted | 2025-04-30 | Graduation log populated |
| [004](ADR-004-Authorization-Strategy-RLS.md) | Authorization Strategy: Row Level Security | Accepted | 2025-05-01 | Runtime scoping mechanism described in ADR-014's OWASP table and `internal/repository/postgres/scope.go` |
| [005](ADR-005-Handling-Primary-Organization-Selection.md) | Primary Organization Selection (two-step query) | Accepted | 2025-05-01 | |
| [006](ADR-006-Task-Automation-Taskfile.md) | Task Automation: Taskfile | Accepted | 2025-05-01 | |
| [007](ADR-007-Frontend-Stack-Selection.md) | Frontend Stack (HTMX + Alpine + Tailwind) | Accepted | 2025-05-01 | Rendering engine amended by ADR-017 |
| [008](ADR-008-UI-Composition-and-Templates.md) | UI Composition and Templates | **Superseded by ADR-017** | 2025-05-01 | |
| [009](ADR-009-Accessibility-and-Responsive-Design.md) | Accessibility and Responsive Design | Accepted | 2025-05-01 | |
| [010](ADR-010-Testing-and-Code-Quality.md) | Testing, Linting, and Code Quality | Accepted | 2025-05-01 | Philosophy in ADR-023; mutation testing in ADR-032 |
| [011](ADR-011-Documentation-Standards.md) | Documentation Standards | Accepted | 2025-05-01 | Guides moved to `docs/guides/` (amendment 2026-09-30) |
| [012](ADR-012-Routing-and-UI-Patterns.md) | Routing, Handler, and UI Patterns | Accepted | 2025-05-01 | Template engine amended by ADR-017 |
| [013](ADR-013-Error-Handling-and-Observability.md) | Error Handling and Observability | Accepted | 2025-11-15 | §2 logging amended by ADR-026 |
| [014](ADR-014-Security-Patterns-and-Threat-Model.md) | Security Patterns and Threat Model | Accepted | 2025-11-15 | Amended by ADR-025 (§1), ADR-027 (§4), ADR-028 (§6), ADR-034 (429 contract) |
| [015](ADR-015-Configuration-Management-Strategy.md) | Configuration Management (env only) | Accepted | 2025-11-15 | Amended by ADR-025 (secrets on the container host) |
| [016](ADR-016-Caching-Strategy.md) | Caching Strategy | Accepted (largely unimplemented, [#135](https://github.com/clownware/go-performance-starter/issues/135)) | 2025-11-15 | Session bullet amended by ADR-025 §3 |
| [017](ADR-017-Templ-Adoption.md) | Templ Adoption | Accepted (supersedes ADR-008) | 2026-04-04 | Also amends ADR-007 and ADR-012 |
| [018](ADR-018-Layered-AI-Constitution.md) | Layered AI Constitution | Accepted | 2026-06-11 | `CLAUDE.md` + `.claude/*.md` |
| [019](ADR-019-Template-Scope-Boundary.md) | Template Scope Boundary | Accepted | 2026-06-11 | `fly.toml` exception added by ADR-025 §6 |
| [020](ADR-020-Agent-Roles.md) | Agent Roles and Three-Pass Workflow | Accepted | 2026-06-11 | |
| [021](ADR-021-Halt-On-Violation-Quality-Gate.md) | Halt-On-Violation Quality Gate (`task ci`) | Accepted | 2026-06-11 | |
| [022](ADR-022-Cross-Tool-Agents-Spine.md) | Cross-Tool AGENTS.md Spine | Accepted | 2026-06-11 | |
| [023](ADR-023-Testing-Philosophy.md) | Testing Philosophy | Accepted | 2026-06-11 | |
| [024](ADR-024-Demo-Application-Direction.md) | Demo Application Direction | Accepted | 2026-06-11 (accepted 2026-07-05) | Surfaces proven live by ADR-034 |
| [025](ADR-025-Deployment-Target.md) | Deployment Target and Production Topology | Accepted (supersedes ADR-001 §5) | 2026-07-05 | Amends ADR-014 §1, ADR-015, ADR-016, ADR-019 |
| [026](ADR-026-Logging-Standardization.md) | Logging Standardized on `log/slog` | Accepted (supersedes ADR-001 §3) | 2026-07-05 | Amends ADR-013 §2 |
| [027](ADR-027-Trusted-Proxy-Client-IP.md) | Trusted-Proxy Client-IP Resolution | Accepted | 2026-07-07 | Amends ADR-014 §4; refines ADR-025 §2 |
| [028](ADR-028-CSP-Alpine-Compatibility.md) | CSP Compatibility with Alpine.js | Accepted | 2026-07-12 | Amends ADR-014 §6 |
| [029](ADR-029-Role-Based-Design-Tokens.md) | Role-Based Design Tokens | Accepted | 2026-07-12 | Graduation log populated; `docs/design-system.md` |
| [030](ADR-030-Versions-Manifest-Contract.md) | `versions.json` Manifest Contract | Accepted | 2026-07-12 | |
| [031](ADR-031-Public-Demo-Operations.md) | Public Demo Operations | Accepted | 2026-07-12 | |
| [032](ADR-032-Mutation-Testing.md) | Mutation Testing | Accepted | 2026-07-12 | |
| [033](ADR-033-ADR-Enforcement-Architecture.md) | ADR Enforcement Architecture | Accepted | 2026-07-12 | Governs every `## Enforcement` section; `checks/enforcement.config.json` |
| [034](ADR-034-Live-Proof-Surfaces.md) | Live Proof Surfaces | Accepted | 2026-09-09 | Amends ADR-000 (asset lists) and ADR-014 (`Retry-After`) |

Metadata format: ADR-030 onward use `**Date**: YYYY-MM-DD` under the title, then `## Status`; that is the canonical form for ADR-035+ and what `TEMPLATE.md` produces. Older ADRs keep their original layout (five variants exist); `adr011-adr-metadata` accepts any of them and checks only the filename pattern and the presence of a Status marker.
