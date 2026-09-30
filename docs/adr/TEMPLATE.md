# ADR-NNN: Title

**Date**: YYYY-MM-DD

## Status

Proposed

<!-- Vocabulary: Proposed | Accepted | Accepted (supersedes ADR-NNN) | Accepted (amends ADR-NNN §N) | Superseded by ADR-NNN | Deprecated.
     An ADR that supersedes or amends another gets a dated note appended to the OTHER ADR's Status as well — links go both ways. -->

## Context

What problem or force makes this decision necessary. Constraints that shaped it. Link the ADRs it builds on.

## Decision

The decision, stated so that a reader can tell whether a change complies with it. Name the concrete technology, pattern, file, or convention.

## Consequences

What becomes easier, what becomes harder, what is now forbidden. Costs are consequences too.

## Alternatives Considered

- **Alternative.** Why it was not chosen.

## References

- Related ADRs, issues, PRs, external docs.

## Enforcement
<!-- added YYYY-MM-DD, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: <a verifiable assertion about the repo — an import that must not appear, a file that must exist, a config value>
- **Checks:**
  - TC-1 → <`scripts/adrcheck` check name, a test, or a `task` target> (status: **warn**)
- **Not machine-checkable:** <the intent-level rules a reviewer has to judge, named honestly>
- **Graduation log:** _(empty)_

<!-- New checks start at warn and are promoted to block in checks/enforcement.config.json after 7+ clean days or one real catch (ADR-033 §4); log the promotion here and in CHANGELOG. If nothing about this decision can be checked by a machine, say so under "Not machine-checkable" rather than inventing a check. -->
