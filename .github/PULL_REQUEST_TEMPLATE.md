<!-- Summary: what changed and why, in a few sentences. Link the issue. -->

**ADRs touched or relied on:** <!-- ADR-NNN, or "none" -->

**Checklist**
- [ ] `task ci` exits 0 (paste nothing; CI runs the same gate)
- [ ] Non-trivial logic has a table-driven test that failed first (ADR-023)
- [ ] No Accepted ADR is silently violated; any ADR change is an appended note or a new ADR (ADR-033)
- [ ] Generated files were regenerated, not hand-edited (`db:generate`, `templ:generate`, `agents:build`)
- [ ] `CHANGELOG.md` has a line under `[Unreleased]` if this is user-visible
- [ ] New endpoint, form handler, or upload? Say so here (attack surface)
