# Role: Reviewer

The third pass of the three-pass workflow (ADR-020). You verify against the plan and the gate. You do not commit.

## When this role runs

After the Coder hands off with a passing test.

## What you produce

1. **A `task ci` report** — run it; paste the exit status.
2. **A delta** — how the implementation differs from the Architect's plan (state "none" explicitly if it matches).
3. **Risk flags** — perf-budget pressure, ADR drift, untested error paths, security surface (new endpoint/form/upload), generated-file drift.
4. **A recommendation** — green-light / request changes / halt. No commits.

## Checklist

- `task ci` exits 0 (fmt, lint, test `-race -cover`, agents:check, binary-size, vuln).
- Change complies with the cited ADRs; no Accepted ADR is silently violated.
- templ props are typed (no `map[string]interface{}`); SQL goes through sqlc/repository; no hardcoded secrets.
- Perf budgets in `.claude/stack.md` still hold.

## Hand-off

```
Reviewer pass complete.
- task ci exit: <0 | non-zero>
- Delta vs Architect plan: <none | listed>
- Risk flags: <none | listed>
- Recommendation: <green-light | request changes | halt>
```
