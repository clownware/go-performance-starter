# Documentation map

| Path | What lives there | When to read it |
|---|---|---|
| [`adr/`](adr/README.md) | Architecture Decision Records ADR-000 … ADR-034, the [template](adr/TEMPLATE.md), and an index. Every rule the repo enforces traces back to one of these. | Before proposing an architectural change (CLAUDE.md rule 1). |
| [`guides/`](guides/README.md) | Topic guides for working in the codebase: data access, view layer, auth, testing, deployment, … Each links the ADRs it explains rather than restating them. | When you need to add something and want the paved path. |
| [`design-system.md`](design-system.md) | The role-based token system (ADR-029): roles, rules, how to restyle. | Before touching a `.templ` file's classes. |
| [`troubleshooting.md`](troubleshooting.md) | Symptoms that have bitten more than once, with cause and fix. | When something fails and you suspect it is not your code. |
| [`personalization-guide.md`](personalization-guide.md) | The forker's checklist: module rename, deploy identity, brand strings, what to strip. | Right after cloning. |
| `updates/` | Two historical design specs ([`demo-design-brief.md`](updates/demo-design-brief.md), [`ux-overhaul-spec.md`](updates/ux-overhaul-spec.md)) linked from ADR-024 and the patterns handler. Superseded where the banners say so; the shipped code is the source of truth. | Only for the history of the demo direction. |

Repo-root documents: [`README.md`](../README.md) (what the template is, quick start), [`CHANGELOG.md`](../CHANGELOG.md) (Keep a Changelog), [`CLAUDE.md`](../CLAUDE.md) + [`.claude/`](../.claude/) (the layered AI constitution, ADR-018), [`AGENTS.md`](../AGENTS.md) (generated from those layers — never hand-edit), [`versions.json`](../versions.json) (the public manifest, ADR-030).

`docs/` is read-only for agents unless a change is explicitly asked for (ADR-019); the one standing exception is appending a graduation-log entry to an ADR when a check is promoted (ADR-033 §4).
